# Terraform Provider for the Gitea Vault Secrets Engine

Manage the [Gitea secrets engine](https://github.com/IUCLabs/vault-plugin-secrets-gitea)
on HashiCorp Vault or OpenBao from Terraform. The provider mounts the engine,
writes its config (Gitea URL, TLS settings, seeded site-admin credentials) and
manages the roles that mint short-lived, scoped Gitea access tokens.

The provider is published as `IUCLabs/vault-secrets-gitea`; the registry type is
`vault-secrets-gitea`. Its resources are prefixed `gitea_`, so declare it under
the local name `gitea`.

## Quick start

Prebuilt provider archives are on the [releases page](https://github.com/IUCLabs/terraform-provider-vault-secrets-gitea/releases).

```hcl
terraform {
  required_providers {
    gitea = {
      source = "IUCLabs/vault-secrets-gitea"
    }
  }
}

# address and token fall back to VAULT_ADDR and VAULT_TOKEN.
provider "gitea" {}

resource "gitea_secret_backend" "gitea" {
  path           = "gitea"
  gitea_url      = "https://gitea.example.com"
  admin_username = "vault-admin"
  admin_password = var.gitea_admin_password
}

resource "gitea_secret_backend_role" "ci_bot" {
  backend  = gitea_secret_backend.gitea.path
  name     = "ci-bot"
  username = "ci-bot"
  scopes   = ["read:repository", "write:issue"]
  ttl      = 3600
  max_ttl  = 28800
}
```

Then `vault read gitea/creds/ci-bot` returns a token that is deleted from Gitea
when its lease is revoked or expires.

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 1.0 (or OpenTofu)
- Vault or OpenBao with [`vault-plugin-secrets-gitea`](https://github.com/IUCLabs/vault-plugin-secrets-gitea)
  registered in the plugin catalog. The end-to-end tests run against Vault
  1.18; other versions and OpenBao are untested with this provider.
- A Gitea site-admin account (local user, no 2FA) for the engine to seed with
- [Go](https://go.dev/doc/install) >= 1.25 to build the provider

## Documentation

- [Provider](docs/index.md)
- [`gitea_secret_backend`](docs/resources/secret_backend.md)
- [`gitea_secret_backend_role`](docs/resources/secret_backend_role.md)
- [Secrets engine paths and parameters](https://github.com/IUCLabs/vault-plugin-secrets-gitea/tree/main/docs)

## Development

```sh
make build   # build the provider binary
make test    # unit tests
make docs    # regenerate docs/ from the schema and examples/
make e2e     # terraform apply against Vault and a mock Gitea (Docker)
```

## License

[MIT](LICENSE)
