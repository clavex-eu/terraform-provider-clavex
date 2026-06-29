# clavex_mdoc_iaca_root (Resource)

Manages a trusted IACA root certificate for mdoc verification. Immutable — changes force replacement.

## Example Usage

```terraform
resource "clavex_mdoc_iaca_root" "eu_root" {
  org_id    = var.org_id
  label     = "EU PID IACA Root"
  pem       = file("iaca_root.pem")
  doc_types = ["org.iso.18013.5.1.mDL", "eu.europa.ec.eudi.pid.1"]
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `label` (String) Human-readable label.
- `pem` (String) PEM-encoded IACA root certificate.

### Optional

- `doc_types` Doc types this root is trusted for.

### Read-Only

- `id` Root ID.
- `subject_dn` Certificate subject DN.
- `sha256_fingerprint` SHA-256 fingerprint.
- `is_active` Whether the root is active.

## Import

```shell
terraform import clavex_mdoc_iaca_root.example <org_id>/<root_id>
```
