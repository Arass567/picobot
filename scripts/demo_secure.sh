#!/usr/bin/env bash
set -euo pipefail

PORT="${PORT:-8083}"
DB_URL="${AZIMUTPLUS_DATABASE_URL:-postgres://admin@127.0.0.1:5432/azimutplus?sslmode=disable}"
ADMIN_TOKEN="${AZIMUTPLUS_ADMIN_TOKEN:-adm-demo}"
API_KEYS="${AZIMUTPLUS_API_KEYS:-key-old}"
WEBHOOK_SECRETS="${AZIMUTPLUS_WEBHOOK_SECRETS:-sec-old}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cleanup() {
  if [[ -n "${API_PID:-}" ]] && kill -0 "$API_PID" 2>/dev/null; then
    kill "$API_PID" >/dev/null 2>&1 || true
    wait "$API_PID" 2>/dev/null || true
  fi
}
trap cleanup EXIT

echo "[demo] starting API on :$PORT"
(
  cd "$ROOT_DIR"
  AZIMUTPLUS_DATABASE_URL="$DB_URL" \
  AZIMUTPLUS_API_KEYS="$API_KEYS" \
  AZIMUTPLUS_WEBHOOK_SECRETS="$WEBHOOK_SECRETS" \
  AZIMUTPLUS_ADMIN_TOKEN="$ADMIN_TOKEN" \
  AZIMUTPLUS_RATE_LIMIT_PER_MINUTE="300" \
  AZIMUTPLUS_RATE_LIMIT_WINDOW_SECONDS="60" \
  go run ./cmd/picobot azimutplus-api --addr ":$PORT" --business-id 1 --timezone Europe/Paris --seed >/tmp/azimutplus_demo_secure.log 2>&1
) &
API_PID=$!

for _ in $(seq 1 40); do
  if curl -fsS "http://127.0.0.1:${PORT}/healthz" >/dev/null 2>&1; then
    break
  fi
  sleep 0.25
done

echo "[demo] create campaign with old key"
CREATE_JSON=$(curl -sS -X POST "http://127.0.0.1:${PORT}/api/v1/campaigns/fill-slot" \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: key-old' \
  -d '{"slot_time":"2026-03-04T14:00:00Z","offer_label":"Promo express","offer_type":"discount","offer_value":"-15%","slot_value_cents":15000}')

echo "$CREATE_JSON"
CAMPAIGN_ID=$(printf '%s' "$CREATE_JSON" | sed -n 's/.*"campaign_id":\([0-9][0-9]*\).*/\1/p')
if [[ -z "$CAMPAIGN_ID" ]]; then
  echo "[demo] failed to parse campaign_id"
  exit 1
fi

echo "[demo] rotate keys/secrets"
curl -sS -X POST "http://127.0.0.1:${PORT}/api/v1/admin/security/rotate" \
  -H 'Content-Type: application/json' \
  -H "X-Admin-Token: ${ADMIN_TOKEN}" \
  -d '{"api_keys":"key-new","webhook_secrets":"sec-new"}'
echo

echo "[demo] admin security status"
curl -sS "http://127.0.0.1:${PORT}/api/v1/admin/security/status" \
  -H "X-Admin-Token: ${ADMIN_TOKEN}"
echo

echo "[demo] old key should fail (401)"
OLD_CODE=$(curl -s -o /dev/null -w '%{http_code}' -X POST "http://127.0.0.1:${PORT}/api/v1/campaigns/${CAMPAIGN_ID}/send" \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: key-old' \
  -d '{"max_recipients":2}')
echo "old_key_http_code=${OLD_CODE}"

echo "[demo] new key send campaign"
curl -sS -X POST "http://127.0.0.1:${PORT}/api/v1/campaigns/${CAMPAIGN_ID}/send" \
  -H 'Content-Type: application/json' \
  -H 'X-API-Key: key-new' \
  -d '{"max_recipients":2}'
echo

echo "[demo] signed webhook with new secret"
BODY=$(printf '{"message_id":"demo-sec-%s","campaign_id":%s,"from_phone_e164":"+33600000001","text":"OUI"}' "$(date +%s)" "$CAMPAIGN_ID")
SIG=$(printf '%s' "$BODY" | openssl dgst -sha256 -hmac 'sec-new' -binary | xxd -p -c 256)
curl -sS -X POST "http://127.0.0.1:${PORT}/api/v1/webhooks/whatsapp" \
  -H 'Content-Type: application/json' \
  -H "X-WhatsApp-Signature: sha256=${SIG}" \
  -d "$BODY"
echo

echo "[demo] KPI snapshot"
curl -sS "http://127.0.0.1:${PORT}/api/v1/kpis/daily?from=2026-03-01&to=2026-03-31" \
  -H 'X-API-Key: key-new'
echo

echo "[demo] completed"
