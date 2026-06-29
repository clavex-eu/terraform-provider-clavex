resource "clavex_saml_sp" "salesforce" {
  org_id         = var.org_id
  entity_id      = "https://saml.salesforce.com"
  name           = "Salesforce"
  acs_url        = "https://login.salesforce.com/saml/acs"
  name_id_format = "urn:oasis:names:tc:SAML:1.1:nameid-format:emailAddress"
}
