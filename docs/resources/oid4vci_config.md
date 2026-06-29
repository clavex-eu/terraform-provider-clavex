# clavex_oid4vci_config (Resource)

Manages an OID4VCI credential-type configuration (SD-JWT VC issuance). Core attributes are immutable; only `claims_mapping` updates in place.

## Example Usage

```terraform
resource "clavex_oid4vci_config" "diploma" {
  org_id       = var.org_id
  vct          = "https://example.edu/credentials/diploma"
  display_name = "University Diploma"
  description  = "Bachelor degree credential"
  ttl_seconds  = 31536000
  category     = "qualification"
  claims_mapping = {
    given_name  = "metadata.given_name"
    family_name = "metadata.family_name"
    degree      = "metadata.degree"
  }
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `vct` (String) Verifiable Credential Type URI. Immutable.
- `display_name` (String) Display name. Immutable.

### Optional

- `description` Description. Immutable.
- `ttl_seconds` Credential lifetime in seconds.
- `category` `identity` | `training` | `qualification` | `badge`. Immutable.
- `claims_mapping` Maps credential field names to source paths. Updatable.

### Read-Only

- `id` Config ID.
- `is_active` Whether the config is active.

## Import

```shell
terraform import clavex_oid4vci_config.example <org_id>/<config_id>
```
