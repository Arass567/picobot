package service

import "time"

const (
	CampaignStatusDraft  = "draft"
	CampaignStatusSent   = "sent"
	CampaignStatusLocked = "locked"
)

type CreateCampaignInput struct {
	SlotTime       time.Time `json:"slot_time"`
	OfferLabel     string    `json:"offer_label"`
	OfferType      string    `json:"offer_type"`
	OfferValue     string    `json:"offer_value"`
	SlotValueCents int64     `json:"slot_value_cents"`
	MaxRecipients  int       `json:"max_recipients"`
}

type Campaign struct {
	ID               int64      `json:"id"`
	BusinessID       int64      `json:"business_id"`
	SlotTime         time.Time  `json:"slot_time"`
	OfferLabel       string     `json:"offer_label"`
	OfferType        string     `json:"offer_type"`
	OfferValue       string     `json:"offer_value"`
	SlotValueCents   int64      `json:"slot_value_cents"`
	Status           string     `json:"status"`
	WinnerCustomerID *int64     `json:"winner_customer_id,omitempty"`
	LockedAt         *time.Time `json:"locked_at,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

type Customer struct {
	ID            int64
	BusinessID    int64
	Name          string
	PhoneE164     string
	Score         int
	OptInWhatsApp bool
}

type Recipient struct {
	CampaignID int64
	CustomerID int64
	RankScore  int
}

type InboundWebhook struct {
	MessageID     string `json:"message_id"`
	CampaignID    int64  `json:"campaign_id"`
	FromPhoneE164 string `json:"from_phone_e164"`
	Text          string `json:"text"`
}

type DailyKPI struct {
	FromDate              string `json:"from_date"`
	ToDate                string `json:"to_date"`
	RecoveredRevenueCents int64  `json:"recovered_revenue_cents"`
	FilledSlotsCount      int64  `json:"filled_slots_count"`
	SentMessagesCount     int64  `json:"sent_messages_count"`
	ReplyYesCount         int64  `json:"reply_yes_count"`
}
