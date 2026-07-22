package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func nsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ns",
		Short: "Manage personal (glue-record) nameservers",
	}
	cmd.AddCommand(nsCreateCmd(app), nsUpdateCmd(app), nsDeleteCmd(app), nsInfoCmd(app))
	return cmd
}

func nsService(cmd *cobra.Command, app *App) (*services.NSService, error) {
	if app.NS == nil {
		return nil, fmt.Errorf("ns service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.NS(cmd.Context(), profile, sandbox)
}

func nsCreateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "create <domain> <host> <ip>",
		Short: "Register a personal nameserver",
		Args:  cobra.ExactArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := nsService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Create(cmd.Context(), args[0], args[1], args[2])
		},
	}
}

func nsUpdateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "update <domain> <host> <old-ip> <new-ip>",
		Short: "Change a personal nameserver's IP",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := nsService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Update(cmd.Context(), args[0], args[1], args[2], args[3])
		},
	}
}

func nsDeleteCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "delete <domain> <host>",
		Short: "Delete a personal nameserver",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := nsService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Delete(cmd.Context(), args[0], args[1])
		},
	}
}

func nsInfoCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "info <domain> <host>",
		Short: "Show a personal nameserver's details",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := nsService(cmd, app)
			if err != nil {
				return err
			}
			info, err := svc.Info(cmd.Context(), args[0], args[1])
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return renderJSON(cmd.OutOrStdout(), map[string]any{
					"host": info.Host, "ip": info.IP, "statuses": info.Statuses,
				})
			}
			w := cmd.OutOrStdout()
			_, _ = fmt.Fprintf(w, "Host:     %s\n", info.Host)
			_, _ = fmt.Fprintf(w, "IP:       %s\n", info.IP)
			_, _ = fmt.Fprintf(w, "Statuses: %s\n", strings.Join(info.Statuses, ", "))
			return nil
		},
	}
}
