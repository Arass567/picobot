# AzimutPlus MVP Module

This module adds a revenue-first HTTP API for urgent slot filling.

## Endpoints
- `POST /api/v1/campaigns/fill-slot`
- `POST /api/v1/campaigns/{id}/send`
- `GET /api/v1/kpis/daily?from=YYYY-MM-DD&to=YYYY-MM-DD`
- `POST /api/v1/webhooks/whatsapp`
- `POST /api/v1/admin/security/rotate` (optional, requires admin token)
- `GET /api/v1/admin/security/status` (optional, requires admin token)
- `GET /healthz`

## Run locally
1. Start PostgreSQL.
2. Export DSN:
   - `export AZIMUTPLUS_DATABASE_URL='postgres://admin@127.0.0.1:5432/azimutplus?sslmode=disable'`
   - `export AZIMUTPLUS_API_KEYS='old-key,new-key'` (optional, rotation supported)
   - `export AZIMUTPLUS_WEBHOOK_SECRETS='old-secret,new-secret'` (optional, rotation supported)
   - `export AZIMUTPLUS_ADMIN_TOKEN='change-me-admin'` (optional, enables runtime rotation endpoint)
   - `export AZIMUTPLUS_RATE_LIMIT_PER_MINUTE='180'` (optional)
   - `export AZIMUTPLUS_RATE_LIMIT_WINDOW_SECONDS='60'` (optional)
3. Start API:
   - `go run ./cmd/picobot azimutplus-api --addr :8080 --business-id 1 --timezone Europe/Paris --seed`

Schema bootstrap and demo customer seeding are executed automatically on startup.

## WhatsApp mode (MVP)
Outgoing messages are mocked in logs (`MockSender`). Incoming replies are accepted through `/api/v1/webhooks/whatsapp`.
The first YES-like reply locks the campaign and increments daily recovered revenue KPI.

If `AZIMUTPLUS_API_KEY` or `AZIMUTPLUS_API_KEYS` is set, all non-health endpoints require header:
- `X-API-Key: <value>`

If `AZIMUTPLUS_WEBHOOK_SECRET` or `AZIMUTPLUS_WEBHOOK_SECRETS` is set, webhook requires:
- `X-WhatsApp-Signature: sha256=<hex_hmac_of_raw_body>`

If `AZIMUTPLUS_ADMIN_TOKEN` is set, runtime rotation endpoint is enabled:
- `POST /api/v1/admin/security/rotate`
- Header: `X-Admin-Token: <admin_token>`
- Body (one or both fields): `{"api_keys":"k1,k2","webhook_secrets":"s1,s2"}`
- `GET /api/v1/admin/security/status` returns counts and rate limit settings (no secret values exposed).

## Hardening MVP
- Basic IP rate limit is enabled by default (fixed window, 180 req/min per IP, except `/healthz`).
- Each response includes `X-Request-ID` for troubleshooting.
- API logs include method/path/status/latency/IP (`azimutplus_audit` log line).

## Demo script
- Run full secure demo flow:
  - `make demo-secure`
- Run commercial one-click demo (API + frontend + browser):
  - `make demo-oneclick`
