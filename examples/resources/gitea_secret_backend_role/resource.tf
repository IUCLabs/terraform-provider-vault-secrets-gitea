# Each read of gitea/creds/ci-bot mints a token for the "ci-bot" user that is
# deleted from Gitea when the Vault lease is revoked or expires.
resource "gitea_secret_backend_role" "ci_bot" {
  backend  = gitea_secret_backend.gitea.path
  name     = "ci-bot"
  username = "ci-bot"
  scopes   = ["read:repository", "write:issue"]

  ttl     = 3600
  max_ttl = 28800
}
