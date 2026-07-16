# Short-lived AWS credentials for a Terraform CI run, federated via Clavex.
#
# Clavex never calls AWS STS itself: it only issues a short-lived, signed
# OIDC id_token audienced to "sts.amazonaws.com". The aws provider then
# performs its own native OIDC federation (AssumeRoleWithWebIdentity) using
# that token. One-time prerequisite (outside Terraform): register Clavex's
# issuer (https://<org_slug>.<your-domain>) as a trusted OIDC provider in
# AWS IAM, and set "sts.amazonaws.com" in the org's
# agent_token_allowed_audiences allowlist.
resource "clavex_agent_token" "terraform_ci" {
  org_id      = var.org_id
  user_id     = var.ci_service_owner_id # human owner who authorized this workload
  agent_id    = "terraform-prod"
  agent_name  = "Terraform CI (prod)"
  scope       = "infra:write"
  ttl_seconds = 900 # 15 minutes — mint fresh for every CI run, do not reuse
  audience    = "sts.amazonaws.com"
}

provider "aws" {
  region = "eu-west-1"

  assume_role_with_web_identity {
    role_arn            = var.aws_terraform_role_arn
    session_name        = "clavex-terraform-ci"
    web_identity_token  = clavex_agent_token.terraform_ci.token
  }
}
