#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=read-postgres-config.sh
source "$SCRIPT_DIR/read-postgres-config.sh" --load

if [[ -z "$PSQL_EXE" || ! -x "$PSQL_EXE" ]]; then
    echo "[ERROR] psql not found: $PSQL_EXE" >&2
    exit 1
fi

if [[ $# -gt 0 ]]; then
    export PGPASSWORD="${POSTGRES_PASSWORD:-}"
    exec "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" "$@"
fi

menu() {
    echo "========================================"
    echo "PostgreSQL client (psql)"
    echo "Host: ${POSTGRES_HOST}:${PG_PORT}"
    echo "User: $POSTGRES_USER"
    echo "========================================"
    echo "  1. Connect game DB ($POSTGRES_DB)"
    if [[ -n "${POSTGRES_DB_AUTH:-}" ]]; then
        echo "  2. Connect auth DB ($POSTGRES_DB_AUTH)"
    else
        echo "  2. Connect auth DB (not configured)"
    fi
    echo "  3. Connect as superuser ($POSTGRES_SUPERUSER, postgres DB)"
    echo "  0. Exit"
    echo "========================================"
    read -r -p "Select [0-3]: " choice
    case "$choice" in
        1)
            export PGPASSWORD="${POSTGRES_PASSWORD:-}"
            "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB"
            ;;
        2)
            if [[ -z "${POSTGRES_DB_AUTH:-}" ]]; then
                echo "Application.db_auth is empty in config.ini"
                return 0
            fi
            export PGPASSWORD="${POSTGRES_PASSWORD:-}"
            "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB_AUTH"
            ;;
        3)
            if [[ -n "${POSTGRES_SUPERUSER_PASSWORD:-}" ]]; then
                export PGPASSWORD="$POSTGRES_SUPERUSER_PASSWORD"
            else
                unset PGPASSWORD || true
            fi
            "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_SUPERUSER" -d postgres
            ;;
        0)
            exit 0
            ;;
        *)
            echo "invalid choice"
            ;;
    esac
}

while true; do
    menu
done
