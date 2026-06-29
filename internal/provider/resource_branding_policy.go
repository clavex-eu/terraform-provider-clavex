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

// ── clavex_branding ───────────────────────────────────────────────────────────

var _ resource.Resource = &BrandingResource{}
var _ resource.ResourceWithConfigure = &BrandingResource{}
var _ resource.ResourceWithImportState = &BrandingResource{}

type BrandingResource struct{ client *clavex.Client }

func NewBrandingResource() resource.Resource { return &BrandingResource{} }

type brandingModel struct {
	ID              types.String `tfsdk:"id"`
	OrgID           types.String `tfsdk:"org_id"`
	PrimaryColor    types.String `tfsdk:"primary_color"`
	BackgroundColor types.String `tfsdk:"background_color"`
	LogoURL         types.String `tfsdk:"logo_url"`
	FaviconURL      types.String `tfsdk:"favicon_url"`
	CustomCSS       types.String `tfsdk:"custom_css"`
}

func (r *BrandingResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_branding"
}

func (r *BrandingResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the login-page branding for a Clavex organisation (singleton per org).",
		Attributes: map[string]schema.Attribute{
			"id":               schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":           schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"primary_color":    schema.StringAttribute{Optional: true, Description: "Hex colour code for primary UI elements (e.g. \"#1D9E75\")."},
			"background_color": schema.StringAttribute{Optional: true},
			"logo_url":         schema.StringAttribute{Optional: true},
			"favicon_url":      schema.StringAttribute{Optional: true},
			"custom_css":       schema.StringAttribute{Optional: true, Description: "Raw CSS injected into the login page."},
		},
	}
}

func (r *BrandingResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *BrandingResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan brandingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := r.client.Branding.Put(ctx, plan.OrgID.ValueString(), clavex.Branding{
		PrimaryColor:    strAttrToPtr(plan.PrimaryColor),
		BackgroundColor: strAttrToPtr(plan.BackgroundColor),
		LogoURL:         strAttrToPtr(plan.LogoURL),
		FaviconURL:      strAttrToPtr(plan.FaviconURL),
		CustomCSS:       strAttrToPtr(plan.CustomCSS),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error setting branding", err.Error())
		return
	}
	flattenBranding(b, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BrandingResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state brandingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := r.client.Branding.Get(ctx, state.OrgID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading branding", err.Error())
		return
	}
	flattenBranding(b, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *BrandingResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan brandingModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	b, err := r.client.Branding.Put(ctx, plan.OrgID.ValueString(), clavex.Branding{
		PrimaryColor:    strAttrToPtr(plan.PrimaryColor),
		BackgroundColor: strAttrToPtr(plan.BackgroundColor),
		LogoURL:         strAttrToPtr(plan.LogoURL),
		FaviconURL:      strAttrToPtr(plan.FaviconURL),
		CustomCSS:       strAttrToPtr(plan.CustomCSS),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating branding", err.Error())
		return
	}
	flattenBranding(b, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *BrandingResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Branding cannot be fully deleted — reset to defaults.
	var state brandingModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.Branding.Put(ctx, state.OrgID.ValueString(), clavex.Branding{}); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error resetting branding", err.Error())
	}
}

func (r *BrandingResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Import ID is the org_id.
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(req.ID))...)
}

func flattenBranding(b *clavex.Branding, m *brandingModel) {
	m.ID = types.StringValue(b.OrgID)
	m.OrgID = types.StringValue(b.OrgID)
	m.PrimaryColor = ptrToStrAttr(b.PrimaryColor)
	m.BackgroundColor = ptrToStrAttr(b.BackgroundColor)
	m.LogoURL = ptrToStrAttr(b.LogoURL)
	m.FaviconURL = ptrToStrAttr(b.FaviconURL)
	m.CustomCSS = ptrToStrAttr(b.CustomCSS)
}

// ── clavex_password_policy ────────────────────────────────────────────────────

var _ resource.Resource = &PasswordPolicyResource{}
var _ resource.ResourceWithConfigure = &PasswordPolicyResource{}
var _ resource.ResourceWithImportState = &PasswordPolicyResource{}

type PasswordPolicyResource struct{ client *clavex.Client }

func NewPasswordPolicyResource() resource.Resource { return &PasswordPolicyResource{} }

type passwordPolicyModel struct {
	ID             types.String `tfsdk:"id"`
	OrgID          types.String `tfsdk:"org_id"`
	MinLength      types.Int64  `tfsdk:"min_length"`
	RequireUpper   types.Bool   `tfsdk:"require_upper"`
	RequireLower   types.Bool   `tfsdk:"require_lower"`
	RequireNumber  types.Bool   `tfsdk:"require_number"`
	RequireSpecial types.Bool   `tfsdk:"require_special"`
	MaxAgeDays     types.Int64  `tfsdk:"max_age_days"`
	HistoryCount   types.Int64  `tfsdk:"history_count"`
}

func (r *PasswordPolicyResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_password_policy"
}

func (r *PasswordPolicyResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages the password policy for a Clavex organisation (singleton per org).",
		Attributes: map[string]schema.Attribute{
			"id":              schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":          schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"min_length":      schema.Int64Attribute{Optional: true, Computed: true},
			"require_upper":   schema.BoolAttribute{Optional: true, Computed: true},
			"require_lower":   schema.BoolAttribute{Optional: true, Computed: true},
			"require_number":  schema.BoolAttribute{Optional: true, Computed: true},
			"require_special": schema.BoolAttribute{Optional: true, Computed: true},
			"max_age_days":    schema.Int64Attribute{Optional: true, Computed: true, Description: "0 = no expiry."},
			"history_count":   schema.Int64Attribute{Optional: true, Computed: true, Description: "Number of previous passwords to remember."},
		},
	}
}

func (r *PasswordPolicyResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *PasswordPolicyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan passwordPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pp, err := r.client.PasswordPolicy.Put(ctx, plan.OrgID.ValueString(), ppFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error setting password policy", err.Error())
		return
	}
	flattenPP(pp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PasswordPolicyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state passwordPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pp, err := r.client.PasswordPolicy.Get(ctx, state.OrgID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading password policy", err.Error())
		return
	}
	flattenPP(pp, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *PasswordPolicyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan passwordPolicyModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	pp, err := r.client.PasswordPolicy.Put(ctx, plan.OrgID.ValueString(), ppFromModel(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating password policy", err.Error())
		return
	}
	flattenPP(pp, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PasswordPolicyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Reset to defaults.
	var state passwordPolicyModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if _, err := r.client.PasswordPolicy.Put(ctx, state.OrgID.ValueString(), clavex.PasswordPolicy{MinLength: 8}); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error resetting password policy", err.Error())
	}
}

func (r *PasswordPolicyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(req.ID))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(req.ID))...)
}

func ppFromModel(m passwordPolicyModel) clavex.PasswordPolicy {
	return clavex.PasswordPolicy{
		MinLength:      int(m.MinLength.ValueInt64()),
		RequireUpper:   m.RequireUpper.ValueBool(),
		RequireLower:   m.RequireLower.ValueBool(),
		RequireNumber:  m.RequireNumber.ValueBool(),
		RequireSpecial: m.RequireSpecial.ValueBool(),
		MaxAgeDays:     int(m.MaxAgeDays.ValueInt64()),
		HistoryCount:   int(m.HistoryCount.ValueInt64()),
	}
}

func flattenPP(pp *clavex.PasswordPolicy, m *passwordPolicyModel) {
	m.ID = types.StringValue(pp.OrgID)
	m.OrgID = types.StringValue(pp.OrgID)
	m.MinLength = types.Int64Value(int64(pp.MinLength))
	m.RequireUpper = types.BoolValue(pp.RequireUpper)
	m.RequireLower = types.BoolValue(pp.RequireLower)
	m.RequireNumber = types.BoolValue(pp.RequireNumber)
	m.RequireSpecial = types.BoolValue(pp.RequireSpecial)
	m.MaxAgeDays = types.Int64Value(int64(pp.MaxAgeDays))
	m.HistoryCount = types.Int64Value(int64(pp.HistoryCount))
}
