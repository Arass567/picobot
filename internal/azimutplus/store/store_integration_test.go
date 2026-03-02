package store

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/local/picobot/internal/azimutplus/azerrors"
	"github.com/local/picobot/internal/azimutplus/service"
)

func TestRecordInboundAndLockIfFirstYes_Integration(t *testing.T) {
	dsn := os.Getenv("AZIMUTPLUS_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("AZIMUTPLUS_TEST_DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	s := New(db)

	if err := s.InitSchema(ctx); err != nil {
		t.Fatalf("init schema: %v", err)
	}

	_, err = db.ExecContext(ctx, `
TRUNCATE whatsapp_inbound_events, campaign_recipients, campaigns, customers, kpi_metrics, businesses RESTART IDENTITY CASCADE;
`)
	if err != nil {
		t.Fatalf("truncate: %v", err)
	}

	if err := s.EnsureBusiness(ctx, 1, "Test Biz"); err != nil {
		t.Fatalf("ensure business: %v", err)
	}
	if err := s.SeedCustomers(ctx, 1); err != nil {
		t.Fatalf("seed customers: %v", err)
	}

	camp, err := s.CreateCampaign(ctx, service.CreateCampaignInput{
		SlotTime:       time.Now().Add(2 * time.Hour).UTC(),
		OfferLabel:     "Promo",
		OfferType:      "discount",
		OfferValue:     "-20%",
		SlotValueCents: 12000,
	}, 1)
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	customers, err := s.ListEligibleCustomers(ctx, 1, 2)
	if err != nil {
		t.Fatalf("list customers: %v", err)
	}
	if len(customers) < 2 {
		t.Fatalf("expected at least 2 customers, got %d", len(customers))
	}

	recipients := []service.Recipient{
		{CampaignID: camp.ID, CustomerID: customers[0].ID, RankScore: customers[0].Score},
		{CampaignID: camp.ID, CustomerID: customers[1].ID, RankScore: customers[1].Score},
	}
	if err := s.MarkCampaignSentAndRecipients(ctx, camp.ID, recipients); err != nil {
		t.Fatalf("mark sent recipients: %v", err)
	}

	locked, err := s.RecordInboundAndLockIfFirstYes(ctx, 1, service.InboundWebhook{
		MessageID:     "msg-1",
		CampaignID:    camp.ID,
		FromPhoneE164: customers[0].PhoneE164,
		Text:          "OUI",
	}, true)
	if err != nil {
		t.Fatalf("first yes lock: %v", err)
	}
	if !locked {
		t.Fatalf("expected first yes to lock campaign")
	}

	locked, err = s.RecordInboundAndLockIfFirstYes(ctx, 1, service.InboundWebhook{
		MessageID:     "msg-2",
		CampaignID:    camp.ID,
		FromPhoneE164: customers[1].PhoneE164,
		Text:          "OUI",
	}, true)
	if !errors.Is(err, azerrors.ErrAlreadyLocked) {
		t.Fatalf("expected ErrAlreadyLocked on second yes, got err=%v", err)
	}
	if locked {
		t.Fatalf("expected second yes to not lock")
	}

	now := time.Now().UTC().Format("2006-01-02")
	kpi, err := s.GetDailyKPI(ctx, 1, now, now)
	if err != nil {
		t.Fatalf("get daily kpi: %v", err)
	}
	if kpi.RecoveredRevenueCents != 12000 {
		t.Fatalf("expected recovered revenue 12000, got %d", kpi.RecoveredRevenueCents)
	}
	if kpi.FilledSlotsCount != 1 {
		t.Fatalf("expected filled slots 1, got %d", kpi.FilledSlotsCount)
	}
}
