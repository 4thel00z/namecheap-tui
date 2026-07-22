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

	client := func(ctx context.Context, profile string, sandbox bool) (*namecheap.Client, string, error) {
		creds, err := profiles.Current(ctx, profile)
		if err != nil {
			return nil, "", err
		}
		if sandbox {
			creds.Endpoint = account.EndpointSandbox
		}
		return namecheap.New(creds), creds.Name, nil
	}
	domainsFactory := func(ctx context.Context, profile string, sandbox bool) (*services.DomainService, error) {
		c, name, err := client(ctx, profile, sandbox)
		if err != nil {
			return nil, err
		}
		return services.NewDomainService(c, store.Cache(), name), nil
	}
	dnsFactory := func(ctx context.Context, profile string, sandbox bool) (*services.DNSService, error) {
		c, name, err := client(ctx, profile, sandbox)
		if err != nil {
			return nil, err
		}
		return services.NewDNSService(c, store.Cache(), name), nil
	}

	app := &cli.App{
		Profiles: profiles,
		Domains:  domainsFactory,
		DNS:      dnsFactory,
		NS: func(ctx context.Context, profile string, sandbox bool) (*services.NSService, error) {
			c, _, err := client(ctx, profile, sandbox)
			if err != nil {
				return nil, err
			}
			return services.NewNSService(c), nil
		},
		Transfers: func(ctx context.Context, profile string, sandbox bool) (*services.TransferService, error) {
			c, _, err := client(ctx, profile, sandbox)
			if err != nil {
				return nil, err
			}
			return services.NewTransferService(c), nil
		},
		SSL: func(ctx context.Context, profile string, sandbox bool) (*services.SSLService, error) {
			c, _, err := client(ctx, profile, sandbox)
			if err != nil {
				return nil, err
			}
			return services.NewSSLService(c), nil
		},
		Privacy: func(ctx context.Context, profile string, sandbox bool) (*services.PrivacyService, error) {
			c, _, err := client(ctx, profile, sandbox)
			if err != nil {
				return nil, err
			}
			return services.NewPrivacyService(c), nil
		},
		Account: func(ctx context.Context, profile string, sandbox bool) (*services.AccountService, error) {
			c, name, err := client(ctx, profile, sandbox)
			if err != nil {
				return nil, err
			}
			return services.NewAccountService(c, store.Cache(), name), nil
		},
		RunTUI: func(ctx context.Context, profile string, sandbox bool) error {
			c, name, err := client(ctx, profile, sandbox)
			if err != nil {
				return err
			}
			deps := tui.DashboardDeps{
				Domains:   services.NewDomainService(c, store.Cache(), name),
				SSL:       services.NewSSLService(c),
				Transfers: services.NewTransferService(c),
				Account:   services.NewAccountService(c, store.Cache(), name),
			}
			return tui.Run(tui.NewDashboard(deps, name, version.Version))
		},
		RunZoneEditor: func(ctx context.Context, profile string, sandbox bool, domain string) error {
			svc, err := dnsFactory(ctx, profile, sandbox)
			if err != nil {
				return err
			}
			return tui.Run(tui.NewZoneEditor(svc, domain))
		},
		Settings: store.Settings(),
		Sync:     store.Sync,
		Version:  version.Version,
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
