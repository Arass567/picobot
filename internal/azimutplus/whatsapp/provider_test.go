package whatsapp

import (
	"os"
	"testing"
)

func TestNewSenderFromEnvMockDefault(t *testing.T) {
	t.Setenv("AZIMUTPLUS_WHATSAPP_PROVIDER", "")
	s, err := NewSenderFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := s.(MockSender); !ok {
		t.Fatalf("expected MockSender by default")
	}
}

func TestNewSenderFromEnvTwilioMissingCreds(t *testing.T) {
	t.Setenv("AZIMUTPLUS_WHATSAPP_PROVIDER", "twilio")
	_ = os.Unsetenv("AZIMUTPLUS_WHATSAPP_TWILIO_ACCOUNT_SID")
	_ = os.Unsetenv("AZIMUTPLUS_WHATSAPP_TWILIO_AUTH_TOKEN")
	_ = os.Unsetenv("AZIMUTPLUS_WHATSAPP_TWILIO_FROM")

	_, err := NewSenderFromEnv()
	if err == nil {
		t.Fatalf("expected error on missing twilio credentials")
	}
}
