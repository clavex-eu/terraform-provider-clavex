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

var _ resource.Resource = &OrganizationResource{}
var _ resource.ResourceWithConfigure = &OrganizationResource{}
var _ resource.ResourceWithImportState = &OrganizationResource{}

// OrganizationResource manages a clavex_organization.
type OrganizationResource struct{ client *clavex.Client }

func NewOrganizationResource() resource.Resource { return &OrganizationResource{} }

type organizationModel struct {
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Slug        types.String `tfsdk:"slug"`
	LogoURL     types.String `tfsdk:"logo_url"`
	IsActive    types.Bool   `tfsdk:"is_active"`
	MFARequired types.Bool   `tfsdk:"mfa_required"`
}

func (r *OrganizationResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_organization"
}

func (r *OrganizationResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Clavex organisation (tenant). Requires superadmin privileges.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Computed:    true,
				Description: "UUID of the organisation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{Required: true, Description: "Display name."},
			"slug": schema.StringAttribute{
				Required:    true,
				Description: "URL-safe identifier used in auth flows. Immutable after creation.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"logo_url":     schema.StringAttribute{Optional: true, Description: "URL of the organisation logo."},
			"is_active":    schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Whether the organisation is enabled."},
			"mfa_required": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Enforce MFA for all users."},
		},
	}
}

func (r *OrganizationResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *OrganizationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan organizationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.Organizations.Create(ctx, clavex.CreateOrgParams{
		Name:    plan.Name.ValueString(),
		Slug:    plan.Slug.ValueString(),
		LogoURL: strAttrToPtr(plan.LogoURL),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating organisation", err.Error())
		return
	}
	flattenOrg(org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state organizationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	org, err := r.client.Organizations.Get(ctx, state.ID.ValueString())
	if err != nil {
		if clavex.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading organisation", err.Error())
		return
	}
	flattenOrg(org, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *OrganizationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan organizationModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	name := plan.Name.ValueString()
	isActive := plan.IsActive.ValueBool()
	mfaReq := plan.MFARequired.ValueBool()

	org, err := r.client.Organizations.Update(ctx, plan.ID.ValueString(), clavex.UpdateOrgParams{
		Name:        &name,
		LogoURL:     strAttrToPtr(plan.LogoURL),
		IsActive:    &isActive,
		MFARequired: &mfaReq,
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating organisation", err.Error())
		return
	}
	flattenOrg(org, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *OrganizationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state organizationModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.Organizations.Delete(ctx, state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting organisation", err.Error())
	}
}

func (r *OrganizationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(req.ID))...)
}

func flattenOrg(o *clavex.Organization, m *organizationModel) {
	m.ID = types.StringValue(o.ID)
	m.Name = types.StringValue(o.Name)
	m.Slug = types.StringValue(o.Slug)
	m.LogoURL = ptrToStrAttr(o.LogoURL)
	m.IsActive = types.BoolValue(o.IsActive)
	m.MFARequired = types.BoolValue(o.MFARequired)
}
