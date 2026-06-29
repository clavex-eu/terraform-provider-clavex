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

var _ resource.Resource = &WsfedRpResource{}
var _ resource.ResourceWithConfigure = &WsfedRpResource{}
var _ resource.ResourceWithImportState = &WsfedRpResource{}

// WsfedRpResource manages a clavex_wsfed_rp (WS-Federation relying party).
type WsfedRpResource struct{ client *clavex.Client }

func NewWsfedRpResource() resource.Resource { return &WsfedRpResource{} }

type wsfedRpModel struct {
	ID                   types.String `tfsdk:"id"`
	OrgID                types.String `tfsdk:"org_id"`
	Name                 types.String `tfsdk:"name"`
	Realm                types.String `tfsdk:"realm"`
	WreplyURIs           types.List   `tfsdk:"wreply_uris"`
	TokenLifetimeSeconds types.Int64  `tfsdk:"token_lifetime_seconds"`
	ClaimsMapping        types.Map    `tfsdk:"claims_mapping"`
	IsActive             types.Bool   `tfsdk:"is_active"`
}

func (r *WsfedRpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_wsfed_rp"
}

func (r *WsfedRpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a WS-Federation relying party.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":                 schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":                   schema.StringAttribute{Required: true},
			"realm":                  schema.StringAttribute{Required: true, Description: "WS-Fed realm (wtrealm)."},
			"wreply_uris":            schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"token_lifetime_seconds": schema.Int64Attribute{Optional: true, Computed: true},
			"claims_mapping":         schema.MapAttribute{Optional: true, ElementType: types.StringType},
			"is_active":              schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *WsfedRpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WsfedRpResource) params(ctx context.Context, plan *wsfedRpModel) clavex.WsfedRpParams {
	return clavex.WsfedRpParams{
		Name:                 plan.Name.ValueString(),
		Realm:                plan.Realm.ValueString(),
		WreplyURIs:           listToStrings(ctx, plan.WreplyURIs),
		TokenLifetimeSeconds: int(plan.TokenLifetimeSeconds.ValueInt64()),
		ClaimsMapping:        mapToStrMap(ctx, plan.ClaimsMapping),
	}
}

func (r *WsfedRpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan wsfedRpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rp, err := r.client.WsfedRPs.Create(ctx, plan.OrgID.ValueString(), r.params(ctx, &plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating WS-Fed RP", err.Error())
		return
	}
	flattenWsfedRp(rp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WsfedRpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state wsfedRpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rp, err := r.client.WsfedRPs.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading WS-Fed RP", err.Error())
		return
	}
	flattenWsfedRp(rp, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *WsfedRpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan wsfedRpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rp, err := r.client.WsfedRPs.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), r.params(ctx, &plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating WS-Fed RP", err.Error())
		return
	}
	flattenWsfedRp(rp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WsfedRpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state wsfedRpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.WsfedRPs.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting WS-Fed RP", err.Error())
	}
}

func (r *WsfedRpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<rp_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenWsfedRp(rp *clavex.WsfedRP, m *wsfedRpModel) {
	m.ID = types.StringValue(rp.ID)
	m.OrgID = types.StringValue(rp.OrgID)
	m.Name = types.StringValue(rp.Name)
	m.Realm = types.StringValue(rp.Realm)
	m.WreplyURIs = stringsToList(rp.WreplyURIs)
	m.TokenLifetimeSeconds = types.Int64Value(int64(rp.TokenLifetimeSeconds))
	m.IsActive = types.BoolValue(rp.IsActive)
	if len(rp.ClaimsMapping) > 0 {
		m.ClaimsMapping = strMapToMap(rp.ClaimsMapping)
	}
}
