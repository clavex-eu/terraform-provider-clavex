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

var _ resource.Resource = &ClientScopeResource{}
var _ resource.ResourceWithConfigure = &ClientScopeResource{}
var _ resource.ResourceWithImportState = &ClientScopeResource{}

// ClientScopeResource manages a clavex_client_scope.
type ClientScopeResource struct{ client *clavex.Client }

func NewClientScopeResource() resource.Resource { return &ClientScopeResource{} }

type clientScopeModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Protocol    types.String `tfsdk:"protocol"`
	IsDefault   types.Bool   `tfsdk:"is_default"`
}

func (r *ClientScopeResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client_scope"
}

func (r *ClientScopeResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a custom OIDC/SAML client scope for a Clavex organisation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name":        schema.StringAttribute{Required: true, Description: "Scope name (e.g. \"profile\")."},
			"description": schema.StringAttribute{Optional: true},
			"protocol":    schema.StringAttribute{Optional: true, Computed: true, Description: "\"openid-connect\" or \"saml\"."},
			"is_default":  schema.BoolAttribute{Computed: true},
		},
	}
}

func (r *ClientScopeResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ClientScopeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan clientScopeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sc, err := r.client.ClientScopes.Create(ctx, plan.OrgID.ValueString(), clavex.CreateClientScopeParams{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Protocol:    plan.Protocol.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating client scope", err.Error())
		return
	}
	flattenClientScope(sc, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ClientScopeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state clientScopeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	scopes, err := r.client.ClientScopes.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing client scopes", err.Error())
		return
	}
	for i := range scopes {
		if scopes[i].ID == state.ID.ValueString() {
			flattenClientScope(&scopes[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *ClientScopeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan clientScopeModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	sc, err := r.client.ClientScopes.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.CreateClientScopeParams{
		Name:        plan.Name.ValueString(),
		Description: plan.Description.ValueString(),
		Protocol:    plan.Protocol.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating client scope", err.Error())
		return
	}
	flattenClientScope(sc, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ClientScopeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state clientScopeModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.ClientScopes.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting client scope", err.Error())
	}
}

func (r *ClientScopeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<scope_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenClientScope(sc *clavex.ClientScope, m *clientScopeModel) {
	m.ID = types.StringValue(sc.ID)
	m.OrgID = types.StringValue(sc.OrgID)
	m.Name = types.StringValue(sc.Name)
	m.Description = ptrToStrAttr(sc.Description)
	m.Protocol = types.StringValue(sc.Protocol)
	m.IsDefault = types.BoolValue(sc.IsDefault)
}
