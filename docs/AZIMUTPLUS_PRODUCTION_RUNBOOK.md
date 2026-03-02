# AzimutPlus Production Runbook

## 1. Objectif
Ce document décrit le déploiement production AzimutPlus intégré à `azimutcode.com` en mode lien caché (prospection), sans exposition sur la navigation principale.

## 2. Architecture cible
- Edge: Cloudflare (DNS/proxy/TLS public).
- Origin VPS: Hetzner Ubuntu 24.04.
- Reverse proxy: Nginx.
- Backend: `picobot azimutplus-api` (Go) dans Docker.
- Base de données: PostgreSQL 16 (Docker).
- Frontend: React/Vite build statique servi par Nginx.

### 2.1 Flux
1. Prospect ouvre l'URL secrète `https://azimutcode.com/<secret_slug>/`.
2. Le frontend appelle `/<secret_slug>/api/api/v1/...`.
3. Nginx proxifie vers `127.0.0.1:18080` et injecte `X-API-Key`.
4. API AzimutPlus lit/écrit PostgreSQL.

## 3. Emplacements serveurs
- Source backend build: `/opt/azimutplus/src`
- Compose: `/opt/azimutplus/docker-compose.yml`
- Variables sensibles: `/opt/azimutplus/.env` (chmod 600)
- Frontend publié: `/var/www/azimutplus/current`
- Nginx site: `/etc/nginx/sites-available/azimutcode.conf`
- Backups DB: `/var/backups/azimutplus`
- Script backup: `/opt/azimutplus/scripts/backup_postgres.sh`

## 4. Variables d'environnement backend
- `AZIMUTPLUS_DATABASE_URL`
- `AZIMUTPLUS_API_KEYS`
- `AZIMUTPLUS_WEBHOOK_SECRETS`
- `AZIMUTPLUS_ADMIN_TOKEN`
- `AZIMUTPLUS_ALLOWED_ORIGINS`
- `AZIMUTPLUS_RATE_LIMIT_PER_MINUTE`
- `AZIMUTPLUS_RATE_LIMIT_WINDOW_SECONDS`

## 5. Déploiement

### 5.1 Backend
```bash
cd /opt/azimutplus
docker compose --env-file /opt/azimutplus/.env up -d --build
```

### 5.2 Frontend (build local puis publication)
```bash
cd web
VITE_BASE_PATH="/<secret_slug>/" VITE_API_BASE_URL="/<secret_slug>/api" npm run build
# copier dist -> /var/www/azimutplus/current
```

### 5.3 Nginx
- Conserver `location /` pour le site principal.
- Ajouter:
  - `location ^~ /<secret_slug>/` (frontend caché)
  - `location ^~ /<secret_slug>/api/` (proxy backend)
  - blocage explicite de `/azimutplus` et `/api/v1/`

Validation:
```bash
nginx -t && systemctl reload nginx
```

## 6. Vérification post-déploiement

### 6.1 Services
```bash
cd /opt/azimutplus
docker compose ps
docker logs --tail=100 azimutplus-api
```

### 6.2 HTTP
- `GET /<secret_slug>/` -> 200
- `GET /azimutplus` -> 404
- `GET /api/v1/kpis/daily` -> 404
- `GET /<secret_slug>/api/healthz` -> 200

### 6.3 Smoke test métier
1. `POST /<secret_slug>/api/api/v1/campaigns/fill-slot`
2. `POST /<secret_slug>/api/api/v1/campaigns/{id}/send`
3. `GET /<secret_slug>/api/api/v1/kpis/daily?...`

## 7. Sécurité appliquée
- URL non discoverable via navigation.
- API backend non exposée publiquement (bind loopback).
- Auth API par clé injectée côté Nginx.
- CORS restreint (`AZIMUTPLUS_ALLOWED_ORIGINS`).
- En-têtes de sécurité côté route secrète:
  - `X-Robots-Tag: noindex, nofollow, noarchive`
  - `X-Frame-Options: DENY`
  - `X-Content-Type-Options: nosniff`
  - `Referrer-Policy: strict-origin-when-cross-origin`
  - `Content-Security-Policy` dédiée
- Rate limit API applicatif.

## 8. Backups
- Cron: `30 3 * * * /opt/azimutplus/scripts/backup_postgres.sh`
- Rétention: 7 jours.
- Emplacement: `/var/backups/azimutplus/*.sql.gz`

Restauration (exemple):
```bash
gunzip -c /var/backups/azimutplus/<dump>.sql.gz | docker exec -i azimutplus-db sh -lc 'psql -U "$POSTGRES_USER" -d "$POSTGRES_DB"'
```

## 9. Rotation des secrets
1. Générer nouvelles valeurs (API/webhook/admin).
2. Mettre à jour `/opt/azimutplus/.env`.
3. Redémarrer API:
```bash
cd /opt/azimutplus
docker compose --env-file /opt/azimutplus/.env up -d --build azimutplus-api
```

## 10. Opérations courantes

### Restart stack
```bash
cd /opt/azimutplus
docker compose restart
```

### Stop/start
```bash
docker compose stop
docker compose start
```

### Mettre à jour code backend
```bash
# synchroniser /opt/azimutplus/src
cd /opt/azimutplus
docker compose --env-file /opt/azimutplus/.env up -d --build azimutplus-api
```

## 11. Troubleshooting
- API 502 via Nginx:
  - vérifier `docker compose ps`
  - vérifier `docker logs azimutplus-api`
  - vérifier `curl http://127.0.0.1:18080/healthz` sur le VPS
- KPI indisponibles:
  - vérifier connectivité DB (`AZIMUTPLUS_DATABASE_URL`)
  - vérifier migration auto (`InitSchema`) au démarrage
- Assets front cassés:
  - vérifier `VITE_BASE_PATH`
  - vérifier `try_files` dans location `/<secret_slug>/`

## 12. Limites actuelles
- Le provider WhatsApp reste en `MockSender` (MVP démonstration).
- Les règles Cloudflare cache/WAF ne sont pas provisionnées automatiquement dans ce repo.
