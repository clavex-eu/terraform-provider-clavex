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

var _ resource.Resource = &AppFamilyResource{}
var _ resource.ResourceWithConfigure = &AppFamilyResource{}
var _ resource.ResourceWithImportState = &AppFamilyResource{}

// AppFamilyResource manages a clavex_app_family. Membership is managed via the
// API separately.
type AppFamilyResource struct{ client *clavex.Client }

func NewAppFamilyResource() resource.Resource { return &AppFamilyResource{} }

type appFamilyModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func (r *AppFamilyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_family"
}

func (r *AppFamilyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an application family (grouped SSO clients). Members are managed separately.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
		},
	}
}

func (r *AppFamilyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AppFamilyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan appFamilyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fam, err := r.client.AppFamilies.Create(ctx, plan.OrgID.ValueString(), clavex.AppFamilyParams{
		Name:        plan.Name.ValueString(),
		Description: strAttrToPtr(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating app family", err.Error())
		return
	}
	flattenAppFamily(fam, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppFamilyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state appFamilyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fam, err := r.client.AppFamilies.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading app family", err.Error())
		return
	}
	flattenAppFamily(fam, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *AppFamilyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan appFamilyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	fam, err := r.client.AppFamilies.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.AppFamilyParams{
		Name:        plan.Name.ValueString(),
		Description: strAttrToPtr(plan.Description),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating app family", err.Error())
		return
	}
	flattenAppFamily(fam, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AppFamilyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state appFamilyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AppFamilies.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting app family", err.Error())
	}
}

func (r *AppFamilyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<family_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenAppFamily(fam *clavex.AppFamily, m *appFamilyModel) {
	m.ID = types.StringValue(fam.ID)
	m.OrgID = types.StringValue(fam.OrgID)
	m.Name = types.StringValue(fam.Name)
	m.Description = ptrToStrAttr(fam.Description)
}
