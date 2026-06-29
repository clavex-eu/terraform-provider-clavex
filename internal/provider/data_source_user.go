package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	dsschema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &UserDataSource{}
var _ datasource.DataSourceWithConfigure = &UserDataSource{}

type UserDataSource struct{ client *clavex.Client }

func NewUserDataSource() datasource.DataSource { return &UserDataSource{} }

type userDataModel struct {
	ID          types.String `tfsdk:"id"`
	OrgID       types.String `tfsdk:"org_id"`
	Email       types.String `tfsdk:"email"`
	FirstName   types.String `tfsdk:"first_name"`
	LastName    types.String `tfsdk:"last_name"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	MFARequired types.Bool   `tfsdk:"mfa_required"`
}

func (d *UserDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_user"
}

func (d *UserDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = dsschema.Schema{
		Description: "Looks up a Clavex user by e-mail within an organisation.",
		Attributes: map[string]dsschema.Attribute{
			"id":           dsschema.StringAttribute{Computed: true},
			"org_id":       dsschema.StringAttribute{Required: true},
			"email":        dsschema.StringAttribute{Required: true},
			"first_name":   dsschema.StringAttribute{Computed: true},
			"last_name":    dsschema.StringAttribute{Computed: true},
			"is_active":    dsschema.BoolAttribute{Computed: true},
			"mfa_required": dsschema.BoolAttribute{Computed: true},
		},
	}
}

func (d *UserDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *UserDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state userDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	users, err := d.client.Users.List(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing users", err.Error())
		return
	}
	for _, u := range users {
		if u.Email == state.Email.ValueString() {
			state.ID = types.StringValue(u.ID)
			state.FirstName = ptrToStrAttr(u.FirstName)
			state.LastName = ptrToStrAttr(u.LastName)
			state.IsActive = types.BoolValue(u.IsActive)
			state.MFARequired = types.BoolValue(u.MFARequired)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.Diagnostics.AddError("User not found", fmt.Sprintf("No user with email %q in org %q.", state.Email.ValueString(), state.OrgID.ValueString()))
}
