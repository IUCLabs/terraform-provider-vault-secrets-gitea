# A role that mints short-lived tokens for the "teabot" bot user, scoped to
# repositories and issues. Reading gitea/creds/teabot returns a lease-bound token
# that is deleted from Gitea on revoke.
resource "gitea_secret_backend_role" "teabot" {
  backend  = gitea_secret_backend.gitea.path
  name     = "teabot"
  username = "teabot"
  scopes   = ["read:repository", "write:issue"]

  ttl     = 3600  # 1h
  max_ttl = 28800 # 8h
}
