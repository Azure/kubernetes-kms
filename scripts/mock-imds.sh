#!/usr/bin/env bash

# Minimal mock IMDS for CI.
#
# Serves a real Key Vault access token (minted from the Workload Identity
# Federation `az login` session) at the managed-identity IMDS token endpoint, so
# the KMS plugin's unchanged `useManagedIdentityExtension` path keeps working now
# that the pool managed identity is gone. The plugin's IMDS calls
# (169.254.169.254) are redirected here with an iptables DNAT rule.
#
# Usage:
#   scripts/mock-imds.sh up host   # plugin runs on the agent host (integration test)
#   scripts/mock-imds.sh up kind   # plugin runs in the KIND node static pod (e2e)
#   scripts/mock-imds.sh down      # stop and remove redirects (best effort)
#
# For "kind" the KMS plugin is a static pod that the apiserver depends on during
# `kubeadm init`, so the mock must be serving before the cluster finishes coming
# up. The in-node redirect needs the node container, which only exists once
# `kind create cluster` starts, so `up kind` returns immediately after launching
# a detached watcher that injects the redirect the moment the node appears. Run
# `up kind` BEFORE creating the cluster.

set -o errexit
set -o nounset
set -o pipefail

PORT="${MOCK_IMDS_PORT:-8080}"
RESOURCE="${KEYVAULT_RESOURCE:-https://vault.azure.net}"
KIND_NODE="${KIND_NODE:-kms-control-plane}"
KIND_NETWORK="${KIND_NETWORK:-kind}"
STATE_DIR="${MOCK_IMDS_STATE_DIR:-tests/e2e/generated_manifests}"
DOC_ROOT="${STATE_DIR}/mock-imds"
TOKEN_FILE="${DOC_ROOT}/metadata/identity/oauth2/token"
PID_FILE="${STATE_DIR}/mock-imds.pid"
WATCHER_PID_FILE="${STATE_DIR}/mock-imds-watcher.pid"
LOG_FILE="${STATE_DIR}/mock-imds.log"
IMDS_IP="169.254.169.254"

log() { echo "[mock-imds] $*"; }

# Mint a Key Vault token from the WIF az session and write it in IMDS response
# shape. The federated idToken is short lived, but the resulting Key Vault token
# is valid ~1h, which covers a CI run, so we fetch once.
build_token() {
    command -v python3 >/dev/null 2>&1 || sudo tdnf install -y python3
    local cmd=(az account get-access-token --resource "${RESOURCE}")
    # The WIF login uses --allow-no-subscriptions, so scope the token by tenant.
    local tenant="${ARM_TENANT_ID:-${AZURE_TENANT_ID:-}}"
    [[ -n "${tenant}" ]] && cmd+=(--tenant "${tenant}")
    local access_token
    access_token="$("${cmd[@]}" --query accessToken -o tsv)"
    mkdir -p "$(dirname "${TOKEN_FILE}")"
    # 0600 so the served token is not world-readable on the shared agent.
    (umask 077 && printf '{"access_token":"%s","expires_in":"3600","expires_on":"%s","token_type":"Bearer","resource":"%s"}' \
        "${access_token}" "$(($(date +%s) + 3600))" "${RESOURCE}" >"${TOKEN_FILE}")
}

# Bind only to the address the plugin reaches (loopback for the host test, the
# KIND bridge gateway for e2e) so the token is not exposed on other interfaces.
serve() {
    local bind="$1"
    nohup python3 -m http.server "${PORT}" --bind "${bind}" --directory "${DOC_ROOT}" >>"${LOG_FILE}" 2>&1 &
    echo "$!" >"${PID_FILE}"
    sleep 1
    if ! curl -sf "http://${bind}:${PORT}/metadata/identity/oauth2/token" >/dev/null; then
        log "mock server did not come up; log:"
        cat "${LOG_FILE}" || true
        return 1
    fi
    log "serving on ${bind}:${PORT} (pid $(cat "${PID_FILE}"))"
}

# Host: redirect locally generated traffic (e.g. the plugin running under sudo).
dnat_host() {
    sudo sysctl -w net.ipv4.conf.all.route_localnet=1 >/dev/null 2>&1 || true
    sudo iptables -t nat "$1" OUTPUT -p tcp -d "${IMDS_IP}" --dport 80 \
        -j DNAT --to-destination "127.0.0.1:${PORT}"
}

# The host IP reachable from the node: the node's own default gateway (the KIND
# bridge host side). The mock binds this on the host and the node DNATs to it.
# Read from the node's routing table, which is more reliable than parsing the
# docker network IPAM gateway (which can be empty).
kind_gateway() {
    docker exec "${KIND_NODE}" ip route show default 2>/dev/null | awk '{print $3; exit}'
}

# KIND: the static pod uses hostNetwork, so redirect from the node netns to the
# mock running on the host (reachable via the KIND bridge gateway).
dnat_kind() {
    docker exec "${KIND_NODE}" iptables -t nat "$1" OUTPUT -p tcp -d "${IMDS_IP}" --dport 80 \
        -j DNAT --to-destination "$2:${PORT}"
}

# Some agents default the INPUT policy to DROP; allow the node to reach the mock
# on the host. The mock only binds the bridge gateway IP, so this exposes nothing
# beyond the KIND network.
host_input() {
    sudo iptables "$1" INPUT -p tcp --dport "${PORT}" -j ACCEPT
}

wait_for_node() {
    for _ in $(seq 180); do
        if [[ "$(docker inspect -f '{{.State.Running}}' "${KIND_NODE}" 2>/dev/null)" == "true" ]]; then
            return 0
        fi
        sleep 1
    done
    log "timed out waiting for node container ${KIND_NODE}"
    return 1
}

# Verify the redirect actually works from inside the node netns, using bash
# /dev/tcp since the node image has no curl. Logs OK or FAILED for diagnostics.
verify_from_node() {
    if docker exec "${KIND_NODE}" timeout 5 bash -c \
        'exec 3<>/dev/tcp/'"${IMDS_IP}"'/80 && printf "GET /metadata/identity/oauth2/token HTTP/1.0\r\nHost: imds\r\n\r\n" >&3 && head -c 64 <&3' \
        >>"${LOG_FILE}" 2>&1; then
        log "node -> mock redirect OK"
    else
        log "node -> mock redirect FAILED (check DNAT, host firewall, or gateway $1)"
    fi
}

# Runs detached: wait for the node, serve, and inject the redirect before the
# apiserver (kubeadm init) gives up on the kms static pod.
watch_kind() {
    log "watcher: waiting for node ${KIND_NODE}"
    wait_for_node || return 1
    local gateway
    gateway="$(kind_gateway)"
    [[ -n "${gateway}" ]] || {
        log "watcher: could not determine gateway from node ${KIND_NODE}"
        return 1
    }
    log "watcher: node up, host gateway=${gateway}"
    host_input -D 2>/dev/null || true
    host_input -I
    serve "${gateway}" || return 1
    local added=""
    for _ in $(seq 30); do
        dnat_kind -D "${gateway}" 2>/dev/null || true
        if dnat_kind -A "${gateway}" 2>/dev/null; then
            added=1
            break
        fi
        sleep 2
    done
    [[ -n "${added}" ]] || {
        log "watcher: failed to add node DNAT after retries"
        return 1
    }
    log "watcher: DNAT added (${KIND_NODE} ${IMDS_IP}:80 -> ${gateway}:${PORT})"
    verify_from_node "${gateway}"
}

up() {
    mkdir -p "${STATE_DIR}"
    : >"${LOG_FILE}"
    build_token
    case "$1" in
    host)
        serve 127.0.0.1
        dnat_host -D 2>/dev/null || true
        dnat_host -A
        log "mock IMDS ready (host)"
        ;;
    kind)
        # Fully detach (new session) so the agent does not reap the watcher when
        # this step ends; it must outlive the step to inject the redirect during
        # cluster creation.
        if command -v setsid >/dev/null 2>&1; then
            setsid "$0" __watch_kind >>"${LOG_FILE}" 2>&1 </dev/null &
        else
            nohup "$0" __watch_kind >>"${LOG_FILE}" 2>&1 </dev/null &
            disown || true
        fi
        echo "$!" >"${WATCHER_PID_FILE}"
        log "kind watcher started (pid $(cat "${WATCHER_PID_FILE}")); redirect lands once ${KIND_NODE} appears"
        ;;
    *)
        log "unknown mode: $1 (expected host|kind)"
        return 1
        ;;
    esac
}

down() {
    local pidfile
    for pidfile in "${WATCHER_PID_FILE}" "${PID_FILE}"; do
        if [[ -f "${pidfile}" ]]; then
            kill "$(cat "${pidfile}")" 2>/dev/null || true
            rm -f "${pidfile}"
        fi
    done
    dnat_host -D 2>/dev/null || true
    host_input -D 2>/dev/null || true
    local gateway
    gateway="$(kind_gateway 2>/dev/null || true)"
    if [[ -n "${gateway}" ]]; then
        dnat_kind -D "${gateway}" 2>/dev/null || true
    fi
    rm -rf "${DOC_ROOT}"
    log "mock IMDS torn down"
}

case "${1:-}" in
up) up "${2:-}" ;;
down) down ;;
__watch_kind) watch_kind ;;
*)
    echo "usage: $0 {up host|up kind|down}" >&2
    exit 1
    ;;
esac
