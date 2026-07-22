package tui_test

import (
	"bytes"
	"context"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/4thel00z/namecheap-tui/internal/adapters/tui"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type editableDNS struct {
	mu      sync.Mutex
	records []dns.HostRecord
	sets    int
}

func (f *editableDNS) GetHosts(context.Context, registrar.DomainName) ([]dns.HostRecord, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]dns.HostRecord, len(f.records))
	copy(out, f.records)
	return out, nil
}

func (f *editableDNS) SetHosts(_ context.Context, _ registrar.DomainName, records []dns.HostRecord) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.records = records
	f.sets++
	return nil
}

func (f *editableDNS) GetNameservers(context.Context, registrar.DomainName) (dns.NameserverInfo, error) {
	return dns.NameserverInfo{}, nil
}
func (f *editableDNS) SetDefaultNS(context.Context, registrar.DomainName) error { return nil }
func (f *editableDNS) SetCustomNS(context.Context, registrar.DomainName, []string) error {
	return nil
}

func (f *editableDNS) GetEmailForwarding(context.Context, registrar.DomainName) ([]dns.EmailForward, error) {
	return nil, nil
}

func (f *editableDNS) SetEmailForwarding(context.Context, registrar.DomainName, []dns.EmailForward) error {
	return nil
}

func (f *editableDNS) snapshot() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.records), f.sets
}

func TestZoneEditorStageDeleteAndApply(t *testing.T) {
	api := &editableDNS{records: []dns.HostRecord{
		{Name: "@", Type: dns.A, Value: "1.2.3.4", TTL: 1799},
		{Name: "www", Type: dns.A, Value: "1.2.3.5", TTL: 1799},
	}}
	svc := services.NewDNSService(api, nilCache{}, "test")
	m := tui.NewZoneEditor(svc, "alpha.com")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(120, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("1.2.3.4"))
	}, teatest.WithDuration(3*time.Second))

	// Stage a delete on the first row, apply, confirm.
	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("-1")) // "staged: +0 ~0 -1"
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("apply alpha.com?"))
	}, teatest.WithDuration(2*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("applied"))
	}, teatest.WithDuration(3*time.Second))

	count, sets := api.snapshot()
	if sets != 1 || count != 1 {
		t.Errorf("after apply: records=%d sets=%d, want 1/1", count, sets)
	}

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
