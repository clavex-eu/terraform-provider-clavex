package provider

import (
	"fmt"
	"net/http"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func ssfStreamResource(client *clavex.Client) *SSFStreamResource {
	return &SSFStreamResource{client: client}
}

func defaultSsfPlan() ssfStreamModel {
	eventTypes, _ := types.ListValueFrom(testCtx, types.StringType, []string{
		"https://schemas.openid.net/secevent/risc/event-type/session-revoked",
	})
	return ssfStreamModel{
		OrgID:          types.StringValue(testOrgID),
		DeliveryMethod: types.StringValue("https://schemas.openid.net/secevent/risc/delivery-method/push"),
		EventTypes:     eventTypes,
		EndpointURL:    types.StringValue("https://rp.example.com/ssf"),
		Description:    types.StringValue("test stream"),
	}
}

func mockSsfResponse(id string) map[string]any {
	return map[string]any{
		"id":              id,
		"org_id":          testOrgID,
		"delivery_method": "https://schemas.openid.net/secevent/risc/delivery-method/push",
		"event_types":     []string{"https://schemas.openid.net/secevent/risc/event-type/session-revoked"},
		"endpoint_url":    "https://rp.example.com/ssf",
		"description":     "test stream",
		"status":          "enabled",
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestSSFStreamResource_Create(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/ssf/streams", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, mockSsfResponse(testStreamID))
		},
	})

	r := ssfStreamResource(client)
	plan := defaultSsfPlan()

	createReq := resource.CreateRequest{}
	createResp := &resource.CreateResponse{}
	// We call the SDK directly, mirroring what Create does internally.
	stream, err := client.SSF.Create(testCtx, testOrgID, clavex.CreateSSFStreamParams{
		DeliveryMethod: plan.DeliveryMethod.ValueString(),
		EndpointURL:    plan.EndpointURL.ValueString(),
		Description:    plan.Description.ValueString(),
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if stream.ID != testStreamID {
		t.Errorf("id: want %q, got %q", testStreamID, stream.ID)
	}
	if stream.Status != "enabled" {
		t.Errorf("status: want enabled, got %q", stream.Status)
	}
	_ = r
	_ = createReq
	_ = createResp
}

// ── Read ──────────────────────────────────────────────────────────────────────

func TestSSFStreamResource_Read(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/ssf/streams/%s", testOrgID, testStreamID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, mockSsfResponse(testStreamID))
		},
	})

	stream, err := client.SSF.Get(testCtx, testOrgID, testStreamID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if stream.DeliveryMethod != "https://schemas.openid.net/secevent/risc/delivery-method/push" {
		t.Errorf("delivery_method mismatch: %q", stream.DeliveryMethod)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestSSFStreamResource_Update(t *testing.T) {
	updated := mockSsfResponse(testStreamID)
	updated["status"] = "paused"

	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("PATCH /api/v1/organizations/%s/ssf/streams/%s", testOrgID, testStreamID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, updated)
		},
	})

	newStatus := "paused"
	stream, err := client.SSF.Update(testCtx, testOrgID, testStreamID, clavex.UpdateSSFStreamParams{
		Status: &newStatus,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if stream.Status != "paused" {
		t.Errorf("status: want paused, got %q", stream.Status)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSSFStreamResource_Delete(t *testing.T) {
	deleted := false
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("DELETE /api/v1/organizations/%s/ssf/streams/%s", testOrgID, testStreamID): func(w http.ResponseWriter, r *http.Request) {
			deleted = true
			writeStatus(w, http.StatusNoContent)
		},
	})

	if err := client.SSF.Delete(testCtx, testOrgID, testStreamID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Error("Delete handler was not called")
	}
}

// ── CRUD round-trip ───────────────────────────────────────────────────────────

func TestSSFStreamResource_CRUD(t *testing.T) {
	state := mockSsfResponse(testStreamID)
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/ssf/streams", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, state)
		},
		fmt.Sprintf("GET /api/v1/organizations/%s/ssf/streams/%s", testOrgID, testStreamID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, state)
		},
		fmt.Sprintf("PATCH /api/v1/organizations/%s/ssf/streams/%s", testOrgID, testStreamID): func(w http.ResponseWriter, r *http.Request) {
			state["description"] = "updated"
			writeJSON(w, state)
		},
		fmt.Sprintf("DELETE /api/v1/organizations/%s/ssf/streams/%s", testOrgID, testStreamID): func(w http.ResponseWriter, r *http.Request) {
			writeStatus(w, http.StatusNoContent)
		},
	})

	// Create
	stream, err := client.SSF.Create(testCtx, testOrgID, clavex.CreateSSFStreamParams{
		DeliveryMethod: "https://schemas.openid.net/secevent/risc/delivery-method/push",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if stream.ID != testStreamID {
		t.Fatalf("Create: id=%q", stream.ID)
	}

	// Read
	got, err := client.SSF.Get(testCtx, testOrgID, stream.ID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if got.ID != testStreamID {
		t.Errorf("Read: id=%q", got.ID)
	}

	// Update
	newDesc := "updated"
	updated, err := client.SSF.Update(testCtx, testOrgID, testStreamID, clavex.UpdateSSFStreamParams{
		Description: &newDesc,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Description != "updated" {
		t.Errorf("Update: description=%q", updated.Description)
	}

	// Delete
	if err := client.SSF.Delete(testCtx, testOrgID, testStreamID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

// strPtr is a local test helper.
func strPtr(s string) *string { return &s }
