#!/bin/bash

# Clear terminal screen
clear

echo "=================================================="
echo "          OMNISYNC WMS DECOUPLED SUITE            "
echo "=================================================="
echo "Initializing environment..."

# 1. Set Go module caching locally inside the workspace to keep system clean
export GOMODCACHE="$PWD/go_cache/pkg/mod"
export GOCACHE="$PWD/go_cache/build"

# ── Helper: load a .env file into the current shell session ───────────────────
load_env() {
    local env_file="$1"
    if [ ! -f "$env_file" ]; then
        echo "ERROR: $env_file not found. Copy .env.example to .env and fill in values."
        exit 1
    fi
    # Export all non-comment, non-empty KEY=VALUE lines
    set -a
    # shellcheck disable=SC1090
    source <(grep -v '^\s*#' "$env_file" | grep -v '^\s*$')
    set +a
}

# Load env vars from both service directories
load_env "$PWD/auth_services/.env"
load_env "$PWD/wms_dashboard/.env"

# Configure dynamic ports (allow override, fall back to .env / defaults)
AUTH_PORT=${AUTH_PORT:-8000}
WMS_PORT=${WMS_PORT:-9901}

echo "Dependencies path: $GOMODCACHE"
echo ""

# 2. Boot Auth Service with all required env vars forwarded
echo "[1/2] Booting Standalone Auth Service (Port $AUTH_PORT)..."
(
  cd auth_services
  PORT=$AUTH_PORT \
  JWT_SECRET_KEY=$JWT_SECRET_KEY \
  ALLOWED_ORIGIN="http://localhost:$WMS_PORT,http://127.0.0.1:$WMS_PORT" \
  DB_TYPE=$DB_TYPE \
  AUTH_DATABASE_URL=$AUTH_DATABASE_URL \
  go run cmd/main.go > ../auth_service.log 2>&1
) &
AUTH_PID=$!

# 3. Boot WMS Dashboard with all required env vars forwarded
echo "[2/2] Booting WMS Dashboard Service (Port $WMS_PORT)..."
(
  cd wms_dashboard
  PORT=$WMS_PORT \
  JWT_SECRET_KEY=$JWT_SECRET_KEY \
  AUTH_API_URL="http://localhost:$AUTH_PORT" \
  DB_TYPE=$DB_TYPE \
  WMS_DATABASE_URL=$WMS_DATABASE_URL \
  go run cmd/main.go > ../wms_dashboard.log 2>&1
) &
WMS_PID=$!

echo ""
echo "--------------------------------------------------"
echo "SUCCESS: Both services launched in the background!"
echo "1. Authentication API: http://localhost:$AUTH_PORT  (Logs: auth_service.log)"
echo "2. WMS Dashboard:      http://localhost:$WMS_PORT  (Logs: wms_dashboard.log)"
echo "--------------------------------------------------"
echo "Press Ctrl+C to cleanly stop both services."
echo "=================================================="

# Graceful shutdown of background jobs on Ctrl+C (SIGINT)
trap "echo -e '\nStopping services...'; kill $AUTH_PID $WMS_PID; exit" INT
wait
