package cli

import (
	"fmt"
	"os"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

type contactYAML struct {
	FirstName     string `yaml:"first_name" json:"first_name"`
	LastName      string `yaml:"last_name" json:"last_name"`
	Organization  string `yaml:"organization,omitempty" json:"organization,omitempty"`
	Address1      string `yaml:"address1" json:"address1"`
	Address2      string `yaml:"address2,omitempty" json:"address2,omitempty"`
	City          string `yaml:"city" json:"city"`
	StateProvince string `yaml:"state_province" json:"state_province"`
	PostalCode    string `yaml:"postal_code" json:"postal_code"`
	Country       string `yaml:"country" json:"country"`
	Phone         string `yaml:"phone" json:"phone"`
	Email         string `yaml:"email" json:"email"`
}

func (c contactYAML) contact() registrar.Contact {
	return registrar.Contact{
		FirstName: c.FirstName, LastName: c.LastName, Organization: c.Organization,
		Address1: c.Address1, Address2: c.Address2, City: c.City,
		StateProvince: c.StateProvince, PostalCode: c.PostalCode,
		Country: c.Country, Phone: c.Phone, Email: c.Email,
	}
}

func toContactYAML(c registrar.Contact) contactYAML {
	return contactYAML{
		FirstName: c.FirstName, LastName: c.LastName, Organization: c.Organization,
		Address1: c.Address1, Address2: c.Address2, City: c.City,
		StateProvince: c.StateProvince, PostalCode: c.PostalCode,
		Country: c.Country, Phone: c.Phone, Email: c.Email,
	}
}

type contactSetYAML struct {
	Registrant *contactYAML `yaml:"registrant" json:"registrant"`
	Tech       *contactYAML `yaml:"tech" json:"tech"`
	Admin      *contactYAML `yaml:"admin" json:"admin"`
	AuxBilling *contactYAML `yaml:"aux_billing" json:"aux_billing"`
}

// loadContacts reads a contact file: either a full four-role mapping or a
// single flat contact applied uniformly.
func loadContacts(path string) (registrar.ContactSet, error) {
	body, err := os.ReadFile(path)
	if err != nil {
		return registrar.ContactSet{}, err
	}
	var set contactSetYAML
	if err := yaml.Unmarshal(body, &set); err == nil && set.Registrant != nil {
		out := registrar.UniformContacts(set.Registrant.contact())
		if set.Tech != nil {
			out.Tech = set.Tech.contact()
		}
		if set.Admin != nil {
			out.Admin = set.Admin.contact()
		}
		if set.AuxBilling != nil {
			out.AuxBilling = set.AuxBilling.contact()
		}
		return out, nil
	}
	var flat contactYAML
	if err := yaml.Unmarshal(body, &flat); err != nil {
		return registrar.ContactSet{}, fmt.Errorf("parse contacts file: %w", err)
	}
	if flat.FirstName == "" {
		return registrar.ContactSet{}, fmt.Errorf("contacts file needs a registrant or a flat contact")
	}
	return registrar.UniformContacts(flat.contact()), nil
}

// contactWizard collects a single contact interactively.
func contactWizard(c *contactYAML) error {
	return huh.NewForm(
		huh.NewGroup(
			huh.NewInput().Title("First name").Value(&c.FirstName),
			huh.NewInput().Title("Last name").Value(&c.LastName),
			huh.NewInput().Title("Organization (optional)").Value(&c.Organization),
			huh.NewInput().Title("Address line 1").Value(&c.Address1),
			huh.NewInput().Title("City").Value(&c.City),
		),
		huh.NewGroup(
			huh.NewInput().Title("State/Province").Value(&c.StateProvince),
			huh.NewInput().Title("Postal code").Value(&c.PostalCode),
			huh.NewInput().Title("Country (ISO code, e.g. DE)").Value(&c.Country),
			huh.NewInput().Title("Phone (+NN.NNNNNNNNN)").Value(&c.Phone),
			huh.NewInput().Title("Email").Value(&c.Email),
		),
	).Run()
}

func domainsRegisterCmd(app *App) *cobra.Command {
	var years int
	var privacy bool
	var contactsFile string
	cmd := &cobra.Command{
		Use:   "register <domain>",
		Short: "Register a new domain (interactive contact wizard unless --contacts-file)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			name, err := registrar.Parse(args[0])
			if err != nil {
				return err
			}
			var contacts registrar.ContactSet
			if contactsFile != "" {
				if contacts, err = loadContacts(contactsFile); err != nil {
					return err
				}
			} else {
				var c contactYAML
				if err := contactWizard(&c); err != nil {
					return err
				}
				contacts = registrar.UniformContacts(c.contact())
			}
			res, err := svc.Register(cmd.Context(), registrar.Registration{
				Name: name, Years: years, Contacts: contacts, Privacy: privacy,
			})
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return renderJSON(cmd.OutOrStdout(), map[string]any{
					"domain": res.Domain, "registered": res.Registered,
					"charged_amount": res.ChargedAmount, "order_id": res.OrderID,
				})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "registered %s (charged %.2f, order %s)\n",
				res.Domain, res.ChargedAmount, res.OrderID)
			return nil
		},
	}
	cmd.Flags().IntVar(&years, "years", 1, "registration years (1-10)")
	cmd.Flags().BoolVar(&privacy, "privacy", true, "enable free domain privacy")
	cmd.Flags().StringVar(&contactsFile, "contacts-file", "", "YAML/JSON contact file (skips the wizard)")
	return cmd
}

func domainsRenewCmd(app *App) *cobra.Command {
	var years int
	cmd := &cobra.Command{
		Use:   "renew <domain>",
		Short: "Renew a domain registration",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			res, err := svc.Renew(cmd.Context(), args[0], years)
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return renderJSON(cmd.OutOrStdout(), map[string]any{
					"charged_amount": res.ChargedAmount, "order_id": res.OrderID,
					"expires": res.Expires.Format("2006-01-02"),
				})
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "renewed %s until %s (charged %.2f)\n",
				args[0], res.Expires.Format("2006-01-02"), res.ChargedAmount)
			return nil
		},
	}
	cmd.Flags().IntVar(&years, "years", 1, "years to add")
	return cmd
}

func domainsReactivateCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "reactivate <domain>",
		Short: "Reactivate an expired domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			return svc.Reactivate(cmd.Context(), args[0])
		},
	}
}

func domainsLockCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Manage the registrar lock",
	}
	get := &cobra.Command{
		Use:   "get <domain>",
		Short: "Show the registrar-lock status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			locked, err := svc.Lock(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return renderJSON(cmd.OutOrStdout(), map[string]bool{"locked": locked})
			}
			state := "unlocked"
			if locked {
				state = "locked"
			}
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), state)
			return nil
		},
	}
	setLock := func(use string, locked bool) *cobra.Command {
		return &cobra.Command{
			Use:   use + " <domain>",
			Short: "Turn the registrar lock " + use,
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				svc, err := domainsService(cmd, app)
				if err != nil {
					return err
				}
				return svc.SetLock(cmd.Context(), args[0], locked)
			},
		}
	}
	cmd.AddCommand(get, setLock("on", true), setLock("off", false))
	return cmd
}

func domainsTldsCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "tlds",
		Short: "List TLDs available for registration",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			tlds, err := svc.TLDs(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					Name         string `json:"name"`
					MinYears     int    `json:"min_years"`
					MaxYears     int    `json:"max_years"`
					Registerable bool   `json:"registerable"`
				}
				out := make([]row, len(tlds))
				for i, t := range tlds {
					out[i] = row{t.Name, t.MinYears, t.MaxYears, t.Registerable}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(tlds))
			for i, t := range tlds {
				rows[i] = []string{
					t.Name, strconv.Itoa(t.MinYears), strconv.Itoa(t.MaxYears),
					boolMark(t.Registerable),
				}
			}
			renderTable(cmd.OutOrStdout(), []string{"TLD", "MIN YEARS", "MAX YEARS", "API REGISTERABLE"}, rows)
			return nil
		},
	}
}

func domainsContactsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contacts",
		Short: "Manage WHOIS contacts",
	}
	get := &cobra.Command{
		Use:   "get <domain>",
		Short: "Show WHOIS contacts (YAML; pipe to a file for contacts set)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			contacts, err := svc.Contacts(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			out := contactSetYAML{
				Registrant: ptr(toContactYAML(contacts.Registrant)),
				Tech:       ptr(toContactYAML(contacts.Tech)),
				Admin:      ptr(toContactYAML(contacts.Admin)),
				AuxBilling: ptr(toContactYAML(contacts.AuxBilling)),
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				return renderJSON(cmd.OutOrStdout(), out)
			}
			return yaml.NewEncoder(cmd.OutOrStdout()).Encode(out)
		},
	}
	set := &cobra.Command{
		Use:   "set <domain>",
		Short: "Replace WHOIS contacts from a YAML/JSON file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			file, _ := cmd.Flags().GetString("file")
			if file == "" {
				return fmt.Errorf("--file is required")
			}
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			contacts, err := loadContacts(file)
			if err != nil {
				return err
			}
			return svc.SetContacts(cmd.Context(), args[0], contacts)
		},
	}
	set.Flags().StringP("file", "f", "", "contact file (single contact or per-role mapping)")
	cmd.AddCommand(get, set)
	return cmd
}

func ptr[T any](v T) *T { return &v }
