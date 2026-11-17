terraform {
  required_providers {
    gitea = {
      source = "IUCLabs/vault-secrets-gitea"
    }
  }
}

provider "gitea" {
  address = "http://127.0.0.1:8200"
  token   = "root"
}

resource "gitea_secret_backend" "gitea" {
  path        = "gitea"
  description = "Gitea token engine (e2e)"
  # Reachable from inside the vault container, where the plugin runs.
  gitea_url       = "http://gitea:3000"
  tls_skip_verify = true

  admin_username = "bot-admin"
  admin_password = "seed-password"
}

resource "gitea_secret_backend_role" "ci-bot" {
  backend  = gitea_secret_backend.gitea.path
  name     = "ci-bot"
  username = "ci-bot"
  scopes   = ["read:repository", "write:issue"]
  ttl      = 3600
  max_ttl  = 86400
}
