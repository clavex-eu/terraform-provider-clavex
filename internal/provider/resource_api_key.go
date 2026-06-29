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

var _ resource.Resource = &APIKeyResource{}
var _ resource.ResourceWithConfigure = &APIKeyResource{}
var _ resource.ResourceWithImportState = &APIKeyResource{}

// APIKeyResource manages a clavex_api_key.
//
// API keys are superadmin credentials used for machine-to-machine integrations
// (CI/CD pipelines, IaC, provisioning tools) as an alternative to JWT bearer
// tokens. The plaintext secret is only available at creation time; it is stored
// in Terraform state as a sensitive computed attribute.
type APIKeyResource struct{ client *clavex.Client }

func NewAPIKeyResource() resource.Resource { return &APIKeyResource{} }

type apiKeyModel struct {
	ID        types.String `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	ExpiresIn types.Int64  `tfsdk:"expires_in"`
	Prefix    types.String `tfsdk:"prefix"`
	Secret    types.String `tfsdk:"secret"`
	ExpiresAt types.String `tfsdk:"expires_at"`
	CreatedAt types.String `tfsdk:"created_at"`
}

func (r *APIKeyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_api_key"
}

func (r *APIKeyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clavex superadmin API key. " +
			"API keys can be used instead of JWT bearer tokens for machine-to-machine " +
			"integrations. The plaintext secret is only available at creation time — " +
			"it is stored in Terraform state (sensitive) and will never be returned by " +
			"the API again. Changing any attribute replaces the key (revoke + new).",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name": schema.StringAttribute{
				Required: true,
				Description: "Human-readable label (e.g. \"terraform\", \"ci-pipeline\"). " +
					"Changing this value forces a new resource.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"expires_in": schema.Int64Attribute{
				Optional: true,
				Description: "Key lifetime in seconds. 0 or omitted = no expiry. " +
					"Changing this value forces a new resource.",
			},
			"prefix": schema.StringAttribute{
				Computed:      true,
				Description:   "Display prefix (e.g. \"cvx_sk_...\"). Safe to log; not the secret.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"secret": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "Plaintext API key. Only populated at creation. Store securely — never returned by the API again.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"expires_at": schema.StringAttribute{
				Computed:      true,
				Description:   "RFC 3339 expiry timestamp, or empty string if the key never expires.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"created_at": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *APIKeyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *APIKeyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan apiKeyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	result, err := r.client.APIKeys.Create(ctx, clavex.CreateAPIKeyParams{
		Name:      plan.Name.ValueString(),
		ExpiresIn: int(plan.ExpiresIn.ValueInt64()),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating API key", err.Error())
		return
	}
	flattenAPIKey(&result.APIKey, &plan)
	plan.Secret = types.StringValue(result.Secret)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *APIKeyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state apiKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	keys, err := r.client.APIKeys.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing API keys", err.Error())
		return
	}
	for _, key := range keys {
		if key.ID == state.ID.ValueString() {
			flattenAPIKey(&key, &state)
			// Secret is write-once; do not overwrite with empty from the API.
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

// Update is never called because all non-computed attributes carry RequiresReplace.
func (r *APIKeyResource) Update(_ context.Context, _ resource.UpdateRequest, resp *resource.UpdateResponse) {
	resp.Diagnostics.AddError(
		"API key update not supported",
		"API keys are immutable. Changing attributes forces a new resource (revoke + create).",
	)
}

func (r *APIKeyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state apiKeyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.APIKeys.Revoke(ctx, state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error revoking API key", err.Error())
	}
}

func (r *APIKeyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(req.ID))...)
}

func flattenAPIKey(key *clavex.APIKey, m *apiKeyModel) {
	m.ID = types.StringValue(key.ID)
	m.Name = types.StringValue(key.Name)
	m.Prefix = types.StringValue(key.Prefix)
	if key.ExpiresAt != nil {
		m.ExpiresAt = types.StringValue(key.ExpiresAt.Format("2006-01-02T15:04:05Z"))
	} else {
		m.ExpiresAt = types.StringValue("")
	}
	m.CreatedAt = types.StringValue(key.CreatedAt.Format("2006-01-02T15:04:05Z"))
}
