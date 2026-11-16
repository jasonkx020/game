#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
bash "$SCRIPT_DIR/read-postgres-config.sh" --print-summary

# shellcheck source=read-postgres-config.sh
source "$SCRIPT_DIR/read-postgres-config.sh" --load

echo "========================================"

if command -v ss &>/dev/null; then
    if ss -tln 2>/dev/null | grep -q ":${PG_PORT} "; then
        echo "Port ${PG_PORT}: listening"
    else
        echo "Port ${PG_PORT}: not listening"
    fi
elif command -v netstat &>/dev/null; then
    if netstat -tln 2>/dev/null | grep -q ":${PG_PORT} "; then
        echo "Port ${PG_PORT}: listening"
    else
        echo "Port ${PG_PORT}: not listening"
    fi
fi

case "$RUN_MODE" in
    systemd|service)
        systemctl status "$SYSTEMD_SERVICE_NAME" --no-pager -l 2>/dev/null | head -n 3 || true
        ;;
esac

if [[ -n "$PG_ISREADY_EXE" ]]; then
    if "$PG_ISREADY_EXE" -h "$POSTGRES_HOST" -p "$PG_PORT" -U "$POSTGRES_SUPERUSER"; then
        echo "pg_isready: accepting connections"
    else
        echo "pg_isready: not ready"
    fi
else
    echo "pg_isready: not found"
fi

if [[ -f "$DATA_PATH/PG_VERSION" ]]; then
    echo "Cluster data: $DATA_PATH (PG_VERSION present)"
elif [[ "$RUN_MODE" == "pg_ctl" ]]; then
    echo "Cluster data: not initialized (run start-postgres.sh)"
fi

if [[ -n "$PSQL_EXE" && -x "$PSQL_EXE" ]]; then
    echo "Client: $PSQL_EXE"
else
    echo "Client: psql not found (install postgresql-client-${PG_VERSION})"
fi
