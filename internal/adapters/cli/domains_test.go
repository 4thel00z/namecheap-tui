package cli_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type memRegistrar struct {
	ports.RegistrarAPI // unimplemented methods panic if called
	domains            []registrar.Domain
}

func (f *memRegistrar) ListDomains(context.Context) ([]registrar.Domain, error) {
	return f.domains, nil
}

func (f *memRegistrar) CheckDomains(_ context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	out := make([]registrar.Availability, len(names))
	for i, n := range names {
		out[i] = registrar.Availability{Name: n, Available: n.SLD == "free"}
	}
	return out, nil
}

func (f *memRegistrar) DomainInfo(_ context.Context, name registrar.DomainName) (registrar.Details, error) {
	return registrar.Details{
		Domain: registrar.Domain{Name: name, Expires: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		Status: "Ok", DNSProvider: "CUSTOM",
		Nameservers: []string{"ns1.example.net"},
	}, nil
}

type noCache struct{}

func (noCache) Get(context.Context, string, string, string, time.Duration) ([]byte, bool, error) {
	return nil, false, nil
}
func (noCache) Put(context.Context, string, string, string, []byte) error { return nil }
func (noCache) Invalidate(context.Context, string, string) error          { return nil }

func domainsApp(reg *memRegistrar) *cli.App {
	app := testApp(newMemRepo())
	app.Domains = func(context.Context, string, bool) (*services.DomainService, error) {
		return services.NewDomainService(reg, noCache{}, "t"), nil
	}
	return app
}

func TestDomainsList(t *testing.T) {
	n, _ := registrar.Parse("alpha.com")
	reg := &memRegistrar{domains: []registrar.Domain{{
		Name: n, Expires: time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC), AutoRenew: true,
	}}}
	out, err := run(t, domainsApp(reg), "domains", "list")
	if err != nil || !strings.Contains(out, "alpha.com") {
		t.Errorf("output %q, %v", out, err)
	}
	out, err = run(t, domainsApp(reg), "domains", "list", "--json")
	if err != nil || !strings.Contains(out, `"name": "alpha.com"`) {
		t.Errorf("json output %q, %v", out, err)
	}
}

func TestDomainsCheck(t *testing.T) {
	out, err := run(t, domainsApp(&memRegistrar{}), "domains", "check", "free.com", "taken.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "free.com") || !strings.Contains(out, "taken.com") {
		t.Errorf("output %q", out)
	}
}

func TestDomainsInfo(t *testing.T) {
	out, err := run(t, domainsApp(&memRegistrar{}), "domains", "info", "alpha.com", "--json")
	if err != nil || !strings.Contains(out, `"dns_provider": "CUSTOM"`) {
		t.Errorf("output %q, %v", out, err)
	}
}
