resource "gitea_secret_backend" "gitea" {
  path      = "gitea"
  gitea_url = "https://gitea.example.com"
  ca_cert   = file("${path.module}/gitea-ca.pem")

  admin_username = "vault-admin"
  admin_password = var.gitea_admin_password

  request_timeout_seconds = 30
}

variable "gitea_admin_password" {
  type      = string
  sensitive = true
}
