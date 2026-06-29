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

### Read-Only

- `id` Token record ID.
- `token` Signed JWT (sensitive, set only at creation).
- `expires_at` Expiry timestamp.

## Import

```shell
terraform import clavex_agent_token.example <org_id>/<token_id>
```
