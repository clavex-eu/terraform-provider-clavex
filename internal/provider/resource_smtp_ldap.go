package provider

import (
	"context"
	"fmt"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ── clavex_smtp_config ────────────────────────────────────────────────────────

var _ resource.Resource = &SMTPConfigResource{}
var _ resource.ResourceWithConfigure = &SMTPConfigResource{}
var _ resource.ResourceWithImportState = &SMTPConfigResource{}

type SMTPConfigResource struct{ client *clavex.Client }

func NewSMTPConfigResource() resource.Resource { return &SMTPConfigResource{} }

type smtpConfigModel struct {
	ID       types.String `tfsdk:"id"`
	OrgID    types.String `tfsdk:"org_id"`
	Host     types.String `tfsdk:"host"`
	Port     types.Int64  `tfsdk:"port"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	UseTLS   types.Bool   `tfsdk:"use_tls"`
	FromName types.String `tfsdk:"from_name"`
	FromAddr types.String `tfsdk:"from_addr"`
}

func (r *SMTPConfigResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_smtp_config"
}

func (r *SMTPConfigResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the outbound SMTP settings for a Clavex organisation (singleton per org).",
		Attributes: map[string]schema.Attribute{
			"id":        schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":    schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"host":      schema.StringAttribute{Required: true},
			"port":      schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(587)},
			"username":  schema.StringAttribute{Optional: true},
			"password":  schema.StringAttribute{Optional: true, Sensitive: true},
			"use_tls":   schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"from_name": schema.StringAttribute{Optional: true},
			"from_addr": schema.StringAttribute{Required: true},
		},
	}
}

func (r *SMTPConfigResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SMTPConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan smtpConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.SMTP.Put(ctx, plan.OrgID.ValueString(), smtpFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error setting SMTP config", err.Error())
		return
	}
	flattenSMTP(cfg, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SMTPConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state smtpConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.SMTP.Get(ctx, state.OrgID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading SMTP config", err.Error())
		return
	}
	flattenSMTP(cfg, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *SMTPConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan smtpConfigModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cfg, err := r.client.SMTP.Put(ctx, plan.OrgID.ValueString(), smtpFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating SMTP config", err.Error())
		return
	}
	flattenSMTP(cfg, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *SMTPConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// SMTP has no DELETE; reset to empty defaults.
	var state smtpConfigModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.SMTP.Put(ctx, state.OrgID.ValueString(), clavex.SMTPConfig{}); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error removing SMTP config", err.Error())
	}
}

func (r *SMTPConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(req.ID))...)
}

func smtpFromModel(m smtpConfigModel) clavex.SMTPConfig {
	return clavex.SMTPConfig{
		Host:     m.Host.ValueString(),
		Port:     int(m.Port.ValueInt64()),
		Username: m.Username.ValueString(),
		UseTLS:   m.UseTLS.ValueBool(),
		FromName: m.FromName.ValueString(),
		FromAddr: m.FromAddr.ValueString(),
	}
}

func flattenSMTP(s *clavex.SMTPConfig, m *smtpConfigModel) {
	m.ID = types.StringValue(s.OrgID)
	m.OrgID = types.StringValue(s.OrgID)
	m.Host = types.StringValue(s.Host)
	m.Port = types.Int64Value(int64(s.Port))
	m.Username = types.StringValue(s.Username)
	m.UseTLS = types.BoolValue(s.UseTLS)
	m.FromName = types.StringValue(s.FromName)
	m.FromAddr = types.StringValue(s.FromAddr)
	// password is write-only
}

// ── clavex_ldap_connection ────────────────────────────────────────────────────

var _ resource.Resource = &LDAPConnectionResource{}
var _ resource.ResourceWithConfigure = &LDAPConnectionResource{}
var _ resource.ResourceWithImportState = &LDAPConnectionResource{}

type LDAPConnectionResource struct{ client *clavex.Client }

func NewLDAPConnectionResource() resource.Resource { return &LDAPConnectionResource{} }

type ldapConnectionModel struct {
	ID           types.String `tfsdk:"id"`
	OrgID        types.String `tfsdk:"org_id"`
	Name         types.String `tfsdk:"name"`
	Host         types.String `tfsdk:"host"`
	Port         types.Int64  `tfsdk:"port"`
	UseTLS       types.Bool   `tfsdk:"use_tls"`
	BindDN       types.String `tfsdk:"bind_dn"`
	BindPassword types.String `tfsdk:"bind_password"`
	BaseDN       types.String `tfsdk:"base_dn"`
	UserFilter   types.String `tfsdk:"user_filter"`
	IsActive     types.Bool   `tfsdk:"is_active"`
}

func (r *LDAPConnectionResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ldap_connection"
}

func (r *LDAPConnectionResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an LDAP directory federation connection for a Clavex organisation.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id": schema.StringAttribute{
				Required:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"name":          schema.StringAttribute{Required: true},
			"host":          schema.StringAttribute{Required: true},
			"port":          schema.Int64Attribute{Optional: true, Computed: true, Default: int64default.StaticInt64(389)},
			"use_tls":       schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"bind_dn":       schema.StringAttribute{Required: true},
			"bind_password": schema.StringAttribute{Required: true, Sensitive: true},
			"base_dn":       schema.StringAttribute{Required: true},
			"user_filter":   schema.StringAttribute{Optional: true, Computed: true, Description: "LDAP filter for user objects (e.g. \"(objectClass=person)\")."},
			"is_active":     schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		},
	}
}

func (r *LDAPConnectionResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *LDAPConnectionResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan ldapConnectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := r.client.LDAP.Create(ctx, plan.OrgID.ValueString(), clavex.CreateLDAPParams{
		Name:         plan.Name.ValueString(),
		Host:         plan.Host.ValueString(),
		Port:         int(plan.Port.ValueInt64()),
		UseTLS:       plan.UseTLS.ValueBool(),
		BindDN:       plan.BindDN.ValueString(),
		BindPassword: plan.BindPassword.ValueString(),
		UsersDN:      plan.BaseDN.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating LDAP connection", err.Error())
		return
	}
	flattenLDAP(conn, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LDAPConnectionResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state ldapConnectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := r.client.LDAP.Get(ctx, state.OrgID.ValueString(), state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading LDAP connection", err.Error())
		return
	}
	flattenLDAP(conn, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *LDAPConnectionResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan ldapConnectionModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	conn, err := r.client.LDAP.Update(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.CreateLDAPParams{
		Name:         plan.Name.ValueString(),
		Host:         plan.Host.ValueString(),
		Port:         int(plan.Port.ValueInt64()),
		UseTLS:       plan.UseTLS.ValueBool(),
		BindDN:       plan.BindDN.ValueString(),
		BindPassword: plan.BindPassword.ValueString(),
		UsersDN:      plan.BaseDN.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating LDAP connection", err.Error())
		return
	}
	flattenLDAP(conn, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *LDAPConnectionResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state ldapConnectionModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.LDAP.Delete(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting LDAP connection", err.Error())
	}
}

func (r *LDAPConnectionResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<ldap_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

func flattenLDAP(conn *clavex.LDAPConnection, m *ldapConnectionModel) {
	m.ID = types.StringValue(conn.ID)
	m.OrgID = types.StringValue(conn.OrgID)
	m.Name = types.StringValue(conn.Name)
	m.Host = types.StringValue(conn.Host)
	m.Port = types.Int64Value(int64(conn.Port))
	m.UseTLS = types.BoolValue(conn.UseTLS)
	m.BindDN = ptrToStrAttr(conn.BindDN)
	m.BaseDN = types.StringValue(conn.BaseDN)
	m.UserFilter = types.StringValue(conn.UserFilter)
	m.IsActive = types.BoolValue(conn.IsActive)
	// bind_password is write-only
}
