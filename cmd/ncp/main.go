// Command ncp is a TUI and CLI for the Namecheap API.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/fang"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/adapters/iputil"
	"github.com/4thel00z/namecheap-tui/internal/adapters/namecheap"
	"github.com/4thel00z/namecheap-tui/internal/adapters/tui"
	"github.com/4thel00z/namecheap-tui/internal/adapters/turso"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
	"github.com/4thel00z/namecheap-tui/internal/version"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	verify := func(ctx context.Context, creds account.Credentials) error {
		_, err := namecheap.New(creds).ListDomains(ctx)
		return err
	}
	profiles := services.NewProfileService(store.Profiles(), iputil.New(), verify)

	domainsFactory := func(ctx context.Context, profile string, sandbox bool) (*services.DomainService, error) {
		creds, err := profiles.Current(ctx, profile)
		if err != nil {
			return nil, err
		}
		if sandbox {
			creds.Endpoint = account.EndpointSandbox
		}
		return services.NewDomainService(namecheap.New(creds), store.Cache(), creds.Name), nil
	}

	app := &cli.App{
		Profiles: profiles,
		Domains:  domainsFactory,
		RunTUI: func(ctx context.Context, profile string, sandbox bool) error {
			svc, err := domainsFactory(ctx, profile, sandbox)
			if err != nil {
				return err
			}
			creds, err := profiles.Current(ctx, profile)
			if err != nil {
				return err
			}
			return tui.Run(tui.NewDashboard(svc, creds.Name, version.Version))
		},
		Version: version.Version,
	}
	return fang.Execute(ctx, cli.Root(app), fang.WithVersion(version.Version))
}

// openStore opens the config DB — an embedded Turso replica when
// TURSO_DATABASE_URL is set, a plain local file otherwise.
func openStore(ctx context.Context) (*turso.Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "ncp", "ncp.db")
	if primary := os.Getenv("TURSO_DATABASE_URL"); primary != "" {
		return turso.OpenReplica(ctx, path, primary, os.Getenv("TURSO_AUTH_TOKEN"))
	}
	return turso.Open(ctx, path)
}
