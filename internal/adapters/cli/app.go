// Package cli is the cobra-based driving adapter.
package cli

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

// App carries the wired services into the command tree. The composition
// root (cmd/ncp) fills the factories so cli never imports other adapters.
type App struct {
	Profiles *services.ProfileService
	Domains  func(ctx context.Context, profile string, sandbox bool) (*services.DomainService, error)
	RunTUI   func(ctx context.Context, profile string, sandbox bool) error
	Version  string
}
