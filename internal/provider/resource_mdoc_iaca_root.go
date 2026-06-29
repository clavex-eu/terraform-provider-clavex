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

var _ resource.Resource = &MdocIACARootResource{}
var _ resource.ResourceWithConfigure = &MdocIACARootResource{}
var _ resource.ResourceWithImportState = &MdocIACARootResource{}

// MdocIACARootResource manages a clavex_mdoc_iaca_root trust anchor.
// Immutable — changes force replacement.
type MdocIACARootResource struct{ client *clavex.Client }

func NewMdocIACARootResource() resource.Resource { return &MdocIACARootResource{} }

type mdocIACARootModel struct {
	ID                types.String `tfsdk:"id"`
	OrgID             types.String `tfsdk:"org_id"`
	Label             types.String `tfsdk:"label"`
	PEM               types.String `tfsdk:"pem"`
	DocTypes          types.List   `tfsdk:"doc_types"`
	SubjectDN         types.String `tfsdk:"subject_dn"`
	SHA256Fingerprint types.String `tfsdk:"sha256_fingerprint"`
	IsActive          types.Bool   `tfsdk:"is_active"`
}

func (r *MdocIACARootResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mdoc_iaca_root"
}

func (r *MdocIACARootResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages a trusted IACA root certificate for mdoc verification. Immutable — changes force replacement.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id": schema.StringAttribute{Required: true, PlanModifiers: replace},
			"label":  schema.StringAttribute{Required: true, PlanModifiers: replace},
			"pem":    schema.StringAttribute{Required: true, PlanModifiers: replace, Description: "PEM-encoded IACA root certificate."},
			"doc_types": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Doc types this root is trusted for.",
			},
			"subject_dn":         schema.StringAttribute{Computed: true},
			"sha256_fingerprint": schema.StringAttribute{Computed: true},
			"is_active":          schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *MdocIACARootResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *MdocIACARootResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mdocIACARootModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	root, err := r.client.Mdoc.CreateIACARoot(ctx, plan.OrgID.ValueString(), clavex.CreateIACARootParams{
		Label:    plan.Label.ValueString(),
		PEM:      plan.PEM.ValueString(),
		DocTypes: listToStrings(ctx, plan.DocTypes),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating IACA root", err.Error())
		return
	}
	flattenIACARoot(root, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MdocIACARootResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mdocIACARootModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	roots, err := r.client.Mdoc.ListIACARoots(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing IACA roots", err.Error())
		return
	}
	for i := range roots {
		if roots[i].ID == state.ID.ValueString() {
			flattenIACARoot(&roots[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *MdocIACARootResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mdocIACARootModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *MdocIACARootResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mdocIACARootModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Mdoc.DeleteIACARoot(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting IACA root", err.Error())
	}
}

func (r *MdocIACARootResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<root_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenIACARoot(root *clavex.IACARoot, m *mdocIACARootModel) {
	m.ID = types.StringValue(root.ID)
	m.OrgID = types.StringValue(root.OrgID)
	m.Label = types.StringValue(root.Label)
	m.PEM = types.StringValue(root.PEM)
	m.DocTypes = stringsToList(root.DocTypes)
	m.SubjectDN = types.StringValue(root.SubjectDN)
	m.SHA256Fingerprint = types.StringValue(root.SHA256Fingerprint)
	m.IsActive = types.BoolValue(root.IsActive)
}
