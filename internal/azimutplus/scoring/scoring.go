package scoring

import "sort"

type Customer struct {
	ID            int64
	Score         int
	OptInWhatsApp bool
}

// RankCustomersForUrgentFill returns customers in descending score order.
func RankCustomersForUrgentFill(customers []Customer) []Customer {
	ranked := make([]Customer, 0, len(customers))
	for _, c := range customers {
		if !c.OptInWhatsApp {
			continue
		}
		ranked = append(ranked, c)
	}

	sort.SliceStable(ranked, func(i, j int) bool {
		if ranked[i].Score == ranked[j].Score {
			return ranked[i].ID < ranked[j].ID
		}
		return ranked[i].Score > ranked[j].Score
	})

	return ranked
}
