package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

func configCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Local settings and Turso sync",
	}
	cmd.AddCommand(configGetCmd(app), configSetCmd(app), configSyncCmd(app))
	return cmd
}

func settingsRepo(app *App) (ports.SettingsRepo, error) {
	if app.Settings == nil {
		return nil, fmt.Errorf("settings not wired")
	}
	return app.Settings, nil
}

func configGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Read a setting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := settingsRepo(app)
			if err != nil {
				return err
			}
			v, err := repo.Get(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), v)
			return nil
		},
	}
}

func configSetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Write a setting",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := settingsRepo(app)
			if err != nil {
				return err
			}
			return repo.Set(cmd.Context(), args[0], args[1])
		},
	}
}

func configSyncCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "sync",
		Short: "Sync the local DB with the Turso primary (no-op without TURSO_DATABASE_URL)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.Sync == nil {
				return fmt.Errorf("sync not wired")
			}
			if err := app.Sync(); err != nil {
				return err
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "synced")
			return nil
		},
	}
}
