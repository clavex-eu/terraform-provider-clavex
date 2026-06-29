# clavex_federation_subordinate (Resource)

Manages an OpenID Federation subordinate entity under a Clavex trust anchor.

## Example Usage

```terraform
resource "clavex_federation_subordinate" "partner" {
  org_id       = var.org_id
  entity_id    = "https://partner.example.com"
  name         = "Partner RP"
  entity_types = ["openid_relying_party"]
  status       = "active"
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `entity_id` (String) Subordinate entity identifier URL. Immutable natural key.
- `name` (String) Display name.
- `entity_types` (String) Entity types, e.g. `["openid_relying_party"]`.

### Optional

- `jwks_uri` Subordinate JWKS URI.
- `trust_mark_ids` Trust mark IDs to assign.
- `status` `active` | `suspended` | `revoked`.
- `statement_lifetime_seconds` Entity statement lifetime.

### Read-Only

- `id` Subordinate record ID.

## Import

```shell
terraform import clavex_federation_subordinate.example <org_id>/<subordinate_id>
```
