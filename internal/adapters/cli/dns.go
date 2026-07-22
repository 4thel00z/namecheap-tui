package cli

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func dnsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dns",
		Short: "Manage DNS records, nameservers, and email forwarding",
	}
	cmd.AddCommand(
		dnsGetCmd(app), dnsAddCmd(app), dnsRmCmd(app), dnsImportCmd(app),
		dnsExportCmd(app), dnsUseDefaultCmd(app), dnsUseCustomCmd(app),
		dnsEmailFwdCmd(app), dnsEditCmd(app),
	)
	return cmd
}

func dnsService(cmd *cobra.Command, app *App) (*services.DNSService, error) {
	if app.DNS == nil {
		return nil, fmt.Errorf("dns service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.DNS(cmd.Context(), profile, sandbox)
}

type recordYAML struct {
	Name   string `yaml:"name" json:"name"`
	Type   string `yaml:"type" json:"type"`
	Value  string `yaml:"value" json:"value"`
	TTL    int    `yaml:"ttl,omitempty" json:"ttl,omitempty"`
	MXPref int    `yaml:"mx_pref,omitempty" json:"mx_pref,omitempty"`
}

type zoneYAML struct {
	Domain  string       `yaml:"domain" json:"domain"`
	Records []recordYAML `yaml:"records" json:"records"`
}

func toZoneYAML(z dns.Zone) zoneYAML {
	out := zoneYAML{Domain: z.Domain.String(), Records: make([]recordYAML, len(z.Records))}
	for i, r := range z.Records {
		out.Records[i] = recordYAML{
			Name: r.Name, Type: string(r.Type), Value: r.Value,
			TTL: r.TTL, MXPref: r.MXPref,
		}
	}
	return out
}

func fromZoneYAML(zy zoneYAML) ([]dns.HostRecord, error) {
	records := make([]dns.HostRecord, len(zy.Records))
	for i, r := range zy.Records {
		rt, err := dns.ParseRecordType(r.Type)
		if err != nil {
			return nil, fmt.Errorf("record %d: %w", i+1, err)
		}
		records[i] = dns.HostRecord{
			Name: r.Name, Type: rt, Value: r.Value, TTL: r.TTL, MXPref: r.MXPref,
		}
	}
	return records, nil
}

func renderZone(cmd *cobra.Command, z dns.Zone) error {
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		return renderJSON(cmd.OutOrStdout(), toZoneYAML(z))
	}
	rows := make([][]string, len(z.Records))
	for i, r := range z.Records {
		ttl := strconv.Itoa(r.TTL)
		if r.TTL == 0 {
			ttl = "auto"
		}
		pref := "-"
		if r.Type == dns.MX {
			pref = strconv.Itoa(r.MXPref)
		}
		rows[i] = []string{r.Name, string(r.Type), r.Value, ttl, pref}
	}
	renderTable(cmd.OutOrStdout(), []string{"NAME", "TYPE", "VALUE", "TTL", "MX PREF"}, rows)
	return nil
}

func dnsGetCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "get <domain>",
		Short: "List the domain's DNS records",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			noCache, _ := cmd.Flags().GetBool("no-cache")
			z, err := svc.Zone(cmd.Context(), args[0], noCache)
			if err != nil {
				return err
			}
			return renderZone(cmd, z)
		},
	}
}

func dnsAddCmd(app *App) *cobra.Command {
	var ttl, mxPref int
	cmd := &cobra.Command{
		Use:   "add <domain> <type> <name> <value>",
		Short: "Add a DNS record",
		Args:  cobra.ExactArgs(4),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			rt, err := dns.ParseRecordType(args[1])
			if err != nil {
				return err
			}
			z, err := svc.Add(cmd.Context(), args[0], dns.HostRecord{
				Name: args[2], Type: rt, Value: args[3], TTL: ttl, MXPref: mxPref,
			})
			if err != nil {
				return err
			}
			return renderZone(cmd, z)
		},
	}
	cmd.Flags().IntVar(&ttl, "ttl", 0, "record TTL in seconds (default: provider default)")
	cmd.Flags().IntVar(&mxPref, "mx-pref", 10, "MX preference (MX records only)")
	return cmd
}

func dnsRmCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <domain> <type> <name> [value]",
		Short: "Remove a DNS record (value required when ambiguous)",
		Args:  cobra.RangeArgs(3, 4),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			value := ""
			if len(args) == 4 {
				value = args[3]
			}
			z, err := svc.Remove(cmd.Context(), args[0], args[2], args[1], value)
			if err != nil {
				return err
			}
			return renderZone(cmd, z)
		},
	}
}

func dnsExportCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "export <domain>",
		Short: "Export the zone as YAML (redirect to a file to snapshot)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			z, err := svc.Zone(cmd.Context(), args[0], true)
			if err != nil {
				return err
			}
			return yaml.NewEncoder(cmd.OutOrStdout()).Encode(toZoneYAML(z))
		},
	}
}

func dnsImportCmd(app *App) *cobra.Command {
	var file string
	cmd := &cobra.Command{
		Use:     "import <domain>",
		Aliases: []string{"set"},
		Short:   "REPLACE the zone from a YAML snapshot (-f file, or stdin)",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			src := cmd.InOrStdin()
			if file != "" {
				f, err := os.Open(file)
				if err != nil {
					return err
				}
				defer func() { _ = f.Close() }()
				src = f
			}
			var zy zoneYAML
			if err := yaml.NewDecoder(src).Decode(&zy); err != nil {
				return fmt.Errorf("parse zone yaml: %w", err)
			}
			records, err := fromZoneYAML(zy)
			if err != nil {
				return err
			}
			z, err := svc.Set(cmd.Context(), args[0], records)
			if err != nil {
				return err
			}
			return renderZone(cmd, z)
		},
	}
	cmd.Flags().StringVarP(&file, "file", "f", "", "zone YAML file (default: stdin)")
	return cmd
}

func dnsUseDefaultCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "use-default <domain>",
		Short: "Switch to Namecheap's default nameservers",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			return svc.UseDefault(cmd.Context(), args[0])
		},
	}
}

func dnsUseCustomCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "use-custom <domain> <ns>...",
		Short: "Switch to custom nameservers (at least 2)",
		Args:  cobra.MinimumNArgs(3),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			return svc.UseCustom(cmd.Context(), args[0], args[1:])
		},
	}
}

func dnsEmailFwdCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "emailfwd",
		Short: "Manage email forwarding",
	}
	get := &cobra.Command{
		Use:   "get <domain>",
		Short: "List mailbox forwards",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			fwds, err := svc.EmailForwards(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					Mailbox   string `json:"mailbox"`
					ForwardTo string `json:"forward_to"`
				}
				out := make([]row, len(fwds))
				for i, f := range fwds {
					out[i] = row{f.Mailbox, f.ForwardTo}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(fwds))
			for i, f := range fwds {
				rows[i] = []string{f.Mailbox, f.ForwardTo}
			}
			renderTable(cmd.OutOrStdout(), []string{"MAILBOX", "FORWARD TO"}, rows)
			return nil
		},
	}
	set := &cobra.Command{
		Use:   "set <domain> <mailbox=dest>...",
		Short: "REPLACE mailbox forwards (e.g. info=team@example.org)",
		Args:  cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := dnsService(cmd, app)
			if err != nil {
				return err
			}
			fwds := make([]dns.EmailForward, 0, len(args)-1)
			for _, arg := range args[1:] {
				mailbox, dest, ok := strings.Cut(arg, "=")
				if !ok || mailbox == "" || dest == "" {
					return fmt.Errorf("invalid forward %q, want mailbox=dest", arg)
				}
				fwds = append(fwds, dns.EmailForward{Mailbox: mailbox, ForwardTo: dest})
			}
			return svc.SetEmailForwards(cmd.Context(), args[0], fwds)
		},
	}
	cmd.AddCommand(get, set)
	return cmd
}

func dnsEditCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "edit <domain>",
		Short: "Open the interactive zone editor",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.RunZoneEditor == nil {
				return fmt.Errorf("zone editor not wired")
			}
			profile, _ := cmd.Flags().GetString("profile")
			sandbox, _ := cmd.Flags().GetBool("sandbox")
			return app.RunZoneEditor(cmd.Context(), profile, sandbox, args[0])
		},
	}
}
