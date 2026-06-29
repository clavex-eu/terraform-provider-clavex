# clavex_protocol_mapper (Resource)

Manages a protocol mapper attached to an OIDC/SAML client.

## Example Usage

```terraform
resource "clavex_protocol_mapper" "email" {
  org_id      = var.org_id
  client_id   = clavex_oidc_client.app.client_id
  name        = "email-mapper"
  protocol    = "openid-connect"
  mapper_type = "user-attribute"
  config = {
    user_attribute = "email"
    claim_name     = "email"
  }
}
```

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `client_id` (String) Client the mapper attaches to. Forces replacement.
- `name` (String) Mapper name.
- `protocol` (String) `openid-connect` or `saml`.
- `mapper_type` (String) Mapper type, e.g. `user-attribute`.

### Optional

- `config` Map of mapper configuration values.

### Read-Only

- `id` Mapper ID.

## Import

```shell
terraform import clavex_protocol_mapper.example <org_id>/<client_id>/<mapper_id>
```
