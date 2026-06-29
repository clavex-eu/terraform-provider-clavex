# Clavex Provider

The Clavex provider manages resources on a [Clavex](https://clavex.eu) identity
platform deployment: organisations, users, OIDC/SAML clients, identity providers,
EUDI wallet issuance (OID4VCI / mdoc / OpenID Federation), fine-grained
authorization, privileged access, and identity governance.

## Example Usage

```terraform
terraform {
  required_providers {
    clavex = {
      source = "clavex-eu/clavex"
    }
  }
}

provider "clavex" {
  base_url = "https://id.example.com"
  token    = var.clavex_token
}
```

## Authentication

Provide either a pre-acquired admin token or email/password credentials. Each
attribute falls back to an environment variable when unset.

| Attribute  | Env var            | Description |
|------------|--------------------|-------------|
| `base_url` | `CLAVEX_BASE_URL`  | Base URL of the Clavex server. |
| `token`    | `CLAVEX_TOKEN`     | Admin JWT or API key (preferred for automation). |
| `email`    | `CLAVEX_EMAIL`     | Admin email (password auth). |
| `password` | `CLAVEX_PASSWORD`  | Admin password (password auth). |
| `org_slug` | `CLAVEX_ORG_SLUG`  | Org slug for password auth (omit for superadmin). |

## Schema

### Optional

- `base_url` (String) Base URL of the Clavex server.
- `token` (String, Sensitive) Admin JWT or API key.
- `email` (String) Admin email for password authentication.
- `password` (String, Sensitive) Admin password.
- `org_slug` (String) Organisation slug for password authentication.
