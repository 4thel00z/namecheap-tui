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

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(0, 1)
	helpStyle   = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dangerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

type domainsMsg struct {
	domains   []registrar.Domain
	fromCache bool
}

type errMsg struct{ err error }

// Dashboard is the top-level TUI model.
type Dashboard struct {
	svc     *services.DomainService
	profile string
	version string

	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
	width   int
}

// NewDashboard builds the dashboard model.
func NewDashboard(svc *services.DomainService, profileName, version string) Dashboard {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "DOMAIN", Width: 32},
			{Title: "EXPIRES", Width: 12},
			{Title: "DAYS", Width: 6},
			{Title: "AUTORENEW", Width: 10},
			{Title: "PRIVACY", Width: 8},
			{Title: "LOCKED", Width: 7},
		}),
		table.WithFocused(true),
	)
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return Dashboard{svc: svc, profile: profileName, version: version, table: t, spinner: s, loading: true}
}

func (m Dashboard) load(refresh bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		domains, err := m.svc.List(ctx, refresh)
		if err != nil {
			return errMsg{err}
		}
		return domainsMsg{domains: domains, fromCache: !refresh}
	}
}

// Init loads cached data instantly, then refreshes in the background.
func (m Dashboard) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.load(false))
}

// Update handles messages.
func (m Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.table.SetHeight(msg.Height - 4)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.load(true))
		}
	case domainsMsg:
		m.err = nil
		m.table.SetRows(toRows(msg.domains))
		if msg.fromCache {
			// stale-while-revalidate: kick a background refresh
			return m, m.load(true)
		}
		m.loading = false
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
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func toRows(domains []registrar.Domain) []table.Row {
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
	out := headerStyle.Render(head) + "\n"
	if m.err != nil {
		out += errStyle.Render("error: "+m.err.Error()) + "\n"
	}
	out += m.table.View() + "\n"
	out += helpStyle.Render("q quit · r refresh · ↑/↓ navigate")
	return out
}

// Run starts the program in the alternate screen.
func Run(m tea.Model) error {
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
