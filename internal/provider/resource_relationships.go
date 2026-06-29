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

// ── clavex_role_assignment ────────────────────────────────────────────────────

var _ resource.Resource = &RoleAssignmentResource{}
var _ resource.ResourceWithConfigure = &RoleAssignmentResource{}
var _ resource.ResourceWithImportState = &RoleAssignmentResource{}

// RoleAssignmentResource grants a role to a user.
type RoleAssignmentResource struct{ client *clavex.Client }

func NewRoleAssignmentResource() resource.Resource { return &RoleAssignmentResource{} }

type roleAssignmentModel struct {
	ID     types.String `tfsdk:"id"`
	OrgID  types.String `tfsdk:"org_id"`
	RoleID types.String `tfsdk:"role_id"`
	UserID types.String `tfsdk:"user_id"`
}

func (r *RoleAssignmentResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role_assignment"
}

func (r *RoleAssignmentResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNew := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Assigns a role to a user within an organisation. " +
			"All attributes are immutable — changing any field destroys and recreates the assignment.",
		Attributes: map[string]schema.Attribute{
			"id":      schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":  schema.StringAttribute{Required: true, PlanModifiers: forceNew},
			"role_id": schema.StringAttribute{Required: true, PlanModifiers: forceNew},
			"user_id": schema.StringAttribute{Required: true, PlanModifiers: forceNew},
		},
	}
}

func (r *RoleAssignmentResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RoleAssignmentResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan roleAssignmentModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Roles.AssignToUser(ctx, plan.OrgID.ValueString(), plan.RoleID.ValueString(), plan.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error assigning role", err.Error())
		return
	}
	plan.ID = types.StringValue(joinID(plan.OrgID.ValueString(), plan.RoleID.ValueString(), plan.UserID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

// Read reconstructs the model from the composite ID without an API call.
// Drift will be corrected on the next apply.
func (r *RoleAssignmentResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.Append(req.State.Get(ctx, &roleAssignmentModel{})...)
}

func (r *RoleAssignmentResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
	// All attributes are ForceNew — Update is never called.
}

func (r *RoleAssignmentResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state roleAssignmentModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Roles.UnassignFromUser(ctx, state.OrgID.ValueString(), state.RoleID.ValueString(), state.UserID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error unassigning role", err.Error())
	}
}

func (r *RoleAssignmentResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<role_id>/<user_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tfsdk("role_id"), types.StringValue(parts[1]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tfsdk("user_id"), types.StringValue(parts[2]))...)
}

// ── clavex_group_member ───────────────────────────────────────────────────────

var _ resource.Resource = &GroupMemberResource{}
var _ resource.ResourceWithConfigure = &GroupMemberResource{}
var _ resource.ResourceWithImportState = &GroupMemberResource{}

// GroupMemberResource adds a user to a group.
type GroupMemberResource struct{ client *clavex.Client }

func NewGroupMemberResource() resource.Resource { return &GroupMemberResource{} }

type groupMemberModel struct {
	ID      types.String `tfsdk:"id"`
	OrgID   types.String `tfsdk:"org_id"`
	GroupID types.String `tfsdk:"group_id"`
	UserID  types.String `tfsdk:"user_id"`
}

func (r *GroupMemberResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_group_member"
}

func (r *GroupMemberResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	forceNew := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Adds a user to a group. All attributes are immutable.",
		Attributes: map[string]schema.Attribute{
			"id":       schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":   schema.StringAttribute{Required: true, PlanModifiers: forceNew},
			"group_id": schema.StringAttribute{Required: true, PlanModifiers: forceNew},
			"user_id":  schema.StringAttribute{Required: true, PlanModifiers: forceNew},
		},
	}
}

func (r *GroupMemberResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *GroupMemberResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan groupMemberModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Groups.AddMember(ctx, plan.OrgID.ValueString(), plan.GroupID.ValueString(), plan.UserID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error adding group member", err.Error())
		return
	}
	plan.ID = types.StringValue(joinID(plan.OrgID.ValueString(), plan.GroupID.ValueString(), plan.UserID.ValueString()))
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *GroupMemberResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	resp.Diagnostics.Append(req.State.Get(ctx, &groupMemberModel{})...)
}

func (r *GroupMemberResource) Update(_ context.Context, _ resource.UpdateRequest, _ *resource.UpdateResponse) {
}

func (r *GroupMemberResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state groupMemberModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Groups.RemoveMember(ctx, state.OrgID.ValueString(), state.GroupID.ValueString(), state.UserID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error removing group member", err.Error())
	}
}

func (r *GroupMemberResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<group_id>/<user_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tfsdk("group_id"), types.StringValue(parts[1]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tfsdk("user_id"), types.StringValue(parts[2]))...)
}
