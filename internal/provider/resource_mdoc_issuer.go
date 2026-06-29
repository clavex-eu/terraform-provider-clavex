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

var _ resource.Resource = &MdocIssuerResource{}
var _ resource.ResourceWithConfigure = &MdocIssuerResource{}
var _ resource.ResourceWithImportState = &MdocIssuerResource{}

// MdocIssuerResource manages a clavex_mdoc_issuer. The issuer is immutable;
// any change forces replacement.
type MdocIssuerResource struct{ client *clavex.Client }

func NewMdocIssuerResource() resource.Resource { return &MdocIssuerResource{} }

type mdocIssuerModel struct {
	ID                 types.String `tfsdk:"id"`
	OrgID              types.String `tfsdk:"org_id"`
	DisplayName        types.String `tfsdk:"display_name"`
	DocType            types.String `tfsdk:"doc_type"`
	DSPrivateKeyPEM    types.String `tfsdk:"ds_private_key_pem"`
	DSCertificatePEM   types.String `tfsdk:"ds_certificate_pem"`
	IACACertificatePEM types.String `tfsdk:"iaca_certificate_pem"`
	ValidityHours      types.Int64  `tfsdk:"validity_hours"`
	IsActive           types.Bool   `tfsdk:"is_active"`
}

func (r *MdocIssuerResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mdoc_issuer"
}

func (r *MdocIssuerResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages an ISO 18013-5 mdoc issuer (Document Signer). Immutable — changes force replacement.",
		Attributes: map[string]schema.Attribute{
			"id":                   schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":               schema.StringAttribute{Required: true, PlanModifiers: replace},
			"display_name":         schema.StringAttribute{Required: true, PlanModifiers: replace},
			"doc_type":             schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "e.g. \"org.iso.18013.5.1.mDL\"."},
			"ds_private_key_pem":   schema.StringAttribute{Required: true, Sensitive: true, PlanModifiers: replace, Description: "PKCS#8 PEM DS private key."},
			"ds_certificate_pem":   schema.StringAttribute{Required: true, PlanModifiers: replace},
			"iaca_certificate_pem": schema.StringAttribute{Optional: true, PlanModifiers: replace},
			"validity_hours":       schema.Int64Attribute{Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{}},
			"is_active":            schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *MdocIssuerResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MdocIssuerResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mdocIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	iss, err := r.client.Mdoc.CreateIssuer(ctx, plan.OrgID.ValueString(), clavex.CreateIssuerParams{
		DisplayName:        plan.DisplayName.ValueString(),
		DocType:            plan.DocType.ValueString(),
		DSPrivateKeyPEM:    plan.DSPrivateKeyPEM.ValueString(),
		DSCertificatePEM:   plan.DSCertificatePEM.ValueString(),
		IACACertificatePEM: strAttrToPtr(plan.IACACertificatePEM),
		ValidityHours:      int(plan.ValidityHours.ValueInt64()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating mdoc issuer", err.Error())
		return
	}
	flattenMdocIssuer(iss, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MdocIssuerResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mdocIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	issuers, err := r.client.Mdoc.ListIssuers(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing mdoc issuers", err.Error())
		return
	}
	for i := range issuers {
		if issuers[i].ID == state.ID.ValueString() {
			flattenMdocIssuer(&issuers[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *MdocIssuerResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// All mutable attributes force replacement; no in-place update exists.
	var plan mdocIssuerModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MdocIssuerResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mdocIssuerModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Mdoc.DeleteIssuer(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting mdoc issuer", err.Error())
	}
}

func (r *MdocIssuerResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<issuer_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenMdocIssuer(iss *clavex.MdocIssuer, m *mdocIssuerModel) {
	m.ID = types.StringValue(iss.ID)
	m.OrgID = types.StringValue(iss.OrgID)
	m.DisplayName = types.StringValue(iss.DisplayName)
	m.DocType = types.StringValue(iss.DocType)
	m.DSCertificatePEM = types.StringValue(iss.DSCertificatePEM)
	m.IACACertificatePEM = ptrToStrAttr(iss.IACACertificatePEM)
	m.ValidityHours = types.Int64Value(int64(iss.ValidityHours))
	m.IsActive = types.BoolValue(iss.IsActive)
	// ds_private_key_pem is write-only; never returned by the API.
}
