package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/local/picobot/internal/azimutplus/azerrors"
	"github.com/local/picobot/internal/azimutplus/scoring"
	"github.com/local/picobot/internal/azimutplus/whatsapp"
)

type repository interface {
	CreateCampaign(ctx context.Context, in CreateCampaignInput, businessID int64) (Campaign, error)
	GetCampaign(ctx context.Context, campaignID, businessID int64) (Campaign, error)
	ListEligibleCustomers(ctx context.Context, businessID int64, limit int) ([]Customer, error)
	MarkCampaignSentAndRecipients(ctx context.Context, campaignID int64, recipients []Recipient) error
	RecordInboundAndLockIfFirstYes(ctx context.Context, businessID int64, in InboundWebhook, intentYes bool) (bool, error)
	GetDailyKPI(ctx context.Context, businessID int64, fromDate, toDate string) (DailyKPI, error)
}

type sender interface {
	SendCampaignMessage(ctx context.Context, customerID int64, phone string, campaignID int64, offerLabel string, slotTime time.Time) error
}

type Service struct {
	repo       repository
	sender     sender
	businessID int64
	location   *time.Location
}

func New(repo repository, sender sender, businessID int64, location *time.Location) *Service {
	if location == nil {
		location = time.UTC
	}
	return &Service{repo: repo, sender: sender, businessID: businessID, location: location}
}

func (s *Service) CreateCampaign(ctx context.Context, in CreateCampaignInput) (Campaign, error) {
	if in.SlotValueCents <= 0 {
		return Campaign{}, fmt.Errorf("slot_value_cents must be > 0")
	}
	if in.OfferLabel == "" {
		return Campaign{}, fmt.Errorf("offer_label is required")
	}
	if in.OfferType == "" {
		in.OfferType = "discount"
	}
	if in.OfferValue == "" {
		in.OfferValue = "-20%"
	}
	if in.MaxRecipients <= 0 {
		in.MaxRecipients = 25
	}

	return s.repo.CreateCampaign(ctx, in, s.businessID)
}

func (s *Service) SendCampaign(ctx context.Context, campaignID int64, maxRecipients int) (int, error) {
	campaign, err := s.repo.GetCampaign(ctx, campaignID, s.businessID)
	if err != nil {
		return 0, fmt.Errorf("load campaign: %w", err)
	}
	if campaign.Status == CampaignStatusLocked {
		return 0, fmt.Errorf("campaign already locked")
	}

	if maxRecipients <= 0 {
		maxRecipients = 25
	}

	customers, err := s.repo.ListEligibleCustomers(ctx, s.businessID, maxRecipients)
	if err != nil {
		return 0, fmt.Errorf("list customers: %w", err)
	}

	scoringInput := make([]scoring.Customer, 0, len(customers))
	customerByID := make(map[int64]Customer, len(customers))
	for _, c := range customers {
		scoringInput = append(scoringInput, scoring.Customer{
			ID:            c.ID,
			Score:         c.Score,
			OptInWhatsApp: c.OptInWhatsApp,
		})
		customerByID[c.ID] = c
	}

	ranked := scoring.RankCustomersForUrgentFill(scoringInput)
	recipients := make([]Recipient, 0, len(ranked))
	for _, customer := range ranked {
		fullCustomer, ok := customerByID[customer.ID]
		if !ok {
			continue
		}
		if err := s.sender.SendCampaignMessage(
			ctx,
			fullCustomer.ID,
			fullCustomer.PhoneE164,
			campaign.ID,
			campaign.OfferLabel,
			campaign.SlotTime,
		); err != nil {
			return 0, fmt.Errorf("send whatsapp message to customer=%d: %w", customer.ID, err)
		}
		recipients = append(recipients, Recipient{
			CampaignID: campaign.ID,
			CustomerID: fullCustomer.ID,
			RankScore:  fullCustomer.Score,
		})
	}

	if err := s.repo.MarkCampaignSentAndRecipients(ctx, campaignID, recipients); err != nil {
		return 0, fmt.Errorf("mark sent: %w", err)
	}

	return len(recipients), nil
}

func (s *Service) HandleInboundWebhook(ctx context.Context, in InboundWebhook) (bool, error) {
	if in.MessageID == "" {
		return false, fmt.Errorf("message_id is required")
	}
	if in.CampaignID <= 0 {
		return false, fmt.Errorf("campaign_id is required")
	}
	if in.FromPhoneE164 == "" {
		return false, fmt.Errorf("from_phone_e164 is required")
	}
	intentYes := whatsapp.ParseYesIntent(in.Text)
	locked, err := s.repo.RecordInboundAndLockIfFirstYes(ctx, s.businessID, in, intentYes)
	if err != nil {
		if errors.Is(err, azerrors.ErrAlreadyLocked) {
			return false, nil
		}
		return false, fmt.Errorf("record inbound: %w", err)
	}
	return locked, nil
}

func (s *Service) GetDailyKPI(ctx context.Context, fromDate, toDate string) (DailyKPI, error) {
	if fromDate == "" || toDate == "" {
		now := time.Now().In(s.location)
		fromDate = now.Format("2006-01-01")
		toDate = now.Format("2006-01-02")
	}
	return s.repo.GetDailyKPI(ctx, s.businessID, fromDate, toDate)
}
