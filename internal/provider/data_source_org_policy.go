package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &OrgPolicyDataSource{}
var _ datasource.DataSourceWithConfigure = &OrgPolicyDataSource{}

// OrgPolicyDataSource reads an existing org auth-policy rule by name.
// Use clavex_org_policy to manage the rule lifecycle.
type OrgPolicyDataSource struct{ client *clavex.Client }

func NewOrgPolicyDataSource() datasource.DataSource { return &OrgPolicyDataSource{} }

type orgPolicyDataModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Name         types.String `tfsdk:"name"`
	Priority     types.Int64  `tfsdk:"priority"`
	Action       types.String `tfsdk:"action"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	IPCIDRs      types.List   `tfsdk:"ip_cidrs"`
	Countries    types.List   `tfsdk:"countries"`
	NotCountries types.List   `tfsdk:"not_countries"`
	ClientIDs    types.List   `tfsdk:"client_ids"`
}

func (d *OrgPolicyDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_policy"
}

func (d *OrgPolicyDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Read-only lookup of an existing Clavex auth-flow policy rule by name. " +
			"Use clavex_org_policy to manage the rule lifecycle.",
		Attributes: map[string]dsschema.Attribute{
			"id":            dsschema.StringAttribute{Computed: true},
			"org_id":        dsschema.StringAttribute{Required: true, Description: "UUID of the owning organisation."},
			"name":          dsschema.StringAttribute{Required: true, Description: "Exact rule name to look up."},
			"priority":      dsschema.Int64Attribute{Computed: true, Description: "Evaluation order (lowest first)."},
			"action":        dsschema.StringAttribute{Computed: true, Description: `Outcome: "allow", "deny", "require_mfa", or "step_up".`},
			"enabled":       dsschema.BoolAttribute{Computed: true},
			"ip_cidrs":      dsschema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "IP CIDR allowlist condition."},
			"countries":     dsschema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Country allowlist (ISO 3166-1 alpha-2)."},
			"not_countries": dsschema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "Country denylist (ISO 3166-1 alpha-2)."},
			"client_ids":    dsschema.ListAttribute{Computed: true, ElementType: types.StringType, Description: "OIDC client_id scope restriction."},
		},
	}
}

func (d *OrgPolicyDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*clavex.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *clavex.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *OrgPolicyDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state orgPolicyDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	rules, err := d.client.Policies.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing policy rules", err.Error())
		return
	}

	for _, r := range rules {
		if r.Name != state.Name.ValueString() {
			continue
		}
		state.ID = types.StringValue(r.ID)
		state.Priority = types.Int64Value(int64(r.Priority))
		state.Action = types.StringValue(r.Action)
		state.Enabled = types.BoolValue(r.Enabled)

		ipCIDRs, diags := types.ListValueFrom(ctx, types.StringType, r.Conditions.IPCIDRs)
		resp.Diagnostics.Append(diags...)
		state.IPCIDRs = ipCIDRs

		countries, diags := types.ListValueFrom(ctx, types.StringType, r.Conditions.Countries)
		resp.Diagnostics.Append(diags...)
		state.Countries = countries

		notCountries, diags := types.ListValueFrom(ctx, types.StringType, r.Conditions.NotCountries)
		resp.Diagnostics.Append(diags...)
		state.NotCountries = notCountries

		clientIDs, diags := types.ListValueFrom(ctx, types.StringType, r.Conditions.ClientIDs)
		resp.Diagnostics.Append(diags...)
		state.ClientIDs = clientIDs

		if resp.Diagnostics.HasError() {
			return
		}
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	resp.Diagnostics.AddError("Policy rule not found", fmt.Sprintf("No rule named %q in org %q.", state.Name.ValueString(), state.OrgID.ValueString()))
}
