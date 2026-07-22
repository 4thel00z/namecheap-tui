package cli

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func domainsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Manage registered domains",
	}
	cmd.AddCommand(domainsListCmd(app), domainsCheckCmd(app), domainsInfoCmd(app))
	return cmd
}

// domainsService builds the domain service from the persistent flags.
func domainsService(cmd *cobra.Command, app *App) (*services.DomainService, error) {
	if app.Domains == nil {
		return nil, fmt.Errorf("domains service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.Domains(cmd.Context(), profile, sandbox)
}

type domainJSON struct {
	Name      string `json:"name"`
	Expires   string `json:"expires"`
	AutoRenew bool   `json:"auto_renew"`
	Privacy   bool   `json:"privacy"`
	Locked    bool   `json:"locked"`
	Expired   bool   `json:"expired"`
}

func toDomainJSON(d registrar.Domain) domainJSON {
	return domainJSON{
		Name: d.Name.String(), Expires: d.Expires.Format(time.DateOnly),
		AutoRenew: d.AutoRenew, Privacy: d.Privacy, Locked: d.Locked, Expired: d.Expired,
	}
}

func domainsListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all domains in the account",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			noCache, _ := cmd.Flags().GetBool("no-cache")
			domains, err := svc.List(cmd.Context(), noCache)
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				out := make([]domainJSON, len(domains))
				for i, d := range domains {
					out[i] = toDomainJSON(d)
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(domains))
			for i, d := range domains {
				days := strconv.Itoa(int(time.Until(d.Expires).Hours() / 24))
				rows[i] = []string{
					d.Name.String(), d.Expires.Format(time.DateOnly), days,
					boolMark(d.AutoRenew), boolMark(d.Privacy), boolMark(d.Locked),
				}
			}
			renderTable(cmd.OutOrStdout(),
				[]string{"DOMAIN", "EXPIRES", "DAYS", "AUTORENEW", "PRIVACY", "LOCKED"}, rows)
			return nil
		},
	}
}

func boolMark(b bool) string {
	if b {
		return "✓"
	}
	return "-"
}

func domainsCheckCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "check <domain>...",
		Short: "Check domain availability",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			results, err := svc.Check(cmd.Context(), args)
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					Name         string  `json:"name"`
					Available    bool    `json:"available"`
					Premium      bool    `json:"premium"`
					PremiumPrice float64 `json:"premium_price,omitempty"`
				}
				out := make([]row, len(results))
				for i, r := range results {
					out[i] = row{r.Name.String(), r.Available, r.Premium, r.PremiumPrice}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(results))
			for i, r := range results {
				price := "-"
				if r.Premium {
					price = strconv.FormatFloat(r.PremiumPrice, 'f', 2, 64)
				}
				rows[i] = []string{r.Name.String(), boolMark(r.Available), boolMark(r.Premium), price}
			}
			renderTable(cmd.OutOrStdout(),
				[]string{"DOMAIN", "AVAILABLE", "PREMIUM", "PREMIUM PRICE"}, rows)
			return nil
		},
	}
}

func domainsInfoCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "info <domain>",
		Short: "Show full details for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			details, err := svc.Info(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type infoJSON struct {
					domainJSON
					Status      string   `json:"status"`
					DNSProvider string   `json:"dns_provider"`
					Nameservers []string `json:"nameservers"`
				}
				return renderJSON(cmd.OutOrStdout(), infoJSON{
					domainJSON: toDomainJSON(details.Domain),
					Status:     details.Status, DNSProvider: details.DNSProvider,
					Nameservers: details.Nameservers,
				})
			}
			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "Domain:       %s\n", details.Name)
			_, _ = fmt.Fprintf(w, "Status:       %s\n", details.Status)
			_, _ = fmt.Fprintf(w, "Created:      %s\n", details.Created.Format(time.DateOnly))
			_, _ = fmt.Fprintf(w, "Expires:      %s\n", details.Expires.Format(time.DateOnly))
			_, _ = fmt.Fprintf(w, "Privacy:      %s\n", boolMark(details.Privacy))
			_, _ = fmt.Fprintf(w, "DNS provider: %s\n", details.DNSProvider)
			for _, ns := range details.Nameservers {
				_, _ = fmt.Fprintf(w, "Nameserver:   %s\n", ns)
			}
			return nil
		},
	}
}
