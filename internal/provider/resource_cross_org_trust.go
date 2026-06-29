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

var _ resource.Resource = &CrossOrgTrustResource{}
var _ resource.ResourceWithConfigure = &CrossOrgTrustResource{}
var _ resource.ResourceWithImportState = &CrossOrgTrustResource{}

// CrossOrgTrustResource manages a clavex_cross_org_trust.
//
// Cross-org trusts implement RFC 8693 (OAuth 2.0 Token Exchange) between two
// Clavex organisations. Once established, users authenticated in org A can
// obtain tokens valid in org B without re-authenticating — enabling federated
// multi-tenant architectures (e.g. a partner portal that accepts tokens from a
// customer's own Clavex tenant).
type CrossOrgTrustResource struct{ client *clavex.Client }

func NewCrossOrgTrustResource() resource.Resource { return &CrossOrgTrustResource{} }

type crossOrgTrustModel struct {
	ID             types.String `tfsdk:"id"`
	OrgID          types.String `tfsdk:"org_id"`
	TrustedOrgSlug types.String `tfsdk:"trusted_org_slug"`
	TrustedOrgID   types.String `tfsdk:"trusted_org_id"`
	AllowedScopes  types.List   `tfsdk:"allowed_scopes"`
	CreatedAt      types.String `tfsdk:"created_at"`
}

func (r *CrossOrgTrustResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cross_org_trust"
}

func (r *CrossOrgTrustResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an RFC 8693 token-exchange trust relationship between two Clavex " +
			"organisations. When established, users authenticated in org_id can exchange " +
			"their tokens for ones valid in the trusted organisation without re-authenticating. " +
			"Trust relationships are immutable — any change forces a revoke + recreate.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required: true,
				Description: "ID of the organisation that initiates the trust (the token issuer). " +
					"Changing this value forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"trusted_org_slug": schema.StringAttribute{
				Required: true,
				Description: "Slug of the organisation whose tokens will be accepted (the relying party). " +
					"Changing this value forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"trusted_org_id": schema.StringAttribute{
				Computed:      true,
				Description:   "UUID of the trusted organisation (resolved by the server).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"allowed_scopes": schema.ListAttribute{
				Optional:    true,
				ElementType: types.StringType,
				Description: "Limit which OAuth2 scopes can be exchanged. Empty = all scopes. " +
					"Changing this value forces a new resource.",
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *CrossOrgTrustResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CrossOrgTrustResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan crossOrgTrustModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	trust, err := r.client.CrossOrgTrust.Create(ctx, plan.OrgID.ValueString(), clavex.CreateCrossOrgTrustParams{
		TrustedOrgSlug: plan.TrustedOrgSlug.ValueString(),
		AllowedScopes:  listToStrings(ctx, plan.AllowedScopes),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating cross-org trust", err.Error())
		return
	}
	flattenCrossOrgTrust(trust, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *CrossOrgTrustResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state crossOrgTrustModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	trusts, err := r.client.CrossOrgTrust.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing cross-org trusts", err.Error())
		return
	}
	for _, trust := range trusts {
		if trust.ID == state.ID.ValueString() {
			flattenCrossOrgTrust(&trust, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

// Update is never called because all non-computed attributes carry RequiresReplace.
func (r *CrossOrgTrustResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"Cross-org trust update not supported",
		"Trust relationships are immutable. Changing attributes forces a new resource (revoke + create).",
	)
}

func (r *CrossOrgTrustResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state crossOrgTrustModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.CrossOrgTrust.Revoke(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error revoking cross-org trust", err.Error())
	}
}

func (r *CrossOrgTrustResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<trust_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenCrossOrgTrust(t *clavex.CrossOrgTrust, m *crossOrgTrustModel) {
	m.ID = types.StringValue(t.ID)
	m.OrgID = types.StringValue(t.OrgID)
	m.TrustedOrgSlug = types.StringValue(t.TrustedOrgSlug)
	m.TrustedOrgID = types.StringValue(t.TrustedOrgID)
	m.AllowedScopes = stringsToList(t.AllowedScopes)
	m.CreatedAt = types.StringValue(t.CreatedAt.Format("2006-01-02T15:04:05Z"))
}
