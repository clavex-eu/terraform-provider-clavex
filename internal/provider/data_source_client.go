package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &ClientDataSource{}
var _ datasource.DataSourceWithConfigure = &ClientDataSource{}

// ClientDataSource reads an existing OIDC/OAuth2 client by client_id.
// Use clavex_oidc_client to manage the client lifecycle.
type ClientDataSource struct{ client *clavex.Client }

func NewClientDataSource() datasource.DataSource { return &ClientDataSource{} }

type clientDataModel struct {
	ID                  types.String `tfsdk:"id"`
	OrgID               types.String `tfsdk:"org_id"`
	ClientID            types.String `tfsdk:"client_id"`
	Name                types.String `tfsdk:"name"`
	RedirectURIs        types.List   `tfsdk:"redirect_uris"`
	PostLogoutRedirects types.List   `tfsdk:"post_logout_redirect_uris"`
	GrantTypes          types.List   `tfsdk:"grant_types"`
	IsPublic            types.Bool   `tfsdk:"is_public"`
	IsActive            types.Bool   `tfsdk:"is_active"`
}

func (d *ClientDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_client"
}

func (d *ClientDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Read-only lookup of an existing Clavex OIDC/OAuth2 client by client_id. " +
			"Use clavex_oidc_client to manage the client lifecycle.",
		Attributes: map[string]dsschema.Attribute{
			"id":                        dsschema.StringAttribute{Computed: true, Description: "Internal UUID of the client record."},
			"org_id":                    dsschema.StringAttribute{Required: true, Description: "UUID of the owning organisation."},
			"client_id":                 dsschema.StringAttribute{Required: true, Description: "OAuth2 client_id to look up."},
			"name":                      dsschema.StringAttribute{Computed: true},
			"redirect_uris":             dsschema.ListAttribute{Computed: true, ElementType: types.StringType},
			"post_logout_redirect_uris": dsschema.ListAttribute{Computed: true, ElementType: types.StringType},
			"grant_types":               dsschema.ListAttribute{Computed: true, ElementType: types.StringType},
			"is_public":                 dsschema.BoolAttribute{Computed: true},
			"is_active":                 dsschema.BoolAttribute{Computed: true},
		},
	}
}

func (d *ClientDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*clavex.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected provider data", fmt.Sprintf("Expected *clavex.Client, got %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *ClientDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state clientDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	c, err := d.client.Clients.Get(ctx, state.OrgID.ValueString(), state.ClientID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading OIDC client", err.Error())
		return
	}

	state.ID = types.StringValue(c.ClientID)
	state.Name = types.StringValue(c.Name)
	state.IsPublic = types.BoolValue(c.IsPublic)
	state.IsActive = types.BoolValue(c.IsActive)

	redirectURIs, diags := types.ListValueFrom(ctx, types.StringType, c.RedirectURIs)
	resp.Diagnostics.Append(diags...)
	state.RedirectURIs = redirectURIs

	postLogout, diags := types.ListValueFrom(ctx, types.StringType, c.PostLogoutRedirectURIs)
	resp.Diagnostics.Append(diags...)
	state.PostLogoutRedirects = postLogout

	grantTypes, diags := types.ListValueFrom(ctx, types.StringType, c.GrantTypes)
	resp.Diagnostics.Append(diags...)
	state.GrantTypes = grantTypes

	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
