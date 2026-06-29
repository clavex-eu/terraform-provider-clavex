package provider

import (
	"fmt"
	"net/http"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
)

// ── helpers ───────────────────────────────────────────────────────────────────

const testTrustedOrgID = "ffffffff-bbbb-cccc-dddd-000000000099"

func mockTrustResponse(id string) map[string]any {
	return map[string]any{
		"id":               id,
		"org_id":           testOrgID,
		"trusted_org_slug": "partner",
		"trusted_org_id":   testTrustedOrgID,
		"allowed_scopes":   []string{"openid", "profile"},
		"created_at":       "2026-05-09T00:00:00Z",
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestCrossOrgTrustResource_Create(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/cross-org-trusts", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, mockTrustResponse(testTrustID))
		},
	})

	trust, err := client.CrossOrgTrust.Create(testCtx, testOrgID, clavex.CreateCrossOrgTrustParams{
		TrustedOrgSlug: "partner",
		AllowedScopes:  []string{"openid", "profile"},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if trust.ID != testTrustID {
		t.Errorf("id: want %q, got %q", testTrustID, trust.ID)
	}
	if trust.TrustedOrgSlug != "partner" {
		t.Errorf("trusted_org_slug: want partner, got %q", trust.TrustedOrgSlug)
	}
}

// ── Read / List ───────────────────────────────────────────────────────────────

func TestCrossOrgTrustResource_List(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/cross-org-trusts", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{mockTrustResponse(testTrustID)})
		},
	})

	trusts, err := client.CrossOrgTrust.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(trusts) != 1 {
		t.Fatalf("List: want 1, got %d", len(trusts))
	}
	if trusts[0].TrustedOrgID != testTrustedOrgID {
		t.Errorf("trusted_org_id: want %q, got %q", testTrustedOrgID, trusts[0].TrustedOrgID)
	}
}

// ── Inbound list ──────────────────────────────────────────────────────────────

func TestCrossOrgTrustResource_ListInbound(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/cross-org-trusts/inbound", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{mockTrustResponse(testTrustID)})
		},
	})

	inbound, err := client.CrossOrgTrust.ListInbound(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("ListInbound: %v", err)
	}
	if len(inbound) != 1 {
		t.Fatalf("ListInbound: want 1, got %d", len(inbound))
	}
}

// ── Revoke (Delete) ───────────────────────────────────────────────────────────

func TestCrossOrgTrustResource_Revoke(t *testing.T) {
	revoked := false
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("DELETE /api/v1/organizations/%s/cross-org-trusts/%s", testOrgID, testTrustID): func(w http.ResponseWriter, r *http.Request) {
			revoked = true
			writeStatus(w, http.StatusNoContent)
		},
	})

	if err := client.CrossOrgTrust.Revoke(testCtx, testOrgID, testTrustID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !revoked {
		t.Error("Revoke handler was not called")
	}
}

// ── CRUD round-trip ───────────────────────────────────────────────────────────

func TestCrossOrgTrustResource_CRUD(t *testing.T) {
	trustRecord := mockTrustResponse(testTrustID)
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/cross-org-trusts", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, trustRecord)
		},
		fmt.Sprintf("GET /api/v1/organizations/%s/cross-org-trusts", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{trustRecord})
		},
		fmt.Sprintf("DELETE /api/v1/organizations/%s/cross-org-trusts/%s", testOrgID, testTrustID): func(w http.ResponseWriter, r *http.Request) {
			writeStatus(w, http.StatusNoContent)
		},
	})

	// Create
	trust, err := client.CrossOrgTrust.Create(testCtx, testOrgID, clavex.CreateCrossOrgTrustParams{
		TrustedOrgSlug: "partner",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Read
	trusts, err := client.CrossOrgTrust.List(testCtx, testOrgID)
	if err != nil || len(trusts) == 0 {
		t.Fatalf("List: %v (len=%d)", err, len(trusts))
	}
	found := false
	for _, tr := range trusts {
		if tr.ID == trust.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Created trust %q not found in list", trust.ID)
	}

	// Cross-org trust has no Update (immutable; change forces replace).
	// Delete
	if err := client.CrossOrgTrust.Revoke(testCtx, testOrgID, trust.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}
