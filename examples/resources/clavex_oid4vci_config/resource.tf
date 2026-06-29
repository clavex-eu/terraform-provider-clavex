resource "clavex_oid4vci_config" "diploma" {
  org_id       = var.org_id
  vct          = "https://example.edu/credentials/diploma"
  display_name = "University Diploma"
  description  = "Bachelor degree credential"
  ttl_seconds  = 31536000
  category     = "qualification"
  claims_mapping = {
    given_name  = "metadata.given_name"
    family_name = "metadata.family_name"
    degree      = "metadata.degree"
  }
}
