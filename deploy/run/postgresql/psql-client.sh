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
    export PGPASSWORD="$POSTGRES_PASSWORD"
    exec "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB_EDGE" "$@"
fi

menu() {
    echo "========================================"
    echo "PostgreSQL client (psql)"
    echo "Host: ${POSTGRES_HOST}:${PG_PORT}"
    echo "User: $POSTGRES_USER"
    echo "========================================"
    echo "  1. Connect EDGE  ($POSTGRES_DB_EDGE)"
    echo "  2. Connect CLOUD ($POSTGRES_DB_CLOUD)"
    echo "  3. Connect as superuser ($POSTGRES_SUPERUSER, postgres DB)"
    echo "  4. Show migration status (tenant)"
    echo "  0. Exit"
    echo "========================================"
    read -r -p "Select [0-4]: " choice
    case "$choice" in
        1)
            export PGPASSWORD="$POSTGRES_PASSWORD"
            "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB_EDGE"
            ;;
        2)
            export PGPASSWORD="$POSTGRES_PASSWORD"
            "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB_CLOUD"
            ;;
        3)
            export PGPASSWORD="$POSTGRES_SUPERUSER_PASSWORD"
            "$PSQL_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_SUPERUSER" -d postgres
            ;;
        4)
            if [[ -f "$DB_MIGRATE_SCRIPT" ]]; then
                bash "$DB_MIGRATE_SCRIPT" status tenant
            else
                echo "Missing $DB_MIGRATE_SCRIPT"
            fi
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
