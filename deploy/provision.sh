#!/usr/bin/env bash
# Idempotent provisioning + deploy script for notesview.
#
# Runs on the target VPS (Debian 12 / Ubuntu 24.04). Called by the GitHub
# Actions deploy workflow after it rsyncs this directory and the freshly built
# binary to /opt/notesview-deploy/. Safe to re-run: first run installs Caddy,
# creates users/dirs, writes configs, and starts the service; subsequent runs
# only swap the binary and reload changed configs.
#
# Required env vars (passed via /opt/notesview-deploy/.env, written by CI):
#   CADDY_DOMAIN         - hostname for Caddy auto-TLS (e.g. notes.fffeeder.com)
#   BASIC_AUTH_ENABLED   - "1" to require basic auth, "0" to disable
#   BASIC_AUTH_USER      - username (required when enabled)
#   BASIC_AUTH_HASH      - bcrypt hash from `caddy hash-password` (required when enabled)

set -euo pipefail

DEPLOY_DIR="/opt/notesview-deploy"
BIN_DIR="/var/lib/notesview/bin"
NOTES_DIR="/var/lib/notesview/notes"
CADDY_FILE="/etc/caddy/Caddyfile"
SYSTEMD_UNIT="/etc/systemd/system/notesview.service"

# shellcheck disable=SC1091
[ -f "${DEPLOY_DIR}/.env" ] && source "${DEPLOY_DIR}/.env"

: "${CADDY_DOMAIN:?CADDY_DOMAIN not set}"
: "${BASIC_AUTH_ENABLED:=0}"

log() { printf '==> %s\n' "$*"; }

# Record config state so handlers restart/reload only when files change.
changed_caddy=0
changed_unit=0

# --- packages ---------------------------------------------------------------
log "ensuring apt packages"
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get install -y --no-install-recommends \
    ca-certificates curl gnupg debian-keyring debian-archive-keyring \
    apt-transport-https rsync ufw

if [ ! -f /usr/share/keyrings/caddy-stable-archive-keyring.gpg ]; then
    log "adding Caddy apt repo"
    curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/gpg.key \
        | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt \
        | tee /etc/apt/sources.list.d/caddy-stable.list >/dev/null
    apt-get update -qq
fi
apt-get install -y --no-install-recommends caddy

# --- firewall (idempotent; no-op if already configured) ---------------------
log "ensuring UFW rules"
ufw allow OpenSSH >/dev/null
ufw allow 80/tcp >/dev/null
ufw allow 443/tcp >/dev/null
ufw --force enable >/dev/null

# --- user and directories ---------------------------------------------------
if ! id notesview >/dev/null 2>&1; then
    log "creating notesview system user"
    useradd --system --home /var/lib/notesview --shell /usr/sbin/nologin notesview
fi
install -d -o notesview -g notesview -m 0755 /var/lib/notesview
install -d -o notesview -g notesview -m 0755 "${BIN_DIR}"
install -d -o notesview -g notesview -m 0755 "${NOTES_DIR}"

# --- systemd unit -----------------------------------------------------------
if ! cmp -s "${DEPLOY_DIR}/files/notesview.service" "${SYSTEMD_UNIT}"; then
    log "updating systemd unit"
    install -m 0644 "${DEPLOY_DIR}/files/notesview.service" "${SYSTEMD_UNIT}"
    changed_unit=1
fi

# --- Caddyfile (rendered from env) ------------------------------------------
tmp_caddy="$(mktemp)"
trap 'rm -f "${tmp_caddy}"' EXIT

{
    printf '%s {\n' "${CADDY_DOMAIN}"
    printf '    encode gzip\n'
    if [ "${BASIC_AUTH_ENABLED}" = "1" ]; then
        : "${BASIC_AUTH_USER:?BASIC_AUTH_USER required when auth enabled}"
        : "${BASIC_AUTH_HASH:?BASIC_AUTH_HASH required when auth enabled}"
        printf '    basic_auth {\n'
        printf '        %s %s\n' "${BASIC_AUTH_USER}" "${BASIC_AUTH_HASH}"
        printf '    }\n'
    fi
    # flush_interval -1 disables proxy buffering so the SSE live-reload stream
    # reaches the browser in real time.
    printf '    reverse_proxy 127.0.0.1:8080 {\n'
    printf '        flush_interval -1\n'
    printf '    }\n'
    printf '}\n'
} > "${tmp_caddy}"

if ! cmp -s "${tmp_caddy}" "${CADDY_FILE}"; then
    log "updating Caddyfile (auth enabled=${BASIC_AUTH_ENABLED})"
    install -m 0644 "${tmp_caddy}" "${CADDY_FILE}"
    changed_caddy=1
fi

# --- binary (atomic swap) ---------------------------------------------------
if [ ! -f "${DEPLOY_DIR}/notesview" ]; then
    echo "error: ${DEPLOY_DIR}/notesview missing; CI must upload the binary" >&2
    exit 1
fi
log "installing notesview binary"
install -m 0755 -o notesview -g notesview \
    "${DEPLOY_DIR}/notesview" "${BIN_DIR}/notesview.new"
mv -f "${BIN_DIR}/notesview.new" "${BIN_DIR}/notesview"

# --- reloads / restarts -----------------------------------------------------
if [ "${changed_unit}" = "1" ]; then
    log "reloading systemd"
    systemctl daemon-reload
fi
systemctl enable --now notesview.service >/dev/null
# Binary was replaced, so always restart notesview.
log "restarting notesview"
systemctl restart notesview.service

systemctl enable --now caddy.service >/dev/null
if [ "${changed_caddy}" = "1" ]; then
    log "reloading Caddy"
    systemctl reload caddy.service
fi

log "done"
