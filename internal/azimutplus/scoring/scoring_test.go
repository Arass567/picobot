package scoring

import (
	"testing"
)

func TestRankCustomersForUrgentFill(t *testing.T) {
	customers := []Customer{
		{ID: 1, Score: 80, OptInWhatsApp: true},
		{ID: 2, Score: 95, OptInWhatsApp: true},
		{ID: 3, Score: 99, OptInWhatsApp: false},
		{ID: 4, Score: 95, OptInWhatsApp: true},
	}

	ranked := RankCustomersForUrgentFill(customers)
	if len(ranked) != 3 {
		t.Fatalf("expected 3 opted-in customers, got %d", len(ranked))
	}

	if ranked[0].ID != 2 || ranked[1].ID != 4 || ranked[2].ID != 1 {
		t.Fatalf("unexpected ranking order: %+v", ranked)
	}
}
