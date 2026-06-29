resource "clavex_pam_credential" "db_root" {
  org_id                 = var.org_id
  name                   = "prod-db-root"
  credential_type        = "password"
  username               = "root"
  secret                 = var.db_root_password
  target_host            = "db.prod.internal"
  checkout_duration      = 3600
  require_access_request  = true
  rotation_interval_days  = 30
}
