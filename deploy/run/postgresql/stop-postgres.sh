#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=read-postgres-config.sh
source "$SCRIPT_DIR/read-postgres-config.sh" --load

log_info() { echo "[INFO] $*"; }
log_error() { echo "[ERROR] $*" >&2; }

maybe_sudo() {
    if [[ "${EUID:-1}" -eq 0 ]]; then
        "$@"
    elif command -v sudo &>/dev/null; then
        sudo "$@"
    else
        log_error "root or sudo required"
        return 1
    fi
}

stop_systemd() {
    log_info "stopping systemd service: $SYSTEMD_SERVICE_NAME"
    if systemctl is-active --quiet "$SYSTEMD_SERVICE_NAME" 2>/dev/null; then
        maybe_sudo systemctl stop "$SYSTEMD_SERVICE_NAME"
        log_info "PostgreSQL service stopped"
    else
        log_info "service already stopped"
    fi
}

stop_pg_ctl() {
    if [[ -z "$PG_CTL_EXE" ]]; then
        log_error "pg_ctl not found: $PG_CTL_EXE"
        exit 1
    fi
    if [[ ! -f "$DATA_PATH/PG_VERSION" ]]; then
        log_info "no cluster at $DATA_PATH"
        exit 0
    fi
    log_info "stopping cluster: $DATA_PATH"
    if "$PG_CTL_EXE" stop -D "$DATA_PATH" -m fast -w; then
        log_info "PostgreSQL stopped"
    else
        log_error "pg_ctl stop failed or server not running"
        exit 1
    fi
}

case "$RUN_MODE" in
    systemd)
        stop_systemd
        ;;
    pg_ctl)
        stop_pg_ctl
        ;;
    service)
        stop_systemd
        ;;
    *)
        log_error "unknown run_mode: $RUN_MODE"
        exit 1
        ;;
esac
