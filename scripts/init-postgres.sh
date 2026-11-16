#!/usr/bin/env bash
# Thin wrapper → deploy/postgresql/db-init.sh
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
exec bash "$ROOT/deploy/postgresql/db-init.sh" "$@"
