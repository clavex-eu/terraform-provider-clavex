resource "clavex_login_flow" "passwordless" {
  org_id      = var.org_id
  name        = "Passwordless"
  description = "WebAuthn-first login"
  is_default  = false
  is_active   = true
}
