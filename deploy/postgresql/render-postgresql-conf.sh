#!/usr/bin/env bash
# Render postgresql.conf / pg_hba.conf from templates (pg_ctl mode only)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONFIG_INI="${CONFIG_INI:-$SCRIPT_DIR/config.ini}"

# shellcheck source=read-postgres-config.sh
source "$SCRIPT_DIR/read-postgres-config.sh" --load

template_conf="$SCRIPT_DIR/postgresql.conf.template"
template_hba="$SCRIPT_DIR/pg_hba.conf.template"
output_conf="$DATA_PATH/postgresql.conf"
output_hba="$DATA_PATH/pg_hba.conf"

mkdir -p "$DATA_PATH" "$LOG_DIR"

log_dir_unix="${LOG_DIR//\\//}"
super_password="$POSTGRES_SUPERUSER_PASSWORD"
if [[ -n "$super_password" ]]; then
    local_auth="scram-sha-256"
    auth_hint="# scram-sha-256 (superuser password configured in config.ini)"
else
    local_auth="trust"
    auth_hint="# trust on 127.0.0.1 when [Superuser] password is empty (local dev only)"
fi

if [[ -f "$template_conf" ]]; then
    content="$(<"$template_conf")"
    content="${content//__LISTEN_BIND__/$LISTEN_BIND}"
    content="${content//__PORT__/$PG_PORT}"
    content="${content//__LOG_DIR__/$log_dir_unix}"
    printf '%s\n' "$content" > "$output_conf"
    echo "Rendered: $output_conf"
fi

if [[ -f "$template_hba" ]]; then
    hba="$(<"$template_hba")"
    hba="${hba//__AUTH_HINT__/$auth_hint}"
    hba="${hba//__LOCAL_AUTH__/$local_auth}"
    printf '%s\n' "$hba" > "$output_hba"
    echo "Rendered: $output_hba (local auth: $local_auth)"
fi
