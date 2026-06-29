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

var _ resource.Resource = &ProtocolMapperResource{}
var _ resource.ResourceWithConfigure = &ProtocolMapperResource{}
var _ resource.ResourceWithImportState = &ProtocolMapperResource{}

// ProtocolMapperResource manages a clavex_protocol_mapper.
type ProtocolMapperResource struct{ client *clavex.Client }

func NewProtocolMapperResource() resource.Resource { return &ProtocolMapperResource{} }

type protocolMapperModel struct {
	ID         types.String `tfsdk:"id"`
	OrgID      types.String `tfsdk:"org_id"`
	ClientID   types.String `tfsdk:"client_id"`
	Name       types.String `tfsdk:"name"`
	Protocol   types.String `tfsdk:"protocol"`
	MapperType types.String `tfsdk:"mapper_type"`
	Config     types.Map    `tfsdk:"config"`
}

func (r *ProtocolMapperResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_protocol_mapper"
}

func (r *ProtocolMapperResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a protocol mapper attached to a Clavex OIDC/SAML client.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Description:   "The client this mapper is attached to.",
			},
			"name":        schema.StringAttribute{Required: true},
			"protocol":    schema.StringAttribute{Required: true, Description: "\"openid-connect\" or \"saml\"."},
			"mapper_type": schema.StringAttribute{Required: true, Description: "Mapper type (e.g. \"user-attribute\")."},
			"config":      schema.MapAttribute{Optional: true, ElementType: types.StringType},
		},
	}
}

func (r *ProtocolMapperResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ProtocolMapperResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan protocolMapperModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pm, err := r.client.ProtocolMappers.Create(ctx, plan.OrgID.ValueString(), plan.ClientID.ValueString(), clavex.CreateProtocolMapperParams{
		Name:       plan.Name.ValueString(),
		Protocol:   plan.Protocol.ValueString(),
		MapperType: plan.MapperType.ValueString(),
		Config:     mapToStrMap(ctx, plan.Config),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating protocol mapper", err.Error())
		return
	}
	flattenProtocolMapper(pm, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProtocolMapperResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state protocolMapperModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	mappers, err := r.client.ProtocolMappers.List(ctx, state.OrgID.ValueString(), state.ClientID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing protocol mappers", err.Error())
		return
	}
	for i := range mappers {
		if mappers[i].ID == state.ID.ValueString() {
			flattenProtocolMapper(&mappers[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *ProtocolMapperResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan protocolMapperModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pm, err := r.client.ProtocolMappers.Update(ctx, plan.OrgID.ValueString(), plan.ClientID.ValueString(), plan.ID.ValueString(), clavex.CreateProtocolMapperParams{
		Name:       plan.Name.ValueString(),
		Protocol:   plan.Protocol.ValueString(),
		MapperType: plan.MapperType.ValueString(),
		Config:     mapToStrMap(ctx, plan.Config),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating protocol mapper", err.Error())
		return
	}
	flattenProtocolMapper(pm, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *ProtocolMapperResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state protocolMapperModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.ProtocolMappers.Delete(ctx, state.OrgID.ValueString(), state.ClientID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting protocol mapper", err.Error())
	}
}

func (r *ProtocolMapperResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 3)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<client_id>/<mapper_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, tfsdk("client_id"), types.StringValue(parts[1]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[2]))...)
}

func flattenProtocolMapper(pm *clavex.ProtocolMapper, m *protocolMapperModel) {
	m.ID = types.StringValue(pm.ID)
	m.ClientID = types.StringValue(pm.ClientID)
	m.Name = types.StringValue(pm.Name)
	m.Protocol = types.StringValue(pm.Protocol)
	m.MapperType = types.StringValue(pm.MapperType)
	// Config values are stringified to fit the string-typed map attribute.
	cfg := make(map[string]string, len(pm.Config))
	for k, v := range pm.Config {
		cfg[k] = fmt.Sprintf("%v", v)
	}
	if len(cfg) > 0 {
		m.Config = strMapToMap(cfg)
	}
}
