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

var _ resource.Resource = &OrgPolicyResource{}
var _ resource.ResourceWithConfigure = &OrgPolicyResource{}
var _ resource.ResourceWithImportState = &OrgPolicyResource{}

// OrgPolicyResource manages a clavex_org_policy.
type OrgPolicyResource struct{ client *clavex.Client }

func NewOrgPolicyResource() resource.Resource { return &OrgPolicyResource{} }

type orgPolicyModel struct {
	ID       types.String `tfsdk:"id"`
	OrgID    types.String `tfsdk:"org_id"`
	Name     types.String `tfsdk:"name"`
	Priority types.Int64  `tfsdk:"priority"`
	Action   types.String `tfsdk:"action"`
	Enabled  types.Bool   `tfsdk:"enabled"`
	// Conditions
	IPCIDRs         types.List   `tfsdk:"ip_cidrs"`
	Countries       types.List   `tfsdk:"countries"`
	NotCountries    types.List   `tfsdk:"not_countries"`
	ClientIDs       types.List   `tfsdk:"client_ids"`
	DaysOfWeek      types.List   `tfsdk:"days_of_week"`
	LastLoginBefore types.String `tfsdk:"last_login_before"`
	MFAEnrolled     types.Bool   `tfsdk:"mfa_enrolled"`
	NewCountry      types.Bool   `tfsdk:"new_country"`
	HourFrom        types.Int64  `tfsdk:"hour_from"`
	HourTo          types.Int64  `tfsdk:"hour_to"`
}

func (r *OrgPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_org_policy"
}

func (r *OrgPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an auth-flow policy rule for a Clavex organisation. " +
			"Rules are evaluated in priority order (lowest number first) during every " +
			"authentication event and determine whether to allow, deny, or require MFA.",
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
				Description: "Human-readable rule name.",
			},
			"priority": schema.Int64Attribute{
				Optional:    true,
				Computed:    true,
				Description: "Evaluation order. Lower number = evaluated first. Assigned by server when omitted.",
			},
			"action": schema.StringAttribute{
				Required:    true,
				Description: `Outcome when the rule matches: "allow", "deny", "require_mfa", or "step_up".`,
			},
			"enabled": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
				Description: "Whether the rule is active.",
			},
			// ── Conditions ──────────────────────────────────────────────────
			"ip_cidrs": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Match requests from these IP CIDR ranges (e.g. [\"10.0.0.0/8\"]).",
			},
			"countries": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "ISO 3166-1 alpha-2 country allowlist (e.g. [\"IT\", \"DE\"]).",
			},
			"not_countries": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "ISO 3166-1 alpha-2 country denylist (e.g. [\"CN\", \"RU\"]).",
			},
			"client_ids": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Restrict the rule to specific OIDC client_id values.",
			},
			"days_of_week": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: `Restrict to UTC days of week: "Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun".`,
			},
			"last_login_before": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: `Match users whose last login is older than this Go duration (e.g. "720h"). Users who never logged in always match.`,
			},
			"mfa_enrolled": schema.BoolAttribute{
				Optional:    true,
				Description: "Match users who are (true) or are not (false) enrolled in MFA. Null = no constraint.",
			},
			"new_country": schema.BoolAttribute{
				Optional:    true,
				Description: "Match logins from a country not seen in the user's 90-day baseline. Null = no constraint.",
			},
			"hour_from": schema.Int64Attribute{
				Optional:    true,
				Description: "Start of UTC hour window (0–23). Must be set together with hour_to.",
			},
			"hour_to": schema.Int64Attribute{
				Optional:    true,
				Description: "End of UTC hour window (0–23, inclusive). Must be set together with hour_from.",
			},
		},
	}
}

func (r *OrgPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrgPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan orgPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.Policies.Create(ctx, plan.OrgID.ValueString(), clavex.CreatePolicyRuleParams{
		Name:       plan.Name.ValueString(),
		Priority:   int(plan.Priority.ValueInt64()),
		Action:     plan.Action.ValueString(),
		Conditions: orgPolicyConditionsFromPlan(ctx, &plan),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating org policy", err.Error())
		return
	}
	flattenOrgPolicy(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state orgPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rules, err := r.client.Policies.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing org policies", err.Error())
		return
	}
	for _, rule := range rules {
		if rule.ID == state.ID.ValueString() {
			flattenOrgPolicy(&rule, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *OrgPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan orgPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	action := plan.Action.ValueString()
	enabled := plan.Enabled.ValueBool()
	priority := int(plan.Priority.ValueInt64())
	name := plan.Name.ValueString()
	rule, err := r.client.Policies.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.UpdatePolicyRuleParams{
		Name:       &name,
		Priority:   &priority,
		Action:     &action,
		Enabled:    &enabled,
		Conditions: orgPolicyConditionsFromPlan(ctx, &plan),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating org policy", err.Error())
		return
	}
	flattenOrgPolicy(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrgPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state orgPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Policies.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting org policy", err.Error())
	}
}

func (r *OrgPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<rule_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func orgPolicyConditionsFromPlan(ctx context.Context, m *orgPolicyModel) clavex.PolicyConditions {
	cond := clavex.PolicyConditions{
		IPCIDRs:         listToStrings(ctx, m.IPCIDRs),
		Countries:       listToStrings(ctx, m.Countries),
		NotCountries:    listToStrings(ctx, m.NotCountries),
		ClientIDs:       listToStrings(ctx, m.ClientIDs),
		DaysOfWeek:      listToStrings(ctx, m.DaysOfWeek),
		LastLoginBefore: m.LastLoginBefore.ValueString(),
		MFAEnrolled:     boolAttrToPtr(m.MFAEnrolled),
		NewCountry:      boolAttrToPtr(m.NewCountry),
	}
	if !m.HourFrom.IsNull() && !m.HourTo.IsNull() {
		cond.HourRange = &clavex.HourRange{
			From: int(m.HourFrom.ValueInt64()),
			To:   int(m.HourTo.ValueInt64()),
		}
	}
	return cond
}

func flattenOrgPolicy(rule *clavex.PolicyRule, m *orgPolicyModel) {
	m.ID = types.StringValue(rule.ID)
	m.OrgID = types.StringValue(rule.OrgID)
	m.Name = types.StringValue(rule.Name)
	m.Priority = types.Int64Value(int64(rule.Priority))
	m.Action = types.StringValue(rule.Action)
	m.Enabled = types.BoolValue(rule.Enabled)

	c := rule.Conditions
	m.IPCIDRs = stringsToList(c.IPCIDRs)
	m.Countries = stringsToList(c.Countries)
	m.NotCountries = stringsToList(c.NotCountries)
	m.ClientIDs = stringsToList(c.ClientIDs)
	m.DaysOfWeek = stringsToList(c.DaysOfWeek)
	m.LastLoginBefore = types.StringValue(c.LastLoginBefore)
	m.MFAEnrolled = ptrToBoolAttr(c.MFAEnrolled)
	m.NewCountry = ptrToBoolAttr(c.NewCountry)
	if c.HourRange != nil {
		m.HourFrom = types.Int64Value(int64(c.HourRange.From))
		m.HourTo = types.Int64Value(int64(c.HourRange.To))
	} else {
		m.HourFrom = types.Int64Null()
		m.HourTo = types.Int64Null()
	}
}
