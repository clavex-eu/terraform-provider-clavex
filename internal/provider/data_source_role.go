package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &RoleDataSource{}
var _ datasource.DataSourceWithConfigure = &RoleDataSource{}

// RoleDataSource reads an existing role by name within an organisation.
// Useful for referencing roles managed outside Terraform (e.g. seeded at deploy time).
// Use clavex_role to manage the role lifecycle.
type RoleDataSource struct{ client *clavex.Client }

func NewRoleDataSource() datasource.DataSource { return &RoleDataSource{} }

type roleDataModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	IsSystem    types.Bool   `tfsdk:"is_system"`
}

func (d *RoleDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_role"
}

func (d *RoleDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Read-only lookup of an existing Clavex role by name within an organisation. " +
			"Use clavex_role to manage the role lifecycle.",
		Attributes: map[string]dsschema.Attribute{
			"id":          dsschema.StringAttribute{Computed: true, Description: "UUID of the role."},
			"org_id":      dsschema.StringAttribute{Required: true, Description: "UUID of the owning organisation."},
			"name":        dsschema.StringAttribute{Required: true, Description: "Exact role name to look up."},
			"description": dsschema.StringAttribute{Computed: true, Description: "Optional human-readable description."},
			"is_system":   dsschema.BoolAttribute{Computed: true, Description: "True if this is a built-in system role."},
		},
	}
}

func (d *RoleDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *RoleDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state roleDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	roles, err := d.client.Roles.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing roles", err.Error())
		return
	}

	for _, r := range roles {
		if r.Name != state.Name.ValueString() {
			continue
		}
		state.ID = types.StringValue(r.ID)
		state.Description = ptrToStrAttr(r.Description)
		state.IsSystem = types.BoolValue(r.IsSystem)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
		return
	}

	resp.Diagnostics.AddError("Role not found", fmt.Sprintf("No role named %q in org %q.", state.Name.ValueString(), state.OrgID.ValueString()))
}
