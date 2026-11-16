#!/usr/bin/env bash
# Install PostgreSQL from Ubuntu apt (optional first-time setup)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=read-postgres-config.sh
source "$SCRIPT_DIR/read-postgres-config.sh" --load

log_info() { echo "[INFO] $*"; }
log_error() { echo "[ERROR] $*" >&2; }

check_root() {
    if [[ "${EUID:-1}" -ne 0 ]]; then
        log_error "run as root: sudo $0"
        exit 1
    fi
}

check_os() {
    if [[ ! -f /etc/os-release ]]; then
        log_error "missing /etc/os-release"
        exit 1
    fi
    # shellcheck disable=SC1091
    source /etc/os-release
    if [[ "${ID:-}" != "ubuntu" ]] || [[ "${VERSION_ID:-}" != "24.04" ]]; then
        log_error "only Ubuntu 24.04 LTS supported; current: ${PRETTY_NAME:-unknown}"
        exit 1
    fi
}

main() {
    check_root
    check_os
    export DEBIAN_FRONTEND=noninteractive
    log_info "installing PostgreSQL ${PG_VERSION} ..."
    apt-get update -qq
    apt-get install -y --no-install-recommends \
        "postgresql-${PG_VERSION}" \
        "postgresql-client-${PG_VERSION}" \
        "postgresql-contrib-${PG_VERSION}"
    log_info "installed. Set run_mode=systemd in config.ini and run start-postgres.sh"
    systemctl is-active --quiet "$SYSTEMD_SERVICE_NAME" 2>/dev/null && \
        log_info "service $SYSTEMD_SERVICE_NAME is already running"
}

main "$@"
