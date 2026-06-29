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

var _ resource.Resource = &SSFStreamResource{}
var _ resource.ResourceWithConfigure = &SSFStreamResource{}
var _ resource.ResourceWithImportState = &SSFStreamResource{}

// SSFStreamResource manages a clavex_ssf_stream.
type SSFStreamResource struct{ client *clavex.Client }

func NewSSFStreamResource() resource.Resource { return &SSFStreamResource{} }

type ssfStreamModel struct {
	ID             types.String `tfsdk:"id"`
	OrgID          types.String `tfsdk:"org_id"`
	DeliveryMethod types.String `tfsdk:"delivery_method"`
	EventTypes     types.List   `tfsdk:"event_types"`
	EndpointURL    types.String `tfsdk:"endpoint_url"`
	Description    types.String `tfsdk:"description"`
	Status         types.String `tfsdk:"status"`
}

func (r *SSFStreamResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssf_stream"
}

func (r *SSFStreamResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Shared Signals Framework (RFC 8935 / RFC 8936) stream. " +
			"SSF lets your application receive real-time security events " +
			"(session revocation, credential changes, account compromise) from Clavex " +
			"via push (HTTP POST) or pull (polling) delivery.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"delivery_method": schema.StringAttribute{
				Required: true,
				Description: "RISC delivery method URI. " +
					"Push: \"https://schemas.openid.net/secevent/risc/delivery-method/push\". " +
					"Poll: \"https://schemas.openid.net/secevent/risc/delivery-method/poll\". " +
					"Changing this value forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"event_types": schema.ListAttribute{
				Required:    true,
				ElementType: types.StringType,
				Description: "SET event-type URIs to subscribe to. " +
					"Example: [\"https://schemas.openid.net/secevent/risc/event-type/session-revoked\"].",
			},
			"endpoint_url": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "HTTPS endpoint that receives push payloads. Required when delivery_method is push.",
			},
			"description": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: "Human-readable label for the stream.",
			},
			"status": schema.StringAttribute{
				Optional:    true,
				Computed:    true,
				Description: `Stream state: "enabled", "paused", or "disabled". Defaults to "enabled" on creation.`,
			},
		},
	}
}

func (r *SSFStreamResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SSFStreamResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ssfStreamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	stream, err := r.client.SSF.Create(ctx, plan.OrgID.ValueString(), clavex.CreateSSFStreamParams{
		EventTypes:     listToStrings(ctx, plan.EventTypes),
		DeliveryMethod: plan.DeliveryMethod.ValueString(),
		EndpointURL:    plan.EndpointURL.ValueString(),
		Description:    plan.Description.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating SSF stream", err.Error())
		return
	}
	flattenSSFStream(stream, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SSFStreamResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ssfStreamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	stream, err := r.client.SSF.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading SSF stream", err.Error())
		return
	}
	flattenSSFStream(stream, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SSFStreamResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ssfStreamModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	status := plan.Status.ValueString()
	description := plan.Description.ValueString()
	endpointURL := plan.EndpointURL.ValueString()
	stream, err := r.client.SSF.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.UpdateSSFStreamParams{
		EventTypes:  listToStrings(ctx, plan.EventTypes),
		Status:      &status,
		Description: &description,
		EndpointURL: &endpointURL,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating SSF stream", err.Error())
		return
	}
	flattenSSFStream(stream, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SSFStreamResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ssfStreamModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.SSF.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting SSF stream", err.Error())
	}
}

func (r *SSFStreamResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<stream_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenSSFStream(s *clavex.SSFStream, m *ssfStreamModel) {
	m.ID = types.StringValue(s.ID)
	m.OrgID = types.StringValue(s.OrgID)
	m.DeliveryMethod = types.StringValue(s.DeliveryMethod)
	m.EventTypes = stringsToList(s.EventTypes)
	m.EndpointURL = ptrToStrAttr(s.EndpointURL)
	m.Description = types.StringValue(s.Description)
	m.Status = types.StringValue(s.Status)
}
