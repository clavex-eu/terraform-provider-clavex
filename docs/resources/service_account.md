# clavex_service_account (Resource)

Manages a machine-to-machine service account. The client secret is only returned at creation.

## Example Usage

```terraform
resource "clavex_service_account" "ci" {
  org_id      = var.org_id
  name        = "ci-pipeline"
  description = "GitHub Actions deploy bot"
  scopes      = ["openid", "clavex.admin"]
}

output "sa_client_id" {
  value = clavex_service_account.ci.client_id
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `name` (String) Service account name.

### Optional

- `description` Description.
- `scopes` Granted scopes.
- `is_active` Whether the account is active (default `true`).

### Read-Only

- `id` Service account ID.
- `client_id` OAuth client ID.
- `client_secret` Plaintext secret (sensitive, set only at creation).

## Import

```shell
terraform import clavex_service_account.example <org_id>/<service_account_id>
```
