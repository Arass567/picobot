package whatsapp

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderMock   = "mock"
	ProviderTwilio = "twilio"
)

func NewSenderFromEnv() (Sender, error) {
	provider := strings.ToLower(strings.TrimSpace(os.Getenv("AZIMUTPLUS_WHATSAPP_PROVIDER")))
	if provider == "" || provider == ProviderMock {
		return MockSender{}, nil
	}

	if provider != ProviderTwilio {
		return nil, fmt.Errorf("unsupported whatsapp provider %q", provider)
	}

	timeout := 10 * time.Second
	if raw := strings.TrimSpace(os.Getenv("AZIMUTPLUS_WHATSAPP_TWILIO_TIMEOUT_SECONDS")); raw != "" {
		secs, err := strconv.Atoi(raw)
		if err != nil || secs <= 0 {
			return nil, fmt.Errorf("invalid AZIMUTPLUS_WHATSAPP_TWILIO_TIMEOUT_SECONDS=%q", raw)
		}
		timeout = time.Duration(secs) * time.Second
	}

	cfg := TwilioConfig{
		AccountSID: strings.TrimSpace(os.Getenv("AZIMUTPLUS_WHATSAPP_TWILIO_ACCOUNT_SID")),
		AuthToken:  strings.TrimSpace(os.Getenv("AZIMUTPLUS_WHATSAPP_TWILIO_AUTH_TOKEN")),
		From:       strings.TrimSpace(os.Getenv("AZIMUTPLUS_WHATSAPP_TWILIO_FROM")),
		BaseURL:    strings.TrimSpace(os.Getenv("AZIMUTPLUS_WHATSAPP_TWILIO_BASE_URL")),
		HTTPClient: &http.Client{Timeout: timeout},
	}

	return NewTwilioSender(cfg)
}
