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

var _ resource.Resource = &AuthZenPolicyResource{}
var _ resource.ResourceWithConfigure = &AuthZenPolicyResource{}
var _ resource.ResourceWithImportState = &AuthZenPolicyResource{}

// AuthZenPolicyResource manages a clavex_authzen_policy.
//
// AuthZen policies are fine-grained authorization rules evaluated by the
// Clavex PDP (POST /access/v1/evaluation) when a resource server calls the
// AuthZen API. They model application-level RBAC ("can user X perform action Y
// on resource type Z?") using the same underlying policy engine as org-level
// auth-flow rules, but scoped to the AuthZen evaluation context.
//
// The action_names attribute maps to the AuthZen action.name field and is
// matched against the policy engine's client_id signal — this is how the PDP
// routes evaluation requests to the correct rules.
type AuthZenPolicyResource struct{ client *clavex.Client }

func NewAuthZenPolicyResource() resource.Resource { return &AuthZenPolicyResource{} }

type authzenPolicyModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Name         types.String `tfsdk:"name"`
	Priority     types.Int64  `tfsdk:"priority"`
	Decision     types.String `tfsdk:"decision"`
	Enabled      types.Bool   `tfsdk:"enabled"`
	ActionNames  types.List   `tfsdk:"action_names"`
	Countries    types.List   `tfsdk:"countries"`
	NotCountries types.List   `tfsdk:"not_countries"`
	IPCIDRs      types.List   `tfsdk:"ip_cidrs"`
}

func (r *AuthZenPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_authzen_policy"
}

func (r *AuthZenPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a fine-grained authorization rule for the Clavex AuthZen PDP " +
			"(POST /:org_slug/access/v1/evaluation). Use this resource to declare what " +
			"subjects are allowed or denied when performing specific actions, optionally " +
			"scoped by country or IP CIDR. " +
			"action_names maps to the AuthZen action.name field so the PDP can route " +
			"evaluation requests to the correct rule.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name": schema.StringAttribute{
				Required:    true,
				Description: "Human-readable rule name (used in audit logs and evaluation traces).",
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Evaluation order. Lower number = evaluated first. Assigned by server when omitted.",
			},
			"decision": schema.StringAttribute{
				Required:    true,
				Description: `Authorization decision when the rule matches: "allow" or "deny".`,
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the rule is active.",
			},
			"action_names": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "AuthZen action.name values this rule applies to (e.g. [\"read\", \"list\"]). " +
					"Empty = applies to all actions.",
			},
			"countries": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "ISO 3166-1 alpha-2 country allowlist for context-aware authorization.",
			},
			"not_countries": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "ISO 3166-1 alpha-2 country denylist for context-aware authorization.",
			},
			"ip_cidrs": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Restrict this rule to requests from these IP CIDR ranges.",
			},
		},
	}
}

func (r *AuthZenPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AuthZenPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan authzenPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.Policies.Create(ctx, plan.OrgID.ValueString(), clavex.CreatePolicyRuleParams{
		Name:     plan.Name.ValueString(),
		Priority: int(plan.Priority.ValueInt64()),
		Action:   plan.Decision.ValueString(), // decision maps to the policy action
		Conditions: clavex.PolicyConditions{
			ClientIDs:    listToStrings(ctx, plan.ActionNames), // action_names → client_id signal
			Countries:    listToStrings(ctx, plan.Countries),
			NotCountries: listToStrings(ctx, plan.NotCountries),
			IPCIDRs:      listToStrings(ctx, plan.IPCIDRs),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating AuthZen policy", err.Error())
		return
	}
	flattenAuthZenPolicy(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthZenPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state authzenPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, err := r.client.Policies.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing policies", err.Error())
		return
	}
	for _, rule := range rules {
		if rule.ID == state.ID.ValueString() {
			flattenAuthZenPolicy(&rule, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *AuthZenPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan authzenPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	action := plan.Decision.ValueString()
	enabled := plan.Enabled.ValueBool()
	priority := int(plan.Priority.ValueInt64())
	name := plan.Name.ValueString()
	rule, err := r.client.Policies.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.UpdatePolicyRuleParams{
		Name:     &name,
		Priority: &priority,
		Action:   &action,
		Enabled:  &enabled,
		Conditions: clavex.PolicyConditions{
			ClientIDs:    listToStrings(ctx, plan.ActionNames),
			Countries:    listToStrings(ctx, plan.Countries),
			NotCountries: listToStrings(ctx, plan.NotCountries),
			IPCIDRs:      listToStrings(ctx, plan.IPCIDRs),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating AuthZen policy", err.Error())
		return
	}
	flattenAuthZenPolicy(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AuthZenPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state authzenPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Policies.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting AuthZen policy", err.Error())
	}
}

func (r *AuthZenPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<rule_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenAuthZenPolicy(rule *clavex.PolicyRule, m *authzenPolicyModel) {
	m.ID = types.StringValue(rule.ID)
	m.OrgID = types.StringValue(rule.OrgID)
	m.Name = types.StringValue(rule.Name)
	m.Priority = types.Int64Value(int64(rule.Priority))
	m.Decision = types.StringValue(rule.Action)
	m.Enabled = types.BoolValue(rule.Enabled)

	c := rule.Conditions
	m.ActionNames = stringsToList(c.ClientIDs)
	m.Countries = stringsToList(c.Countries)
	m.NotCountries = stringsToList(c.NotCountries)
	m.IPCIDRs = stringsToList(c.IPCIDRs)
}
