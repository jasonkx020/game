#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=read-postgres-config.sh
source "$SCRIPT_DIR/read-postgres-config.sh" --load

SCOPE=""
WITH_SCHEMA=0

usage() {
    cat <<EOF
Usage: $0 [tenant|platform|all] [--schema]

Wraps scripts/db-init.sh using config.ini (writes local postgres.env).
EOF
}

while [[ $# -gt 0 ]]; do
    case "$1" in
        tenant|platform|all) SCOPE="$1"; shift ;;
        --schema|-Schema) WITH_SCHEMA=1; shift ;;
        -h|--help) usage; exit 0 ;;
        *) echo "unknown argument: $1" >&2; usage; exit 1 ;;
    esac
done

if [[ -z "$SCOPE" ]]; then
    SCOPE="$DEFAULT_SCOPE"
fi

ENV_FILE="$SCRIPT_DIR/postgres.env"

if [[ "$RUN_MODE" == "pg_ctl" ]]; then
    bash "$SCRIPT_DIR/render-postgresql-conf.sh"
    if [[ -f "$DATA_PATH/PG_VERSION" && -n "$PG_CTL_EXE" ]]; then
        if "$PG_CTL_EXE" status -D "$DATA_PATH" &>/dev/null; then
            echo "[INFO] reloading pg_hba.conf ..."
            "$PG_CTL_EXE" reload -D "$DATA_PATH" -s
        fi
    fi
fi

bash "$SCRIPT_DIR/read-postgres-config.sh" --write-env-file "$ENV_FILE"

if [[ ! -f "$DB_INIT_SCRIPT" ]]; then
    echo "[ERROR] missing: $DB_INIT_SCRIPT" >&2
    exit 1
fi

if [[ -z "$PSQL_EXE" || ! -x "$PSQL_EXE" ]]; then
    echo "[ERROR] psql not found; run install-postgres.sh or set install_path in config.ini" >&2
    exit 1
fi

export POSTGRES_CONFIG_FILE="$ENV_FILE"
export YCWL_SCRIPT_DIR="$REPO_ROOT/scripts"
export YCWL_REPO_ROOT="$REPO_ROOT"

args=("$SCOPE")
if [[ "$WITH_SCHEMA" -eq 1 ]]; then
    args+=(--schema)
fi

echo "[INFO] running db-init: scope=$SCOPE schema=$WITH_SCHEMA"
bash "$DB_INIT_SCRIPT" "${args[@]}"

echo ""
echo "Database init completed."
echo "Verify: bash $DB_MIGRATE_SCRIPT status tenant"
