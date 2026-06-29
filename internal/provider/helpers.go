package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ── List helpers ──────────────────────────────────────────────────────────────

// stringsToList converts a []string to types.List with element type StringType.
func stringsToList(ss []string) types.List {
	vals := make([]attr.Value, len(ss))
	for i, s := range ss {
		vals[i] = types.StringValue(s)
	}
	list, _ := types.ListValue(types.StringType, vals)
	return list
}

// listToStrings converts a types.List to []string, ignoring diagnostics.
func listToStrings(ctx context.Context, list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	var out []string
	list.ElementsAs(ctx, &out, false) //nolint:errcheck
	return out
}

// ── Map helpers ───────────────────────────────────────────────────────────────

// strMapToMap converts a map[string]string to types.Map with element type StringType.
func strMapToMap(m map[string]string) types.Map {
	if len(m) == 0 {
		return types.MapValueMust(types.StringType, map[string]attr.Value{})
	}
	vals := make(map[string]attr.Value, len(m))
	for k, v := range m {
		vals[k] = types.StringValue(v)
	}
	result, _ := types.MapValue(types.StringType, vals)
	return result
}

// mapToStrMap converts a types.Map to map[string]string, ignoring diagnostics.
func mapToStrMap(ctx context.Context, m types.Map) map[string]string {
	if m.IsNull() || m.IsUnknown() {
		return nil
	}
	var out map[string]string
	m.ElementsAs(ctx, &out, false) //nolint:errcheck
	return out
}

// ── String pointer helpers ────────────────────────────────────────────────────

// ptrString returns a pointer to s, or nil if s is "".
func ptrString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// derefString dereferences a *string, returning "" for nil.
func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// strAttrToPtr converts a types.String to *string (nil when null/unknown/empty).
func strAttrToPtr(attr types.String) *string {
	if attr.IsNull() || attr.IsUnknown() {
		return nil
	}
	v := attr.ValueString()
	if v == "" {
		return nil
	}
	return &v
}

// ptrToStrAttr converts *string to types.String (null when nil).
func ptrToStrAttr(s *string) types.String {
	if s == nil {
		return types.StringNull()
	}
	return types.StringValue(*s)
}

// ── Bool pointer helpers ──────────────────────────────────────────────────────

// boolAttrToPtr converts types.Bool to *bool (nil when null/unknown).
func boolAttrToPtr(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

// ptrToBoolAttr converts *bool to types.Bool (null when nil).
func ptrToBoolAttr(b *bool) types.Bool {
	if b == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*b)
}

// ── Composite ID helpers ──────────────────────────────────────────────────────

// joinID joins multiple parts with "/" to form a composite resource ID.
func joinID(parts ...string) string { return strings.Join(parts, "/") }

// splitID splits a composite resource ID by "/", returning an error if the
// number of parts doesn't match expected.
func splitID(id string, expected int) ([]string, error) {
	parts := strings.SplitN(id, "/", expected)
	if len(parts) != expected {
		return nil, fmt.Errorf("unexpected resource ID format %q (want %d parts)", id, expected)
	}
	return parts, nil
}
