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

var _ resource.Resource = &PAMCredentialResource{}
var _ resource.ResourceWithConfigure = &PAMCredentialResource{}
var _ resource.ResourceWithImportState = &PAMCredentialResource{}

// PAMCredentialResource manages a clavex_pam_credential. The secret is
// write-only and never returned by the API.
type PAMCredentialResource struct{ client *clavex.Client }

func NewPAMCredentialResource() resource.Resource { return &PAMCredentialResource{} }

type pamCredentialModel struct {
	ID                   types.String `tfsdk:"id"`
	OrgID                types.String `tfsdk:"org_id"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	CredentialType       types.String `tfsdk:"credential_type"`
	Username             types.String `tfsdk:"username"`
	Secret               types.String `tfsdk:"secret"`
	TargetHost           types.String `tfsdk:"target_host"`
	CheckoutDuration     types.Int64  `tfsdk:"checkout_duration"`
	RequireAccessRequest types.Bool   `tfsdk:"require_access_request"`
	IsActive             types.Bool   `tfsdk:"is_active"`
	RotationIntervalDays types.Int64  `tfsdk:"rotation_interval_days"`
}

func (r *PAMCredentialResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_pam_credential"
}

func (r *PAMCredentialResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a vaulted privileged credential. The secret is write-only.",
		Attributes: map[string]schema.Attribute{
			"id":                     schema.StringAttribute{Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"org_id":                 schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
			"name":                   schema.StringAttribute{Required: true},
			"description":            schema.StringAttribute{Optional: true},
			"credential_type":        schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Description: "e.g. password | ssh_key | api_key."},
			"username":               schema.StringAttribute{Optional: true},
			"secret":                 schema.StringAttribute{Required: true, Sensitive: true, Description: "Write-only secret material."},
			"target_host":            schema.StringAttribute{Optional: true},
			"checkout_duration":      schema.Int64Attribute{Required: true, Description: "Checkout lease duration in seconds."},
			"require_access_request": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
			"is_active":              schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
			"rotation_interval_days": schema.Int64Attribute{Optional: true},
		},
	}
}

func (r *PAMCredentialResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func intAttrToPtr(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := int(v.ValueInt64())
	return &n
}

func (r *PAMCredentialResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan pamCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	cred, err := r.client.PAM.CreateCredential(ctx, plan.OrgID.ValueString(), clavex.CreateCredentialParams{
		Name:                 plan.Name.ValueString(),
		Description:          strAttrToPtr(plan.Description),
		CredentialType:       plan.CredentialType.ValueString(),
		Username:             strAttrToPtr(plan.Username),
		Secret:               plan.Secret.ValueString(),
		TargetHost:           strAttrToPtr(plan.TargetHost),
		CheckoutDuration:     int(plan.CheckoutDuration.ValueInt64()),
		RequireAccessRequest: plan.RequireAccessRequest.ValueBool(),
		RotationIntervalDays: intAttrToPtr(plan.RotationIntervalDays),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error creating PAM credential", err.Error())
		return
	}
	flattenPAMCredential(cred, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PAMCredentialResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state pamCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	creds, err := r.client.PAM.ListCredentials(ctx, state.OrgID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing PAM credentials", err.Error())
		return
	}
	for i := range creds {
		if creds[i].ID == state.ID.ValueString() {
			flattenPAMCredential(&creds[i], &state)
			resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
			return
		}
	}
	resp.State.RemoveResource(ctx)
}

func (r *PAMCredentialResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan pamCredentialModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	isActive := plan.IsActive.ValueBool()
	requireAR := plan.RequireAccessRequest.ValueBool()
	checkout := int(plan.CheckoutDuration.ValueInt64())
	err := r.client.PAM.UpdateCredential(ctx, plan.OrgID.ValueString(), plan.ID.ValueString(), clavex.UpdateCredentialParams{
		Name:                 plan.Name.ValueStringPointer(),
		Description:          strAttrToPtr(plan.Description),
		Username:             strAttrToPtr(plan.Username),
		Secret:               strAttrToPtr(plan.Secret),
		TargetHost:           strAttrToPtr(plan.TargetHost),
		CheckoutDuration:     &checkout,
		RequireAccessRequest: &requireAR,
		IsActive:             &isActive,
		RotationIntervalDays: intAttrToPtr(plan.RotationIntervalDays),
	})
	if err != nil {
		resp.Diagnostics.AddError("Error updating PAM credential", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *PAMCredentialResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state pamCredentialModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.PAM.DeleteCredential(ctx, state.OrgID.ValueString(), state.ID.ValueString()); err != nil && !clavex.IsNotFound(err) {
		resp.Diagnostics.AddError("Error deleting PAM credential", err.Error())
	}
}

func (r *PAMCredentialResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parts, err := splitID(req.ID, 2)
	if err != nil {
		resp.Diagnostics.AddError("Invalid import ID", "Format: <org_id>/<credential_id>")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, orgIDPath(), types.StringValue(parts[0]))...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, idPath(), types.StringValue(parts[1]))...)
}

// flattenPAMCredential maps server fields to state, preserving the write-only secret.
func flattenPAMCredential(cred *clavex.PAMCredential, m *pamCredentialModel) {
	m.ID = types.StringValue(cred.ID)
	m.OrgID = types.StringValue(cred.OrgID)
	m.Name = types.StringValue(cred.Name)
	m.Description = ptrToStrAttr(cred.Description)
	m.CredentialType = types.StringValue(cred.CredentialType)
	m.Username = ptrToStrAttr(cred.Username)
	m.TargetHost = ptrToStrAttr(cred.TargetHost)
	m.CheckoutDuration = types.Int64Value(int64(cred.CheckoutDuration))
	m.RequireAccessRequest = types.BoolValue(cred.RequireAccessRequest)
	m.IsActive = types.BoolValue(cred.IsActive)
	if cred.RotationIntervalDays != nil {
		m.RotationIntervalDays = types.Int64Value(int64(*cred.RotationIntervalDays))
	}
}
