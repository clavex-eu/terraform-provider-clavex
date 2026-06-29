resource "clavex_scim_token" "provisioning" {
  org_id = var.org_id
}

output "scim_token" {
  value     = clavex_scim_token.provisioning.token
  sensitive = true
}
