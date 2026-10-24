#!/usr/bin/env bash
set -euo pipefail
app_dir="$(cd "$(dirname "$0")" && pwd)"
cd "$app_dir/frontend"
if [ ! -d node_modules ]; then npm ci --no-audit --no-fund; fi
npm run build
cd "$app_dir/backend"
export GOCACHE="${GOCACHE:-$app_dir/.cache/go-build}"
mkdir -p "$app_dir/.cache"
go build -o "$app_dir/.cache/math-server" ./cmd/server
exec "$app_dir/.cache/math-server" -addr "${MATH_ADDR:-127.0.0.1:8080}" -content "$app_dir/content" -web "$app_dir/frontend/dist" -db "$app_dir/var/math.db"
