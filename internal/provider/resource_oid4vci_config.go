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

var _ resource.Resource = &OID4VCIConfigResource{}
var _ resource.ResourceWithConfigure = &OID4VCIConfigResource{}
var _ resource.ResourceWithImportState = &OID4VCIConfigResource{}

// OID4VCIConfigResource manages a clavex_oid4vci_config (credential type config).
type OID4VCIConfigResource struct{ client *clavex.Client }

func NewOID4VCIConfigResource() resource.Resource { return &OID4VCIConfigResource{} }

type oid4vciConfigModel struct {
	ID            types.String `tfsdk:"id"`
	OrgID         types.String `tfsdk:"org_id"`
	VCT           types.String `tfsdk:"vct"`
	DisplayName   types.String `tfsdk:"display_name"`
	Description   types.String `tfsdk:"description"`
	TTLSeconds    types.Int64  `tfsdk:"ttl_seconds"`
	Category      types.String `tfsdk:"category"`
	ClaimsMapping types.Map    `tfsdk:"claims_mapping"`
	IsActive      types.Bool   `tfsdk:"is_active"`
}

func (r *OID4VCIConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oid4vci_config"
}

func (r *OID4VCIConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replaceStr := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages an OID4VCI credential-type configuration. Core attributes are immutable; only claims_mapping can be updated in place.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":         schema.StringAttribute{Required: true, PlanModifiers: replaceStr},
			"vct":            schema.StringAttribute{Required: true, PlanModifiers: replaceStr, Description: "Verifiable Credential Type URI (immutable)."},
			"display_name":   schema.StringAttribute{Required: true, PlanModifiers: replaceStr},
			"description":    schema.StringAttribute{Optional: true, PlanModifiers: replaceStr},
			"ttl_seconds":    schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{}},
			"category":       schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replaceStr, Description: "identity | training | qualification | badge."},
			"claims_mapping": schema.MapAttribute{Optional: true, ElementType: types.StringType, Description: "Maps credential field names to source paths."},
			"is_active":      schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *OID4VCIConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func claimsMapToIface(ctx context.Context, m types.Map) map[string]interface{} {
	sm := mapToStrMap(ctx, m)
	if sm == nil {
		return nil
	}
	out := make(map[string]interface{}, len(sm))
	for k, v := range sm {
		out[k] = v
	}
	return out
}

func (r *OID4VCIConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oid4vciConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.OID4VCI.CreateConfig(ctx, plan.OrgID.ValueString(), clavex.CreateVCIConfigParams{
		VCT:           plan.VCT.ValueString(),
		DisplayName:   plan.DisplayName.ValueString(),
		Description:   strAttrToPtr(plan.Description),
		TTLSeconds:    int(plan.TTLSeconds.ValueInt64()),
		Category:      plan.Category.ValueString(),
		ClaimsMapping: claimsMapToIface(ctx, plan.ClaimsMapping),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating OID4VCI config", err.Error())
		return
	}
	flattenVCIConfig(cfg, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OID4VCIConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oid4vciConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	configs, err := r.client.OID4VCI.ListConfigs(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing OID4VCI configs", err.Error())
		return
	}
	for i := range configs {
		if configs[i].ID == state.ID.ValueString() {
			flattenVCIConfig(&configs[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *OID4VCIConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oid4vciConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Only claims_mapping is mutable in place via PatchConfig.
	cfg, err := r.client.OID4VCI.PatchConfig(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.PatchVCIConfigParams{
		ClaimsMapping: claimsMapToIface(ctx, plan.ClaimsMapping),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating OID4VCI config", err.Error())
		return
	}
	flattenVCIConfig(cfg, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OID4VCIConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oid4vciConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.OID4VCI.DeleteConfig(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting OID4VCI config", err.Error())
	}
}

func (r *OID4VCIConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<config_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenVCIConfig(cfg *clavex.VCICredentialConfig, m *oid4vciConfigModel) {
	m.ID = types.StringValue(cfg.ID)
	m.OrgID = types.StringValue(cfg.OrgID)
	m.VCT = types.StringValue(cfg.VCT)
	m.DisplayName = types.StringValue(cfg.DisplayName)
	m.Description = ptrToStrAttr(cfg.Description)
	m.TTLSeconds = types.Int64Value(int64(cfg.TTLSeconds))
	m.Category = types.StringValue(cfg.Category)
	m.IsActive = types.BoolValue(cfg.IsActive)
	cm := make(map[string]string, len(cfg.ClaimsMapping))
	for k, v := range cfg.ClaimsMapping {
		cm[k] = fmt.Sprintf("%v", v)
	}
	if len(cm) > 0 {
		m.ClaimsMapping = strMapToMap(cm)
	}
}
