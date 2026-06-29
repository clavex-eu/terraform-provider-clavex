resource "clavex_federation_subordinate" "partner" {
  org_id       = var.org_id
  entity_id    = "https://partner.example.com"
  name         = "Partner RP"
  entity_types = ["openid_relying_party"]
  status       = "active"
}
