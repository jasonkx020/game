#!/usr/bin/env bash
set -euo pipefail

NGINX_ROOT="$(cd "$(dirname "$0")" && pwd)"
cd "$NGINX_ROOT"

mkdir -p logs temp/client_body_temp temp/proxy_temp temp/fastcgi_temp temp/uwsgi_temp temp/scgi_temp

if ! command -v nginx >/dev/null 2>&1; then
    echo "Error: nginx not found in PATH" >&2
    exit 1
fi

if [[ ! -f conf/nginx.conf ]]; then
    echo "Error: conf/nginx.conf not found" >&2
    exit 1
fi

if [[ ! -f conf/ssl/dev-server.crt ]]; then
    echo "Warning: conf/ssl/dev-server.crt not found; TLS listeners will fail until certs are present" >&2
fi

echo "Testing configuration (prefix: $NGINX_ROOT)..."
nginx -p "$NGINX_ROOT" -c conf/nginx.conf -t

echo "Starting Nginx..."
nginx -p "$NGINX_ROOT" -c conf/nginx.conf

echo "Nginx started."
echo "  Tenant:        https://127.0.0.1:8081/"
echo "  MinIO API:     https://127.0.0.1:9002/"
echo "  MinIO Console: https://127.0.0.1:9003/"
echo "  Platform admin: http://127.0.0.1:8098/"
echo "  Logs:          $NGINX_ROOT/logs/"
