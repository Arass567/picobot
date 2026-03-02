#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
API_PORT="${API_PORT:-8080}"
WEB_PORT="${WEB_PORT:-5173}"
DB_URL="${AZIMUTPLUS_DATABASE_URL:-postgres://admin@127.0.0.1:5432/azimutplus?sslmode=disable}"

# Demo defaults (override with env if needed)
export AZIMUTPLUS_API_KEYS="${AZIMUTPLUS_API_KEYS:-demo-key}"
export AZIMUTPLUS_WEBHOOK_SECRETS="${AZIMUTPLUS_WEBHOOK_SECRETS:-demo-secret}"
export AZIMUTPLUS_ADMIN_TOKEN="${AZIMUTPLUS_ADMIN_TOKEN:-demo-admin}"
export AZIMUTPLUS_RATE_LIMIT_PER_MINUTE="${AZIMUTPLUS_RATE_LIMIT_PER_MINUTE:-300}"
export AZIMUTPLUS_RATE_LIMIT_WINDOW_SECONDS="${AZIMUTPLUS_RATE_LIMIT_WINDOW_SECONDS:-60}"
export AZIMUTPLUS_DATABASE_URL="$DB_URL"

API_PID=""
WEB_PID=""

cleanup() {
  echo
  echo "[one-click] arrêt des services..."
  if [[ -n "$API_PID" ]] && kill -0 "$API_PID" 2>/dev/null; then
    kill "$API_PID" >/dev/null 2>&1 || true
    wait "$API_PID" 2>/dev/null || true
  fi
  if [[ -n "$WEB_PID" ]] && kill -0 "$WEB_PID" 2>/dev/null; then
    kill "$WEB_PID" >/dev/null 2>&1 || true
    wait "$WEB_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT INT TERM

command -v go >/dev/null 2>&1 || { echo "[one-click] go introuvable"; exit 1; }
command -v npm >/dev/null 2>&1 || { echo "[one-click] npm introuvable"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "[one-click] curl introuvable"; exit 1; }

if command -v brew >/dev/null 2>&1; then
  echo "[one-click] démarrage PostgreSQL (brew services)..."
  brew services start postgresql@16 >/dev/null 2>&1 || true
fi

if command -v /usr/local/opt/postgresql@16/bin/createdb >/dev/null 2>&1; then
  /usr/local/opt/postgresql@16/bin/createdb -h 127.0.0.1 -U admin azimutplus >/dev/null 2>&1 || true
fi

echo "[one-click] préparation frontend"
cd "$ROOT_DIR/web"
npm install >/dev/null
cat > .env <<ENV
VITE_API_BASE_URL=http://127.0.0.1:${API_PORT}
VITE_API_KEY=${AZIMUTPLUS_API_KEYS%%,*}
ENV

cd "$ROOT_DIR"
echo "[one-click] démarrage backend sur :${API_PORT}"
go run ./cmd/picobot azimutplus-api \
  --addr ":${API_PORT}" \
  --business-id 1 \
  --timezone Europe/Paris \
  --seed >/tmp/azimutplus_oneclick_api.log 2>&1 &
API_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${API_PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

echo "[one-click] backend OK"

echo "[one-click] démarrage frontend sur :${WEB_PORT}"
cd "$ROOT_DIR/web"
npm run dev -- --host 127.0.0.1 --port "$WEB_PORT" >/tmp/azimutplus_oneclick_web.log 2>&1 &
WEB_PID=$!

for _ in $(seq 1 60); do
  if curl -fsS "http://127.0.0.1:${WEB_PORT}" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

URL="http://127.0.0.1:${WEB_PORT}/"
echo "[one-click] ouverture navigateur: ${URL}"
open "$URL" >/dev/null 2>&1 || true

echo "[one-click] prêt pour la démo"
echo "[one-click] API: ${AZIMUTPLUS_DATABASE_URL}"
echo "[one-click] X-API-Key: ${AZIMUTPLUS_API_KEYS%%,*}"
echo "[one-click] X-Admin-Token: ${AZIMUTPLUS_ADMIN_TOKEN}"
echo "[one-click] logs API: /tmp/azimutplus_oneclick_api.log"
echo "[one-click] logs Web: /tmp/azimutplus_oneclick_web.log"
echo "[one-click] Ctrl+C pour tout arrêter"

# Keep session alive
while true; do sleep 1; done
