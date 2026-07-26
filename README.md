# terraform-provider-giteavaultsecret

A Terraform provider that manages the **Gitea token secrets engine**
([`vault-plugin-secrets-gitea`](https://git.unkin.net/unkin/vault-plugin-secrets-gitea))
on HashiCorp Vault or OpenBao, so the engine's mount, config, and roles can be
driven declaratively (e.g. from `terraform-vault`).

Source address: `git.unkin.net/unkin/giteavaultsecret` (declare it under the
local name `gitea`, so its resources are `gitea_*`).

## Resources

| Resource | Manages |
|----------|---------|
| `gitea_secret_backend` | Mounts the engine at a path and writes its `config` (Gitea URL, TLS, seeded admin username/password). |
| `gitea_secret_backend_role` | A role: target Gitea `username`, `scopes`, `token_name_prefix`, `ttl`, `max_ttl`. |

## Usage

```hcl
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

resource "gitea_secret_backend" "gitea" {
  path           = "gitea"
  gitea_url      = "https://git.example.com"
  admin_username = "bot-admin"
  admin_password = var.gitea_admin_password
}

resource "gitea_secret_backend_role" "teabot" {
  backend  = gitea_secret_backend.gitea.path
  name     = "teabot"
  username = "teabot"
  scopes   = ["read:repository", "write:issue"]
  ttl      = 3600
  max_ttl  = 28800
}
```

### Notes

- `admin_password` is write-only: Vault never returns it, so it is preserved in
  Terraform state and does not show drift. If you rotate the admin password out
  of band with `vault write -f gitea/config/rotate-root`, do **not** re-apply the
  backend resource without also updating `admin_password`, or Terraform will set
  the password back to the value in your configuration. `ca_cert` is likewise
  write-only and preserved.
- Writing `config` makes the engine verify the credentials against Gitea (it must
  be a reachable site admin), so a bad URL/username/password fails the apply.

## Import

```sh
terraform import gitea_secret_backend.gitea gitea
terraform import gitea_secret_backend_role.teabot gitea/roles/teabot
```

## Development

```sh
make build      # build the provider binary
make install    # install into ~/.terraform.d/plugins for local use
make test       # unit tests (race)
make e2e        # apply real terraform against Vault + a mock Gitea (Docker)
make package    # build the release zip
```

Releases are tag-driven (`make patch|minor|major`): a Woodpecker pipeline builds
`terraform-provider-giteavaultsecret_<version>_linux_amd64.zip` and PUTs it to the
artifactapi terraform registry
(`.../api/v2/remotes/terraform-unkin/files/unkin/giteavaultsecret/<file>`), which
signs it server-side. Install it via the bare `source` address above.
