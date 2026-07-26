terraform {
  required_providers {
    gitea = {
      source = "git.unkin.net/unkin/giteavaultsecret"
    }
  }
}

provider "gitea" {
  # address / token fall back to VAULT_ADDR / VAULT_TOKEN.
}

# Mount the Gitea secrets engine and seed it with a site-admin credential.
resource "gitea_secret_backend" "gitea" {
  path      = "gitea"
  gitea_url = "https://git.example.com"
  ca_cert   = file("${path.module}/gitea-ca.pem")

  admin_username = "bot-admin"
  admin_password = var.gitea_admin_password # e.g. sourced from vault_kv_secret

  request_timeout_seconds = 30
}

variable "gitea_admin_password" {
  type      = string
  sensitive = true
}
