package tui_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/4thel00z/namecheap-tui/internal/adapters/tui"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type fakeRegistrar struct {
	ports.RegistrarAPI // unimplemented methods panic if called
}

func (fakeRegistrar) ListDomains(context.Context) ([]registrar.Domain, error) {
	n, _ := registrar.Parse("alpha.com")
	return []registrar.Domain{{
		Name: n, Expires: time.Now().AddDate(1, 0, 0), AutoRenew: true, Privacy: true,
	}}, nil
}

func (fakeRegistrar) CheckDomains(_ context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	return nil, nil
}

func (fakeRegistrar) DomainInfo(_ context.Context, name registrar.DomainName) (registrar.Details, error) {
	return registrar.Details{}, nil
}

type nilCache struct{}

func (nilCache) Get(context.Context, string, string, string, time.Duration) ([]byte, bool, error) {
	return nil, false, nil
}
func (nilCache) Put(context.Context, string, string, string, []byte) error { return nil }
func (nilCache) Invalidate(context.Context, string, string) error          { return nil }

func TestDashboardShowsDomains(t *testing.T) {
	svc := services.NewDomainService(fakeRegistrar{}, nilCache{}, "test")
	m := tui.NewDashboard(tui.DashboardDeps{Domains: svc}, "test", "0.0.0")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("alpha.com"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
