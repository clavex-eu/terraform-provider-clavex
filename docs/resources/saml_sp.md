# clavex_saml_sp (Resource)

Manages a SAML service provider (Clavex acting as IdP). Immutable — changes force replacement.

## Example Usage

```terraform
resource "clavex_saml_sp" "salesforce" {
  org_id         = var.org_id
  entity_id      = "https://saml.salesforce.com"
  name           = "Salesforce"
  acs_url        = "https://login.salesforce.com/saml/acs"
  name_id_format = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `entity_id` (String) SP entity ID.
- `name` (String) Display name.
- `acs_url` (String) Assertion Consumer Service URL.

### Optional

- `slo_url` Single Logout URL.
- `metadata_xml` SP metadata XML (write-only).
- `name_id_format` NameID format.

### Read-Only

- `id` SP ID.
- `is_active` Whether the SP is active.

## Import

```shell
terraform import clavex_saml_sp.example <org_id>/<sp_id>
```
