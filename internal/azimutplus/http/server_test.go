package http

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/local/picobot/internal/azimutplus/service"
)

type stubService struct{}

func (s stubService) CreateCampaign(ctx context.Context, in service.CreateCampaignInput) (service.Campaign, error) {
	return service.Campaign{ID: 1, Status: service.CampaignStatusDraft}, nil
}

func (s stubService) SendCampaign(ctx context.Context, campaignID int64, maxRecipients int) (int, error) {
	return 1, nil
}

func (s stubService) HandleInboundWebhook(ctx context.Context, in service.InboundWebhook) (bool, error) {
	return true, nil
}

func (s stubService) GetDailyKPI(ctx context.Context, fromDate, toDate string) (service.DailyKPI, error) {
	return service.DailyKPI{FromDate: fromDate, ToDate: toDate}, nil
}

func TestAuthorizeHeaderRequired(t *testing.T) {
	srv := NewServer(stubService{}, Options{APIKeysCSV: "old-key,new-key"})
	h := srv.Router()

	body := []byte(`{"slot_time":"2026-03-02T14:00:00Z","offer_label":"Promo","slot_value_cents":1000}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/campaigns/fill-slot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without api key, got %d", rr.Code)
	}

	req = httptest.NewRequest(nethttp.MethodPost, "/api/v1/campaigns/fill-slot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "new-key")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusCreated {
		t.Fatalf("expected 201 with api key, got %d", rr.Code)
	}
}

func TestWebhookSignatureRequired(t *testing.T) {
	srv := NewServer(stubService{}, Options{WebhookSecretsCSV: "webhook-secret,new-secret"})
	h := srv.Router()

	payload := service.InboundWebhook{
		MessageID:     "msg-1",
		CampaignID:    1,
		FromPhoneE164: "+33600000001",
		Text:          "OUI",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/webhooks/whatsapp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 without signature, got %d", rr.Code)
	}

	mac := hmac.New(sha256.New, []byte("new-secret"))
	mac.Write(body)
	sig := fmt.Sprintf("sha256=%x", mac.Sum(nil))

	req = httptest.NewRequest(nethttp.MethodPost, "/api/v1/webhooks/whatsapp", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-WhatsApp-Signature", sig)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusOK {
		t.Fatalf("expected 200 with valid signature, got %d", rr.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := newRateLimiter(2, time.Minute)
	if !limiter.allow("1.2.3.4") {
		t.Fatalf("first request should be allowed")
	}
	if !limiter.allow("1.2.3.4") {
		t.Fatalf("second request should be allowed")
	}
	if limiter.allow("1.2.3.4") {
		t.Fatalf("third request should be blocked")
	}
}

func TestAdminRotateSecrets(t *testing.T) {
	srv := NewServer(stubService{}, Options{APIKeysCSV: "old-key", AdminToken: "admin-secret"})
	h := srv.Router()

	// old key works initially
	body := []byte(`{"slot_time":"2026-03-02T14:00:00Z","offer_label":"Promo","slot_value_cents":1000}`)
	req := httptest.NewRequest(nethttp.MethodPost, "/api/v1/campaigns/fill-slot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "old-key")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusCreated {
		t.Fatalf("expected 201 with old key before rotation, got %d", rr.Code)
	}

	rotateBody := []byte(`{"api_keys":"new-key"}`)
	req = httptest.NewRequest(nethttp.MethodPost, "/api/v1/admin/security/rotate", bytes.NewReader(rotateBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin-secret")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusOK {
		t.Fatalf("expected 200 on rotate, got %d", rr.Code)
	}

	// old key no longer works
	req = httptest.NewRequest(nethttp.MethodPost, "/api/v1/campaigns/fill-slot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "old-key")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusUnauthorized {
		t.Fatalf("expected 401 with old key after rotation, got %d", rr.Code)
	}

	// new key works
	req = httptest.NewRequest(nethttp.MethodPost, "/api/v1/campaigns/fill-slot", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-API-Key", "new-key")
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusCreated {
		t.Fatalf("expected 201 with new key after rotation, got %d", rr.Code)
	}
}

func TestAdminSecurityStatus(t *testing.T) {
	srv := NewServer(stubService{}, Options{
		APIKeysCSV:             "k1,k2",
		WebhookSecretsCSV:      "s1",
		AdminToken:             "admin-secret",
		RateLimitPerMinute:     123,
		RateLimitWindowSeconds: 45,
	})
	h := srv.Router()

	req := httptest.NewRequest(nethttp.MethodGet, "/api/v1/admin/security/status", nil)
	req.Header.Set("X-Admin-Token", "admin-secret")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != nethttp.StatusOK {
		t.Fatalf("expected 200 status endpoint, got %d", rr.Code)
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal status response: %v", err)
	}
	if int(out["api_keys_count"].(float64)) != 2 {
		t.Fatalf("expected api_keys_count=2, got %v", out["api_keys_count"])
	}
	if int(out["webhook_secrets_count"].(float64)) != 1 {
		t.Fatalf("expected webhook_secrets_count=1, got %v", out["webhook_secrets_count"])
	}
	if int(out["rate_limit_per_minute"].(float64)) != 123 {
		t.Fatalf("expected rate_limit_per_minute=123, got %v", out["rate_limit_per_minute"])
	}
	if int(out["rate_limit_window_secs"].(float64)) != 45 {
		t.Fatalf("expected rate_limit_window_secs=45, got %v", out["rate_limit_window_secs"])
	}
}
