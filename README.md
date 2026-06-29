# Terraform Provider for Clavex

[![release](https://github.com/clavex-eu/terraform-provider-clavex/actions/workflows/release.yml/badge.svg)](https://github.com/clavex-eu/terraform-provider-clavex/actions/workflows/release.yml)

Manage [Clavex](https://clavex.eu) identity-platform resources as code:
organizations, users, roles & groups, OIDC/SAML/WS-Fed clients, identity
providers, EUDI wallet issuance (OID4VCI / mdoc / OpenID Federation),
fine-grained authorization, privileged access management, and identity
governance.

Built on the [Clavex Go SDK](https://github.com/clavex-eu/clavex-sdk-go) and the
[terraform-plugin-framework](https://github.com/hashicorp/terraform-plugin-framework)
(protocol v6).

## Usage

```terraform
terraform {
  required_providers {
    clavex = {
      source  = "clavex-eu/clavex"
      version = "~> 0.1"
    }
  }
}

provider "clavex" {
  base_url = "https://id.example.com"
  token    = var.clavex_token # or env CLAVEX_TOKEN
}
```

See [`docs/`](docs/) for the full resource and data-source reference, and
[`examples/`](examples/) for runnable configurations.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0
- [Go](https://go.dev/dl/) >= 1.25 (to build from source)

## Building locally

```bash
go install .
```

To test the provider against a local build, add a `dev_overrides` block to your
`~/.terraformrc` pointing `clavex-eu/clavex` at your `GOBIN`.

## Releasing

Releases are produced by GoReleaser via the `release` workflow on every `vX.Y.Z`
tag and published to the Terraform Registry. The repository must define two
secrets:

| Secret            | Description                                  |
|-------------------|----------------------------------------------|
| `GPG_PRIVATE_KEY` | ASCII-armored private key (signs checksums). |
| `PASSPHRASE`      | Passphrase for that key.                      |

The corresponding public key must be registered with the Terraform Registry.

## License

Business Source License 1.1 — see [LICENSE](LICENSE).
