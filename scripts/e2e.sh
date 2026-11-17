#!/usr/bin/env bash
#
# End-to-end test for terraform-provider-vault-secrets-gitea.
#
# Builds the sibling gitea plugin and this provider, boots Vault + a mock Gitea
# REST API in Docker, then runs a real `terraform apply` through the provider to
# mount the engine, seed the admin credential, and create a role. It asserts a
# token can be minted from the role, then `terraform destroy` and verifies the
# mount is gone.
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
PLUGIN_REPO="${PLUGIN_REPO:-${ROOT_DIR}/../vault-plugin-secrets-gitea}"
COMPOSE_FILE="${ROOT_DIR}/test/docker-compose.yml"
COMPOSE="docker compose -f ${COMPOSE_FILE}"
TF="${TF:-terraform}"
E2E_DIR="${ROOT_DIR}/test/e2e"
PLUGIN_BIN="vault-plugin-secrets-gitea"
PROVIDER_BIN="terraform-provider-vault-secrets-gitea"

GITEA_ADDR="http://127.0.0.1:3000"
export VAULT_ADDR="http://127.0.0.1:8200"
export VAULT_TOKEN="root"
export PLUGIN_SRC="${PLUGIN_REPO}"

red()   { printf '\033[31m%s\033[0m\n' "$*"; }
green() { printf '\033[32m%s\033[0m\n' "$*"; }
blue()  { printf '\033[34m==> %s\033[0m\n' "$*"; }
fail()  { red "FAIL: $*"; exit 1; }

cleanup() {
  blue "Cleaning up"
  if [ -d "${E2E_DIR}" ]; then
    (cd "${E2E_DIR}" && TF_CLI_CONFIG_FILE="${ROOT_DIR}/test/dev.tfrc" "${TF}" destroy -auto-approve >/dev/null 2>&1 || true)
    rm -f "${E2E_DIR}"/terraform.tfstate* "${E2E_DIR}"/.terraform.lock.hcl
    rm -rf "${E2E_DIR}/.terraform"
  fi
  ${COMPOSE} down -v >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_for() {
  local desc="$1"; shift
  local retries="${WAIT_RETRIES:-90}" i=0
  until "$@" >/dev/null 2>&1; do
    i=$((i + 1))
    [ "$i" -ge "$retries" ] && fail "timed out waiting for ${desc}"
    sleep 2
  done
  green "ready: ${desc}"
}

jq_field() { python3 -c "import sys,json;print(json.load(sys.stdin)$1)"; }

# ---------------------------------------------------------------------------
blue "Building gitea plugin from ${PLUGIN_REPO}"
[ -d "${PLUGIN_REPO}" ] || fail "plugin repo not found at ${PLUGIN_REPO} (set PLUGIN_REPO)"
mkdir -p "${ROOT_DIR}/test/plugins"
( cd "${PLUGIN_REPO}" && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags "-s -w" \
    -o "${ROOT_DIR}/test/plugins/${PLUGIN_BIN}" ./cmd/vault-plugin-secrets-gitea )

blue "Building the provider"
( cd "${ROOT_DIR}" && go build -o "${PROVIDER_BIN}" . )

blue "Writing terraform dev_overrides config"
cat > "${ROOT_DIR}/test/dev.tfrc" <<EOF
provider_installation {
  dev_overrides {
    "IUCLabs/vault-secrets-gitea" = "${ROOT_DIR}"
  }
  direct {}
}
EOF
export TF_CLI_CONFIG_FILE="${ROOT_DIR}/test/dev.tfrc"

blue "Starting Docker stack"
${COMPOSE} up -d --build
wait_for "gitea" curl -fsS "${GITEA_ADDR}/healthz"
wait_for "vault" ${COMPOSE} exec -T vault vault status -address=http://127.0.0.1:8200

blue "Registering the gitea plugin in Vault"
SHA="$(sha256sum "${ROOT_DIR}/test/plugins/${PLUGIN_BIN}" | awk '{print $1}')"
${COMPOSE} exec -T vault vault plugin register -sha256="${SHA}" secret "${PLUGIN_BIN}" >/dev/null

# ---------------------------------------------------------------------------
blue "terraform apply (mount engine + config + role via the provider)"
( cd "${E2E_DIR}" && "${TF}" apply -auto-approve )
green "apply succeeded"

blue "Verifying the mount and role exist"
${COMPOSE} exec -T vault vault secrets list 2>/dev/null | grep -q '^gitea/' \
  || fail "gitea mount not found after apply"
${COMPOSE} exec -T vault vault read gitea/roles/ci-bot >/dev/null \
  || fail "role ci-bot not found after apply"
green "mount + role present"

blue "Minting a token from the terraform-managed role"
JSON="$(${COMPOSE} exec -T vault vault read -format=json gitea/creds/ci-bot)"
TOKEN="$(printf '%s' "${JSON}" | jq_field '["data"]["token"]')"
USER="$(printf '%s' "${JSON}" | jq_field '["data"]["username"]')"
LEASE="$(printf '%s' "${JSON}" | jq_field '["lease_id"]')"
[ -n "${TOKEN}" ] || fail "no token minted"
[ "${USER}" = "ci-bot" ] || fail "unexpected username ${USER}"
green "minted token ${TOKEN:0:10}... for ${USER} (lease ${LEASE})"

blue "Revoking the lease"
${COMPOSE} exec -T vault vault lease revoke "${LEASE}" >/dev/null
green "lease revoked"

# ---------------------------------------------------------------------------
blue "terraform destroy (unmount engine)"
( cd "${E2E_DIR}" && "${TF}" destroy -auto-approve )
${COMPOSE} exec -T vault vault secrets list 2>/dev/null | grep -q '^gitea/' \
  && fail "gitea mount still present after destroy" || true
green "mount removed by destroy"

green "ALL PROVIDER END-TO-END CHECKS PASSED"
