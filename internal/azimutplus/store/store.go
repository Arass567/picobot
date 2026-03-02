package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/local/picobot/internal/azimutplus/azerrors"
	"github.com/local/picobot/internal/azimutplus/service"
)

type Store struct {
	db *sql.DB
}

func New(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) InitSchema(ctx context.Context) error {
	schema := `
CREATE TABLE IF NOT EXISTS businesses (
	id BIGINT PRIMARY KEY,
	name TEXT NOT NULL,
	theme_prefs JSONB,
	whatsapp_number TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS customers (
	id BIGSERIAL PRIMARY KEY,
	business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
	name TEXT NOT NULL,
	phone_e164 TEXT NOT NULL,
	last_visit TIMESTAMPTZ,
	score INT NOT NULL DEFAULT 0,
	opt_in_whatsapp BOOLEAN NOT NULL DEFAULT TRUE,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (business_id, phone_e164)
);

CREATE TABLE IF NOT EXISTS campaigns (
	id BIGSERIAL PRIMARY KEY,
	business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
	slot_time TIMESTAMPTZ NOT NULL,
	offer_label TEXT NOT NULL,
	offer_type TEXT NOT NULL,
	offer_value TEXT NOT NULL,
	slot_value_cents BIGINT NOT NULL,
	status TEXT NOT NULL,
	winner_customer_id BIGINT,
	locked_at TIMESTAMPTZ,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS campaign_recipients (
	id BIGSERIAL PRIMARY KEY,
	campaign_id BIGINT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
	rank_score INT NOT NULL,
	sent_at TIMESTAMPTZ,
	delivery_status TEXT NOT NULL DEFAULT 'queued',
	reply_text TEXT,
	reply_intent TEXT,
	replied_at TIMESTAMPTZ,
	UNIQUE (campaign_id, customer_id)
);

CREATE TABLE IF NOT EXISTS whatsapp_inbound_events (
	id BIGSERIAL PRIMARY KEY,
	message_id TEXT NOT NULL UNIQUE,
	campaign_id BIGINT NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
	customer_id BIGINT NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
	body TEXT NOT NULL,
	intent TEXT NOT NULL,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS kpi_metrics (
	id BIGSERIAL PRIMARY KEY,
	business_id BIGINT NOT NULL REFERENCES businesses(id) ON DELETE CASCADE,
	day DATE NOT NULL,
	recovered_revenue_cents BIGINT NOT NULL DEFAULT 0,
	filled_slots_count BIGINT NOT NULL DEFAULT 0,
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (business_id, day)
);
`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return fmt.Errorf("init schema: %w", err)
	}
	return nil
}

func (s *Store) EnsureBusiness(ctx context.Context, businessID int64, name string) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO businesses (id, name)
VALUES ($1, $2)
ON CONFLICT (id) DO NOTHING
`, businessID, name)
	if err != nil {
		return fmt.Errorf("ensure business: %w", err)
	}
	return nil
}

func (s *Store) CreateCampaign(ctx context.Context, in service.CreateCampaignInput, businessID int64) (service.Campaign, error) {
	var out service.Campaign
	err := s.db.QueryRowContext(ctx, `
INSERT INTO campaigns (business_id, slot_time, offer_label, offer_type, offer_value, slot_value_cents, status)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id, business_id, slot_time, offer_label, offer_type, offer_value, slot_value_cents, status, created_at
`, businessID, in.SlotTime, in.OfferLabel, in.OfferType, in.OfferValue, in.SlotValueCents, service.CampaignStatusDraft).Scan(
		&out.ID,
		&out.BusinessID,
		&out.SlotTime,
		&out.OfferLabel,
		&out.OfferType,
		&out.OfferValue,
		&out.SlotValueCents,
		&out.Status,
		&out.CreatedAt,
	)
	if err != nil {
		return service.Campaign{}, fmt.Errorf("create campaign: %w", err)
	}
	return out, nil
}

func (s *Store) GetCampaign(ctx context.Context, campaignID, businessID int64) (service.Campaign, error) {
	var out service.Campaign
	var winner sql.NullInt64
	var lockedAt sql.NullTime
	err := s.db.QueryRowContext(ctx, `
SELECT id, business_id, slot_time, offer_label, offer_type, offer_value, slot_value_cents, status, winner_customer_id, locked_at, created_at
FROM campaigns
WHERE id = $1 AND business_id = $2
`, campaignID, businessID).Scan(
		&out.ID,
		&out.BusinessID,
		&out.SlotTime,
		&out.OfferLabel,
		&out.OfferType,
		&out.OfferValue,
		&out.SlotValueCents,
		&out.Status,
		&winner,
		&lockedAt,
		&out.CreatedAt,
	)
	if err != nil {
		return service.Campaign{}, fmt.Errorf("get campaign: %w", err)
	}
	if winner.Valid {
		out.WinnerCustomerID = &winner.Int64
	}
	if lockedAt.Valid {
		out.LockedAt = &lockedAt.Time
	}
	return out, nil
}

func (s *Store) ListEligibleCustomers(ctx context.Context, businessID int64, limit int) ([]service.Customer, error) {
	if limit <= 0 {
		limit = 50
	}

	rows, err := s.db.QueryContext(ctx, `
SELECT id, business_id, name, phone_e164, score, opt_in_whatsapp
FROM customers
WHERE business_id = $1 AND opt_in_whatsapp = TRUE
ORDER BY score DESC, id ASC
LIMIT $2
`, businessID, limit)
	if err != nil {
		return nil, fmt.Errorf("list customers: %w", err)
	}
	defer rows.Close()

	customers := make([]service.Customer, 0, limit)
	for rows.Next() {
		var c service.Customer
		if err := rows.Scan(&c.ID, &c.BusinessID, &c.Name, &c.PhoneE164, &c.Score, &c.OptInWhatsApp); err != nil {
			return nil, fmt.Errorf("scan customer: %w", err)
		}
		customers = append(customers, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate customers: %w", err)
	}
	return customers, nil
}

func (s *Store) MarkCampaignSentAndRecipients(ctx context.Context, campaignID int64, recipients []service.Recipient) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx send campaign: %w", err)
	}
	defer tx.Rollback()

	for _, r := range recipients {
		_, err := tx.ExecContext(ctx, `
INSERT INTO campaign_recipients (campaign_id, customer_id, rank_score, sent_at, delivery_status)
VALUES ($1, $2, $3, NOW(), 'sent')
ON CONFLICT (campaign_id, customer_id)
DO UPDATE SET rank_score = EXCLUDED.rank_score, sent_at = NOW(), delivery_status = 'sent'
`, r.CampaignID, r.CustomerID, r.RankScore)
		if err != nil {
			return fmt.Errorf("insert recipient: %w", err)
		}
	}

	_, err = tx.ExecContext(ctx, `
UPDATE campaigns
SET status = $2
WHERE id = $1 AND status IN ($3, $2)
`, campaignID, service.CampaignStatusSent, service.CampaignStatusDraft)
	if err != nil {
		return fmt.Errorf("mark campaign sent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit send campaign: %w", err)
	}
	return nil
}

func (s *Store) FindCustomerByPhone(ctx context.Context, businessID int64, phone string) (service.Customer, error) {
	var c service.Customer
	err := s.db.QueryRowContext(ctx, `
SELECT id, business_id, name, phone_e164, score, opt_in_whatsapp
FROM customers
WHERE business_id = $1 AND phone_e164 = $2
`, businessID, phone).Scan(&c.ID, &c.BusinessID, &c.Name, &c.PhoneE164, &c.Score, &c.OptInWhatsApp)
	if err != nil {
		return service.Customer{}, fmt.Errorf("find customer by phone: %w", err)
	}
	return c, nil
}

func (s *Store) RecordInboundAndLockIfFirstYes(ctx context.Context, businessID int64, in service.InboundWebhook, intentYes bool) (bool, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return false, fmt.Errorf("begin inbound tx: %w", err)
	}
	defer tx.Rollback()

	var customerID int64
	err = tx.QueryRowContext(ctx, `
SELECT id FROM customers WHERE business_id = $1 AND phone_e164 = $2
`, businessID, in.FromPhoneE164).Scan(&customerID)
	if err != nil {
		return false, fmt.Errorf("lookup inbound customer: %w", err)
	}

	intent := "other"
	if intentYes {
		intent = "yes"
	}

	_, err = tx.ExecContext(ctx, `
INSERT INTO whatsapp_inbound_events (message_id, campaign_id, customer_id, body, intent)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (message_id) DO NOTHING
`, in.MessageID, in.CampaignID, customerID, in.Text, intent)
	if err != nil {
		return false, fmt.Errorf("insert inbound event: %w", err)
	}

	_, err = tx.ExecContext(ctx, `
UPDATE campaign_recipients
SET reply_text = $3, reply_intent = $4, replied_at = NOW()
WHERE campaign_id = $1 AND customer_id = $2
`, in.CampaignID, customerID, in.Text, intent)
	if err != nil {
		return false, fmt.Errorf("update recipient reply: %w", err)
	}

	if !intentYes {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit non-yes inbound: %w", err)
		}
		return false, nil
	}

	var slotValue int64
	var status string
	err = tx.QueryRowContext(ctx, `
SELECT slot_value_cents, status
FROM campaigns
WHERE id = $1 AND business_id = $2
FOR UPDATE
`, in.CampaignID, businessID).Scan(&slotValue, &status)
	if err != nil {
		return false, fmt.Errorf("select campaign for lock: %w", err)
	}

	if status == service.CampaignStatusLocked {
		if err := tx.Commit(); err != nil {
			return false, fmt.Errorf("commit already locked: %w", err)
		}
		return false, azerrors.ErrAlreadyLocked
	}

	var lockedAt time.Time
	err = tx.QueryRowContext(ctx, `
UPDATE campaigns
SET status = $3, winner_customer_id = $2, locked_at = NOW()
WHERE id = $1 AND status <> $3
RETURNING locked_at
`, in.CampaignID, customerID, service.CampaignStatusLocked).Scan(&lockedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if err := tx.Commit(); err != nil {
				return false, fmt.Errorf("commit no lock rows: %w", err)
			}
			return false, azerrors.ErrAlreadyLocked
		}
		return false, fmt.Errorf("lock campaign: %w", err)
	}

	day := lockedAt.Format("2006-01-02")
	_, err = tx.ExecContext(ctx, `
INSERT INTO kpi_metrics (business_id, day, recovered_revenue_cents, filled_slots_count, updated_at)
VALUES ($1, $2, $3, 1, NOW())
ON CONFLICT (business_id, day)
DO UPDATE
SET recovered_revenue_cents = kpi_metrics.recovered_revenue_cents + EXCLUDED.recovered_revenue_cents,
	filled_slots_count = kpi_metrics.filled_slots_count + 1,
	updated_at = NOW()
`, businessID, day, slotValue)
	if err != nil {
		return false, fmt.Errorf("upsert kpi: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("commit inbound lock: %w", err)
	}

	return true, nil
}

func (s *Store) GetDailyKPI(ctx context.Context, businessID int64, fromDate, toDate string) (service.DailyKPI, error) {
	out := service.DailyKPI{
		FromDate: fromDate,
		ToDate:   toDate,
	}

	err := s.db.QueryRowContext(ctx, `
SELECT COALESCE(SUM(recovered_revenue_cents), 0), COALESCE(SUM(filled_slots_count), 0)
FROM kpi_metrics
WHERE business_id = $1 AND day >= $2::date AND day <= $3::date
`, businessID, fromDate, toDate).Scan(&out.RecoveredRevenueCents, &out.FilledSlotsCount)
	if err != nil {
		return service.DailyKPI{}, fmt.Errorf("query kpi metrics: %w", err)
	}

	err = s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM campaign_recipients cr
JOIN campaigns c ON c.id = cr.campaign_id
WHERE c.business_id = $1 AND cr.sent_at::date >= $2::date AND cr.sent_at::date <= $3::date
`, businessID, fromDate, toDate).Scan(&out.SentMessagesCount)
	if err != nil {
		return service.DailyKPI{}, fmt.Errorf("query sent messages: %w", err)
	}

	err = s.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM whatsapp_inbound_events w
JOIN campaigns c ON c.id = w.campaign_id
WHERE c.business_id = $1 AND w.intent = 'yes' AND w.created_at::date >= $2::date AND w.created_at::date <= $3::date
`, businessID, fromDate, toDate).Scan(&out.ReplyYesCount)
	if err != nil {
		return service.DailyKPI{}, fmt.Errorf("query yes replies: %w", err)
	}

	return out, nil
}

func (s *Store) SeedCustomers(ctx context.Context, businessID int64) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO customers (business_id, name, phone_e164, score, opt_in_whatsapp)
VALUES
	($1, 'Nadia M.', '+33600000001', 95, TRUE),
	($1, 'Sonia K.', '+33600000002', 90, TRUE),
	($1, 'Leila A.', '+33600000003', 85, TRUE),
	($1, 'Yann B.', '+33600000004', 72, TRUE),
	($1, 'Client Opt-out', '+33600000005', 99, FALSE)
ON CONFLICT (business_id, phone_e164) DO NOTHING
`, businessID)
	if err != nil {
		return fmt.Errorf("seed customers: %w", err)
	}
	return nil
}
