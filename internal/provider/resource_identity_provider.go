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

var _ resource.Resource = &IdentityProviderResource{}
var _ resource.ResourceWithConfigure = &IdentityProviderResource{}
var _ resource.ResourceWithImportState = &IdentityProviderResource{}

// IdentityProviderResource manages a clavex_identity_provider.
type IdentityProviderResource struct{ client *clavex.Client }

func NewIdentityProviderResource() resource.Resource { return &IdentityProviderResource{} }

type identityProviderModel struct {
	ID                types.String `tfsdk:"id"`
	OrgID             types.String `tfsdk:"org_id"`
	Name              types.String `tfsdk:"name"`
	ProviderType      types.String `tfsdk:"provider_type"`
	ClientID          types.String `tfsdk:"client_id_attr"`
	ClientSecret      types.String `tfsdk:"client_secret"`
	AuthorizationURL  types.String `tfsdk:"authorization_url"`
	TokenURL          types.String `tfsdk:"token_url"`
	UserinfoURL       types.String `tfsdk:"userinfo_url"`
	Scopes            types.String `tfsdk:"scopes"`
	IsActive          types.Bool   `tfsdk:"is_active"`
	AllowJIT          types.Bool   `tfsdk:"allow_jit"`
	RolesClaim        types.String `tfsdk:"roles_claim"`
	RoleClaimMappings types.Map    `tfsdk:"role_claim_mappings"`
}

func (r *IdentityProviderResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_identity_provider"
}

func (r *IdentityProviderResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an upstream OIDC/OAuth2 identity provider for SSO within a Clavex organisation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name":          schema.StringAttribute{Required: true},
			"provider_type": schema.StringAttribute{Required: true, Description: "e.g. \"oidc\", \"saml\", \"google\", \"github\""},
			"client_id_attr": schema.StringAttribute{
				Required:    true,
				Description: "OAuth2 client_id for the upstream provider.",
			},
			"client_secret": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "OAuth2 client_secret for the upstream provider.",
			},
			"authorization_url": schema.StringAttribute{Optional: true},
			"token_url":         schema.StringAttribute{Optional: true},
			"userinfo_url":      schema.StringAttribute{Optional: true},
			"scopes":            schema.StringAttribute{Optional: true, Computed: true, Description: "Space-separated scopes (e.g. \"openid email profile\")."},
			"is_active":         schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"allow_jit": schema.BoolAttribute{
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				Description: "Automatically provision users on first login via this IdP.",
			},
			"roles_claim": schema.StringAttribute{
				Optional:    true,
				Description: "JWT claim name that carries role assignments from the upstream IdP.",
			},
			"role_claim_mappings": schema.MapAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Maps upstream role values to Clavex role IDs.",
			},
		},
	}
}

func (r *IdentityProviderResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *IdentityProviderResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan identityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	cfg := map[string]string{
		"client_id":     plan.ClientID.ValueString(),
		"client_secret": plan.ClientSecret.ValueString(),
	}
	if !plan.AuthorizationURL.IsNull() {
		cfg["authorization_url"] = plan.AuthorizationURL.ValueString()
	}
	if !plan.TokenURL.IsNull() {
		cfg["token_url"] = plan.TokenURL.ValueString()
	}
	if !plan.UserinfoURL.IsNull() {
		cfg["userinfo_url"] = plan.UserinfoURL.ValueString()
	}

	enabled := plan.IsActive.ValueBool()
	idp, err := r.client.IdentityProviders.Create(ctx, plan.OrgID.ValueString(), clavex.CreateIDPParams{
		Name:              plan.Name.ValueString(),
		Type:              plan.ProviderType.ValueString(),
		Config:            cfg,
		Enabled:           &enabled,
		AllowJIT:          plan.AllowJIT.ValueBool(),
		RolesClaim:        plan.RolesClaim.ValueString(),
		RoleClaimMappings: mapToStrMap(ctx, plan.RoleClaimMappings),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating identity provider", err.Error())
		return
	}
	flattenIDP(idp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityProviderResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state identityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	idp, err := r.client.IdentityProviders.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading identity provider", err.Error())
		return
	}
	flattenIDP(idp, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *IdentityProviderResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan identityProviderModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	enabled := plan.IsActive.ValueBool()
	cfg := map[string]string{"client_id": plan.ClientID.ValueString()}
	if !plan.ClientSecret.IsNull() {
		cfg["client_secret"] = plan.ClientSecret.ValueString()
	}
	idp, err := r.client.IdentityProviders.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.CreateIDPParams{
		Name:              plan.Name.ValueString(),
		Type:              plan.ProviderType.ValueString(),
		Config:            cfg,
		Enabled:           &enabled,
		AllowJIT:          plan.AllowJIT.ValueBool(),
		RolesClaim:        plan.RolesClaim.ValueString(),
		RoleClaimMappings: mapToStrMap(ctx, plan.RoleClaimMappings),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating identity provider", err.Error())
		return
	}
	flattenIDP(idp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *IdentityProviderResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state identityProviderModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.IdentityProviders.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting identity provider", err.Error())
	}
}

func (r *IdentityProviderResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<idp_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenIDP(idp *clavex.IdentityProvider, m *identityProviderModel) {
	m.ID = types.StringValue(idp.ID)
	m.OrgID = types.StringValue(idp.OrgID)
	m.Name = types.StringValue(idp.Name)
	m.ProviderType = types.StringValue(idp.ProviderType)
	m.ClientID = types.StringValue(idp.ClientID)
	m.AuthorizationURL = types.StringValue(idp.AuthorizationURL)
	m.TokenURL = types.StringValue(idp.TokenURL)
	m.UserinfoURL = ptrToStrAttr(idp.UserinfoURL)
	m.Scopes = types.StringValue(idp.Scopes)
	m.IsActive = types.BoolValue(idp.IsActive)
	m.AllowJIT = types.BoolValue(idp.AllowJIT)
	m.RolesClaim = ptrToStrAttr(idp.RolesClaim)
	m.RoleClaimMappings = strMapToMap(idp.RoleClaimMappings)
}
