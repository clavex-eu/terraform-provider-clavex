package provider

import (
	"fmt"
	"net/http"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func mockApiKeyResponse(id string) map[string]any {
	return map[string]any{
		"id":         id,
		"name":       "terraform",
		"prefix":     "cvx_sk_test",
		"expires_at": "2099-12-31T23:59:59Z",
		"created_at": "2026-05-09T00:00:00Z",
		"secret":     "cvx_sk_test_supersecret",
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestAPIKeyResource_Create(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		"POST /api/v1/superadmin/api-keys": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{
				"api_key": mockApiKeyResponse(testApiKeyID),
				"secret":  "cvx_sk_test_supersecret",
			})
		},
	})

	result, err := client.APIKeys.Create(testCtx, clavex.CreateAPIKeyParams{
		Name: "terraform",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.APIKey.ID != testApiKeyID {
		t.Errorf("id: want %q, got %q", testApiKeyID, result.APIKey.ID)
	}
	if result.Secret != "cvx_sk_test_supersecret" {
		t.Errorf("secret: want supersecret, got %q", result.Secret)
	}
}

// ── Read / List ───────────────────────────────────────────────────────────────

func TestAPIKeyResource_List(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		"GET /api/v1/superadmin/api-keys": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{mockApiKeyResponse(testApiKeyID)})
		},
	})

	keys, err := client.APIKeys.List(testCtx, "")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("List: want 1 key, got %d", len(keys))
	}
	if keys[0].ID != testApiKeyID {
		t.Errorf("id: want %q, got %q", testApiKeyID, keys[0].ID)
	}
	if keys[0].Prefix != "cvx_sk_test" {
		t.Errorf("prefix: want cvx_sk_test, got %q", keys[0].Prefix)
	}
}

// ── Revoke (Delete) ───────────────────────────────────────────────────────────

func TestAPIKeyResource_Revoke(t *testing.T) {
	revoked := false
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("DELETE /api/v1/superadmin/api-keys/%s", testApiKeyID): func(w http.ResponseWriter, r *http.Request) {
			revoked = true
			writeStatus(w, http.StatusNoContent)
		},
	})

	if err := client.APIKeys.Revoke(testCtx, testApiKeyID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if !revoked {
		t.Error("Revoke handler was not called")
	}
}

// ── CRUD round-trip ───────────────────────────────────────────────────────────

func TestAPIKeyResource_CRUD(t *testing.T) {
	keyRecord := mockApiKeyResponse(testApiKeyID)
	_, client := newMockServer(t, routeMap{
		"POST /api/v1/superadmin/api-keys": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{
				"api_key": keyRecord,
				"secret":  "cvx_sk_test_supersecret",
			})
		},
		"GET /api/v1/superadmin/api-keys": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{keyRecord})
		},
		fmt.Sprintf("DELETE /api/v1/superadmin/api-keys/%s", testApiKeyID): func(w http.ResponseWriter, r *http.Request) {
			writeStatus(w, http.StatusNoContent)
		},
	})

	// Create
	result, err := client.APIKeys.Create(testCtx, clavex.CreateAPIKeyParams{Name: "terraform"})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if result.Secret == "" {
		t.Error("Create: secret must be non-empty on creation")
	}

	// Read (via list, since there is no Get endpoint for API keys)
	keys, err := client.APIKeys.List(testCtx, "")
	if err != nil || len(keys) == 0 {
		t.Fatalf("List: %v (len=%d)", err, len(keys))
	}
	found := false
	for _, k := range keys {
		if k.ID == result.APIKey.ID {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Created key %q not found in list", result.APIKey.ID)
	}

	// Delete (Revoke)
	if err := client.APIKeys.Revoke(testCtx, result.APIKey.ID); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
}
