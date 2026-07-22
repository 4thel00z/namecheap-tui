// Package cli is the cobra-based driving adapter.
package cli

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

// App carries the wired services into the command tree. The composition
// root (cmd/ncp) fills the factories so cli never imports other adapters.
type App struct {
	Profiles      *services.ProfileService
	Domains       func(ctx context.Context, profile string, sandbox bool) (*services.DomainService, error)
	DNS           func(ctx context.Context, profile string, sandbox bool) (*services.DNSService, error)
	NS            func(ctx context.Context, profile string, sandbox bool) (*services.NSService, error)
	Transfers     func(ctx context.Context, profile string, sandbox bool) (*services.TransferService, error)
	SSL           func(ctx context.Context, profile string, sandbox bool) (*services.SSLService, error)
	Privacy       func(ctx context.Context, profile string, sandbox bool) (*services.PrivacyService, error)
	Account       func(ctx context.Context, profile string, sandbox bool) (*services.AccountService, error)
	RunTUI        func(ctx context.Context, profile string, sandbox bool) error
	RunZoneEditor func(ctx context.Context, profile string, sandbox bool, domain string) error
	Version       string
}
