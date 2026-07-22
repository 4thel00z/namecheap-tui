package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func transferCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "transfer",
		Short: "Manage inbound domain transfers",
	}
	cmd.AddCommand(transferCreateCmd(app), transferStatusCmd(app), transferListCmd(app))
	return cmd
}

func transferService(cmd *cobra.Command, app *App) (*services.TransferService, error) {
	if app.Transfers == nil {
		return nil, fmt.Errorf("transfer service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.Transfers(cmd.Context(), profile, sandbox)
}

func renderTransfers(cmd *cobra.Command, transfers []registrar.Transfer) error {
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		type row struct {
			ID       string `json:"id"`
			Name     string `json:"name,omitempty"`
			Status   string `json:"status"`
			StatusID int    `json:"status_id"`
			Date     string `json:"date,omitempty"`
		}
		out := make([]row, len(transfers))
		for i, t := range transfers {
			out[i] = row{t.ID, t.Name, t.Status, t.StatusID, t.Date}
		}
		return renderJSON(cmd.OutOrStdout(), out)
	}
	rows := make([][]string, len(transfers))
	for i, t := range transfers {
		rows[i] = []string{t.ID, t.Name, t.Status, t.Date}
	}
	renderTable(cmd.OutOrStdout(), []string{"ID", "DOMAIN", "STATUS", "DATE"}, rows)
	return nil
}

func transferCreateCmd(app *App) *cobra.Command {
	var years int
	cmd := &cobra.Command{
		Use:   "create <domain> <epp-code>",
		Short: "Start an inbound transfer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := transferService(cmd, app)
			if err != nil {
				return err
			}
			tr, err := svc.Create(cmd.Context(), args[0], args[1], years)
			if err != nil {
				return err
			}
			return renderTransfers(cmd, []registrar.Transfer{tr})
		},
	}
	cmd.Flags().IntVar(&years, "years", 1, "renewal years included with the transfer")
	return cmd
}

func transferStatusCmd(app *App) *cobra.Command {
	var resubmit bool
	cmd := &cobra.Command{
		Use:   "status <transfer-id>",
		Short: "Show (or resubmit) a transfer",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := transferService(cmd, app)
			if err != nil {
				return err
			}
			if resubmit {
				if err := svc.Resubmit(cmd.Context(), args[0]); err != nil {
					return err
				}
			}
			tr, err := svc.Status(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			return renderTransfers(cmd, []registrar.Transfer{tr})
		},
	}
	cmd.Flags().BoolVar(&resubmit, "resubmit", false, "resubmit the transfer before showing status")
	return cmd
}

func transferListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List transfers on the account",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := transferService(cmd, app)
			if err != nil {
				return err
			}
			transfers, err := svc.List(cmd.Context())
			if err != nil {
				return err
			}
			return renderTransfers(cmd, transfers)
		},
	}
}
