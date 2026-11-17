# Import by mount path. Import only sets `path`; `admin_password` and `ca_cert`
# are never read back, so they must be set in configuration and are written to
# the backend again on the next apply.
terraform import gitea_secret_backend.gitea gitea
