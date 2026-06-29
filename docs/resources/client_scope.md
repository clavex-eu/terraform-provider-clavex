# clavex_client_scope (Resource)

Manages a custom OIDC/SAML client scope.

## Example Usage

```terraform
resource "clavex_client_scope" "profile" {
  org_id      = var.org_id
  name        = "profile_extended"
  description = "Extended profile claims"
  protocol    = "openid-connect"
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `name` (String) Scope name.

### Optional

- `description` Human-readable description.
- `protocol` `openid-connect` or `saml`.

### Read-Only

- `id` Scope ID.
- `is_default` Whether the scope is assigned by default.

## Import

```shell
terraform import clavex_client_scope.example <org_id>/<scope_id>
```
