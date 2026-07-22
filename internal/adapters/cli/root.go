package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// Root builds the ncp command tree.
func Root(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:           "ncp",
		Short:         "A fast TUI and CLI for the Namecheap API",
		Version:       app.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.RunTUI == nil {
				return errors.New("TUI not wired; use a subcommand (see ncp --help)")
			}
			profile, _ := cmd.Flags().GetString("profile")
			sandbox, _ := cmd.Flags().GetBool("sandbox")
			return app.RunTUI(cmd.Context(), profile, sandbox)
		},
	}
	pf := root.PersistentFlags()
	pf.String("profile", "", "profile to use (default: the stored default)")
	pf.Bool("json", false, "output JSON instead of tables")
	pf.Bool("no-cache", false, "bypass the local cache")
	pf.Bool("sandbox", false, "use the Namecheap sandbox endpoint")

	root.AddCommand(profileCmd(app), domainsCmd(app), dnsCmd(app), nsCmd(app), transferCmd(app))
	return root
}
