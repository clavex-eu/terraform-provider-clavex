package provider

import (
	"fmt"
	"net/http"
	"testing"
)

// ── data.clavex_organization ──────────────────────────────────────────────────

func TestOrganizationDataSource_ByID(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{
				"id":           testOrgID,
				"slug":         "acme",
				"name":         "ACME Corp",
				"logo_url":     nil,
				"is_active":    true,
				"mfa_required": false,
			})
		},
	})

	org, err := client.Organizations.Get(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if org.ID != testOrgID {
		t.Errorf("id: want %q, got %q", testOrgID, org.ID)
	}
	if org.Slug != "acme" {
		t.Errorf("slug: want acme, got %q", org.Slug)
	}
}

func TestOrganizationDataSource_BySlug(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		"GET /api/v1/organizations": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{
				map[string]any{"id": testOrgID, "slug": "acme", "name": "ACME Corp", "is_active": true, "mfa_required": false},
				map[string]any{"id": "other-id", "slug": "other", "name": "Other", "is_active": true, "mfa_required": false},
			})
		},
	})

	orgs, err := client.Organizations.List(testCtx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, o := range orgs {
		if o.Slug == "acme" {
			found = true
			if o.ID != testOrgID {
				t.Errorf("org id: want %q, got %q", testOrgID, o.ID)
			}
		}
	}
	if !found {
		t.Error("slug acme not found in list")
	}
}

func TestOrganizationDataSource_SlugNotFound(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		"GET /api/v1/organizations": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{}) // empty list — slug not found
		},
	})

	orgs, err := client.Organizations.List(testCtx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, o := range orgs {
		if o.Slug == "nonexistent" {
			t.Error("should not find nonexistent slug")
		}
	}
}

// ── data.clavex_user ──────────────────────────────────────────────────────────

func TestUserDataSource_ByEmail(t *testing.T) {
	const testUserID = "cccccccc-dddd-eeee-ffff-000000000001"
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/users", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{
				map[string]any{
					"id":           testUserID,
					"org_id":       testOrgID,
					"email":        "alice@example.com",
					"first_name":   "Alice",
					"last_name":    "Smith",
					"is_active":    true,
					"mfa_required": false,
				},
				map[string]any{
					"id":        "other-user",
					"org_id":    testOrgID,
					"email":     "bob@example.com",
					"is_active": true,
				},
			})
		},
	})

	users, err := client.Users.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, u := range users {
		if u.Email == "alice@example.com" {
			found = true
			if u.ID != testUserID {
				t.Errorf("id: want %q, got %q", testUserID, u.ID)
			}
		}
	}
	if !found {
		t.Error("user alice@example.com not found")
	}
}

func TestUserDataSource_EmailNotFound(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/users", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{}) // empty
		},
	})

	users, err := client.Users.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, u := range users {
		if u.Email == "missing@example.com" {
			t.Error("should not find missing email")
		}
	}
}

// ── data.clavex_client ────────────────────────────────────────────────────────

func TestClientDataSource_ByClientID(t *testing.T) {
	const clientID = "my-spa"
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/clients/%s", testOrgID, clientID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{
				"client_id":                 clientID,
				"org_id":                    testOrgID,
				"name":                      "My SPA",
				"redirect_uris":             []string{"https://app.example.com/callback"},
				"post_logout_redirect_uris": []string{},
				"grant_types":               []string{"authorization_code"},
				"is_public":                 true,
				"is_active":                 true,
			})
		},
	})

	c, err := client.Clients.Get(testCtx, testOrgID, clientID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if c.ClientID != clientID {
		t.Errorf("client_id: want %q, got %q", clientID, c.ClientID)
	}
	if c.Name != "My SPA" {
		t.Errorf("name: want My SPA, got %q", c.Name)
	}
	if !c.IsPublic {
		t.Error("is_public: want true")
	}
	if len(c.RedirectURIs) != 1 {
		t.Errorf("redirect_uris: want 1, got %d", len(c.RedirectURIs))
	}
}

func TestClientDataSource_ClientNotFound(t *testing.T) {
	const clientID = "nonexistent-client"
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/clients/%s", testOrgID, clientID): func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			writeJSON(w, map[string]any{"error": "client not found"})
		},
	})

	_, err := client.Clients.Get(testCtx, testOrgID, clientID)
	if err == nil {
		t.Error("expected error for missing client, got nil")
	}
}

// ── data.clavex_org_policy ───────────────────────────────────────────────────

func TestOrgPolicyDataSource_ByName(t *testing.T) {
	const ruleID = "rule-aaaa-bbbb-cccc-000000000001"
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{
				map[string]any{
					"id":       ruleID,
					"org_id":   testOrgID,
					"name":     "block-china",
					"priority": 10,
					"action":   "deny",
					"enabled":  true,
					"conditions": map[string]any{
						"country_not": []string{"CN"},
					},
				},
				map[string]any{
					"id": "other-rule", "org_id": testOrgID, "name": "allow-all",
					"priority": 100, "action": "allow", "enabled": true, "conditions": map[string]any{},
				},
			})
		},
	})

	rules, err := client.Policies.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, r := range rules {
		if r.Name == "block-china" {
			found = true
			if r.ID != ruleID {
				t.Errorf("id: want %q, got %q", ruleID, r.ID)
			}
			if r.Action != "deny" {
				t.Errorf("action: want deny, got %q", r.Action)
			}
		}
	}
	if !found {
		t.Error("rule block-china not found")
	}
}

func TestOrgPolicyDataSource_NotFound(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/auth-policies", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{})
		},
	})

	rules, err := client.Policies.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, r := range rules {
		if r.Name == "nonexistent-rule" {
			t.Error("should not find nonexistent rule")
		}
	}
}

// ── data.clavex_role ─────────────────────────────────────────────────────────

func TestRoleDataSource_ByName(t *testing.T) {
	const roleID = "role-aaaa-bbbb-cccc-000000000001"
	desc := "Administrator with full access"
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/roles", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{
				map[string]any{
					"id": roleID, "org_id": testOrgID, "name": "admin",
					"description": desc, "is_system": true,
				},
				map[string]any{
					"id": "viewer-id", "org_id": testOrgID, "name": "viewer",
					"is_system": false,
				},
			})
		},
	})

	roles, err := client.Roles.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	var found bool
	for _, r := range roles {
		if r.Name == "admin" {
			found = true
			if r.ID != roleID {
				t.Errorf("id: want %q, got %q", roleID, r.ID)
			}
			if !r.IsSystem {
				t.Error("is_system: want true")
			}
		}
	}
	if !found {
		t.Error("role admin not found")
	}
}

func TestRoleDataSource_NotFound(t *testing.T) {
	_, client := newMockServer(t, routeMap{
		fmt.Sprintf("GET /api/v1/organizations/%s/roles", testOrgID): func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{})
		},
	})

	roles, err := client.Roles.List(testCtx, testOrgID)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	for _, r := range roles {
		if r.Name == "nonexistent-role" {
			t.Error("should not find nonexistent role")
		}
	}
}
