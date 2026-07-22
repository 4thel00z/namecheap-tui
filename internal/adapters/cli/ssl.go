package cli

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func sslCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ssl",
		Short: "Manage SSL certificates",
	}
	cmd.AddCommand(
		sslListCmd(app), sslCreateCmd(app), sslActivateCmd(app, "activate"),
		sslInfoCmd(app), sslRenewCmd(app), sslActivateCmd(app, "reissue"),
		sslApproversCmd(app), sslResendCmd(app), sslRevokeCmd(app),
	)
	return cmd
}

func sslService(cmd *cobra.Command, app *App) (*services.SSLService, error) {
	if app.SSL == nil {
		return nil, fmt.Errorf("ssl service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.SSL(cmd.Context(), profile, sandbox)
}

func renderCerts(cmd *cobra.Command, certs []ssl.Certificate) error {
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		type row struct {
			ID      string `json:"id"`
			Host    string `json:"host,omitempty"`
			Type    string `json:"type"`
			Status  string `json:"status"`
			Expires string `json:"expires,omitempty"`
			Expired bool   `json:"expired"`
		}
		out := make([]row, len(certs))
		for i, c := range certs {
			expires := ""
			if !c.Expires.IsZero() {
				expires = c.Expires.Format(time.DateOnly)
			}
			out[i] = row{c.ID, c.Host, c.Type, c.Status, expires, c.Expired}
		}
		return renderJSON(cmd.OutOrStdout(), out)
	}
	rows := make([][]string, len(certs))
	for i, c := range certs {
		expires := "-"
		if !c.Expires.IsZero() {
			expires = c.Expires.Format(time.DateOnly)
		}
		rows[i] = []string{c.ID, c.Host, c.Type, c.Status, expires}
	}
	renderTable(cmd.OutOrStdout(), []string{"ID", "HOST", "TYPE", "STATUS", "EXPIRES"}, rows)
	return nil
}

func sslListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List certificates",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			certs, err := svc.List(cmd.Context())
			if err != nil {
				return err
			}
			return renderCerts(cmd, certs)
		},
	}
}

func sslCreateCmd(app *App) *cobra.Command {
	var years int
	cmd := &cobra.Command{
		Use:   "create <type>",
		Short: "Purchase a certificate (e.g. PositiveSSL)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			cert, err := svc.Purchase(cmd.Context(), ssl.Purchase{Type: args[0], Years: years})
			if err != nil {
				return err
			}
			return renderCerts(cmd, []ssl.Certificate{cert})
		},
	}
	cmd.Flags().IntVar(&years, "years", 1, "certificate years")
	return cmd
}

// sslActivateCmd builds both `activate` and `reissue` (same inputs).
func sslActivateCmd(app *App, verb string) *cobra.Command {
	var csrFile, dvMethod, approver, webServer string
	cmd := &cobra.Command{
		Use:   verb + " <certificate-id>",
		Short: strings.ToUpper(verb[:1]) + verb[1:] + " a certificate with a CSR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			csr, err := os.ReadFile(csrFile)
			if err != nil {
				return fmt.Errorf("read csr: %w", err)
			}
			activation := ssl.Activation{
				CertificateID: args[0], CSR: string(csr),
				WebServerType: webServer, DVMethod: ssl.DVMethod(dvMethod),
				ApproverEmail: approver,
			}
			if verb == "reissue" {
				return svc.Reissue(cmd.Context(), activation)
			}
			return svc.Activate(cmd.Context(), activation)
		},
	}
	cmd.Flags().StringVar(&csrFile, "csr-file", "", "path to the CSR PEM file (required)")
	cmd.Flags().StringVar(&dvMethod, "dv", "email", "domain validation method: email, http, or dns")
	cmd.Flags().StringVar(&approver, "approver", "", "approver email (required for --dv email)")
	cmd.Flags().StringVar(&webServer, "web-server", "nginx", "web server type")
	_ = cmd.MarkFlagRequired("csr-file")
	return cmd
}

func sslInfoCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "info <certificate-id>",
		Short: "Show certificate details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			cert, err := svc.Info(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return renderCerts(cmd, []ssl.Certificate{cert})
		},
	}
}

func sslRenewCmd(app *App) *cobra.Command {
	var years int
	cmd := &cobra.Command{
		Use:   "renew <certificate-id> <type>",
		Short: "Renew a certificate",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			cert, err := svc.Renew(cmd.Context(), args[0], ssl.Purchase{Type: args[1], Years: years})
			if err != nil {
				return err
			}
			return renderCerts(cmd, []ssl.Certificate{cert})
		},
	}
	cmd.Flags().IntVar(&years, "years", 1, "renewal years")
	return cmd
}

func sslApproversCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "approvers <domain> <type>",
		Short: "List valid DV approver emails",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			emails, err := svc.Approvers(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return renderJSON(cmd.OutOrStdout(), emails)
			}
			for _, e := range emails {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), e)
			}
			return nil
		},
	}
}

func sslResendCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "resend-approver <certificate-id>",
		Short: "Resend the DV approval email",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			return svc.ResendApproverEmail(cmd.Context(), args[0])
		},
	}
}

func sslRevokeCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "revoke <certificate-id> <type>",
		Short: "Revoke a certificate",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := sslService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Revoke(cmd.Context(), args[0], args[1])
		},
	}
}
