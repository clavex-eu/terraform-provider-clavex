// Package provider contains unit-level tests for the Clavex Terraform provider.
//
// Test strategy: each test spins up a net/http/httptest server that returns
// minimal but valid JSON responses, builds a clavex SDK client that points to
// it, and exercises the resource's Create/Read/Update/Delete methods directly.
// This avoids the need for a live Clavex installation or the TF_ACC flag.
package provider

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	clavex "github.com/clavex-eu/clavex-sdk-go"
)

// ── Test helpers ──────────────────────────────────────────────────────────────

// route maps "METHOD /path" → handler func.
type routeMap map[string]http.HandlerFunc

// newMockServer builds a httptest.Server that dispatches on "METHOD /path".
// Unmapped routes reply 404.
func newMockServer(t *testing.T, routes routeMap) (*httptest.Server, *clavex.Client) {
	t.Helper()
	var mux *http.ServeMux // declared for local route registration; routes map used directly above
	_ = mux

	// We need method-aware routing; httptest.NewServer wraps a plain Handler.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := r.Method + " " + r.URL.Path
		if h, ok := routes[key]; ok {
			w.Header().Set("Content-Type", "application/json")
			h(w, r)
			return
		}
		// Try prefix match (useful for paths with IDs).
		for pattern, h := range routes {
			if len(pattern) > 4 && matchPath(pattern, r.Method, r.URL.Path) {
				w.Header().Set("Content-Type", "application/json")
				h(w, r)
				return
			}
		}
		t.Logf("mock: no route for %s %s", r.Method, r.URL.Path)
		http.NotFound(w, r)
	}))
	_ = mux // prevent unused-variable error (mux registered routes above)

	client, err := clavex.New(srv.URL, clavex.WithToken("test-token"))
	if err != nil {
		t.Fatalf("newMockServer: clavex.New: %v", err)
	}
	t.Cleanup(srv.Close)
	return srv, client
}

// matchPath checks whether the route key (e.g. "GET /api/v1/organizations/*/auth-policies")
// matches the request. We use a simple prefix check on the non-wildcard segment.
func matchPath(pattern, method, path string) bool {
	if len(pattern) < 5 {
		return false
	}
	parts := splitKeyword(pattern)
	if len(parts) != 2 {
		return false
	}
	if parts[0] != method {
		return false
	}
	// Simple: check that the path contains all literal segments.
	// Good enough for test mocking without introducing a router dependency.
	for _, seg := range splitSegments(parts[1]) {
		if seg != "*" && !containsSegment(path, seg) {
			return false
		}
	}
	return true
}

func splitKeyword(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return []string{s}
}

func splitSegments(p string) []string {
	var out []string
	cur := ""
	for _, c := range p {
		if c == '/' {
			if cur != "" {
				out = append(out, cur)
				cur = ""
			}
		} else {
			cur += string(c)
		}
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}

func containsSegment(path, seg string) bool {
	return len(path) >= len(seg) && (path == seg ||
		contains(path, "/"+seg) || contains(path, seg+"/"))
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || func() bool {
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				return true
			}
		}
		return false
	}())
}

func writeJSON(w http.ResponseWriter, v any) {
	_ = json.NewEncoder(w).Encode(v)
}

func writeStatus(w http.ResponseWriter, code int) {
	w.WriteHeader(code)
}

// ── Shared context ────────────────────────────────────────────────────────────

var testCtx = context.Background()

// ── Data source helpers ───────────────────────────────────────────────────────

func testOrgPolicyDiag(t *testing.T) {
	t.Helper()
}

// testOrgID is a stable fake UUID used across all tests.
const (
	testOrgID    = "aaaaaaaa-bbbb-cccc-dddd-000000000001"
	testPolicyID = "aaaaaaaa-bbbb-cccc-dddd-000000000002"
	testStreamID = "aaaaaaaa-bbbb-cccc-dddd-000000000003"
	testApiKeyID = "aaaaaaaa-bbbb-cccc-dddd-000000000004"
	testTrustID  = "aaaaaaaa-bbbb-cccc-dddd-000000000005"
)
