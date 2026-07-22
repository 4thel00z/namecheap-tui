package namecheap

import (
	"context"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

func testContact() registrar.Contact {
	return registrar.Contact{
		FirstName: "Ada", LastName: "Lovelace", Address1: "1 Analytical Way",
		City: "London", StateProvince: "LDN", PostalCode: "E1 6AN",
		Country: "GB", Phone: "+44.2071234567", Email: "ada@example.org",
	}
}

func TestRegisterDomain(t *testing.T) {
	c, q := serveFixtureCapture(t, "domains_create.xml")
	name, _ := registrar.Parse("newdomain.com")
	got, err := c.RegisterDomain(context.Background(), registrar.Registration{
		Name: name, Years: 2,
		Contacts: registrar.UniformContacts(testContact()),
		Privacy:  true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !got.Registered || got.ChargedAmount != 10.87 || got.OrderID != "196074" {
		t.Errorf("result wrong: %+v", got)
	}
	for k, want := range map[string]string{
		"DomainName":           "newdomain.com",
		"Years":                "2",
		"RegistrantFirstName":  "Ada",
		"TechEmailAddress":     "ada@example.org",
		"AdminCity":            "London",
		"AuxBillingPostalCode": "E1 6AN",
		"AddFreeWhoisguard":    "yes",
		"WGEnabled":            "yes",
	} {
		if got := (*q).Get(k); got != want {
			t.Errorf("param %s = %q, want %q", k, got, want)
		}
	}
}

func TestRegisterDomainValidatesFirst(t *testing.T) {
	c := New(testCreds(), WithNoThrottle())
	name, _ := registrar.Parse("x.com")
	_, err := c.RegisterDomain(context.Background(), registrar.Registration{Name: name, Years: 1})
	if err == nil {
		t.Error("empty contacts accepted")
	}
}

func TestRenewDomain(t *testing.T) {
	c, q := serveFixtureCapture(t, "domains_renew.xml")
	name, _ := registrar.Parse("alpha.com")
	got, err := c.RenewDomain(context.Background(), name, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got.ChargedAmount != 14.20 || got.Expires != time.Date(2028, 2, 15, 0, 0, 0, 0, time.UTC) {
		t.Errorf("result wrong: %+v", got)
	}
	if (*q).Get("Years") != "1" {
		t.Errorf("Years param missing")
	}
}

func TestGetContacts(t *testing.T) {
	c, _ := serveFixtureCapture(t, "domains_getcontacts.xml")
	name, _ := registrar.Parse("alpha.com")
	got, err := c.GetContacts(context.Background(), name)
	if err != nil {
		t.Fatal(err)
	}
	if got.Registrant.FirstName != "Ada" || got.Tech.FirstName != "Grace" {
		t.Errorf("contacts wrong: %+v", got)
	}
	if got.Admin.Email != "ada@example.org" {
		t.Errorf("admin email: %q", got.Admin.Email)
	}
}

func TestSetContactsSendsAllRoles(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_ok.xml")
	name, _ := registrar.Parse("alpha.com")
	if err := c.SetContacts(context.Background(), name, registrar.UniformContacts(testContact())); err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"Registrant", "Tech", "Admin", "AuxBilling"} {
		if (*q).Get(role+"FirstName") != "Ada" {
			t.Errorf("%sFirstName missing", role)
		}
	}
}

func TestLock(t *testing.T) {
	c, _ := serveFixtureCapture(t, "domains_lock.xml")
	name, _ := registrar.Parse("alpha.com")
	locked, err := c.LockStatus(context.Background(), name)
	if err != nil || !locked {
		t.Errorf("locked=%v err=%v", locked, err)
	}

	c2, q := serveFixtureCapture(t, "dns_ok.xml")
	if err := c2.SetLock(context.Background(), name, false); err != nil {
		t.Fatal(err)
	}
	if (*q).Get("LockAction") != "UNLOCK" {
		t.Errorf("LockAction = %q", (*q).Get("LockAction"))
	}
}

func TestTLDs(t *testing.T) {
	c, _ := serveFixtureCapture(t, "domains_tldlist.xml")
	got, err := c.TLDs(context.Background())
	if err != nil || len(got) != 3 {
		t.Fatalf("%v, %v", got, err)
	}
	if got[0].Name != "com" || !got[0].Registerable || got[0].MaxYears != 10 {
		t.Errorf("com tld wrong: %+v", got[0])
	}
	if got[2].Registerable {
		t.Errorf("museum should not be registerable: %+v", got[2])
	}
}

func TestTransfers(t *testing.T) {
	c, q := serveFixtureCapture(t, "transfer_create.xml")
	name, _ := registrar.Parse("moving.com")
	tr, err := c.CreateTransfer(context.Background(), name, "EPP123", 1)
	if err != nil {
		t.Fatal(err)
	}
	if tr.ID != "12345" || tr.Status != "WAITINGFOREPPTRANSFER" {
		t.Errorf("transfer wrong: %+v", tr)
	}
	if (*q).Get("EPPCode") != "EPP123" {
		t.Errorf("EPPCode missing")
	}

	c2, _ := serveFixtureCapture(t, "transfer_list.xml")
	list, err := c2.ListTransfers(context.Background())
	if err != nil || len(list) != 1 || list[0].Name != "moving.com" || list[0].StatusID != 5 {
		t.Errorf("list wrong: %+v, %v", list, err)
	}

	c3, q3 := serveFixtureCapture(t, "dns_ok.xml")
	if err := c3.ResubmitTransfer(context.Background(), "12345"); err != nil {
		t.Fatal(err)
	}
	if (*q3).Get("Resubmit") != "true" {
		t.Errorf("Resubmit param missing")
	}
}
