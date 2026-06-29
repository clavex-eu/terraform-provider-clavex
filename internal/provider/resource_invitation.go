package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &InvitationResource{}
var _ resource.ResourceWithConfigure = &InvitationResource{}
var _ resource.ResourceWithImportState = &InvitationResource{}

// InvitationResource manages a pending user invitation.
type InvitationResource struct{ client *clavex.Client }

func NewInvitationResource() resource.Resource { return &InvitationResource{} }

type invitationModel struct {
	ID      types.String `tfsdk:"id"`
	OrgID   types.String `tfsdk:"org_id"`
	Email   types.String `tfsdk:"email"`
	RoleIDs types.List   `tfsdk:"role_ids"`
}

func (r *InvitationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_invitation"
}

func (r *InvitationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNew := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Sends a user invitation email and tracks the pending invite. " +
			"Deleting this resource revokes the invitation. All attributes are immutable.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id": schema.StringAttribute{Required: true, PlanModifiers: forceNew},
			"email":  schema.StringAttribute{Required: true, PlanModifiers: forceNew},
			"role_ids": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Role IDs to pre-assign to the user upon accepting the invite.",
			},
		},
	}
}

func (r *InvitationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *InvitationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan invitationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	inv, err := r.client.Invitations.Create(ctx, plan.OrgID.ValueString(), clavex.CreateInvitationParams{
		Email:   plan.Email.ValueString(),
		RoleIDs: listToStrings(ctx, plan.RoleIDs),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating invitation", err.Error())
		return
	}
	plan.ID = types.StringValue(inv.ID)
	plan.Email = types.StringValue(inv.Email)
	plan.RoleIDs = stringsToList(inv.RoleIDs)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *InvitationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state invitationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	invites, err := r.client.Invitations.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing invitations", err.Error())
		return
	}
	for _, inv := range invites {
		if inv.ID == state.ID.ValueString() {
			state.Email = types.StringValue(inv.Email)
			state.RoleIDs = stringsToList(inv.RoleIDs)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	// Invitation accepted or expired — remove from state.
	resp.State.RemoveResource(ctx)
}

func (r *InvitationResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes are ForceNew.
}

func (r *InvitationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state invitationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Invitations.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error revoking invitation", err.Error())
	}
}

func (r *InvitationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<invitation_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}
