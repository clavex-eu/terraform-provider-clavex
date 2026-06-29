package provider

import (
	"fmt"
	"net/http"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
)

// AuthZenPolicyResource shares the Policies SDK service with OrgPolicyResource,
// but uses action_names → client_ids conditions to scope rules to specific
// AuthZen action.name values. These tests verify the correct API paths and
// payload structure used by the resource's CRUD methods.

// ── helpers ───────────────────────────────────────────────────────────────────

const testAuthZenPolicyID = "aaaaaaaa-bbbb-cccc-dddd-000000000006"

func mockAuthZenPolicyResponse(id string) map[string]any {
	return map[string]any{
		"id":       id,
		"org_id":   testOrgID,
		"name":     "allow-read-for-it",
		"priority": 20,
		"action":   "allow",
		"enabled":  true,
		"conditions": map[string]any{
			"client_id": []string{"read", "list"},
			"country":   []string{"IT", "DE"},
		},
	}
}

// ── Create ────────────────────────────────────────────────────────────────────

func TestAuthZenPolicyResource_Create(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, mockAuthZenPolicyResponse(testAuthZenPolicyID))
		},
	})

	rule, err := client.Policies.Create(testCtx, testOrgID, clavex.CreatePolicyRuleParams{
		Name:     "allow-read-for-it",
		Priority: 20,
		Action:   "allow",
		Conditions: clavex.PolicyConditions{
			ClientIDs: []string{"read", "list"},
			Countries: []string{"IT", "DE"},
		},
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if rule.ID != testAuthZenPolicyID {
		t.Errorf("id: want %q, got %q", testAuthZenPolicyID, rule.ID)
	}
	if rule.Action != "allow" {
		t.Errorf("action: want allow, got %q", rule.Action)
	}
}

// ── Decision = deny ───────────────────────────────────────────────────────────

func TestAuthZenPolicyResource_CreateDeny(t *testing.T) {
	resp := mockAuthZenPolicyResponse(testAuthZenPolicyID)
	resp["action"] = "deny"

	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("POST /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, resp)
		},
	})

	rule, err := client.Policies.Create(testCtx, testOrgID, clavex.CreatePolicyRuleParams{
		Name:   "deny-delete-worldwide",
		Action: "deny",
		Conditions: clavex.PolicyConditions{
			ClientIDs: []string{"delete"},
		},
	})
	if err != nil {
		t.Fatalf("Create deny: %v", err)
	}
	if rule.Action != "deny" {
		t.Errorf("action: want deny, got %q", rule.Action)
	}
}

// ── Read ──────────────────────────────────────────────────────────────────────

func TestAuthZenPolicyResource_Read(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{mockAuthZenPolicyResponse(testAuthZenPolicyID)})
		},
	})

	rules, err := client.Policies.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	found := false
	for _, r := range rules {
		if r.ID == testAuthZenPolicyID {
			found = true
			if r.Name != "allow-read-for-it" {
				t.Errorf("name: want allow-read-for-it, got %q", r.Name)
			}
		}
	}
	if !found {
		t.Errorf("policy %q not found in list", testAuthZenPolicyID)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestAuthZenPolicyResource_Update(t *testing.T) {
	updated := mockAuthZenPolicyResponse(testAuthZenPolicyID)
	updated["action"] = "deny"
	updated["priority"] = 5

	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("PUT /api/v1/organizations/%s/auth-policies/%s", testOrgID, testAuthZenPolicyID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, updated)
		},
	})

	newAction := "deny"
	newPriority := 5
	rule, err := client.Policies.Update(testCtx, testOrgID, testAuthZenPolicyID, clavex.UpdatePolicyRuleParams{
		Action:   &newAction,
		Priority: &newPriority,
	})
	if err != nil {
		t.Fatalf("Update: %v", err)
	}
	if rule.Action != "deny" {
		t.Errorf("action: want deny, got %q", rule.Action)
	}
	if rule.Priority != 5 {
		t.Errorf("priority: want 5, got %d", rule.Priority)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestAuthZenPolicyResource_Delete(t *testing.T) {
	deleted := false
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("DELETE /api/v1/organizations/%s/auth-policies/%s", testOrgID, testAuthZenPolicyID): func(w http.ResponseWriter, r *http.Request) {
			deleted = true
			writeStatus(w, http.StatusNoContent)
		},
	})

	if err := client.Policies.Delete(testCtx, testOrgID, testAuthZenPolicyID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if !deleted {
		t.Error("Delete handler was not called")
	}
}
