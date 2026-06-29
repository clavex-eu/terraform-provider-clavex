package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &OrganizationDataSource{}
var _ datasource.DataSourceWithConfigure = &OrganizationDataSource{}

type OrganizationDataSource struct{ client *clavex.Client }

func NewOrganizationDataSource() datasource.DataSource { return &OrganizationDataSource{} }

type organizationDataModel struct {
	ID          types.String `tfsdk:"id"`
	Slug        types.String `tfsdk:"slug"`
	Name        types.String `tfsdk:"name"`
	LogoURL     types.String `tfsdk:"logo_url"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	MFARequired types.Bool   `tfsdk:"mfa_required"`
}

func (d *OrganizationDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (d *OrganizationDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Looks up a Clavex organisation by ID or slug.",
		Attributes: map[string]dsschema.Attribute{
			"id":           dsschema.StringAttribute{Optional: true, Computed: true, Description: "UUID — used when provided as a filter."},
			"slug":         dsschema.StringAttribute{Optional: true, Computed: true, Description: "URL-safe slug — used when provided as a filter."},
			"name":         dsschema.StringAttribute{Computed: true},
			"logo_url":     dsschema.StringAttribute{Computed: true},
			"is_active":    dsschema.BoolAttribute{Computed: true},
			"mfa_required": dsschema.BoolAttribute{Computed: true},
		},
	}
}

func (d *OrganizationDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *OrganizationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state organizationDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if !state.ID.IsNull() && state.ID.ValueString() != "" {
		org, err := d.client.Organizations.Get(ctx, state.ID.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Error reading organisation", err.Error())
			return
		}
		orgToDataModel(org, &state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	if state.Slug.IsNull() || state.Slug.ValueString() == "" {
		resp.Diagnostics.AddError("Missing filter", "Provide either id or slug.")
		return
	}
	orgs, err := d.client.Organizations.List(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing organisations", err.Error())
		return
	}
	for _, o := range orgs {
		if o.Slug == state.Slug.ValueString() {
			orgToDataModel(&o, &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddError("Organisation not found", fmt.Sprintf("No organisation with slug %q.", state.Slug.ValueString()))
}

func orgToDataModel(o *clavex.Organization, m *organizationDataModel) {
	m.ID = types.StringValue(o.ID)
	m.Slug = types.StringValue(o.Slug)
	m.Name = types.StringValue(o.Name)
	m.LogoURL = ptrToStrAttr(o.LogoURL)
	m.IsActive = types.BoolValue(o.IsActive)
	m.MFARequired = types.BoolValue(o.MFARequired)
}
