// Package tui is the bubbletea driving adapter.
package tui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

var (
	headerStyle    = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	errStyle       = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(0, 1)
	helpStyle      = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	warnStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dangerStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	tabStyle       = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	activeTabStyle = lipgloss.NewStyle().Bold(true).Underline(true).Padding(0, 1)
)

type domainsMsg struct {
	domains   []registrar.Domain
	fromCache bool
}

type (
	certsMsg     struct{ certs []ssl.Certificate }
	transfersMsg struct{ transfers []registrar.Transfer }
	addressesMsg struct{ addresses []account.Address }
	errMsg       struct{ err error }
)

type dashTab int

const (
	tabDomains dashTab = iota
	tabSSL
	tabTransfers
	tabAddresses
	tabCount
)

var tabNames = [tabCount]string{"Domains", "SSL", "Transfers", "Addresses"}

// DashboardDeps carries the services each dashboard tab needs.
type DashboardDeps struct {
	Domains   *services.DomainService
	SSL       *services.SSLService
	Transfers *services.TransferService
	Account   *services.AccountService
}

// Dashboard is the top-level TUI model.
type Dashboard struct {
	deps    DashboardDeps
	profile string
	version string

	tab     dashTab
	tables  [tabCount]table.Model
	loaded  [tabCount]bool
	spinner spinner.Model
	loading bool
	err     error
}

func newTable(cols []table.Column) table.Model {
	return table.New(table.WithColumns(cols), table.WithFocused(true))
}

// NewDashboard builds the dashboard model.
func NewDashboard(deps DashboardDeps, profileName, version string) Dashboard {
	var tables [tabCount]table.Model
	tables[tabDomains] = newTable([]table.Column{
		{Title: "DOMAIN", Width: 32},
		{Title: "EXPIRES", Width: 12},
		{Title: "DAYS", Width: 6},
		{Title: "AUTORENEW", Width: 10},
		{Title: "PRIVACY", Width: 8},
		{Title: "LOCKED", Width: 7},
	})
	tables[tabSSL] = newTable([]table.Column{
		{Title: "ID", Width: 10},
		{Title: "HOST", Width: 30},
		{Title: "TYPE", Width: 16},
		{Title: "STATUS", Width: 12},
		{Title: "EXPIRES", Width: 12},
	})
	tables[tabTransfers] = newTable([]table.Column{
		{Title: "ID", Width: 10},
		{Title: "DOMAIN", Width: 30},
		{Title: "STATUS", Width: 26},
		{Title: "DATE", Width: 12},
	})
	tables[tabAddresses] = newTable([]table.Column{
		{Title: "ID", Width: 10},
		{Title: "NAME", Width: 30},
		{Title: "DEFAULT", Width: 8},
	})
	return Dashboard{
		deps: deps, profile: profileName, version: version,
		tables:  tables,
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		loading: true,
	}
}

func (m Dashboard) loadDomains(refresh bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		domains, err := m.deps.Domains.List(ctx, refresh)
		if err != nil {
			return errMsg{err}
		}
		return domainsMsg{domains: domains, fromCache: !refresh}
	}
}

func (m Dashboard) loadTab(tab dashTab) tea.Cmd {
	load := func(f func(ctx context.Context) (tea.Msg, error)) tea.Cmd {
		return func() tea.Msg {
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()
			msg, err := f(ctx)
			if err != nil {
				return errMsg{err}
			}
			return msg
		}
	}
	switch tab {
	case tabSSL:
		if m.deps.SSL == nil {
			return nil
		}
		return load(func(ctx context.Context) (tea.Msg, error) {
			certs, err := m.deps.SSL.List(ctx)
			return certsMsg{certs}, err
		})
	case tabTransfers:
		if m.deps.Transfers == nil {
			return nil
		}
		return load(func(ctx context.Context) (tea.Msg, error) {
			transfers, err := m.deps.Transfers.List(ctx)
			return transfersMsg{transfers}, err
		})
	case tabAddresses:
		if m.deps.Account == nil {
			return nil
		}
		return load(func(ctx context.Context) (tea.Msg, error) {
			addresses, err := m.deps.Account.Addresses(ctx)
			return addressesMsg{addresses}, err
		})
	default:
		return m.loadDomains(false)
	}
}

// Init loads cached domains instantly, then refreshes in the background.
func (m Dashboard) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadDomains(false))
}

// Update handles messages.
func (m Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		for i := range m.tables {
			m.tables[i].SetHeight(msg.Height - 5)
		}
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			m.loaded[m.tab] = false
			if m.tab == tabDomains {
				return m, tea.Batch(m.spinner.Tick, m.loadDomains(true))
			}
			return m, tea.Batch(m.spinner.Tick, m.loadTab(m.tab))
		case "tab":
			return m.switchTab((m.tab + 1) % tabCount)
		case "1", "2", "3", "4":
			n, _ := strconv.Atoi(msg.String())
			return m.switchTab(dashTab(n - 1))
		}
	case domainsMsg:
		m.err = nil
		m.loaded[tabDomains] = true
		m.tables[tabDomains].SetRows(domainRows(msg.domains))
		if msg.fromCache {
			// stale-while-revalidate: kick a background refresh
			return m, m.loadDomains(true)
		}
		m.loading = false
		return m, nil
	case certsMsg:
		m.err = nil
		m.loading = false
		m.loaded[tabSSL] = true
		m.tables[tabSSL].SetRows(certRows(msg.certs))
		return m, nil
	case transfersMsg:
		m.err = nil
		m.loading = false
		m.loaded[tabTransfers] = true
		m.tables[tabTransfers].SetRows(transferRows(msg.transfers))
		return m, nil
	case addressesMsg:
		m.err = nil
		m.loading = false
		m.loaded[tabAddresses] = true
		m.tables[tabAddresses].SetRows(addressRows(msg.addresses))
		return m, nil
	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.tables[m.tab], cmd = m.tables[m.tab].Update(msg)
	return m, cmd
}

func (m Dashboard) switchTab(tab dashTab) (tea.Model, tea.Cmd) {
	if tab < 0 || tab >= tabCount {
		return m, nil
	}
	m.tab = tab
	if m.loaded[tab] {
		return m, nil
	}
	cmd := m.loadTab(tab)
	if cmd == nil {
		return m, nil
	}
	m.loading = true
	return m, tea.Batch(m.spinner.Tick, cmd)
}

func domainRows(domains []registrar.Domain) []table.Row {
	rows := make([]table.Row, len(domains))
	for i, d := range domains {
		days := int(time.Until(d.Expires).Hours() / 24)
		daysStr := strconv.Itoa(days)
		switch {
		case days < 30:
			daysStr = dangerStyle.Render(daysStr)
		case days < 90:
			daysStr = warnStyle.Render(daysStr)
		}
		rows[i] = table.Row{
			d.Name.String(), d.Expires.Format(time.DateOnly), daysStr,
			mark(d.AutoRenew), mark(d.Privacy), mark(d.Locked),
		}
	}
	return rows
}

func certRows(certs []ssl.Certificate) []table.Row {
	rows := make([]table.Row, len(certs))
	for i, c := range certs {
		expires := "-"
		if !c.Expires.IsZero() {
			expires = c.Expires.Format(time.DateOnly)
		}
		rows[i] = table.Row{c.ID, c.Host, c.Type, c.Status, expires}
	}
	return rows
}

func transferRows(transfers []registrar.Transfer) []table.Row {
	rows := make([]table.Row, len(transfers))
	for i, t := range transfers {
		rows[i] = table.Row{t.ID, t.Name, t.Status, t.Date}
	}
	return rows
}

func addressRows(addresses []account.Address) []table.Row {
	rows := make([]table.Row, len(addresses))
	for i, a := range addresses {
		rows[i] = table.Row{a.ID, a.Name, mark(a.Default)}
	}
	return rows
}

func mark(b bool) string {
	if b {
		return "✓"
	}
	return "-"
}

// View renders the dashboard.
func (m Dashboard) View() string {
	head := fmt.Sprintf("ncp %s — profile %s", m.version, m.profile)
	if m.loading {
		head += "  " + m.spinner.View()
	}
	tabs := ""
	for i, name := range tabNames {
		label := fmt.Sprintf("%d %s", i+1, name)
		if dashTab(i) == m.tab {
			tabs += activeTabStyle.Render(label)
		} else {
			tabs += tabStyle.Render(label)
		}
	}
	out := headerStyle.Render(head) + "\n" + tabs + "\n"
	if m.err != nil {
		out += errStyle.Render("error: "+m.err.Error()) + "\n"
	}
	out += m.tables[m.tab].View() + "\n"
	out += helpStyle.Render("1-4/tab switch · r refresh · ↑/↓ navigate · q quit")
	return out
}

// Run starts the program in the alternate screen.
func Run(m tea.Model) error {
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
