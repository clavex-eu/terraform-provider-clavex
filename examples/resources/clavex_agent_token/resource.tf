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
