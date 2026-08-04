#!/usr/bin/env bash
# Parse deploy/run/postgresql/config.ini (aligned with read-postgres-config.ps1)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../../.." && pwd)"

CONFIG_INI="${CONFIG_INI:-$SCRIPT_DIR/config.ini}"

_ini_current_section=""
_ini_file="$CONFIG_INI"

read_ini_value() {
    local section="$1" key="$2" default="${3:-}"
    local current="" line trim
    if [[ ! -f "$_ini_file" ]]; then
        echo "$default"
        return 0
    fi
    while IFS= read -r line || [[ -n "$line" ]]; do
        trim="${line#"${line%%[![:space:]]*}"}"
        trim="${trim%"${trim##*[![:space:]]}"}"
        [[ -z "$trim" ]] && continue
        [[ "$trim" =~ ^[;#] ]] && continue
        if [[ "$trim" =~ ^\[(.+)\]$ ]]; then
            current="${BASH_REMATCH[1]}"
            continue
        fi
        if [[ "$current" != "$section" ]]; then
            continue
        fi
        if [[ "$trim" =~ ^([^=]+)=(.*)$ ]]; then
            local k="${BASH_REMATCH[1]}"
            k="${k#"${k%%[![:space:]]*}"}"
            k="${k%"${k##*[![:space:]]}"}"
            if [[ "$k" == "$key" ]]; then
                local v="${BASH_REMATCH[2]}"
                v="${v#"${v%%[![:space:]]*}"}"
                v="${v%"${v##*[![:space:]]}"}"
                echo "$v"
                return 0
            fi
        fi
    done < "$_ini_file"
    echo "$default"
}

resolve_config_path() {
    local base="$1" path="$2" fallback="$3"
    local p="${path:-}"
    if [[ -z "$p" ]]; then
        p="$fallback"
    fi
    if [[ "$p" == "." || "$p" == "./" ]]; then
        echo "$base"
        return 0
    fi
    if [[ "$p" == ./* ]]; then
        echo "$base/${p#./}"
        return 0
    fi
    if [[ "$p" != /* ]]; then
        echo "$base/$p"
        return 0
    fi
    echo "$p"
}

resolve_install_path() {
    local configured
    configured="$(read_ini_value PostgreSQL install_path ".")"
    local resolved
    resolved="$(resolve_config_path "$SCRIPT_DIR" "$configured" "$SCRIPT_DIR")"
    if [[ "$configured" != "." && "$configured" != "./" ]]; then
        echo "$resolved"
        return 0
    fi
    local version
    version="$(read_ini_value PostgreSQL version "16")"
    local candidates=(
        "$SCRIPT_DIR"
        "/usr/lib/postgresql/$version"
        "/usr/lib/postgresql"
    )
    local c
    for c in "$candidates"; do
        if [[ -x "$c/bin/psql" || -x "$c/psql" ]]; then
            echo "$c"
            return 0
        fi
    done
    echo "$resolved"
}

resolve_systemd_service_name() {
    local configured version unit
    configured="$(read_ini_value PostgreSQL systemd_service_name "")"
    if [[ -n "$configured" ]]; then
        echo "$configured"
        return 0
    fi
    version="$(read_ini_value PostgreSQL version "16")"
    unit="postgresql@${version}-main"
    if systemctl list-unit-files "${unit}.service" &>/dev/null 2>&1; then
        echo "$unit"
        return 0
    fi
    if systemctl list-unit-files postgresql.service &>/dev/null 2>&1; then
        echo "postgresql"
        return 0
    fi
    echo "$unit"
}

get_pg_exe() {
    local name="$1"
    local install_path candidates c cmd
    install_path="$(resolve_install_path)"
    candidates=(
        "$install_path/bin/$name"
        "$install_path/$name"
        "$SCRIPT_DIR/bin/$name"
        "$SCRIPT_DIR/$name"
    )
    for c in "$candidates"; do
        if [[ -x "$c" ]]; then
            echo "$c"
            return 0
        fi
    done
    if command -v "$name" &>/dev/null; then
        command -v "$name"
        return 0
    fi
    echo ""
}

ycwl_pg_load_config() {
    INSTALL_PATH="$(resolve_install_path)"
    DATA_PATH="$(resolve_config_path "$SCRIPT_DIR" "$(read_ini_value PostgreSQL data_path "data")" "$SCRIPT_DIR/data")"
    LOG_DIR="$(resolve_config_path "$SCRIPT_DIR" "$(read_ini_value PostgreSQL log_dir "log")" "$SCRIPT_DIR/log")"
    PG_PORT="$(read_ini_value PostgreSQL port "5432")"
    LISTEN_BIND="$(read_ini_value PostgreSQL listen_bind "127.0.0.1")"
    PG_VERSION="$(read_ini_value PostgreSQL version "16")"
    RUN_MODE="$(read_ini_value PostgreSQL run_mode "systemd")"
    RUN_MODE="${RUN_MODE,,}"
    # Linux: systemd | pg_ctl; legacy alias service -> systemd
    if [[ "$RUN_MODE" == "service" ]]; then
        RUN_MODE="systemd"
    fi
    SYSTEMD_SERVICE_NAME="$(resolve_systemd_service_name)"
    ADD_FIREWALL="$(read_ini_value PostgreSQL add_firewall_rules "true")"
    ADD_FIREWALL="${ADD_FIREWALL,,}"

    POSTGRES_SUPERUSER="$(read_ini_value Superuser user "postgres")"
    POSTGRES_SUPERUSER_PASSWORD="$(read_ini_value Superuser password "")"

    POSTGRES_USER="$(read_ini_value Application user "freetv")"
    POSTGRES_PASSWORD="$(read_ini_value Application password "changeme")"
    POSTGRES_DB_EDGE="$(read_ini_value Application db_edge "freetv_edge")"
    POSTGRES_DB_CLOUD="$(read_ini_value Application db_cloud "freetv_cloud")"
    DEFAULT_SCOPE="$(read_ini_value Init default_scope "tenant")"

    if [[ "$LISTEN_BIND" == "0.0.0.0" ]]; then
        POSTGRES_HOST="127.0.0.1"
    else
        POSTGRES_HOST="$LISTEN_BIND"
    fi
    POSTGRES_HOST_PORT="$PG_PORT"

    TENANT_DSN="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${PG_PORT}/${POSTGRES_DB_EDGE}?sslmode=disable"
    PLATFORM_DSN="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST}:${PG_PORT}/${POSTGRES_DB_CLOUD}?sslmode=disable"

    PG_CTL_EXE="$(get_pg_exe pg_ctl)"
    PSQL_EXE="$(get_pg_exe psql)"
    PG_ISREADY_EXE="$(get_pg_exe pg_isready)"
    INITDB_EXE="$(get_pg_exe initdb)"

    DB_INIT_SCRIPT="$REPO_ROOT/scripts/db-init.sh"
    DB_MIGRATE_SCRIPT="$REPO_ROOT/scripts/db-migrate.sh"
    CONFIG_FILE="$CONFIG_INI"
    SCRIPT_DIR="$SCRIPT_DIR"
    REPO_ROOT="$REPO_ROOT"
}

print_summary() {
    ycwl_pg_load_config
    echo "PostgreSQL config ($CONFIG_INI)"
    echo "  SCRIPT_DIR = $SCRIPT_DIR"
    echo "  INSTALL_PATH = $INSTALL_PATH"
    echo "  DATA_PATH = $DATA_PATH"
    echo "  LOG_DIR = $LOG_DIR"
    echo "  PG_PORT = $PG_PORT"
    echo "  LISTEN_BIND = $LISTEN_BIND"
    echo "  PG_VERSION = $PG_VERSION"
    echo "  RUN_MODE = $RUN_MODE"
    echo "  SYSTEMD_SERVICE_NAME = $SYSTEMD_SERVICE_NAME"
    echo "  POSTGRES_HOST = $POSTGRES_HOST"
    echo "  POSTGRES_USER = $POSTGRES_USER"
    echo "  POSTGRES_DB_EDGE = $POSTGRES_DB_EDGE"
    echo "  POSTGRES_DB_CLOUD = $POSTGRES_DB_CLOUD"
    echo "  TENANT_DSN = postgres://${POSTGRES_USER}:****@${POSTGRES_HOST}:${PG_PORT}/${POSTGRES_DB_EDGE}?sslmode=disable"
    echo "  PSQL_EXE = $PSQL_EXE"
}

write_env_file() {
    local out="$1"
    ycwl_pg_load_config
    local postgres_bin=""
    if [[ -x "$INSTALL_PATH/bin/psql" ]]; then
        postgres_bin="$INSTALL_PATH/bin"
    fi
    mkdir -p "$(dirname "$out")"
    {
        echo "# Generated by read-postgres-config.sh from config.ini"
        echo "POSTGRES_HOST=$POSTGRES_HOST"
        echo "POSTGRES_HOST_PORT=$PG_PORT"
        echo "POSTGRES_SUPERUSER=$POSTGRES_SUPERUSER"
        echo "POSTGRES_SUPERUSER_PASSWORD=$POSTGRES_SUPERUSER_PASSWORD"
        echo "POSTGRES_USER=$POSTGRES_USER"
        echo "POSTGRES_PASSWORD=$POSTGRES_PASSWORD"
        echo "POSTGRES_DB_EDGE=$POSTGRES_DB_EDGE"
        echo "POSTGRES_DB_CLOUD=$POSTGRES_DB_CLOUD"
        echo "POSTGRES_DB=$POSTGRES_DB_EDGE"
        if [[ -n "$postgres_bin" ]]; then
            echo "POSTGRES_HOME=$INSTALL_PATH"
            echo "POSTGRES_BIN=$postgres_bin"
        fi
        if [[ -n "$PSQL_EXE" ]]; then
            echo "POSTGRES_PSQL=$PSQL_EXE"
        fi
    } > "$out"
    echo "Wrote: $out"
}

emit_shell() {
    ycwl_pg_load_config
    local vars=(
        SCRIPT_DIR REPO_ROOT CONFIG_FILE INSTALL_PATH DATA_PATH LOG_DIR
        PG_PORT LISTEN_BIND PG_VERSION RUN_MODE SYSTEMD_SERVICE_NAME ADD_FIREWALL
        POSTGRES_SUPERUSER POSTGRES_SUPERUSER_PASSWORD POSTGRES_HOST POSTGRES_HOST_PORT
        POSTGRES_USER POSTGRES_PASSWORD POSTGRES_DB_EDGE POSTGRES_DB_CLOUD POSTGRES_DB
        DEFAULT_SCOPE TENANT_DSN PLATFORM_DSN
        PG_CTL_EXE PSQL_EXE PG_ISREADY_EXE INITDB_EXE
        DB_INIT_SCRIPT DB_MIGRATE_SCRIPT
    )
    POSTGRES_DB="$POSTGRES_DB_EDGE"
    local v val escaped
    for v in "${vars[@]}"; do
        val="${!v-}"
        escaped="${val//\'/\'\\\'\'}"
        printf '%s=%q\n' "$v" "$escaped"
    done
}

check_ubuntu_24_04() {
    if [[ ! -f /etc/os-release ]]; then
        echo "[ERROR] cannot detect OS (/etc/os-release missing)" >&2
        return 1
    fi
    # shellcheck disable=SC1091
    source /etc/os-release
    if [[ "${ID:-}" != "ubuntu" ]] || [[ "${VERSION_ID:-}" != "24.04" ]]; then
        echo "[WARN] scripts target Ubuntu 24.04 LTS; current: ${PRETTY_NAME:-unknown}" >&2
    fi
}

main() {
    case "${1:-}" in
        --print-summary|-PrintSummary)
            print_summary
            ;;
        --write-env-file|-WriteEnvFile)
            if [[ -z "${2:-}" ]]; then
                echo "usage: $0 --write-env-file <path>" >&2
                exit 1
            fi
            write_env_file "$2"
            ;;
        --emit-shell|-EmitShell)
            emit_shell
            ;;
        --check-os)
            check_ubuntu_24_04
            ;;
        --load)
            ycwl_pg_load_config
            ;;
        -h|--help)
            cat <<EOF
Usage: $0 [--print-summary | --write-env-file <path> | --emit-shell | --load]

Reads $CONFIG_INI (same format as Windows read-postgres-config.ps1).
Linux run_mode: systemd (apt service) or pg_ctl (local cluster).
EOF
            ;;
        "")
            ycwl_pg_load_config
            ;;
        *)
            echo "unknown option: $1" >&2
            exit 1
            ;;
    esac
}

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
else
    case "${1:-}" in
        --load) ycwl_pg_load_config ;;
        --print-summary|-PrintSummary) print_summary ;;
        --write-env-file|-WriteEnvFile)
            if [[ -z "${2:-}" ]]; then
                echo "usage: source $0 --write-env-file <path>" >&2
                return 1
            fi
            write_env_file "$2"
            ;;
        --emit-shell|-EmitShell) emit_shell ;;
        --check-os) check_ubuntu_24_04 ;;
        "") ycwl_pg_load_config ;;
        *)
            echo "unknown option: $1" >&2
            return 1
            ;;
    esac
fi
