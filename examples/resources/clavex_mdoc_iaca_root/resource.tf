resource "clavex_mdoc_iaca_root" "eu_root" {
  org_id    = var.org_id
  label     = "EU PID IACA Root"
  pem       = file("iaca_root.pem")
  doc_types = ["org.iso.18013.5.1.mDL", "eu.europa.ec.eudi.pid.1"]
}
