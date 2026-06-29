# clavex_mdoc_issuer (Resource)

Manages an ISO 18013-5 mdoc issuer (Document Signer). Immutable — changes force replacement.

## Example Usage

```terraform
resource "clavex_mdoc_issuer" "mdl" {
  org_id             = var.org_id
  display_name       = "National mDL Issuer"
  doc_type           = "org.iso.18013.5.1.mDL"
  ds_private_key_pem = file("ds_key.pem")
  ds_certificate_pem = file("ds_cert.pem")
  validity_hours     = 8760
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `display_name` (String) Issuer display name.
- `doc_type` (String) mdoc doc type, e.g. `org.iso.18013.5.1.mDL`.
- `ds_private_key_pem` (String) PKCS#8 PEM DS private key (write-only, sensitive).
- `ds_certificate_pem` (String) DS certificate PEM.

### Optional

- `iaca_certificate_pem` IACA certificate PEM.
- `validity_hours` Issued credential validity in hours.

### Read-Only

- `id` Issuer ID.
- `is_active` Whether the issuer is active.

## Import

```shell
terraform import clavex_mdoc_issuer.example <org_id>/<issuer_id>
```
