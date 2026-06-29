# clavex_app_family (Resource)

Manages an application family (grouped SSO clients). Members are managed separately.

## Example Usage

```terraform
resource "clavex_app_family" "suite" {
  org_id      = var.org_id
  name        = "Office Suite"
  description = "Coordinated SSO + logout group"
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `name` (String) Family name.

### Optional

- `description` Description.

### Read-Only

- `id` Family ID.

## Import

```shell
terraform import clavex_app_family.example <org_id>/<family_id>
```
