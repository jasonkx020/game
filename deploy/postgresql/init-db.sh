#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

usage() {
    cat <<EOF
Usage: $0 [game|auth|all] [--split-auth] [--write-env|--no-write-env] [--app-password PASS]

Self-contained GameS DB bootstrap (role + database). Does not run migrations.
EOF
}

if [[ "${1:-}" == "-h" || "${1:-}" == "--help" ]]; then
    usage
    exit 0
fi

exec bash "$SCRIPT_DIR/db-init.sh" "$@"
