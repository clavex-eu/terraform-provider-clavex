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
