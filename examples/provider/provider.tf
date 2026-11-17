terraform {
  required_providers {
    gitea = {
      source = "IUCLabs/vault-secrets-gitea"
    }
  }
}

# address and token fall back to VAULT_ADDR and VAULT_TOKEN.
provider "gitea" {
  address = "https://vault.example.com:8200"
}
