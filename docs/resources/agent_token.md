# clavex_agent_token (Resource)

Manages an AI-agent (MCP) access token. Immutable; the token is returned only at creation.

## Example Usage

```terraform
resource "clavex_agent_token" "copilot" {
  org_id      = var.org_id
  user_id     = var.user_id
  agent_id    = "copilot-prod"
  agent_name  = "Production Copilot"
  scope       = "mcp.read mcp.tools"
  ttl_seconds = 3600
}

output "agent_jwt" {
  value     = clavex_agent_token.copilot.token
  sensitive = true
}
```

### Cloud credential federation for Terraform (AWS/Azure/GCP)

`audience` lets a token be used as the input `web_identity_token`/`oidc_token`
for the AWS, azurerm, or google Terraform providers' own native OIDC
federation (`assume_role_with_web_identity`, `use_oidc`, `external_account`).
Clavex never calls the cloud STS/Workload Identity Federation APIs itself —
it only issues a short-lived, signed OIDC id_token; the target cloud
provider performs the actual credential exchange. See
`examples/resources/clavex_agent_token/resource_aws_federation.tf` for a
full AWS example. Requires the target audience (e.g. `sts.amazonaws.com`,
`api://AzureADTokenExchange`, or a GCP Workload Identity Federation pool
provider audience) to be pre-approved in the org's agent-token audience
allowlist; otherwise issuance fails with `invalid_target`.

## Schema

### Required

- `org_id` (String) Organisation ID. Forces replacement.
- `user_id` (String) User the token acts on behalf of. Forces replacement.
- `agent_id` (String) Stable agent identifier. Forces replacement.
- `agent_name` (String) Agent display name. Forces replacement.

### Optional

- `scope` Granted scope. Forces replacement.
- `ttl_seconds` Token lifetime in seconds.
- `mcp_server_id` Associated MCP server ID. Forces replacement.
- `audience` Overrides the token's `aud` claim (default: the issuer). Must
  be pre-approved in the org's agent-token audience allowlist. Forces
  replacement.

### Read-Only

- `id` Token record ID.
- `token` Signed JWT (sensitive, set only at creation).
- `expires_at` Expiry timestamp.

## Import

```shell
terraform import clavex_agent_token.example <org_id>/<token_id>
```
