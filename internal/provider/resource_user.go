package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &UserResource{}
var _ resource.ResourceWithConfigure = &UserResource{}
var _ resource.ResourceWithImportState = &UserResource{}

// UserResource manages a clavex_user.
type UserResource struct{ client *clavex.Client }

func NewUserResource() resource.Resource { return &UserResource{} }

type userModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Email       types.String `tfsdk:"email"`
	FirstName   types.String `tfsdk:"first_name"`
	LastName    types.String `tfsdk:"last_name"`
	Password    types.String `tfsdk:"password"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	MFARequired types.Bool   `tfsdk:"mfa_required"`
}

func (r *UserResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (r *UserResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clavex user account within an organisation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed: true, Description: "UUID of the user.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required: true, Description: "UUID of the organisation.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"email":      schema.StringAttribute{Required: true, Description: "Primary e-mail address."},
			"first_name": schema.StringAttribute{Optional: true, Description: "First name."},
			"last_name":  schema.StringAttribute{Optional: true, Description: "Last name."},
			"password": schema.StringAttribute{
				Optional:  true,
				Sensitive: true,
				Description: "Initial password. Only used on creation; changes are ignored on subsequent applies " +
					"(use SendPasswordReset instead).",
			},
			"is_active":    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the account is enabled."},
			"mfa_required": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Require MFA for this user."},
		},
	}
}

func (r *UserResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*clavex.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *clavex.Client, got %T", req.ProviderData))
		return
	}
	r.client = c
}

func (r *UserResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	isActive := plan.IsActive.ValueBool()
	user, err := r.client.Users.Create(ctx, plan.OrgID.ValueString(), clavex.CreateUserParams{
		Email:     plan.Email.ValueString(),
		FirstName: plan.FirstName.ValueString(),
		LastName:  plan.LastName.ValueString(),
		Password:  plan.Password.ValueString(),
		IsActive:  &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating user", err.Error())
		return
	}
	flattenUser(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	user, err := r.client.Users.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading user", err.Error())
		return
	}
	flattenUser(user, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *UserResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan userModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	email := plan.Email.ValueString()
	isActive := plan.IsActive.ValueBool()
	p := clavex.UpdateUserParams{
		Email:    &email,
		IsActive: &isActive,
	}
	if !plan.FirstName.IsNull() {
		v := plan.FirstName.ValueString()
		p.FirstName = &v
	}
	if !plan.LastName.IsNull() {
		v := plan.LastName.ValueString()
		p.LastName = &v
	}

	user, err := r.client.Users.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), p)
	if err != nil {
		resp.Diagnostics.AddError("Error updating user", err.Error())
		return
	}
	flattenUser(user, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *UserResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state userModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Users.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting user", err.Error())
	}
}

func (r *UserResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import format: "<org_id>/<user_id>"
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenUser(u *clavex.User, m *userModel) {
	m.ID = types.StringValue(u.ID)
	m.OrgID = types.StringValue(u.OrgID)
	m.Email = types.StringValue(u.Email)
	m.FirstName = ptrToStrAttr(u.FirstName)
	m.LastName = ptrToStrAttr(u.LastName)
	m.IsActive = types.BoolValue(u.IsActive)
	m.MFARequired = types.BoolValue(u.MFARequired)
	// password is write-only; never populated from API response
}
