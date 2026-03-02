package whatsapp

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type TwilioConfig struct {
	AccountSID string
	AuthToken  string
	From       string
	BaseURL    string
	HTTPClient *http.Client
}

type TwilioSender struct {
	accountSID string
	authToken  string
	from       string
	baseURL    string
	client     *http.Client
}

func NewTwilioSender(cfg TwilioConfig) (*TwilioSender, error) {
	if cfg.AccountSID == "" {
		return nil, fmt.Errorf("missing twilio account sid")
	}
	if cfg.AuthToken == "" {
		return nil, fmt.Errorf("missing twilio auth token")
	}
	if cfg.From == "" {
		return nil, fmt.Errorf("missing twilio from number")
	}

	base := strings.TrimSuffix(strings.TrimSpace(cfg.BaseURL), "/")
	if base == "" {
		base = "https://api.twilio.com"
	}

	client := cfg.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	return &TwilioSender{
		accountSID: cfg.AccountSID,
		authToken:  cfg.AuthToken,
		from:       normalizeTwilioWhatsAppAddress(cfg.From),
		baseURL:    base,
		client:     client,
	}, nil
}

func (s *TwilioSender) SendCampaignMessage(ctx context.Context, customerID int64, phone string, campaignID int64, offerLabel string, slotTime time.Time) error {
	if strings.TrimSpace(phone) == "" {
		return fmt.Errorf("customer %d has empty phone", customerID)
	}

	form := url.Values{}
	form.Set("To", normalizeTwilioWhatsAppAddress(phone))
	form.Set("From", s.from)
	form.Set("Body", BuildCampaignMessage(campaignID, offerLabel, slotTime))

	endpoint := fmt.Sprintf("%s/2010-04-01/Accounts/%s/Messages.json", s.baseURL, s.accountSID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("build twilio request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.SetBasicAuth(s.accountSID, s.authToken)

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("twilio request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("twilio rejected message status=%d body=%s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return nil
}

func normalizeTwilioWhatsAppAddress(value string) string {
	v := strings.TrimSpace(value)
	if strings.HasPrefix(strings.ToLower(v), "whatsapp:") {
		return v
	}
	return "whatsapp:" + v
}
