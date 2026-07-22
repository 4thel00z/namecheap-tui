package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func accountCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "account",
		Short: "Account balances and pricing",
	}
	cmd.AddCommand(accountBalanceCmd(app), accountPricingCmd(app))
	return cmd
}

func accountService(cmd *cobra.Command, app *App) (*services.AccountService, error) {
	if app.Account == nil {
		return nil, fmt.Errorf("account service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.Account(cmd.Context(), profile, sandbox)
}

func accountBalanceCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "balance",
		Short: "Show account funds",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := accountService(cmd, app)
			if err != nil {
				return err
			}
			noCache, _ := cmd.Flags().GetBool("no-cache")
			b, err := svc.Balance(cmd.Context(), noCache)
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return renderJSON(cmd.OutOrStdout(), map[string]any{
					"currency": b.Currency, "available": b.Available,
					"total": b.Total, "earned": b.Earned, "withdrawable": b.Withdrawable,
				})
			}
			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "Available:    %.2f %s\n", b.Available, b.Currency)
			_, _ = fmt.Fprintf(w, "Total:        %.2f %s\n", b.Total, b.Currency)
			_, _ = fmt.Fprintf(w, "Earned:       %.2f %s\n", b.Earned, b.Currency)
			_, _ = fmt.Fprintf(w, "Withdrawable: %.2f %s\n", b.Withdrawable, b.Currency)
			return nil
		},
	}
}

func accountPricingCmd(app *App) *cobra.Command {
	var category, product string
	cmd := &cobra.Command{
		Use:   "pricing <product-type>",
		Short: "Show pricing (product-type: DOMAIN, SSLCERTIFICATE, WHOISGUARD)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := accountService(cmd, app)
			if err != nil {
				return err
			}
			prices, err := svc.Pricing(cmd.Context(), args[0], category, product)
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					Product  string  `json:"product"`
					Category string  `json:"category"`
					Duration string  `json:"duration"`
					Regular  float64 `json:"regular"`
					Yours    float64 `json:"yours"`
					Currency string  `json:"currency"`
				}
				out := make([]row, len(prices))
				for i, p := range prices {
					out[i] = row{
						p.Product, p.Category,
						strconv.Itoa(p.Duration) + " " + p.DurationType,
						p.Regular, p.Yours, p.Currency,
					}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(prices))
			for i, p := range prices {
				rows[i] = []string{
					p.Product, p.Category, strconv.Itoa(p.Duration) + " " + p.DurationType,
					strconv.FormatFloat(p.Regular, 'f', 2, 64),
					strconv.FormatFloat(p.Yours, 'f', 2, 64), p.Currency,
				}
			}
			renderTable(cmd.OutOrStdout(),
				[]string{"PRODUCT", "CATEGORY", "DURATION", "REGULAR", "YOURS", "CURRENCY"}, rows)
			return nil
		},
	}
	cmd.Flags().StringVar(&category, "category", "", "product category (REGISTER, RENEW, ...)")
	cmd.Flags().StringVar(&product, "product", "", "product name (com, positivessl, ...)")
	return cmd
}

func addressCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "address",
		Short: "Manage the address book",
	}
	cmd.AddCommand(
		addressListCmd(app), addressInfoCmd(app), addressAddCmd(app, "add"),
		addressAddCmd(app, "edit"), addressRmCmd(app), addressDefaultCmd(app),
	)
	return cmd
}

func addressListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List address-book entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := accountService(cmd, app)
			if err != nil {
				return err
			}
			addrs, err := svc.Addresses(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					ID      string `json:"id"`
					Name    string `json:"name"`
					Default bool   `json:"default"`
				}
				out := make([]row, len(addrs))
				for i, a := range addrs {
					out[i] = row{a.ID, a.Name, a.Default}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(addrs))
			for i, a := range addrs {
				rows[i] = []string{a.ID, a.Name, boolMark(a.Default)}
			}
			renderTable(cmd.OutOrStdout(), []string{"ID", "NAME", "DEFAULT"}, rows)
			return nil
		},
	}
}

func addressInfoCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "info <address-id>",
		Short: "Show one address-book entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := accountService(cmd, app)
			if err != nil {
				return err
			}
			addr, err := svc.Address(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			out := map[string]any{
				"id": addr.ID, "name": addr.Name, "contact": toContactYAML(addr.Contact),
			}
			return renderJSON(cmd.OutOrStdout(), out)
		},
	}
}

// addressAddCmd builds both `add` and `edit` (same inputs, edit needs an id).
func addressAddCmd(app *App, verb string) *cobra.Command {
	var name, file string
	var makeDefault bool
	use := verb
	nArgs := cobra.NoArgs
	if verb == "edit" {
		use = "edit <address-id>"
		nArgs = cobra.ExactArgs(1)
	}
	cmd := &cobra.Command{
		Use:   use,
		Short: verb + " an address-book entry (contact wizard unless -f)",
		Args:  nArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := accountService(cmd, app)
			if err != nil {
				return err
			}
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			addr := account.Address{Name: name, Default: makeDefault}
			if verb == "edit" {
				addr.ID = args[0]
			}
			if file != "" {
				contacts, err := loadContacts(file)
				if err != nil {
					return err
				}
				addr.Contact = contacts.Registrant
			} else {
				var c contactYAML
				if err := contactWizard(&c); err != nil {
					return err
				}
				addr.Contact = c.contact()
			}
			if verb == "edit" {
				return svc.UpdateAddress(cmd.Context(), addr)
			}
			return svc.AddAddress(cmd.Context(), addr)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "address name (required)")
	cmd.Flags().StringVarP(&file, "file", "f", "", "contact file (skips the wizard)")
	cmd.Flags().BoolVar(&makeDefault, "default", false, "make this the default address")
	return cmd
}

func addressRmCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <address-id>",
		Short: "Delete an address-book entry",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := accountService(cmd, app)
			if err != nil {
				return err
			}
			return svc.RemoveAddress(cmd.Context(), args[0])
		},
	}
}

func addressDefaultCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "default <address-id>",
		Short: "Mark an entry as the default address",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := accountService(cmd, app)
			if err != nil {
				return err
			}
			return svc.SetDefaultAddress(cmd.Context(), args[0])
		},
	}
}
