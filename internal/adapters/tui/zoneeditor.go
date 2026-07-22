package tui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

var (
	addStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	delStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Strikethrough(true)
	updStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	appliedInfo = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Padding(0, 1)
)

type editorMode int

const (
	modeBrowse editorMode = iota
	modeForm
	modeConfirm
)

type zoneMsg struct{ zone dns.Zone }

type appliedMsg struct{ zone dns.Zone }

// row state markers
const (
	stateNone    = " "
	stateAdded   = "+"
	stateDeleted = "-"
	stateUpdated = "~"
)

// ZoneEditor is the interactive staged-changes DNS record editor.
type ZoneEditor struct {
	svc    *services.DNSService
	domain string

	mode    editorMode
	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
	info    string

	base    []dns.HostRecord       // records as fetched
	deleted map[int]bool           // base index -> staged delete
	updated map[int]dns.HostRecord // base index -> staged replacement
	added   []dns.HostRecord       // staged additions
	form    *recordForm            // active add/edit form
}

// NewZoneEditor builds the editor model for one domain.
func NewZoneEditor(svc *services.DNSService, domain string) ZoneEditor {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "", Width: 2},
			{Title: "NAME", Width: 20},
			{Title: "TYPE", Width: 7},
			{Title: "VALUE", Width: 40},
			{Title: "TTL", Width: 6},
			{Title: "MX", Width: 4},
		}),
		table.WithFocused(true),
	)
	return ZoneEditor{
		svc: svc, domain: domain, table: t,
		spinner: spinner.New(spinner.WithSpinner(spinner.Dot)),
		loading: true,
		deleted: map[int]bool{}, updated: map[int]dns.HostRecord{},
	}
}

func (m ZoneEditor) loadZone() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		z, err := m.svc.Zone(ctx, m.domain, true)
		if err != nil {
			return errMsg{err}
		}
		return zoneMsg{z}
	}
}

func (m ZoneEditor) applyChanges() tea.Cmd {
	cs := m.changeSet()
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
		defer cancel()
		z, err := m.svc.Apply(ctx, m.domain, cs)
		if err != nil {
			return errMsg{err}
		}
		return appliedMsg{z}
	}
}

func (m ZoneEditor) changeSet() dns.ChangeSet {
	var cs dns.ChangeSet
	for i, r := range m.base {
		if m.deleted[i] {
			cs.Remove = append(cs.Remove, r)
			continue
		}
		if upd, ok := m.updated[i]; ok {
			cs.Update = append(cs.Update, dns.RecordUpdate{Old: r, New: upd})
		}
	}
	cs.Add = append(cs.Add, m.added...)
	return cs
}

// Init loads the zone (always fresh — this editor writes).
func (m ZoneEditor) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.loadZone())
}

// Update handles messages.
func (m ZoneEditor) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.table.SetHeight(msg.Height - 6)
		return m, nil
	case zoneMsg:
		m.base = msg.zone.Records
		m.deleted = map[int]bool{}
		m.updated = map[int]dns.HostRecord{}
		m.added = nil
		m.loading = false
		m.refreshRows()
		return m, nil
	case appliedMsg:
		m.mode = modeBrowse
		m.info = fmt.Sprintf("applied — zone now has %d records", len(msg.zone.Records))
		m.base = msg.zone.Records
		m.deleted = map[int]bool{}
		m.updated = map[int]dns.HostRecord{}
		m.added = nil
		m.loading = false
		m.refreshRows()
		return m, nil
	case errMsg:
		m.err = msg.err
		m.loading = false
		m.mode = modeBrowse
		return m, nil
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	switch m.mode {
	case modeForm:
		return m.updateForm(msg)
	case modeConfirm:
		return m.updateConfirm(msg)
	default:
		return m.updateBrowse(msg)
	}
}

func (m ZoneEditor) updateBrowse(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "a":
			m.form = newRecordForm(nil)
			m.mode = modeForm
			return m, m.form.form.Init()
		case "e":
			if idx, base := m.selectedBase(); base {
				rec := m.currentRecord(idx)
				m.form = newRecordForm(&rec)
				m.form.baseIndex = idx
				m.mode = modeForm
				return m, m.form.form.Init()
			}
			return m, nil
		case "d":
			if idx, base := m.selectedBase(); base {
				m.deleted[idx] = !m.deleted[idx]
			} else if idx >= 0 {
				m.added = append(m.added[:idx], m.added[idx+1:]...)
			}
			m.refreshRows()
			return m, nil
		case "u":
			if idx, base := m.selectedBase(); base {
				delete(m.deleted, idx)
				delete(m.updated, idx)
				m.refreshRows()
			}
			return m, nil
		case "A":
			if !m.changeSet().Empty() {
				m.mode = modeConfirm
			}
			return m, nil
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.loadZone())
		}
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func (m ZoneEditor) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	model, cmd := m.form.form.Update(msg)
	if f, ok := model.(*huh.Form); ok {
		m.form.form = f
	}
	switch m.form.form.State {
	case huh.StateCompleted:
		rec, err := m.form.record()
		if err != nil {
			m.err = err
			m.mode = modeBrowse
			return m, nil
		}
		m.err = nil
		if m.form.baseIndex >= 0 {
			m.updated[m.form.baseIndex] = rec
		} else {
			m.added = append(m.added, rec)
		}
		m.mode = modeBrowse
		m.refreshRows()
		return m, nil
	case huh.StateAborted:
		m.mode = modeBrowse
		return m, nil
	}
	return m, cmd
}

func (m ZoneEditor) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "y", "Y", "enter":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.applyChanges())
		case "n", "N", "esc", "q":
			m.mode = modeBrowse
			return m, nil
		}
	}
	return m, nil
}

// selectedBase maps the table cursor to (index, isBaseRecord). Added records
// come after base records in the table; their index is into m.added.
func (m ZoneEditor) selectedBase() (int, bool) {
	cur := m.table.Cursor()
	if cur < 0 {
		return -1, false
	}
	if cur < len(m.base) {
		return cur, true
	}
	addIdx := cur - len(m.base)
	if addIdx < len(m.added) {
		return addIdx, false
	}
	return -1, false
}

// currentRecord returns the effective (possibly staged-updated) base record.
func (m ZoneEditor) currentRecord(idx int) dns.HostRecord {
	if upd, ok := m.updated[idx]; ok {
		return upd
	}
	return m.base[idx]
}

func (m *ZoneEditor) refreshRows() {
	rows := make([]table.Row, 0, len(m.base)+len(m.added))
	for i, r := range m.base {
		state := stateNone
		rec := r
		style := lipgloss.NewStyle()
		switch {
		case m.deleted[i]:
			state, style = stateDeleted, delStyle
		default:
			if upd, ok := m.updated[i]; ok {
				state, rec, style = stateUpdated, upd, updStyle
			}
		}
		rows = append(rows, recordRow(state, rec, style))
	}
	for _, r := range m.added {
		rows = append(rows, recordRow(stateAdded, r, addStyle))
	}
	m.table.SetRows(rows)
}

func recordRow(state string, r dns.HostRecord, style lipgloss.Style) table.Row {
	ttl := strconv.Itoa(r.TTL)
	if r.TTL == 0 {
		ttl = "auto"
	}
	pref := "-"
	if r.Type == dns.MX {
		pref = strconv.Itoa(r.MXPref)
	}
	return table.Row{
		style.Render(state), style.Render(r.Name), style.Render(string(r.Type)),
		style.Render(r.Value), style.Render(ttl), style.Render(pref),
	}
}

// View renders the editor.
func (m ZoneEditor) View() string {
	switch m.mode {
	case modeForm:
		return m.form.form.View()
	case modeConfirm:
		return m.confirmView()
	}
	head := fmt.Sprintf("zone editor — %s", m.domain)
	if m.loading {
		head += "  " + m.spinner.View()
	}
	out := headerStyle.Render(head) + "\n"
	if m.err != nil {
		out += errStyle.Render("error: "+m.err.Error()) + "\n"
	}
	if m.info != "" {
		out += appliedInfo.Render(m.info) + "\n"
	}
	out += m.table.View() + "\n"
	staged := m.changeSet()
	status := "no staged changes"
	if !staged.Empty() {
		status = fmt.Sprintf("staged: +%d ~%d -%d", len(staged.Add), len(staged.Update), len(staged.Remove))
	}
	out += helpStyle.Render(status+"  ·  a add · e edit · d delete · u undo · A apply · r reload · q quit") + "\n"
	return out
}

func (m ZoneEditor) confirmView() string {
	cs := m.changeSet()
	var b strings.Builder
	b.WriteString(headerStyle.Render(fmt.Sprintf("apply %s?", m.domain)) + "\n\n")
	for _, u := range cs.Update {
		b.WriteString(updStyle.Render(fmt.Sprintf("  ~ %s %s %s → %s %s %s",
			u.Old.Type, u.Old.Name, u.Old.Value, u.New.Type, u.New.Name, u.New.Value)) + "\n")
	}
	for _, r := range cs.Remove {
		b.WriteString(delStyle.Render(fmt.Sprintf("  - %s %s %s", r.Type, r.Name, r.Value)) + "\n")
	}
	for _, r := range cs.Add {
		b.WriteString(addStyle.Render(fmt.Sprintf("  + %s %s %s", r.Type, r.Name, r.Value)) + "\n")
	}
	b.WriteString("\n" + helpStyle.Render("setHosts replaces the whole zone — y apply · n cancel"))
	return b.String()
}

// recordForm is a huh-backed add/edit record form.
type recordForm struct {
	form      *huh.Form
	baseIndex int // -1 for add
	name      string
	typ       string
	value     string
	ttl       string
	mxPref    string
}

func newRecordForm(existing *dns.HostRecord) *recordForm {
	f := &recordForm{baseIndex: -1, typ: string(dns.A), ttl: "0", mxPref: "10"}
	if existing != nil {
		f.name = existing.Name
		f.typ = string(existing.Type)
		f.value = existing.Value
		f.ttl = strconv.Itoa(existing.TTL)
		f.mxPref = strconv.Itoa(existing.MXPref)
	}
	options := make([]huh.Option[string], 0, 12)
	for _, t := range []dns.RecordType{
		dns.A, dns.AAAA, dns.ALIAS, dns.CAA, dns.CNAME, dns.MX,
		dns.MXE, dns.NS, dns.TXT, dns.URL, dns.URL301, dns.FRAME,
	} {
		options = append(options, huh.NewOption(string(t), string(t)))
	}
	f.form = huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Name (@, www, …)").Value(&f.name),
		huh.NewSelect[string]().Title("Type").Options(options...).Value(&f.typ),
		huh.NewInput().Title("Value").Value(&f.value),
		huh.NewInput().Title("TTL (0 = auto)").Value(&f.ttl),
		huh.NewInput().Title("MX preference").Value(&f.mxPref),
	))
	return f
}

func (f *recordForm) record() (dns.HostRecord, error) {
	rt, err := dns.ParseRecordType(f.typ)
	if err != nil {
		return dns.HostRecord{}, err
	}
	ttl, err := strconv.Atoi(strings.TrimSpace(f.ttl))
	if err != nil {
		return dns.HostRecord{}, fmt.Errorf("invalid ttl %q", f.ttl)
	}
	pref, err := strconv.Atoi(strings.TrimSpace(f.mxPref))
	if err != nil {
		return dns.HostRecord{}, fmt.Errorf("invalid mx preference %q", f.mxPref)
	}
	rec := dns.HostRecord{Name: f.name, Type: rt, Value: f.value, TTL: ttl, MXPref: pref}
	return rec, rec.Validate()
}
