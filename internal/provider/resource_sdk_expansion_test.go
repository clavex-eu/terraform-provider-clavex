package provider

import (
	"fmt"
	"net/http"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
)

// These tests exercise the SDK client paths that the SDK-expansion Terraform
// resources rely on, using the shared mock-server helper.

func TestClientScopeResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	scope := map[string]any{"id": "sc-1", "org_id": testOrgID, "name": "profile", "protocol": "openid-connect", "is_default": false}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/client-scopes":        func(w http.ResponseWriter, r *http.Request) { writeJSON(w, scope) },
		"GET " + base + "/client-scopes":         func(w http.ResponseWriter, r *http.Request) { writeJSON(w, []any{scope}) },
		"DELETE " + base + "/client-scopes/sc-1": func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusNoContent) },
	})

	sc, err := client.ClientScopes.Create(testCtx, testOrgID, clavex.CreateClientScopeParams{Name: "profile", Protocol: "openid-connect"})
	if err != nil || sc.ID != "sc-1" {
		t.Fatalf("Create: %v %#v", err, sc)
	}
	scopes, err := client.ClientScopes.List(testCtx, testOrgID)
	if err != nil || len(scopes) != 1 {
		t.Fatalf("List: %v %#v", err, scopes)
	}
	if err := client.ClientScopes.Delete(testCtx, testOrgID, "sc-1"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestOID4VCIConfigResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	cfg := map[string]any{"id": "cfg-1", "org_id": testOrgID, "vct": "https://x/d", "display_name": "D", "ttl_seconds": 86400, "is_active": true}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/oid4vci/configs":         func(w http.ResponseWriter, r *http.Request) { writeJSON(w, cfg) },
		"GET " + base + "/oid4vci/configs":          func(w http.ResponseWriter, r *http.Request) { writeJSON(w, []any{cfg}) },
		"DELETE " + base + "/oid4vci/configs/cfg-1": func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusNoContent) },
	})

	c, err := client.OID4VCI.CreateConfig(testCtx, testOrgID, clavex.CreateVCIConfigParams{VCT: "https://x/d", DisplayName: "D"})
	if err != nil || c.ID != "cfg-1" {
		t.Fatalf("CreateConfig: %v %#v", err, c)
	}
	cfgs, err := client.OID4VCI.ListConfigs(testCtx, testOrgID)
	if err != nil || len(cfgs) != 1 {
		t.Fatalf("ListConfigs: %v %#v", err, cfgs)
	}
	if err := client.OID4VCI.DeleteConfig(testCtx, testOrgID, "cfg-1"); err != nil {
		t.Fatalf("DeleteConfig: %v", err)
	}
}

func TestMdocResources_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	issuer := map[string]any{"id": "is-1", "org_id": testOrgID, "display_name": "PID", "doc_type": "org.iso.18013.5.1.mDL", "ds_certificate_pem": "PEM", "validity_hours": 8760, "is_active": true}
	root := map[string]any{"id": "r-1", "org_id": testOrgID, "label": "root", "pem": "PEM", "doc_types": []string{"mDL"}, "subject_dn": "CN=x", "sha256_fingerprint": "ab", "is_active": true}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/mdoc/issuers":    func(w http.ResponseWriter, r *http.Request) { writeJSON(w, issuer) },
		"GET " + base + "/mdoc/issuers":     func(w http.ResponseWriter, r *http.Request) { writeJSON(w, []any{issuer}) },
		"POST " + base + "/mdoc/iaca-roots": func(w http.ResponseWriter, r *http.Request) { writeJSON(w, root) },
		"GET " + base + "/mdoc/iaca-roots":  func(w http.ResponseWriter, r *http.Request) { writeJSON(w, []any{root}) },
	})

	is, err := client.Mdoc.CreateIssuer(testCtx, testOrgID, clavex.CreateIssuerParams{DisplayName: "PID", DocType: "org.iso.18013.5.1.mDL", DSPrivateKeyPEM: "k", DSCertificatePEM: "PEM"})
	if err != nil || is.ID != "is-1" {
		t.Fatalf("CreateIssuer: %v %#v", err, is)
	}
	rt, err := client.Mdoc.CreateIACARoot(testCtx, testOrgID, clavex.CreateIACARootParams{Label: "root", PEM: "PEM"})
	if err != nil || rt.ID != "r-1" {
		t.Fatalf("CreateIACARoot: %v %#v", err, rt)
	}
}

func TestFederationSubordinateResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/federation/subordinates": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"id": "s-1", "entity_id": "https://sub", "name": "Sub", "entity_types": []string{"openid_relying_party"}, "status": "active"})
		},
		"GET " + base + "/federation/subordinates": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"subordinates": []any{map[string]any{"id": "s-1", "entity_id": "https://sub", "name": "Sub", "status": "active"}}, "count": 1})
		},
	})

	sub, err := client.Federation.RegisterSubordinate(testCtx, testOrgID, clavex.RegisterSubordinateParams{EntityID: "https://sub", Name: "Sub", EntityTypes: []string{"openid_relying_party"}})
	if err != nil || sub.ID != "s-1" {
		t.Fatalf("RegisterSubordinate: %v %#v", err, sub)
	}
	subs, err := client.Federation.ListSubordinates(testCtx, testOrgID, "all")
	if err != nil || len(subs) != 1 {
		t.Fatalf("ListSubordinates: %v %#v", err, subs)
	}
}

func TestSCIMTokenResource_CRUD(t *testing.T) {
	base := fmt.Sprintf("/api/v1/organizations/%s", testOrgID)
	tok := map[string]any{"id": "t-1", "org_id": testOrgID, "name": "default", "created_at": "2026-01-01T00:00:00Z"}
	_, client := newMockServer(t, routeMap{
		"POST " + base + "/scim/tokens": func(w http.ResponseWriter, r *http.Request) {
			writeJSON(w, map[string]any{"token": "scim-secret", "scim_token": tok})
		},
		"GET " + base + "/scim/tokens":        func(w http.ResponseWriter, r *http.Request) { writeJSON(w, []any{tok}) },
		"DELETE " + base + "/scim/tokens/t-1": func(w http.ResponseWriter, r *http.Request) { writeStatus(w, http.StatusNoContent) },
	})

	res, err := client.SCIM.CreateToken(testCtx, testOrgID)
	if err != nil || res.Token != "scim-secret" || res.SCIMToken.ID != "t-1" {
		t.Fatalf("CreateToken: %v %#v", err, res)
	}
	if err := client.SCIM.DeleteToken(testCtx, testOrgID, "t-1"); err != nil {
		t.Fatalf("DeleteToken: %v", err)
	}
}
