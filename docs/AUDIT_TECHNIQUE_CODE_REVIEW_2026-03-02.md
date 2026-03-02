# Audit Technique + Code Review

Date: 2026-03-02
Portée: backend Go AzimutPlus, frontend React, intégration Nginx/VPS.

## 1. Findings (ordre de sévérité)

### Haut
1. Provider WhatsApp réel dépend de la configuration runtime (`AZIMUTPLUS_WHATSAPP_PROVIDER` + creds Twilio).
- Impact: si mal configuré, fallback/erreur de démarrage et perte de capacité d'envoi réel.
- Action: sécuriser le provisioning secrets (vault / process d'onboarding client).

### Moyen
1. Route secrète seule (sans PIN/Access).
- Impact: sécurité par obscurité; fuite d'URL possible.
- Mitigation appliquée: non-indexation + absence de liens + slug long + API non exposée.
- Recommandation: option future Cloudflare Access ou PIN léger si taux d'abus.

2. Secret API injecté dans conf Nginx.
- Impact: dépend de la sécurité root VPS.
- Mitigation appliquée: fichier root-only, backend loopback.
- Recommandation: passer à `include` secret séparé non versionné + rotation périodique.

3. Edge Cloudflare non piloté par infra-as-code.
- Impact: drift possible (cache/WAF/règles).
- Recommandation: Terraform/Cloudflare API pour industrialisation.

### Bas
1. Serveur origin Nginx en HTTP local (TLS terminé Cloudflare).
- Acceptable dans ce contexte cloud-proxy.
- Recommandation: Full (strict) + cert origin si durcissement futur.

## 2. Correctifs livrés dans ce cycle
- CORS backend configurable via `AZIMUTPLUS_ALLOWED_ORIGINS`.
- Correction `writeError` (retour JSON unique).
- Ajout flag CLI `--allowed-origins`.
- Frontend:
  - base path compatible URL secrète (`VITE_BASE_PATH`)
  - API base compatible déploiement sous sous-chemin
  - bouton `Retour AzimutCode`
  - meta robots `noindex`
- Infra VPS:
  - stack Docker `azimutplus-api` + `azimutplus-db`
  - publication frontend sous slug secret
  - proxy Nginx `/<secret_slug>/api/` vers backend loopback
  - blocage des chemins publics `/azimutplus` et `/api/v1`
  - headers sécurité + CSP sur route secrète
- backup Postgres quotidien + rétention 7 jours
- provider WhatsApp Twilio implémenté (réel) + mode mock conservé

## 3. Vérifications exécutées
- Tests Go ciblés AzimutPlus: OK
- Build frontend Vite: OK
- `nginx -t`: OK
- `docker compose ps`: API healthy + DB up
- Smoke test public:
  - route secrète: 200
  - route publique `/azimutplus`: 404
  - route publique `/api/v1/*`: 404
  - API via route secrète: create/send/kpi OK

## 4. Risques résiduels
- Pas de monitoring métrique centralisé (Prometheus/Grafana absent).
- Pas d'alerting uptime/API errors.
- Pas de pipeline CI/CD automatisé vers VPS.

## 5. Recommandations prioritaires (Semaine 1)
1. Brancher un fournisseur WhatsApp réel.
2. Ajouter observabilité minimale:
- uptime check externe
- alerte sur erreur backend (tail + webhook)
3. Ajouter workflow de déploiement reproductible (GitHub Actions + SSH).
4. Mettre en place rotation mensuelle des secrets.
