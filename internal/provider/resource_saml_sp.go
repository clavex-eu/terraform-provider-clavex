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

var _ resource.Resource = &SamlSpResource{}
var _ resource.ResourceWithConfigure = &SamlSpResource{}
var _ resource.ResourceWithImportState = &SamlSpResource{}

// SamlSpResource manages a clavex_saml_sp. SAML SPs are immutable; changes
// force replacement.
type SamlSpResource struct{ client *clavex.Client }

func NewSamlSpResource() resource.Resource { return &SamlSpResource{} }

type samlSpModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	EntityID     types.String `tfsdk:"entity_id"`
	Name         types.String `tfsdk:"name"`
	ACSURL       types.String `tfsdk:"acs_url"`
	SLOURL       types.String `tfsdk:"slo_url"`
	MetadataXML  types.String `tfsdk:"metadata_xml"`
	NameIDFormat types.String `tfsdk:"name_id_format"`
	IsActive     types.Bool   `tfsdk:"is_active"`
}

func (r *SamlSpResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_saml_sp"
}

func (r *SamlSpResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a SAML service provider (Clavex as IdP). Immutable — changes force replacement.",
		Attributes: map[string]schema.Attribute{
			"id":             schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":         schema.StringAttribute{Required: true, PlanModifiers: replace},
			"entity_id":      schema.StringAttribute{Required: true, PlanModifiers: replace},
			"name":           schema.StringAttribute{Required: true, PlanModifiers: replace},
			"acs_url":        schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "Assertion Consumer Service URL."},
			"slo_url":        schema.StringAttribute{Optional: true, PlanModifiers: replace},
			"metadata_xml":   schema.StringAttribute{Optional: true, PlanModifiers: replace, Description: "SP metadata XML (write-only)."},
			"name_id_format": schema.StringAttribute{Optional: true, Computed: true, PlanModifiers: replace},
			"is_active":      schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *SamlSpResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SamlSpResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan samlSpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sp, err := r.client.SamlSPs.Create(ctx, plan.OrgID.ValueString(), clavex.CreateSamlSpParams{
		EntityID:     plan.EntityID.ValueString(),
		Name:         plan.Name.ValueString(),
		ACSURL:       plan.ACSURL.ValueString(),
		SLOURL:       strAttrToPtr(plan.SLOURL),
		MetadataXML:  strAttrToPtr(plan.MetadataXML),
		NameIDFormat: plan.NameIDFormat.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating SAML SP", err.Error())
		return
	}
	flattenSamlSp(sp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SamlSpResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state samlSpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sps, err := r.client.SamlSPs.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing SAML SPs", err.Error())
		return
	}
	for i := range sps {
		if sps[i].ID == state.ID.ValueString() {
			flattenSamlSp(&sps[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *SamlSpResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan samlSpModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SamlSpResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state samlSpModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.SamlSPs.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting SAML SP", err.Error())
	}
}

func (r *SamlSpResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<sp_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenSamlSp(sp *clavex.SamlSP, m *samlSpModel) {
	m.ID = types.StringValue(sp.ID)
	m.OrgID = types.StringValue(sp.OrgID)
	m.EntityID = types.StringValue(sp.EntityID)
	m.Name = types.StringValue(sp.Name)
	m.ACSURL = types.StringValue(sp.ACSURL)
	m.SLOURL = ptrToStrAttr(sp.SLOURL)
	m.NameIDFormat = types.StringValue(sp.NameIDFormat)
	m.IsActive = types.BoolValue(sp.IsActive)
	// metadata_xml is write-only; never returned.
}
