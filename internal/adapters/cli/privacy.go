package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func privacyCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "privacy",
		Short: "Manage domain privacy (whoisguard) subscriptions",
	}
	cmd.AddCommand(
		privacyListCmd(app),
		privacySimpleCmd(app, "disable", "Disable privacy", (*services.PrivacyService).Disable),
		privacySimpleCmd(app, "change-email", "Rotate the privacy contact email", (*services.PrivacyService).ChangeEmail),
		privacySimpleCmd(app, "unassign", "Detach a subscription from its domain", (*services.PrivacyService).Unassign),
		privacySimpleCmd(app, "discard", "Discard an unused subscription", (*services.PrivacyService).Discard),
		privacyEnableCmd(app), privacyRenewCmd(app), privacyAssignCmd(app),
	)
	return cmd
}

func privacyService(cmd *cobra.Command, app *App) (*services.PrivacyService, error) {
	if app.Privacy == nil {
		return nil, fmt.Errorf("privacy service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.Privacy(cmd.Context(), profile, sandbox)
}

func privacyListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List privacy subscriptions",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := privacyService(cmd, app)
			if err != nil {
				return err
			}
			subs, err := svc.List(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					ID      string `json:"id"`
					Domain  string `json:"domain,omitempty"`
					Status  string `json:"status"`
					Expires string `json:"expires,omitempty"`
				}
				out := make([]row, len(subs))
				for i, s := range subs {
					expires := ""
					if !s.Expires.IsZero() {
						expires = s.Expires.Format(time.DateOnly)
					}
					out[i] = row{s.ID, s.Domain, s.Status, expires}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(subs))
			for i, s := range subs {
				domain := s.Domain
				if domain == "" {
					domain = "-"
				}
				expires := "-"
				if !s.Expires.IsZero() {
					expires = s.Expires.Format(time.DateOnly)
				}
				rows[i] = []string{s.ID, domain, s.Status, expires}
			}
			renderTable(cmd.OutOrStdout(), []string{"ID", "DOMAIN", "STATUS", "EXPIRES"}, rows)
			return nil
		},
	}
}

// privacySimpleCmd builds a one-ID-argument subcommand.
func privacySimpleCmd(app *App, use, short string,
	action func(*services.PrivacyService, context.Context, string) error,
) *cobra.Command {
	return &cobra.Command{
		Use:   use + " <subscription-id>",
		Short: short,
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := privacyService(cmd, app)
			if err != nil {
				return err
			}
			return action(svc, cmd.Context(), args[0])
		},
	}
}

func privacyEnableCmd(app *App) *cobra.Command {
	var forwardTo string
	cmd := &cobra.Command{
		Use:   "enable <subscription-id>",
		Short: "Enable privacy",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := privacyService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Enable(cmd.Context(), args[0], forwardTo)
		},
	}
	cmd.Flags().StringVar(&forwardTo, "forward-to", "", "email to forward WHOIS mail to")
	return cmd
}

func privacyRenewCmd(app *App) *cobra.Command {
	var years int
	cmd := &cobra.Command{
		Use:   "renew <subscription-id>",
		Short: "Renew a subscription",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := privacyService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Renew(cmd.Context(), args[0], years)
		},
	}
	cmd.Flags().IntVar(&years, "years", 1, "renewal years")
	return cmd
}

func privacyAssignCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "assign <subscription-id> <domain>",
		Short: "Attach a subscription to a domain",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := privacyService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Assign(cmd.Context(), args[0], args[1])
		},
	}
}
