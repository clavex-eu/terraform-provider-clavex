// Package provider implements the Clavex Terraform provider.
// It exposes all Clavex management resources and data sources.
package provider

import (
	"context"
	"os"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ provider.Provider = &ClavexProvider{}

// ClavexProvider is the root Terraform provider struct.
type ClavexProvider struct{ version string }

// New returns a factory function used by providerserver.Serve.
func New(version string) func() provider.Provider {
	return func() provider.Provider { return &ClavexProvider{version: version} }
}

// providerModel mirrors the HCL provider block.
type providerModel struct {
	BaseURL  types.String `tfsdk:"base_url"`
	Email    types.String `tfsdk:"email"`
	Password types.String `tfsdk:"password"`
	OrgSlug  types.String `tfsdk:"org_slug"`
	Token    types.String `tfsdk:"token"`
}

func (p *ClavexProvider) Metadata(_ context.Context, _ provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "clavex"
	resp.Version = p.version
}

func (p *ClavexProvider) Schema(_ context.Context, _ provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The Clavex provider manages users, clients, roles, groups and all other " +
			"resources exposed by the Clavex IAM REST API.",
		Attributes: map[string]schema.Attribute{
			"base_url": schema.StringAttribute{
				Optional:    true,
				Description: "Base URL of the Clavex server (e.g. https://auth.example.com). Falls back to CLAVEX_BASE_URL.",
			},
			"email": schema.StringAttribute{
				Optional:    true,
				Description: "Admin e-mail for password-based login. Falls back to CLAVEX_EMAIL.",
			},
			"password": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Admin password. Falls back to CLAVEX_PASSWORD.",
			},
			"org_slug": schema.StringAttribute{
				Optional:    true,
				Description: "Organisation slug used during login (leave empty for superadmin). Falls back to CLAVEX_ORG_SLUG.",
			},
			"token": schema.StringAttribute{
				Optional:    true,
				Sensitive:   true,
				Description: "Pre-acquired JWT. Supersedes email/password when set. Falls back to CLAVEX_TOKEN.",
			},
		},
	}
}

func (p *ClavexProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var cfg providerModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}

	baseURL := envOrVal(cfg.BaseURL, "CLAVEX_BASE_URL")
	email := envOrVal(cfg.Email, "CLAVEX_EMAIL")
	password := envOrVal(cfg.Password, "CLAVEX_PASSWORD")
	orgSlug := envOrVal(cfg.OrgSlug, "CLAVEX_ORG_SLUG")
	token := envOrVal(cfg.Token, "CLAVEX_TOKEN")

	if baseURL == "" {
		resp.Diagnostics.AddError(
			"Missing base_url",
			"Set base_url in the provider block or the CLAVEX_BASE_URL environment variable.",
		)
		return
	}

	var opts []clavex.Option
	switch {
	case token != "":
		opts = append(opts, clavex.WithToken(token))
	case email != "" && password != "":
		opts = append(opts, clavex.WithCredentials(orgSlug, email, password))
	default:
		resp.Diagnostics.AddError(
			"Missing authentication",
			"Provide either token (or CLAVEX_TOKEN) or email+password (or CLAVEX_EMAIL+CLAVEX_PASSWORD).",
		)
		return
	}

	client, err := clavex.New(baseURL, opts...)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create Clavex client", err.Error())
		return
	}

	resp.ResourceData = client
	resp.DataSourceData = client
}

func (p *ClavexProvider) Resources(_ context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewOrganizationResource,
		NewUserResource,
		NewRoleResource,
		NewGroupResource,
		NewRoleAssignmentResource,
		NewGroupMemberResource,
		NewOIDCClientResource,
		NewIdentityProviderResource,
		NewWebhookResource,
		NewBrandingResource,
		NewPasswordPolicyResource,
		NewSMTPConfigResource,
		NewLDAPConnectionResource,
		NewInvitationResource,
		// Enterprise IaC additions
		NewOrgPolicyResource,
		NewSSFStreamResource,
		NewAuthZenPolicyResource,
		NewAPIKeyResource,
		NewCrossOrgTrustResource,
		// SDK-expansion resources
		NewClientScopeResource,
		NewProtocolMapperResource,
		NewOID4VCIConfigResource,
		NewMdocIssuerResource,
		NewMdocIACARootResource,
		NewFederationSubordinateResource,
		NewSCIMTokenResource,
		NewServiceAccountResource,
		NewLoginFlowResource,
		NewLifecycleRuleResource,
		NewSamlSpResource,
		NewWsfedRpResource,
		NewAppFamilyResource,
		NewAgentTokenResource,
		NewPAMCredentialResource,
	}
}

func (p *ClavexProvider) DataSources(_ context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewOrganizationDataSource,
		NewUserDataSource,
		NewClientDataSource,
		NewOrgPolicyDataSource,
		NewRoleDataSource,
	}
}

// envOrVal returns the terraform attribute value when set, otherwise reads the
// named environment variable. Both a null attr and an empty env var yield "".
func envOrVal(attr types.String, envKey string) string {
	if !attr.IsNull() && !attr.IsUnknown() && attr.ValueString() != "" {
		return attr.ValueString()
	}
	return os.Getenv(envKey)
}
