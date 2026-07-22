// Package ports defines the hexagon's driven interfaces and the error
// contract adapters must honor.
package ports

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

// RegistrarAPI is the domain-registration capability of the registrar.
type RegistrarAPI interface {
	// ListDomains returns all domains in the account (all pages merged).
	ListDomains(ctx context.Context) ([]registrar.Domain, error)
	// CheckDomains checks availability for up to 50 names per call.
	CheckDomains(ctx context.Context, names []registrar.DomainName) ([]registrar.Availability, error)
	// DomainInfo returns full details for one domain.
	DomainInfo(ctx context.Context, name registrar.DomainName) (registrar.Details, error)
}
