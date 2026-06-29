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

var _ resource.Resource = &FederationSubordinateResource{}
var _ resource.ResourceWithConfigure = &FederationSubordinateResource{}
var _ resource.ResourceWithImportState = &FederationSubordinateResource{}

// FederationSubordinateResource manages a clavex_federation_subordinate.
type FederationSubordinateResource struct{ client *clavex.Client }

func NewFederationSubordinateResource() resource.Resource {
	return &FederationSubordinateResource{}
}

type federationSubordinateModel struct {
	ID                       types.String `tfsdk:"id"`
	OrgID                    types.String `tfsdk:"org_id"`
	EntityID                 types.String `tfsdk:"entity_id"`
	Name                     types.String `tfsdk:"name"`
	EntityTypes              types.List   `tfsdk:"entity_types"`
	JWKSURI                  types.String `tfsdk:"jwks_uri"`
	TrustMarkIDs             types.List   `tfsdk:"trust_mark_ids"`
	Status                   types.String `tfsdk:"status"`
	StatementLifetimeSeconds types.Int64  `tfsdk:"statement_lifetime_seconds"`
}

func (r *FederationSubordinateResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_federation_subordinate"
}

func (r *FederationSubordinateResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OpenID Federation subordinate entity registered under a Clavex trust anchor.",
		Attributes: map[string]schema.Attribute{
			"id":     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"entity_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "Subordinate entity identifier URL (immutable natural key).",
			},
			"name":                       schema.StringAttribute{Required: true},
			"entity_types":               schema.ListAttribute{Required: true, ElementType: types.StringType, Description: "e.g. [\"openid_relying_party\"]."},
			"jwks_uri":                   schema.StringAttribute{Optional: true},
			"trust_mark_ids":             schema.ListAttribute{Optional: true, ElementType: types.StringType},
			"status":                     schema.StringAttribute{Optional: true, Computed: true, Description: "active | suspended | revoked."},
			"statement_lifetime_seconds": schema.Int64Attribute{Optional: true, Computed: true},
		},
	}
}

func (r *FederationSubordinateResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *FederationSubordinateResource) params(ctx context.Context, plan *federationSubordinateModel) clavex.RegisterSubordinateParams {
	return clavex.RegisterSubordinateParams{
		EntityID:                 plan.EntityID.ValueString(),
		Name:                     plan.Name.ValueString(),
		EntityTypes:              listToStrings(ctx, plan.EntityTypes),
		JWKSURI:                  plan.JWKSURI.ValueString(),
		TrustMarkIDs:             listToStrings(ctx, plan.TrustMarkIDs),
		Status:                   plan.Status.ValueString(),
		StatementLifetimeSeconds: int(plan.StatementLifetimeSeconds.ValueInt64()),
	}
}

func (r *FederationSubordinateResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan federationSubordinateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sub, err := r.client.Federation.RegisterSubordinate(ctx, plan.OrgID.ValueString(), r.params(ctx, &plan))
	if err != nil {
		resp.Diagnostics.AddError("Error registering subordinate", err.Error())
		return
	}
	flattenSubordinate(sub, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FederationSubordinateResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state federationSubordinateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	subs, err := r.client.Federation.ListSubordinates(ctx, state.OrgID.ValueString(), "all")
	if err != nil {
		resp.Diagnostics.AddError("Error listing subordinates", err.Error())
		return
	}
	for i := range subs {
		if subs[i].ID == state.ID.ValueString() {
			flattenSubordinate(&subs[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *FederationSubordinateResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan federationSubordinateModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sub, err := r.client.Federation.UpdateSubordinate(ctx, plan.OrgID.ValueString(), plan.EntityID.ValueString(), r.params(ctx, &plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating subordinate", err.Error())
		return
	}
	flattenSubordinate(sub, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *FederationSubordinateResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state federationSubordinateModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Federation.RevokeSubordinate(ctx, state.OrgID.ValueString(), state.EntityID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error revoking subordinate", err.Error())
	}
}

func (r *FederationSubordinateResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<subordinate_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenSubordinate(sub *clavex.FederationSubordinate, m *federationSubordinateModel) {
	m.ID = types.StringValue(sub.ID)
	m.EntityID = types.StringValue(sub.EntityID)
	m.Name = types.StringValue(sub.Name)
	m.EntityTypes = stringsToList(sub.EntityTypes)
	m.Status = types.StringValue(sub.Status)
	m.StatementLifetimeSeconds = types.Int64Value(int64(sub.StatementLifetimeSeconds))
	if len(sub.TrustMarkIDs) > 0 {
		m.TrustMarkIDs = stringsToList(sub.TrustMarkIDs)
	}
}
