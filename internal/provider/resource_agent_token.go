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

var _ resource.Resource = &AgentTokenResource{}
var _ resource.ResourceWithConfigure = &AgentTokenResource{}
var _ resource.ResourceWithImportState = &AgentTokenResource{}

// AgentTokenResource manages a clavex_agent_token. Tokens are immutable; the
// signed JWT is only returned at creation.
type AgentTokenResource struct{ client *clavex.Client }

func NewAgentTokenResource() resource.Resource { return &AgentTokenResource{} }

type agentTokenModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	UserID      types.String `tfsdk:"user_id"`
	AgentID     types.String `tfsdk:"agent_id"`
	AgentName   types.String `tfsdk:"agent_name"`
	Scope       types.String `tfsdk:"scope"`
	TTLSeconds  types.Int64  `tfsdk:"ttl_seconds"`
	MCPServerID types.String `tfsdk:"mcp_server_id"`
	Token       types.String `tfsdk:"token"`
	ExpiresAt   types.String `tfsdk:"expires_at"`
}

func (r *AgentTokenResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_agent_token"
}

func (r *AgentTokenResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	replace := []planmodifier.String{stringplanmodifier.RequiresReplace()}
	resp.Schema = schema.Schema{
		Description: "Manages an AI-agent (MCP) access token. Immutable; the token is returned only at creation.",
		Attributes: map[string]schema.Attribute{
			"id":            schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":        schema.StringAttribute{Required: true, PlanModifiers: replace},
			"user_id":       schema.StringAttribute{Required: true, PlanModifiers: replace},
			"agent_id":      schema.StringAttribute{Required: true, PlanModifiers: replace},
			"agent_name":    schema.StringAttribute{Required: true, PlanModifiers: replace},
			"scope":         schema.StringAttribute{Optional: true, PlanModifiers: replace},
			"ttl_seconds":   schema.Int64Attribute{Optional: true, PlanModifiers: []planmodifier.Int64{}},
			"mcp_server_id": schema.StringAttribute{Optional: true, PlanModifiers: replace},
			"token": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Description:   "Signed JWT, only set at creation.",
			},
			"expires_at": schema.StringAttribute{Computed: true},
		},
	}
}

func (r *AgentTokenResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *AgentTokenResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan agentTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	res, err := r.client.AgentTokens.Issue(ctx, plan.OrgID.ValueString(), clavex.IssueAgentTokenParams{
		UserID:      plan.UserID.ValueString(),
		AgentID:     plan.AgentID.ValueString(),
		AgentName:   plan.AgentName.ValueString(),
		Scope:       plan.Scope.ValueString(),
		TTLSeconds:  int(plan.TTLSeconds.ValueInt64()),
		MCPServerID: strAttrToPtr(plan.MCPServerID),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error issuing agent token", err.Error())
		return
	}
	plan.ID = types.StringValue(res.TokenID)
	plan.Token = types.StringValue(res.Token)
	plan.ExpiresAt = types.StringValue(res.ExpiresAt)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentTokenResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state agentTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tokens, err := r.client.AgentTokens.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing agent tokens", err.Error())
		return
	}
	for i := range tokens {
		if tokens[i].ID == state.ID.ValueString() {
			state.UserID = types.StringValue(tokens[i].UserID)
			state.AgentID = types.StringValue(tokens[i].AgentID)
			state.AgentName = types.StringValue(tokens[i].AgentName)
			if tokens[i].Scope != "" {
				state.Scope = types.StringValue(tokens[i].Scope)
			}
			state.MCPServerID = ptrToStrAttr(tokens[i].MCPServerID)
			state.ExpiresAt = types.StringValue(tokens[i].ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *AgentTokenResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan agentTokenModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *AgentTokenResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state agentTokenModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.AgentTokens.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting agent token", err.Error())
	}
}

func (r *AgentTokenResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<token_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}
