resource "clavex_wsfed_rp" "legacy_app" {
  org_id                 = var.org_id
  name                   = "Legacy App"
  realm                  = "urn:legacy:app"
  wreply_uris            = ["https://app.example.com/wsfed"]
  token_lifetime_seconds = 3600
  claims_mapping = {
    email = "email"
    name  = "display_name"
  }
}
