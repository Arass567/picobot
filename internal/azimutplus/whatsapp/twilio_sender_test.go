package whatsapp

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestNewTwilioSenderValidation(t *testing.T) {
	_, err := NewTwilioSender(TwilioConfig{})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestTwilioSenderSendCampaignMessage(t *testing.T) {
	var gotAuth string
	var gotContentType string
	var gotPath string
	var gotForm url.Values

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotContentType = r.Header.Get("Content-Type")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = r.Body.Close()
		gotForm, _ = url.ParseQuery(string(body))
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM123"}`))
	}))
	defer ts.Close()

	sender, err := NewTwilioSender(TwilioConfig{
		AccountSID: "AC123",
		AuthToken:  "token123",
		From:       "+14155238886",
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}

	err = sender.SendCampaignMessage(context.Background(), 1, "+33600000001", 42, "Promo express", time.Date(2026, 3, 2, 14, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("send message: %v", err)
	}

	if gotPath != "/2010-04-01/Accounts/AC123/Messages.json" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotContentType != "application/x-www-form-urlencoded" {
		t.Fatalf("unexpected content-type: %s", gotContentType)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") {
		t.Fatalf("expected basic auth header, got %q", gotAuth)
	}
	if gotForm.Get("To") != "whatsapp:+33600000001" {
		t.Fatalf("unexpected To=%q", gotForm.Get("To"))
	}
	if gotForm.Get("From") != "whatsapp:+14155238886" {
		t.Fatalf("unexpected From=%q", gotForm.Get("From"))
	}
	if !strings.Contains(gotForm.Get("Body"), "Promo express") {
		t.Fatalf("body does not contain offer label: %q", gotForm.Get("Body"))
	}
}

func TestTwilioSenderErrorOnNon2xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"message":"invalid"}`))
	}))
	defer ts.Close()

	sender, err := NewTwilioSender(TwilioConfig{
		AccountSID: "AC123",
		AuthToken:  "token123",
		From:       "whatsapp:+14155238886",
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	})
	if err != nil {
		t.Fatalf("new sender: %v", err)
	}

	err = sender.SendCampaignMessage(context.Background(), 1, "+33600000001", 7, "Promo", time.Now())
	if err == nil {
		t.Fatalf("expected twilio non-2xx error")
	}
}
