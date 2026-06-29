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

var _ resource.Resource = &LoginFlowResource{}
var _ resource.ResourceWithConfigure = &LoginFlowResource{}
var _ resource.ResourceWithImportState = &LoginFlowResource{}

// LoginFlowResource manages a clavex_login_flow (flow metadata; steps are
// managed out of band).
type LoginFlowResource struct{ client *clavex.Client }

func NewLoginFlowResource() resource.Resource { return &LoginFlowResource{} }

type loginFlowModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
	IsActive    types.Bool   `tfsdk:"is_active"`
}

func (r *LoginFlowResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_login_flow"
}

func (r *LoginFlowResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clavex login flow. Steps are managed separately via the API.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"is_default":  schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"is_active":   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		},
	}
}

func (r *LoginFlowResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LoginFlowResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan loginFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lf, err := r.client.LoginFlows.Create(ctx, plan.OrgID.ValueString(), clavex.CreateLoginFlowParams{
		Name:        plan.Name.ValueString(),
		Description: strAttrToPtr(plan.Description),
		IsDefault:   plan.IsDefault.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating login flow", err.Error())
		return
	}
	flattenLoginFlow(lf, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoginFlowResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state loginFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lf, err := r.client.LoginFlows.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading login flow", err.Error())
		return
	}
	flattenLoginFlow(lf, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LoginFlowResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan loginFlowModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	lf, err := r.client.LoginFlows.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.UpdateLoginFlowParams{
		Name:        plan.Name.ValueString(),
		Description: strAttrToPtr(plan.Description),
		IsDefault:   plan.IsDefault.ValueBool(),
		IsActive:    plan.IsActive.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating login flow", err.Error())
		return
	}
	flattenLoginFlow(lf, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LoginFlowResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state loginFlowModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.LoginFlows.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting login flow", err.Error())
	}
}

func (r *LoginFlowResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<flow_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenLoginFlow(lf *clavex.LoginFlow, m *loginFlowModel) {
	m.ID = types.StringValue(lf.ID)
	m.OrgID = types.StringValue(lf.OrgID)
	m.Name = types.StringValue(lf.Name)
	m.Description = ptrToStrAttr(lf.Description)
	m.IsDefault = types.BoolValue(lf.IsDefault)
	m.IsActive = types.BoolValue(lf.IsActive)
}
