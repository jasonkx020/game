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
        log_error "root or sudo required for this operation"
        return 1
    fi
}

add_firewall_rule() {
    if [[ "$ADD_FIREWALL" != "true" ]]; then
        return 0
    fi
    if command -v ufw &>/dev/null && ufw status 2>/dev/null | grep -qi "active"; then
        if ! ufw status | grep -q "${PG_PORT}/tcp"; then
            log_info "ufw allow ${PG_PORT}/tcp"
            maybe_sudo ufw allow "${PG_PORT}/tcp" comment "YCWL PostgreSQL" || true
        fi
    fi
}

wait_ready() {
    local deadline=$((SECONDS + 30))
    if [[ -z "$PG_ISREADY_EXE" ]]; then
        log_error "pg_isready not found"
        return 1
    fi
    while (( SECONDS < deadline )); do
        if "$PG_ISREADY_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_SUPERUSER" &>/dev/null; then
            return 0
        fi
        sleep 1
    done
    log_error "pg_isready timeout on ${POSTGRES_HOST}:${PG_PORT}"
    return 1
}

start_systemd() {
    log_info "starting systemd service: $SYSTEMD_SERVICE_NAME"
    if systemctl is-active --quiet "$SYSTEMD_SERVICE_NAME" 2>/dev/null; then
        log_info "service already running"
        return 0
    fi
    maybe_sudo systemctl start "$SYSTEMD_SERVICE_NAME"
}

start_pg_ctl() {
    if [[ -z "$PG_CTL_EXE" ]]; then
        log_error "pg_ctl not found; set install_path or install postgresql-client-${PG_VERSION}"
        exit 1
    fi
    if [[ ! -f "$DATA_PATH/PG_VERSION" ]]; then
        if [[ -z "$INITDB_EXE" ]]; then
            log_error "initdb not found"
            exit 1
        fi
        log_info "initializing cluster: $DATA_PATH"
        "$INITDB_EXE" -D "$DATA_PATH" -U "$POSTGRES_SUPERUSER" --encoding=UTF8 --locale=C
        bash "$SCRIPT_DIR/render-postgresql-conf.sh"
    fi
    bash "$SCRIPT_DIR/render-postgresql-conf.sh"
    if "$PG_CTL_EXE" status -D "$DATA_PATH" &>/dev/null; then
        log_info "cluster already running: $DATA_PATH"
        log_info "reloading configuration ..."
        "$PG_CTL_EXE" reload -D "$DATA_PATH" -s
        return 0
    fi
    log_info "starting PostgreSQL (pg_ctl) ..."
    "$PG_CTL_EXE" start -D "$DATA_PATH" -l "$LOG_DIR/postgresql.log" -w
}

main() {
    mkdir -p "$LOG_DIR"

    case "$RUN_MODE" in
        systemd)
            start_systemd
            ;;
        pg_ctl)
            start_pg_ctl
            ;;
        *)
            log_error "unknown run_mode: $RUN_MODE (use systemd or pg_ctl)"
            exit 1
            ;;
    esac

    add_firewall_rule
    wait_ready || log_info "server may still be starting"

    echo ""
    echo "========================================"
    echo "PostgreSQL started"
    echo "Mode:     $RUN_MODE"
    echo "Host:     ${POSTGRES_HOST}:${PG_PORT}"
    echo "App user: $POSTGRES_USER"
    echo "EDGE DB:  $POSTGRES_DB_EDGE"
    echo "CLOUD DB: $POSTGRES_DB_CLOUD"
    echo "Next:     ./init-db.sh tenant --schema"
    echo "tenant-api DSN:"
    echo "  $TENANT_DSN"
    echo "========================================"
}

main "$@"
