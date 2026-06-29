# clavex_pam_credential (Resource)

Manages a vaulted privileged credential. The secret is write-only.

## Example Usage

```terraform
resource "clavex_pam_credential" "db_root" {
  org_id                 = var.org_id
  name                   = "prod-db-root"
  credential_type        = "password"
  username               = "root"
  secret                 = var.db_root_password
  target_host            = "db.prod.internal"
  checkout_duration      = 3600
  require_access_request  = true
  rotation_interval_days  = 30
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `name` (String) Credential name.
- `credential_type` (String) `password` | `ssh_key` | `api_key`. Forces replacement.
- `secret` (String) Secret material (write-only, sensitive).
- `checkout_duration` (String) Checkout lease duration in seconds.

### Optional

- `username` Associated username.
- `description` Description.
- `target_host` Target host.
- `require_access_request` Require an approved access request before checkout.
- `is_active` Whether the credential is active (default `true`).
- `rotation_interval_days` Auto-rotation interval in days.

### Read-Only

- `id` Credential ID.

## Import

```shell
terraform import clavex_pam_credential.example <org_id>/<credential_id>
```
