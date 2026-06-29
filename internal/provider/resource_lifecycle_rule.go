package provider

import (
	"context"
	"encoding/json"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &LifecycleRuleResource{}
var _ resource.ResourceWithConfigure = &LifecycleRuleResource{}
var _ resource.ResourceWithImportState = &LifecycleRuleResource{}

// LifecycleRuleResource manages a clavex_lifecycle_rule.
type LifecycleRuleResource struct{ client *clavex.Client }

func NewLifecycleRuleResource() resource.Resource { return &LifecycleRuleResource{} }

type lifecycleRuleModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Trigger     types.String `tfsdk:"trigger"`
	Priority    types.Int64  `tfsdk:"priority"`
	Conditions  types.String `tfsdk:"conditions"`
	Actions     types.String `tfsdk:"actions"`
	IsActive    types.Bool   `tfsdk:"is_active"`
}

func (r *LifecycleRuleResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_lifecycle_rule"
}

func (r *LifecycleRuleResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a joiner/mover/leaver lifecycle automation rule.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":      schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":        schema.StringAttribute{Required: true},
			"description": schema.StringAttribute{Optional: true},
			"trigger":     schema.StringAttribute{Required: true, Description: "joiner | mover | leaver."},
			"priority":    schema.Int64Attribute{Required: true},
			"conditions":  schema.StringAttribute{Required: true, Description: "JSON array of condition objects."},
			"actions":     schema.StringAttribute{Required: true, Description: "JSON array of action objects."},
			"is_active":   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		},
	}
}

func (r *LifecycleRuleResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LifecycleRuleResource) params(plan *lifecycleRuleModel) clavex.LifecycleRuleParams {
	isActive := plan.IsActive.ValueBool()
	return clavex.LifecycleRuleParams{
		Name:        plan.Name.ValueString(),
		Description: strAttrToPtr(plan.Description),
		Trigger:     plan.Trigger.ValueString(),
		Priority:    int(plan.Priority.ValueInt64()),
		Conditions:  json.RawMessage(plan.Conditions.ValueString()),
		Actions:     json.RawMessage(plan.Actions.ValueString()),
		IsActive:    &isActive,
	}
}

func (r *LifecycleRuleResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan lifecycleRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.LifecycleRules.Create(ctx, plan.OrgID.ValueString(), r.params(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error creating lifecycle rule", err.Error())
		return
	}
	flattenLifecycleRule(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LifecycleRuleResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state lifecycleRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.LifecycleRules.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading lifecycle rule", err.Error())
		return
	}
	flattenLifecycleRule(rule, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LifecycleRuleResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan lifecycleRuleModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	rule, err := r.client.LifecycleRules.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), r.params(&plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating lifecycle rule", err.Error())
		return
	}
	flattenLifecycleRule(rule, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LifecycleRuleResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state lifecycleRuleModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.LifecycleRules.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting lifecycle rule", err.Error())
	}
}

func (r *LifecycleRuleResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<rule_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenLifecycleRule(rule *clavex.LifecycleRule, m *lifecycleRuleModel) {
	m.ID = types.StringValue(rule.ID)
	m.OrgID = types.StringValue(rule.OrgID)
	m.Name = types.StringValue(rule.Name)
	m.Description = ptrToStrAttr(rule.Description)
	m.Trigger = types.StringValue(rule.Trigger)
	m.Priority = types.Int64Value(int64(rule.Priority))
	if len(rule.Conditions) > 0 {
		m.Conditions = types.StringValue(string(rule.Conditions))
	}
	if len(rule.Actions) > 0 {
		m.Actions = types.StringValue(string(rule.Actions))
	}
	m.IsActive = types.BoolValue(rule.IsActive)
}
