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

var _ resource.Resource = &WebhookResource{}
var _ resource.ResourceWithConfigure = &WebhookResource{}
var _ resource.ResourceWithImportState = &WebhookResource{}

// WebhookResource manages a clavex_webhook.
type WebhookResource struct{ client *clavex.Client }

func NewWebhookResource() resource.Resource { return &WebhookResource{} }

type webhookModel struct {
	ID       types.String `tfsdk:"id"`
	OrgID    types.String `tfsdk:"org_id"`
	URL      types.String `tfsdk:"url"`
	Events   types.List   `tfsdk:"events"`
	Secret   types.String `tfsdk:"secret"`
	IsActive types.Bool   `tfsdk:"is_active"`
}

func (r *WebhookResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_webhook"
}

func (r *WebhookResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an outbound webhook for a Clavex organisation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"url":    schema.StringAttribute{Required: true, Description: "HTTPS endpoint that receives webhook payloads."},
			"events": schema.ListAttribute{Required: true, ElementType: types.StringType, Description: "List of event types to subscribe to (e.g. [\"user.created\", \"user.deleted\"])."},
			"secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Shared secret used to sign HMAC-SHA256 payloads.",
			},
			"is_active": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		},
	}
}

func (r *WebhookResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *WebhookResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan webhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	isActive := plan.IsActive.ValueBool()
	wh, err := r.client.Webhooks.Create(ctx, plan.OrgID.ValueString(), clavex.CreateWebhookParams{
		URL:      plan.URL.ValueString(),
		Events:   listToStrings(ctx, plan.Events),
		Secret:   plan.Secret.ValueString(),
		IsActive: &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating webhook", err.Error())
		return
	}
	flattenWebhook(wh, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state webhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	// Webhooks are list-only; find by ID.
	hooks, err := r.client.Webhooks.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing webhooks", err.Error())
		return
	}
	for _, wh := range hooks {
		if wh.ID == state.ID.ValueString() {
			flattenWebhook(&wh, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *WebhookResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan webhookModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	isActive := plan.IsActive.ValueBool()
	wh, err := r.client.Webhooks.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.CreateWebhookParams{
		URL:      plan.URL.ValueString(),
		Events:   listToStrings(ctx, plan.Events),
		Secret:   plan.Secret.ValueString(),
		IsActive: &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating webhook", err.Error())
		return
	}
	flattenWebhook(wh, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *WebhookResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state webhookModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Webhooks.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting webhook", err.Error())
	}
}

func (r *WebhookResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<webhook_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenWebhook(wh *clavex.Webhook, m *webhookModel) {
	m.ID = types.StringValue(wh.ID)
	m.OrgID = types.StringValue(wh.OrgID)
	m.URL = types.StringValue(wh.URL)
	m.Events = stringsToList(wh.Events)
	m.IsActive = types.BoolValue(wh.IsActive)
	// Secret is write-only; never returned by the API.
}
