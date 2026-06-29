package provider

import (
	"fmt"
	"net/http"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
)

// Exercises the typed SDK paths the governance Terraform resources rely on.

func TestServiceAccountResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	sa := map[string]any{"id": "sa-1", "org_id": testOrgID, "name": "ci", "client_id": "cli-1", "scopes": []string{"openid"}, "is_active": true}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/service-accounts": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"service_account": sa, "client_secret": "sek", "secret_note": "once"})
		},
		"GET " + base + "/service-accounts/sa-1":    func(w http.ResponseWriter, r *http.Request) { writeJSON(w, sa) },
		"DELETE " + base + "/service-accounts/sa-1": func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusNoContent) },
	})

	res, err := client.ServiceAccounts.Create(testCtx, testOrgID, clavex.CreateServiceAccountParams{Name: "ci"})
	if err != nil || res.ClientSecret != "sek" || res.ServiceAccount.ID != "sa-1" {
		t.Fatalf("Create: %v %#v", err, res)
	}
	got, err := client.ServiceAccounts.Get(testCtx, testOrgID, "sa-1")
	if err != nil || got.ClientID != "cli-1" {
		t.Fatalf("Get: %v %#v", err, got)
	}
	if err := client.ServiceAccounts.Delete(testCtx, testOrgID, "sa-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestLoginFlowResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	lf := map[string]any{"id": "lf-1", "org_id": testOrgID, "name": "default", "is_default": true, "is_active": true}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/login-flows":        func(w http.ResponseWriter, r *http.Request) { writeJSON(w, lf) },
		"GET " + base + "/login-flows/lf-1":    func(w http.ResponseWriter, r *http.Request) { writeJSON(w, lf) },
		"PUT " + base + "/login-flows/lf-1":    func(w http.ResponseWriter, r *http.Request) { writeJSON(w, lf) },
		"DELETE " + base + "/login-flows/lf-1": func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusNoContent) },
	})

	flow, err := client.LoginFlows.Create(testCtx, testOrgID, clavex.CreateLoginFlowParams{Name: "default", IsDefault: true})
	if err != nil || flow.ID != "lf-1" {
		t.Fatalf("Create: %v %#v", err, flow)
	}
	got, err := client.LoginFlows.Get(testCtx, testOrgID, "lf-1")
	if err != nil || !got.IsDefault {
		t.Fatalf("Get: %v %#v", err, got)
	}
}

func TestLifecycleRuleResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	rule := map[string]any{"id": "lr-1", "org_id": testOrgID, "name": "joiner", "trigger": "joiner", "priority": 10, "conditions": []any{}, "actions": []any{}, "is_active": true}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/lifecycle-rules":     func(w http.ResponseWriter, r *http.Request) { writeJSON(w, rule) },
		"GET " + base + "/lifecycle-rules/lr-1": func(w http.ResponseWriter, r *http.Request) { writeJSON(w, rule) },
	})

	rl, err := client.LifecycleRules.Create(testCtx, testOrgID, clavex.LifecycleRuleParams{Name: "joiner", Trigger: "joiner", Priority: 10, Conditions: []byte("[]"), Actions: []byte("[]")})
	if err != nil || rl.ID != "lr-1" {
		t.Fatalf("Create: %v %#v", err, rl)
	}
	got, err := client.LifecycleRules.Get(testCtx, testOrgID, "lr-1")
	if err != nil || got.Trigger != "joiner" {
		t.Fatalf("Get: %v %#v", err, got)
	}
}

func TestWsfedRpResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	rp := map[string]any{"id": "rp-1", "org_id": testOrgID, "name": "App", "realm": "urn:app", "wreply_uris": []string{"https://app/wsfed"}, "token_lifetime_seconds": 3600, "claims_mapping": map[string]string{"email": "email"}, "is_active": true}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/wsfed/relying-parties":     func(w http.ResponseWriter, r *http.Request) { writeJSON(w, rp) },
		"GET " + base + "/wsfed/relying-parties/rp-1": func(w http.ResponseWriter, r *http.Request) { writeJSON(w, rp) },
	})

	created, err := client.WsfedRPs.Create(testCtx, testOrgID, clavex.WsfedRpParams{Name: "App", Realm: "urn:app"})
	if err != nil || created.ID != "rp-1" {
		t.Fatalf("Create: %v %#v", err, created)
	}
	got, err := client.WsfedRPs.Get(testCtx, testOrgID, "rp-1")
	if err != nil || got.Realm != "urn:app" {
		t.Fatalf("Get: %v %#v", err, got)
	}
}

func TestAppFamilyResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	fam := map[string]any{"id": "fam-1", "org_id": testOrgID, "name": "Suite"}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/app-families":      func(w http.ResponseWriter, r *http.Request) { writeJSON(w, fam) },
		"GET " + base + "/app-families/fam-1": func(w http.ResponseWriter, r *http.Request) { writeJSON(w, fam) },
	})

	created, err := client.AppFamilies.Create(testCtx, testOrgID, clavex.AppFamilyParams{Name: "Suite"})
	if err != nil || created.ID != "fam-1" {
		t.Fatalf("Create: %v %#v", err, created)
	}
	got, err := client.AppFamilies.Get(testCtx, testOrgID, "fam-1")
	if err != nil || got.Name != "Suite" {
		t.Fatalf("Get: %v %#v", err, got)
	}
}

func TestAgentTokenResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/agent-tokens": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"token": "jwt", "token_id": "at-1", "expires_at": "2026-12-31T00:00:00Z"})
		},
		"GET " + base + "/agent-tokens": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, []any{map[string]any{"id": "at-1", "org_id": testOrgID, "user_id": "u", "agent_id": "a", "agent_name": "n", "expires_at": "2026-12-31T00:00:00Z"}})
		},
		"DELETE " + base + "/agent-tokens/at-1": func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusNoContent) },
	})

	res, err := client.AgentTokens.Issue(testCtx, testOrgID, clavex.IssueAgentTokenParams{UserID: "u", AgentID: "a", AgentName: "n"})
	if err != nil || res.Token != "jwt" || res.TokenID != "at-1" {
		t.Fatalf("Issue: %v %#v", err, res)
	}
	if err := client.AgentTokens.Delete(testCtx, testOrgID, "at-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestPAMCredentialResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	cred := map[string]any{"id": "cred-1", "org_id": testOrgID, "name": "db", "credential_type": "password", "checkout_duration": 3600, "require_access_request": false, "is_active": true}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/pam/credentials":          func(w http.ResponseWriter, r *http.Request) { writeJSON(w, cred) },
		"GET " + base + "/pam/credentials":           func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]any{"data": []any{cred}}) },
		"PUT " + base + "/pam/credentials/cred-1":    func(w http.ResponseWriter, r *http.Request) { writeJSON(w, map[string]any{"status": "updated"}) },
		"DELETE " + base + "/pam/credentials/cred-1": func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusNoContent) },
	})

	c, err := client.PAM.CreateCredential(testCtx, testOrgID, clavex.CreateCredentialParams{Name: "db", CredentialType: "password", Secret: "s", CheckoutDuration: 3600})
	if err != nil || c.ID != "cred-1" {
		t.Fatalf("CreateCredential: %v %#v", err, c)
	}
	creds, err := client.PAM.ListCredentials(testCtx, testOrgID)
	if err != nil || len(creds) != 1 || creds[0].CredentialType != "password" {
		t.Fatalf("ListCredentials: %v %#v", err, creds)
	}
	if err := client.PAM.UpdateCredential(testCtx, testOrgID, "cred-1", clavex.UpdateCredentialParams{}); err != nil {
		t.Fatalf("UpdateCredential: %v", err)
	}
}
