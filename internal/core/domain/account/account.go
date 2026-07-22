package account

import (
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

// Balance is the account's fund state.
type Balance struct {
	Currency     string
	Available    float64
	Total        float64
	Earned       float64
	Withdrawable float64
}

// Price is one pricing entry from users.getPricing.
type Price struct {
	Product      string
	Category     string
	Duration     int
	DurationType string
	Regular      float64
	Yours        float64
	Currency     string
}

// Address is an address-book entry.
type Address struct {
	ID      string
	Name    string
	Default bool
	Contact registrar.Contact
}

// PrivacySubscription is a domain-privacy (whoisguard) subscription.
type PrivacySubscription struct {
	ID      string
	Domain  string
	Created time.Time
	Expires time.Time
	Status  string
}
