# clavex_login_flow (Resource)

Manages a login flow. Steps are managed separately via the API.

## Example Usage

```terraform
resource "clavex_login_flow" "passwordless" {
  org_id      = var.org_id
  name        = "Passwordless"
  description = "WebAuthn-first login"
  is_default  = false
  is_active   = true
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `name` (String) Flow name.

### Optional

- `description` Description.
- `is_default` Whether this is the org default flow.
- `is_active` Whether the flow is active (default `true`).

### Read-Only

- `id` Flow ID.

## Import

```shell
terraform import clavex_login_flow.example <org_id>/<flow_id>
```
