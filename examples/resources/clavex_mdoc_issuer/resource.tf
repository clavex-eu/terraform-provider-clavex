resource "clavex_mdoc_issuer" "mdl" {
  org_id             = var.org_id
  display_name       = "National mDL Issuer"
  doc_type           = "org.iso.18013.5.1.mDL"
  ds_private_key_pem = file("ds_key.pem")
  ds_certificate_pem = file("ds_cert.pem")
  validity_hours     = 8760
}
