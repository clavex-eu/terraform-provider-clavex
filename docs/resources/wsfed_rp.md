# clavex_wsfed_rp (Resource)

Manages a WS-Federation relying party.

## Example Usage

```terraform
resource "clavex_wsfed_rp" "legacy_app" {
  org_id                 = var.org_id
  name                   = "Legacy App"
  realm                  = "urn:legacy:app"
  wreply_uris            = ["https://app.example.com/wsfed"]
  token_lifetime_seconds = 3600
  claims_mapping = {
    email = "email"
    name  = "display_name"
  }
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `name` (String) Display name.
- `realm` (String) WS-Fed realm (wtrealm).

### Optional

- `wreply_uris` Allowed wreply return URIs.
- `token_lifetime_seconds` Issued token lifetime.
- `claims_mapping` Claim name mapping.

### Read-Only

- `id` RP ID.
- `is_active` Whether the RP is active.

## Import

```shell
terraform import clavex_wsfed_rp.example <org_id>/<rp_id>
```
