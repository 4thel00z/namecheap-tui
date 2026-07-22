package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type lifecycleAPI struct {
	ports.RegistrarAPI
	registered *registrar.Registration
	renewed    int
	locked     *bool
	contacts   *registrar.ContactSet
}

func (f *lifecycleAPI) RegisterDomain(_ context.Context, reg registrar.Registration) (registrar.RegistrationResult, error) {
	f.registered = &reg
	return registrar.RegistrationResult{Domain: reg.Name.String(), Registered: true, ChargedAmount: 10.87, OrderID: "196074"}, nil
}

func (f *lifecycleAPI) RenewDomain(_ context.Context, _ registrar.DomainName, years int) (registrar.RenewalResult, error) {
	f.renewed = years
	return registrar.RenewalResult{ChargedAmount: 14.2, Expires: time.Date(2028, 2, 15, 0, 0, 0, 0, time.UTC)}, nil
}

func (f *lifecycleAPI) ReactivateDomain(context.Context, registrar.DomainName) error { return nil }

func (f *lifecycleAPI) GetContacts(context.Context, registrar.DomainName) (registrar.ContactSet, error) {
	return registrar.UniformContacts(registrar.Contact{
		FirstName: "Ada", LastName: "Lovelace", Address1: "1 Analytical Way",
		City: "London", StateProvince: "LDN", PostalCode: "E1 6AN",
		Country: "GB", Phone: "+44.2071234567", Email: "ada@example.org",
	}), nil
}

func (f *lifecycleAPI) SetContacts(_ context.Context, _ registrar.DomainName, c registrar.ContactSet) error {
	f.contacts = &c
	return nil
}

func (f *lifecycleAPI) LockStatus(context.Context, registrar.DomainName) (bool, error) {
	return true, nil
}

func (f *lifecycleAPI) SetLock(_ context.Context, _ registrar.DomainName, locked bool) error {
	f.locked = &locked
	return nil
}

func (f *lifecycleAPI) TLDs(context.Context) ([]registrar.TLD, error) {
	return []registrar.TLD{{Name: "com", MinYears: 1, MaxYears: 10, Registerable: true}}, nil
}

type memTransfer struct{ resub string }

func (f *memTransfer) CreateTransfer(_ context.Context, name registrar.DomainName, epp string, years int) (registrar.Transfer, error) {
	return registrar.Transfer{ID: "12345", Name: name.String(), Status: "WAITINGFOREPPTRANSFER"}, nil
}

func (f *memTransfer) TransferStatus(_ context.Context, id string) (registrar.Transfer, error) {
	return registrar.Transfer{ID: id, Status: "INPROGRESS", StatusID: 5}, nil
}

func (f *memTransfer) ListTransfers(context.Context) ([]registrar.Transfer, error) {
	return []registrar.Transfer{{ID: "12345", Name: "moving.com", Status: "INPROGRESS"}}, nil
}

func (f *memTransfer) ResubmitTransfer(_ context.Context, id string) error {
	f.resub = id
	return nil
}

func lifecycleApp(api *lifecycleAPI, tr *memTransfer) *cli.App {
	app := testApp(newMemRepo())
	app.Domains = func(context.Context, string, bool) (*services.DomainService, error) {
		return services.NewDomainService(api, noCache{}, "t"), nil
	}
	app.Transfers = func(context.Context, string, bool) (*services.TransferService, error) {
		return services.NewTransferService(tr), nil
	}
	return app
}

const contactYAMLBody = `first_name: Ada
last_name: Lovelace
address1: 1 Analytical Way
city: London
state_province: LDN
postal_code: E1 6AN
country: GB
phone: "+44.2071234567"
email: ada@example.org
`

func TestDomainsRegisterWithContactsFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "contacts.yaml")
	if err := os.WriteFile(path, []byte(contactYAMLBody), 0o600); err != nil {
		t.Fatal(err)
	}
	api := &lifecycleAPI{}
	out, err := run(t, lifecycleApp(api, &memTransfer{}), "domains", "register", "newdomain.com",
		"--contacts-file", path, "--years", "2")
	if err != nil {
		t.Fatal(err)
	}
	if api.registered == nil || api.registered.Years != 2 || !api.registered.Privacy {
		t.Fatalf("registration: %+v", api.registered)
	}
	if api.registered.Contacts.Tech.FirstName != "Ada" {
		t.Error("uniform contact not applied to tech role")
	}
	if !strings.Contains(out, "10.87") {
		t.Errorf("output %q", out)
	}
}

func TestDomainsRenewAndLock(t *testing.T) {
	api := &lifecycleAPI{}
	app := lifecycleApp(api, &memTransfer{})

	out, err := run(t, app, "domains", "renew", "alpha.com", "--years", "3")
	if err != nil || api.renewed != 3 || !strings.Contains(out, "2028-02-15") {
		t.Errorf("renew: %v, years=%d, out=%q", err, api.renewed, out)
	}

	out, err = run(t, app, "domains", "lock", "get", "alpha.com")
	if err != nil || !strings.Contains(out, "locked") {
		t.Errorf("lock get: %q, %v", out, err)
	}
	if _, err := run(t, app, "domains", "lock", "off", "alpha.com"); err != nil {
		t.Fatal(err)
	}
	if api.locked == nil || *api.locked {
		t.Error("lock off not passed through")
	}
}

func TestDomainsContactsRoundtrip(t *testing.T) {
	api := &lifecycleAPI{}
	app := lifecycleApp(api, &memTransfer{})

	out, err := run(t, app, "domains", "contacts", "get", "alpha.com")
	if err != nil || !strings.Contains(out, "Ada") {
		t.Fatalf("contacts get: %q, %v", out, err)
	}

	path := filepath.Join(t.TempDir(), "contacts.yaml")
	if err := os.WriteFile(path, []byte(out), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, app, "domains", "contacts", "set", "alpha.com", "-f", path); err != nil {
		t.Fatal(err)
	}
	if api.contacts == nil || api.contacts.Registrant.FirstName != "Ada" {
		t.Errorf("contacts set: %+v", api.contacts)
	}
}

func TestDomainsTlds(t *testing.T) {
	out, err := run(t, lifecycleApp(&lifecycleAPI{}, &memTransfer{}), "domains", "tlds", "--json")
	if err != nil || !strings.Contains(out, `"name": "com"`) {
		t.Errorf("tlds: %q, %v", out, err)
	}
}

func TestTransferCommands(t *testing.T) {
	tr := &memTransfer{}
	app := lifecycleApp(&lifecycleAPI{}, tr)

	out, err := run(t, app, "transfer", "create", "moving.com", "EPP123")
	if err != nil || !strings.Contains(out, "12345") {
		t.Errorf("create: %q, %v", out, err)
	}
	out, err = run(t, app, "transfer", "list")
	if err != nil || !strings.Contains(out, "moving.com") {
		t.Errorf("list: %q, %v", out, err)
	}
	out, err = run(t, app, "transfer", "status", "12345", "--resubmit")
	if err != nil || tr.resub != "12345" || !strings.Contains(out, "INPROGRESS") {
		t.Errorf("status: %q, %v, resub=%q", out, err, tr.resub)
	}
}
