package cli

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
)

func profileCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage Namecheap account profiles",
	}
	cmd.AddCommand(profileAddCmd(app), profileListCmd(app), profileUseCmd(app), profileRmCmd(app))
	return cmd
}

func profileAddCmd(app *App) *cobra.Command {
	var name, apiUser, username, apiKey, clientIP string
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a profile (interactive form when flags are omitted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// --sandbox is the root's persistent flag; don't redefine it locally.
			sandbox, _ = cmd.Flags().GetBool("sandbox")
			interactive := name == "" && apiUser == "" && apiKey == ""
			if interactive {
				if err := profileForm(cmd, app, &name, &apiUser, &username, &apiKey, &clientIP, &sandbox); err != nil {
					return err
				}
			}
			if username == "" {
				username = apiUser
			}
			endpoint := account.EndpointProduction
			if sandbox {
				endpoint = account.EndpointSandbox
			}
			creds := account.Credentials{
				Profile: account.Profile{
					Name: name, APIUser: apiUser, Username: username,
					ClientIP: clientIP, Endpoint: endpoint,
				},
				APIKey: apiKey,
			}
			if err := app.Profiles.Add(cmd.Context(), creds); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "profile %q saved\n", name)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&name, "name", "", "profile name")
	f.StringVar(&apiUser, "api-user", "", "Namecheap API user")
	f.StringVar(&username, "username", "", "Namecheap username (default: api-user)")
	f.StringVar(&apiKey, "api-key", "", "Namecheap API key")
	f.StringVar(&clientIP, "client-ip", "", "whitelisted client IPv4 (empty = auto-detect)")
	return cmd
}

// profileForm collects profile fields interactively, pre-filling the
// detected public IP.
func profileForm(cmd *cobra.Command, app *App,
	name, apiUser, username, apiKey, clientIP *string, sandbox *bool,
) error {
	if *clientIP == "" {
		if ip, err := app.Profiles.DetectIP(cmd.Context()); err == nil {
			*clientIP = ip
		}
	}
	return huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Profile name").Value(name),
		huh.NewInput().Title("API user").Value(apiUser),
		huh.NewInput().Title("Username (empty = API user)").Value(username),
		huh.NewInput().Title("API key").EchoMode(huh.EchoModePassword).Value(apiKey),
		huh.NewInput().Title("Whitelisted client IP").Value(clientIP),
		huh.NewConfirm().Title("Use sandbox endpoint?").Value(sandbox),
	)).Run()
}

func profileListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := app.Profiles.List(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					Name     string `json:"name"`
					APIUser  string `json:"api_user"`
					Username string `json:"username"`
					ClientIP string `json:"client_ip"`
					Endpoint string `json:"endpoint"`
					Default  bool   `json:"default"`
				}
				out := make([]row, len(profiles))
				for i, p := range profiles {
					out[i] = row{p.Name, p.APIUser, p.Username, p.ClientIP, string(p.Endpoint), p.IsDefault}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(profiles))
			for i, p := range profiles {
				rows[i] = []string{
					p.Name, p.APIUser, p.Username, p.ClientIP,
					string(p.Endpoint), strconv.FormatBool(p.IsDefault),
				}
			}
			renderTable(cmd.OutOrStdout(),
				[]string{"NAME", "API USER", "USERNAME", "CLIENT IP", "ENDPOINT", "DEFAULT"}, rows)
			return nil
		},
	}
}

func profileUseCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Profiles.Use(cmd.Context(), args[0])
		},
	}
}

func profileRmCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Profiles.Remove(cmd.Context(), args[0])
		},
	}
}
