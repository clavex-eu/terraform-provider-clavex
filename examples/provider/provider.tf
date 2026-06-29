terraform {
  required_providers {
    clavex = {
      source = "clavex-eu/clavex"
    }
  }
}

# Token-based auth (recommended for CI / service accounts).
provider "clavex" {
  base_url = "https://id.example.com"
  token    = var.clavex_token # or env CLAVEX_TOKEN
}

variable "clavex_token" {
  type      = string
  sensitive = true
}

variable "org_id" {
  type = string
}
