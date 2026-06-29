# clavex_scim_token (Resource)

Manages a SCIM 2.0 bearer token. The token value is only available at creation.

## Example Usage

```terraform
resource "clavex_scim_token" "provisioning" {
  org_id = var.org_id
}

output "scim_token" {
  value     = clavex_scim_token.provisioning.token
  sensitive = true
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.

### Read-Only

- `id` Token ID.
- `name` Token name.
- `token` Plaintext bearer token (sensitive, set only at creation).

## Import

```shell
terraform import clavex_scim_token.example <org_id>/<token_id>
```
