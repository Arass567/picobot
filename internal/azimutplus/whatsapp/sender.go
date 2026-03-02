package whatsapp

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"
)

type Sender interface {
	SendCampaignMessage(ctx context.Context, customerID int64, phone string, campaignID int64, offerLabel string, slotTime time.Time) error
}

type MockSender struct{}

func (m MockSender) SendCampaignMessage(_ context.Context, customerID int64, phone string, campaignID int64, offerLabel string, slotTime time.Time) error {
	log.Printf("azimutplus mock whatsapp send campaign_id=%d customer_id=%d phone=%s offer=%s slot=%s", campaignID, customerID, phone, offerLabel, slotTime.Format("2006-01-02 15:04"))
	if phone == "" {
		return fmt.Errorf("customer %d has empty phone", customerID)
	}
	return nil
}

func BuildCampaignMessage(campaignID int64, offerLabel string, slotTime time.Time) string {
	slot := slotTime.Format("02/01 15:04")
	return strings.TrimSpace(fmt.Sprintf(
		"AzimutPlus: un créneau vient de se libérer (%s). Offre: %s. Répondez OUI pour réserver. Réf campagne #%d.",
		slot,
		offerLabel,
		campaignID,
	))
}
