resource "clavex_app_family" "suite" {
  org_id      = var.org_id
  name        = "Office Suite"
  description = "Coordinated SSO + logout group"
}
