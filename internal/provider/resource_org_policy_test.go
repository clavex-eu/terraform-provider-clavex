package provider

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
)

// ── helpers ───────────────────────────────────────────────────────────────────

func mockPolicyResponse(id string) map[string]any {
	return map[string]any{
		"id":       id,
		"org_id":   testOrgID,
		"name":     "block-russia",
		"priority": 10,
		"action":   "deny",
		"enabled":  true,
		"conditions": map[string]any{
			"country_not": []string{"RU"},
		},
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestOrgPolicyResource_Create(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["action"] != "deny" {
				t.Errorf("action: want deny, got %v", body["action"])
			}
			writeJSON(w, mockPolicyResponse(testPolicyID))
		},
	})

	rule, err := client.Policies.Create(testCtx, testOrgID, clavex.CreatePolicyRuleParams{
		Name:     "block-russia",
		Priority: 10,
		Action:   "deny",
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rule.ID != testPolicyID {
		t.Errorf("id: want %q, got %q", testPolicyID, rule.ID)
	}
	if rule.Action != "deny" {
		t.Errorf("action: want deny, got %q", rule.Action)
	}
}

// ── Read / List ───────────────────────────────────────────────────────────────

func TestOrgPolicyResource_Read(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{mockPolicyResponse(testPolicyID)})
		},
	})

	rules, err := client.Policies.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rules) != 1 {
		t.Fatalf("List: want 1 rule, got %d", len(rules))
	}
	if rules[0].ID != testPolicyID {
		t.Errorf("id: want %q, got %q", testPolicyID, rules[0].ID)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestOrgPolicyResource_Update(t *testing.T) {
	updated := mockPolicyResponse(testPolicyID)
	updated["action"] = "require_mfa"
	updated["enabled"] = false

	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("PUT /api/v1/organizations/%s/auth-policies/%s", testOrgID, testPolicyID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, updated)
		},
	})

	action := "require_mfa"
	enabled := false
	rule, err := client.Policies.Update(testCtx, testOrgID, testPolicyID, clavex.UpdatePolicyRuleParams{
		Action:  &action,
		Enabled: &enabled,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rule.Action != "require_mfa" {
		t.Errorf("action: want require_mfa, got %q", rule.Action)
	}
	if rule.Enabled {
		t.Error("enabled: want false after update")
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestOrgPolicyResource_Delete(t *testing.T) {
	deleted := false
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("DELETE /api/v1/organizations/%s/auth-policies/%s", testOrgID, testPolicyID): func(w http.ResponseWriter, r *http.Request) {
			deleted = true
			writeStatus(w, http.StatusNoContent)
		},
	})

	if err := client.Policies.Delete(testCtx, testOrgID, testPolicyID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Error("Delete handler was not called")
	}
}

// ── 404 after delete (resource removed from state) ───────────────────────────

func TestOrgPolicyResource_ReadAfterDelete(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			// Return empty list — resource was deleted externally.
			writeJSON(w, []any{})
		},
	})

	rules, err := client.Policies.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.ID == testPolicyID {
			found = true
			break
		}
	}
	// When not found, the resource should be removed from state (no error).
	if found {
		t.Error("deleted resource should not appear in list")
	}
}

// ── CRUD round-trip ───────────────────────────────────────────────────────────

func TestOrgPolicyResource_CRUD(t *testing.T) {
	state := mockPolicyResponse(testPolicyID)
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, state)
		},
		fmt.Sprintf("GET /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{state})
		},
		fmt.Sprintf("PUT /api/v1/organizations/%s/auth-policies/%s", testOrgID, testPolicyID): func(w http.ResponseWriter, r *http.Request) {
			state["action"] = "allow"
			writeJSON(w, state)
		},
		fmt.Sprintf("DELETE /api/v1/organizations/%s/auth-policies/%s", testOrgID, testPolicyID): func(w http.ResponseWriter, r *http.Request) {
			writeStatus(w, http.StatusNoContent)
		},
	})

	// Create
	rule, err := client.Policies.Create(testCtx, testOrgID, clavex.CreatePolicyRuleParams{
		Name: "block-russia", Action: "deny", Priority: 10,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}

	// Read
	rules, err := client.Policies.List(testCtx, testOrgID)
	if err != nil || len(rules) == 0 {
		t.Fatalf("Read: %v (len=%d)", err, len(rules))
	}

	// Update
	newAction := "allow"
	updated, err := client.Policies.Update(testCtx, testOrgID, rule.ID, clavex.UpdatePolicyRuleParams{
		Action: &newAction,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if updated.Action != "allow" {
		t.Errorf("Update action: want allow, got %q", updated.Action)
	}

	// Delete
	if err := client.Policies.Delete(testCtx, testOrgID, rule.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}
