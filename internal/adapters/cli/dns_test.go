package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type memDNS struct {
	records  []dns.HostRecord
	custom   []string
	forwards []dns.EmailForward
}

func (f *memDNS) GetHosts(context.Context, registrar.DomainName) ([]dns.HostRecord, error) {
	out := make([]dns.HostRecord, len(f.records))
	copy(out, f.records)
	return out, nil
}

func (f *memDNS) SetHosts(_ context.Context, _ registrar.DomainName, records []dns.HostRecord) error {
	f.records = records
	return nil
}

func (f *memDNS) GetNameservers(context.Context, registrar.DomainName) (dns.NameserverInfo, error) {
	return dns.NameserverInfo{UsingOurDNS: true}, nil
}

func (f *memDNS) SetDefaultNS(context.Context, registrar.DomainName) error { return nil }

func (f *memDNS) SetCustomNS(_ context.Context, _ registrar.DomainName, ns []string) error {
	f.custom = ns
	return nil
}

func (f *memDNS) GetEmailForwarding(context.Context, registrar.DomainName) ([]dns.EmailForward, error) {
	return f.forwards, nil
}

func (f *memDNS) SetEmailForwarding(_ context.Context, _ registrar.DomainName, fwds []dns.EmailForward) error {
	f.forwards = fwds
	return nil
}

type memNS struct{ created, deleted string }

func (f *memNS) CreateNS(_ context.Context, _ registrar.DomainName, host, ip string) error {
	f.created = host + "=" + ip
	return nil
}

func (f *memNS) UpdateNS(context.Context, registrar.DomainName, string, string, string) error {
	return nil
}

func (f *memNS) DeleteNS(_ context.Context, _ registrar.DomainName, host string) error {
	f.deleted = host
	return nil
}

func (f *memNS) NSInfo(_ context.Context, _ registrar.DomainName, host string) (dns.RegisteredNS, error) {
	return dns.RegisteredNS{Host: host, IP: "203.0.113.10", Statuses: []string{"OK"}}, nil
}

func dnsApp(api *memDNS, nsAPI *memNS) *cli.App {
	app := testApp(newMemRepo())
	app.DNS = func(context.Context, string, bool) (*services.DNSService, error) {
		return services.NewDNSService(api, noCache{}, "t"), nil
	}
	app.NS = func(context.Context, string, bool) (*services.NSService, error) {
		return services.NewNSService(nsAPI), nil
	}
	return app
}

func record(name, typ, value string) dns.HostRecord {
	return dns.HostRecord{Name: name, Type: dns.RecordType(typ), Value: value, TTL: 1799}
}

func TestDNSGet(t *testing.T) {
	api := &memDNS{records: []dns.HostRecord{record("@", "A", "1.2.3.4")}}
	out, err := run(t, dnsApp(api, &memNS{}), "dns", "get", "alpha.com")
	if err != nil || !strings.Contains(out, "1.2.3.4") {
		t.Errorf("output %q, %v", out, err)
	}
	out, err = run(t, dnsApp(api, &memNS{}), "dns", "get", "alpha.com", "--json")
	if err != nil || !strings.Contains(out, `"value": "1.2.3.4"`) {
		t.Errorf("json output %q, %v", out, err)
	}
}

func TestDNSAddAndRm(t *testing.T) {
	api := &memDNS{records: []dns.HostRecord{record("@", "A", "1.2.3.4")}}
	app := dnsApp(api, &memNS{})

	if _, err := run(t, app, "dns", "add", "alpha.com", "TXT", "@", "v=spf1 -all"); err != nil {
		t.Fatal(err)
	}
	if len(api.records) != 2 {
		t.Fatalf("add failed: %+v", api.records)
	}
	if _, err := run(t, app, "dns", "rm", "alpha.com", "TXT", "@"); err != nil {
		t.Fatal(err)
	}
	if len(api.records) != 1 {
		t.Errorf("rm failed: %+v", api.records)
	}
}

func TestDNSExportImportRoundtrip(t *testing.T) {
	api := &memDNS{records: []dns.HostRecord{
		record("@", "A", "1.2.3.4"),
		{Name: "@", Type: dns.MX, Value: "mail.alpha.com", TTL: 1799, MXPref: 10},
	}}
	app := dnsApp(api, &memNS{})

	out, err := run(t, app, "dns", "export", "alpha.com")
	if err != nil || !strings.Contains(out, "mail.alpha.com") {
		t.Fatalf("export: %q, %v", out, err)
	}

	// Import the exported YAML into an empty zone.
	path := filepath.Join(t.TempDir(), "zone.yaml")
	if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	api2 := &memDNS{}
	app2 := dnsApp(api2, &memNS{})
	if _, err := run(t, app2, "dns", "import", "alpha.com", "-f", path); err != nil {
		t.Fatal(err)
	}
	if len(api2.records) != 2 || api2.records[1].MXPref != 10 {
		t.Errorf("roundtrip lost data: %+v", api2.records)
	}
}

func TestDNSUseCustom(t *testing.T) {
	api := &memDNS{}
	if _, err := run(t, dnsApp(api, &memNS{}), "dns", "use-custom", "alpha.com", "ns1.x.com", "ns2.x.com"); err != nil {
		t.Fatal(err)
	}
	if len(api.custom) != 2 {
		t.Errorf("custom = %v", api.custom)
	}
}

func TestDNSEmailFwd(t *testing.T) {
	api := &memDNS{}
	app := dnsApp(api, &memNS{})
	if _, err := run(t, app, "dns", "emailfwd", "set", "alpha.com", "info=team@x.org"); err != nil {
		t.Fatal(err)
	}
	if len(api.forwards) != 1 || api.forwards[0].Mailbox != "info" {
		t.Fatalf("set failed: %+v", api.forwards)
	}
	out, err := run(t, app, "dns", "emailfwd", "get", "alpha.com")
	if err != nil || !strings.Contains(out, "team@x.org") {
		t.Errorf("get output %q, %v", out, err)
	}
	if _, err := run(t, app, "dns", "emailfwd", "set", "alpha.com", "bogus"); err == nil {
		t.Error("invalid forward spec accepted")
	}
}

func TestNSCommands(t *testing.T) {
	nsAPI := &memNS{}
	app := dnsApp(&memDNS{}, nsAPI)
	if _, err := run(t, app, "ns", "create", "alpha.com", "ns1.alpha.com", "203.0.113.10"); err != nil {
		t.Fatal(err)
	}
	if nsAPI.created != "ns1.alpha.com=203.0.113.10" {
		t.Errorf("created = %q", nsAPI.created)
	}
	out, err := run(t, app, "ns", "info", "alpha.com", "ns1.alpha.com", "--json")
	if err != nil || !strings.Contains(out, `"ip": "203.0.113.10"`) {
		t.Errorf("info output %q, %v", out, err)
	}
	if _, err := run(t, app, "ns", "delete", "alpha.com", "ns1.alpha.com"); err != nil {
		t.Fatal(err)
	}
	if nsAPI.deleted != "ns1.alpha.com" {
		t.Errorf("deleted = %q", nsAPI.deleted)
	}
}
