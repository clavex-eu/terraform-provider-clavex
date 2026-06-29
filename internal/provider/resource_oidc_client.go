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

var _ resource.Resource = &OIDCClientResource{}
var _ resource.ResourceWithConfigure = &OIDCClientResource{}
var _ resource.ResourceWithImportState = &OIDCClientResource{}

// OIDCClientResource manages a clavex_oidc_client.
type OIDCClientResource struct{ client *clavex.Client }

func NewOIDCClientResource() resource.Resource { return &OIDCClientResource{} }

type oidcClientModel struct {
	ID                  types.String `tfsdk:"id"`
	OrgID               types.String `tfsdk:"org_id"`
	ClientID            types.String `tfsdk:"client_id"`
	Name                types.String `tfsdk:"name"`
	RedirectURIs        types.List   `tfsdk:"redirect_uris"`
	PostLogoutRedirects types.List   `tfsdk:"post_logout_redirect_uris"`
	GrantTypes          types.List   `tfsdk:"grant_types"`
	IsPublic            types.Bool   `tfsdk:"is_public"`
	IsActive            types.Bool   `tfsdk:"is_active"`
	ClientSecret        types.String `tfsdk:"client_secret"`
}

func (r *OIDCClientResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_oidc_client"
}

func (r *OIDCClientResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an OIDC/OAuth2 client application within a Clavex organisation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:      true,
				Description:   "Internal UUID of the client record.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"client_id": schema.StringAttribute{
				Computed:      true,
				Description:   "OAuth2 client_id (assigned by the server).",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"name":          schema.StringAttribute{Required: true},
			"redirect_uris": schema.ListAttribute{Required: true, ElementType: types.StringType},
			"post_logout_redirect_uris": schema.ListAttribute{
				Optional: true, Computed: true, ElementType: types.StringType,
			},
			"grant_types": schema.ListAttribute{
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Description: "Allowed grant types (e.g. [\"authorization_code\", \"refresh_token\"]).",
			},
			"is_public": schema.BoolAttribute{
				Optional: true, Computed: true, Default: booldefault.StaticBool(false),
				Description: "Public clients do not authenticate with a secret (e.g. SPAs, mobile apps).",
			},
			"is_active": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"client_secret": schema.StringAttribute{
				Computed:      true,
				Sensitive:     true,
				Description:   "Client secret. Only populated on creation (or secret rotation). Stored in state as sensitive.",
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		},
	}
}

func (r *OIDCClientResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OIDCClientResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan oidcClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	result, err := r.client.Clients.Create(ctx, plan.OrgID.ValueString(), clavex.CreateClientParams{
		Name:                plan.Name.ValueString(),
		RedirectURIs:        listToStrings(ctx, plan.RedirectURIs),
		PostLogoutRedirects: listToStrings(ctx, plan.PostLogoutRedirects),
		GrantTypes:          listToStrings(ctx, plan.GrantTypes),
		IsPublic:            plan.IsPublic.ValueBool(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating OIDC client", err.Error())
		return
	}
	flattenOIDCClient(&result.Client, &plan)
	if result.ClientSecret != nil {
		plan.ClientSecret = types.StringValue(*result.ClientSecret)
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OIDCClientResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state oidcClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	client, err := r.client.Clients.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading OIDC client", err.Error())
		return
	}
	flattenOIDCClient(client, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OIDCClientResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan oidcClientModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	name := plan.Name.ValueString()
	isActive := plan.IsActive.ValueBool()
	client, err := r.client.Clients.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.UpdateClientParams{
		Name:                &name,
		RedirectURIs:        listToStrings(ctx, plan.RedirectURIs),
		PostLogoutRedirects: listToStrings(ctx, plan.PostLogoutRedirects),
		GrantTypes:          listToStrings(ctx, plan.GrantTypes),
		IsActive:            &isActive,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating OIDC client", err.Error())
		return
	}
	flattenOIDCClient(client, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OIDCClientResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state oidcClientModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Clients.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting OIDC client", err.Error())
	}
}

func (r *OIDCClientResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<client_uuid>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenOIDCClient(c *clavex.OIDCClient, m *oidcClientModel) {
	m.ID = types.StringValue(c.ClientID) // ClientID is the PK in the API
	m.OrgID = types.StringValue(c.OrgID)
	m.ClientID = types.StringValue(c.ClientID)
	m.Name = types.StringValue(c.Name)
	m.RedirectURIs = stringsToList(c.RedirectURIs)
	m.PostLogoutRedirects = stringsToList(c.PostLogoutRedirectURIs)
	m.GrantTypes = stringsToList(c.GrantTypes)
	m.IsPublic = types.BoolValue(c.IsPublic)
	m.IsActive = types.BoolValue(c.IsActive)
}
