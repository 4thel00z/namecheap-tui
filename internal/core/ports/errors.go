package ports

import (
	"errors"
	"fmt"
)

var (
	// ErrProfileNotFound is returned when a named profile does not exist.
	ErrProfileNotFound = errors.New("profile not found")
	// ErrNoDefaultProfile is returned when no profile is marked default.
	ErrNoDefaultProfile = errors.New("no default profile configured")
	// ErrIPNotWhitelisted maps Namecheap error 1011147.
	ErrIPNotWhitelisted = errors.New("client IP is not whitelisted for API access")
	// ErrAuth maps Namecheap credential errors (e.g. 1011102, 1010104).
	ErrAuth = errors.New("authentication failed")
	// ErrRateLimited maps Namecheap request-throttling errors.
	ErrRateLimited = errors.New("rate limited by namecheap API")
)

// APIError is a Namecheap API error that has no dedicated sentinel.
type APIError struct {
	Number  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("namecheap api error %d: %s", e.Number, e.Message)
}
