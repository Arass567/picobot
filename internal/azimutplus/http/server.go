package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/local/picobot/internal/azimutplus/service"
)

type Service interface {
	CreateCampaign(ctx context.Context, in service.CreateCampaignInput) (service.Campaign, error)
	SendCampaign(ctx context.Context, campaignID int64, maxRecipients int) (int, error)
	HandleInboundWebhook(ctx context.Context, in service.InboundWebhook) (bool, error)
	GetDailyKPI(ctx context.Context, fromDate, toDate string) (service.DailyKPI, error)
}

type Options struct {
	APIKeysCSV             string
	WebhookSecretsCSV      string
	AdminToken             string
	AllowedOriginsCSV      string
	RateLimitPerMinute     int
	RateLimitWindowSeconds int
}

type Server struct {
	svc        Service
	adminToken string
	limiter    *rateLimiter

	secretsMu      sync.RWMutex
	apiKeys        []string
	webhookSecrets []string
	allowedOrigins []string
}

func NewServer(svc Service, opts Options) *Server {
	window := time.Duration(opts.RateLimitWindowSeconds) * time.Second
	return &Server{
		svc:            svc,
		adminToken:     strings.TrimSpace(opts.AdminToken),
		limiter:        newRateLimiter(opts.RateLimitPerMinute, window),
		apiKeys:        splitSecrets(opts.APIKeysCSV),
		webhookSecrets: splitSecrets(opts.WebhookSecretsCSV),
		allowedOrigins: splitSecrets(opts.AllowedOriginsCSV),
	}
}

func (s *Server) Router() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", s.handleHealth)
	mux.HandleFunc("POST /api/v1/campaigns/fill-slot", s.handleCreateCampaign)
	mux.HandleFunc("POST /api/v1/campaigns/", s.handleCampaignActions)
	mux.HandleFunc("GET /api/v1/kpis/daily", s.handleGetDailyKPI)
	mux.HandleFunc("POST /api/v1/webhooks/whatsapp", s.handleWebhook)
	mux.HandleFunc("POST /api/v1/admin/security/rotate", s.handleRotateSecrets)
	mux.HandleFunc("GET /api/v1/admin/security/status", s.handleSecurityStatus)

	var h http.Handler = mux
	h = withJSONContentType(h)
	h = withCORS(s.allowedOrigins, h)
	h = withRateLimit(s.limiter, map[string]bool{"/healthz": true})(h)
	h = withRequestID(h)
	h = withAuditLogs(h)
	return h
}

func (s *Server) handleHealth(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateCampaign(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(r) {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var in service.CreateCampaignInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}

	campaign, err := s.svc.CreateCampaign(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]any{
		"campaign_id": campaign.ID,
		"status":      campaign.Status,
		"campaign":    campaign,
	})
}

func (s *Server) handleCampaignActions(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(r) {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	trimmed := strings.TrimPrefix(r.URL.Path, "/api/v1/campaigns/")
	parts := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(parts) != 2 || parts[1] != "send" {
		writeError(w, http.StatusNotFound, errors.New("not found"))
		return
	}

	campaignID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || campaignID <= 0 {
		writeError(w, http.StatusBadRequest, errors.New("invalid campaign id"))
		return
	}

	var body struct {
		MaxRecipients int `json:"max_recipients"`
	}
	if r.ContentLength > 0 {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
			return
		}
	}

	sentCount, err := s.svc.SendCampaign(r.Context(), campaignID, body.MaxRecipients)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"campaign_id": campaignID,
		"sent_count":  sentCount,
		"status":      service.CampaignStatusSent,
	})
}

func (s *Server) handleGetDailyKPI(w http.ResponseWriter, r *http.Request) {
	if !s.authorize(r) {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	query := r.URL.Query()
	fromDate := query.Get("from")
	toDate := query.Get("to")
	if fromDate == "" || toDate == "" {
		now := time.Now().UTC()
		fromDate = now.Format("2006-01-01")
		toDate = now.Format("2006-01-02")
	}

	kpis, err := s.svc.GetDailyKPI(r.Context(), fromDate, toDate)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, kpis)
}

func (s *Server) handleWebhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("read body: %w", err))
		return
	}
	if !s.verifyWebhookSignature(body, r.Header.Get("X-WhatsApp-Signature")) {
		writeError(w, http.StatusUnauthorized, errors.New("invalid webhook signature"))
		return
	}

	var in service.InboundWebhook
	if err := json.Unmarshal(body, &in); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}

	locked, err := s.svc.HandleInboundWebhook(r.Context(), in)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"campaign_locked": locked,
	})
}

func (s *Server) handleRotateSecrets(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(r) {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	var in struct {
		APIKeys        *string `json:"api_keys"`
		WebhookSecrets *string `json:"webhook_secrets"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid json: %w", err))
		return
	}
	if in.APIKeys == nil && in.WebhookSecrets == nil {
		writeError(w, http.StatusBadRequest, errors.New("at least one of api_keys or webhook_secrets is required"))
		return
	}

	s.secretsMu.Lock()
	if in.APIKeys != nil {
		s.apiKeys = splitSecrets(*in.APIKeys)
	}
	if in.WebhookSecrets != nil {
		s.webhookSecrets = splitSecrets(*in.WebhookSecrets)
	}
	apiCount := len(s.apiKeys)
	secretCount := len(s.webhookSecrets)
	s.secretsMu.Unlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                    true,
		"api_keys_count":        apiCount,
		"webhook_secrets_count": secretCount,
	})
}

func (s *Server) handleSecurityStatus(w http.ResponseWriter, r *http.Request) {
	if !s.authorizeAdmin(r) {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}

	s.secretsMu.RLock()
	apiCount := len(s.apiKeys)
	secretCount := len(s.webhookSecrets)
	s.secretsMu.RUnlock()

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                     true,
		"admin_enabled":          s.adminToken != "",
		"api_keys_count":         apiCount,
		"webhook_secrets_count":  secretCount,
		"rate_limit_per_minute":  s.limiter.limit,
		"rate_limit_window_secs": int(s.limiter.window.Seconds()),
	})
}

func (s *Server) authorize(r *http.Request) bool {
	s.secretsMu.RLock()
	keys := append([]string(nil), s.apiKeys...)
	s.secretsMu.RUnlock()

	if len(keys) == 0 {
		return true
	}
	received := r.Header.Get("X-API-Key")
	for _, allowed := range keys {
		if hmac.Equal([]byte(received), []byte(allowed)) {
			return true
		}
	}
	return false
}

func (s *Server) verifyWebhookSignature(body []byte, headerSignature string) bool {
	s.secretsMu.RLock()
	secrets := append([]string(nil), s.webhookSecrets...)
	s.secretsMu.RUnlock()

	if len(secrets) == 0 {
		return true
	}
	if headerSignature == "" {
		return false
	}

	received := strings.TrimPrefix(headerSignature, "sha256=")
	for _, secret := range secrets {
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write(body)
		expected := fmt.Sprintf("%x", mac.Sum(nil))
		if hmac.Equal([]byte(received), []byte(expected)) {
			return true
		}
	}
	return false
}

func (s *Server) authorizeAdmin(r *http.Request) bool {
	if s.adminToken == "" {
		return false
	}
	return hmac.Equal([]byte(r.Header.Get("X-Admin-Token")), []byte(s.adminToken))
}

func splitSecrets(csv string) []string {
	parts := strings.Split(csv, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func withJSONContentType(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.Header.Get("Content-Type") == "" {
			r.Header.Set("Content-Type", "application/json")
		}
		next.ServeHTTP(w, r)
	})
}

func withCORS(allowedOrigins []string, next http.Handler) http.Handler {
	allowAll := len(allowedOrigins) == 0
	allowedSet := make(map[string]struct{}, len(allowedOrigins))
	for _, origin := range allowedOrigins {
		allowedSet[strings.TrimSpace(origin)] = struct{}{}
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		allowed := allowAll
		if !allowAll {
			_, allowed = allowedSet[origin]
		}

		if allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if allowed {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key, X-WhatsApp-Signature, X-Request-ID, X-Admin-Token")
		if r.Method == http.MethodOptions {
			if !allowAll && !allowed {
				writeError(w, http.StatusForbidden, errors.New("origin not allowed"))
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

var reqCounter atomic.Uint64

func withRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rid := r.Header.Get("X-Request-ID")
		if rid == "" {
			rid = fmt.Sprintf("az-%d-%d", time.Now().UnixNano(), reqCounter.Add(1))
		}
		w.Header().Set("X-Request-ID", rid)
		next.ServeHTTP(w, r)
	})
}

func withAuditLogs(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		latency := time.Since(start)
		rid := rec.Header().Get("X-Request-ID")
		ip := clientIP(r)
		log.Printf("azimutplus_audit request_id=%s method=%s path=%s status=%d latency_ms=%d ip=%s", rid, r.Method, r.URL.Path, rec.status, latency.Milliseconds(), ip)
	})
}

func withRateLimit(l *rateLimiter, skip map[string]bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if skip[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			if !l.allow(clientIP(r)) {
				writeError(w, http.StatusTooManyRequests, errors.New("rate limit exceeded"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

type rateLimiter struct {
	mu      sync.Mutex
	limit   int
	window  time.Duration
	buckets map[string]rateBucket
}

type rateBucket struct {
	start time.Time
	count int
}

func newRateLimiter(limit int, window time.Duration) *rateLimiter {
	if limit <= 0 {
		limit = 180
	}
	if window <= 0 {
		window = time.Minute
	}
	return &rateLimiter{
		limit:   limit,
		window:  window,
		buckets: make(map[string]rateBucket),
	}
}

func (l *rateLimiter) allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()

	b := l.buckets[key]
	if b.start.IsZero() || now.Sub(b.start) >= l.window {
		l.buckets[key] = rateBucket{start: now, count: 1}
		return true
	}
	if b.count >= l.limit {
		return false
	}
	b.count++
	l.buckets[key] = b
	return true
}

func clientIP(r *http.Request) string {
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	if r.RemoteAddr != "" {
		return r.RemoteAddr
	}
	return "unknown"
}
