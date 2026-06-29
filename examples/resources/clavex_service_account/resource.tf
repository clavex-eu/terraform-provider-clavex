resource "clavex_service_account" "ci" {
  org_id      = var.org_id
  name        = "ci-pipeline"
  description = "GitHub Actions deploy bot"
  scopes      = ["openid", "clavex.admin"]
}

output "sa_client_id" {
  value = clavex_service_account.ci.client_id
}
