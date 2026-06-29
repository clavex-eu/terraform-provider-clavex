resource "clavex_client_scope" "profile" {
  org_id      = var.org_id
  name        = "profile_extended"
  description = "Extended profile claims"
  protocol    = "openid-connect"
}
