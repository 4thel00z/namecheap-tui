package namecheap

import (
	"context"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
)

func TestListCertificates(t *testing.T) {
	c, _ := serveFixtureCapture(t, "ssl_list.xml")
	got, err := c.ListCertificates(context.Background())
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	if got[0].ID != "52556" || got[0].Host != "alpha.com" || got[0].Status != "active" {
		t.Errorf("cert wrong: %+v", got[0])
	}
}

func TestPurchaseCertificate(t *testing.T) {
	c, q := serveFixtureCapture(t, "ssl_create.xml")
	got, err := c.PurchaseCertificate(context.Background(), ssl.Purchase{Type: "PositiveSSL", Years: 1})
	if err != nil || got.ID != "52557" {
		t.Fatalf("%+v, %v", got, err)
	}
	if (*q).Get("Type") != "PositiveSSL" || (*q).Get("Years") != "1" {
		t.Errorf("params: %v", *q)
	}
}

func TestActivateCertificateDVMethods(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_ok.xml")
	err := c.ActivateCertificate(context.Background(), ssl.Activation{
		CertificateID: "52557", CSR: "-----BEGIN CERTIFICATE REQUEST-----",
		DVMethod: ssl.DVDNS, WebServerType: "nginx",
	})
	if err != nil {
		t.Fatal(err)
	}
	if (*q).Get("DNSDCValidation") != "true" || (*q).Get("WebServerType") != "nginx" {
		t.Errorf("params: %v", *q)
	}
	// email DV requires approver
	err = c.ActivateCertificate(context.Background(), ssl.Activation{
		CertificateID: "52557", CSR: "x", DVMethod: ssl.DVEmail,
	})
	if err == nil {
		t.Error("email DV without approver accepted")
	}
}

func TestApproverEmails(t *testing.T) {
	c, _ := serveFixtureCapture(t, "ssl_approvers.xml")
	got, err := c.ApproverEmails(context.Background(), "alpha.com", "PositiveSSL")
	if err != nil || len(got) != 3 || got[0] != "admin@alpha.com" {
		t.Errorf("%v, %v", got, err)
	}
}

func TestListPrivacy(t *testing.T) {
	c, _ := serveFixtureCapture(t, "wg_list.xml")
	got, err := c.ListPrivacy(context.Background())
	if err != nil || len(got) != 2 {
		t.Fatalf("%v, %v", got, err)
	}
	if got[0].ID != "7578" || got[0].Domain != "alpha.com" || got[0].Status != "ENABLED" {
		t.Errorf("subscription wrong: %+v", got[0])
	}
}

func TestPrivacyParams(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_ok.xml")
	if err := c.AssignPrivacy(context.Background(), "7579", "beta.com"); err != nil {
		t.Fatal(err)
	}
	if (*q).Get("WhoisguardID") != "7579" || (*q).Get("DomainName") != "beta.com" {
		t.Errorf("params: %v", *q)
	}
}

func TestBalances(t *testing.T) {
	c, _ := serveFixtureCapture(t, "users_balances.xml")
	got, err := c.Balances(context.Background())
	if err != nil || got.Currency != "USD" || got.Available != 4932.96 {
		t.Errorf("%+v, %v", got, err)
	}
}

func TestPricing(t *testing.T) {
	c, q := serveFixtureCapture(t, "users_pricing.xml")
	got, err := c.Pricing(context.Background(), "DOMAIN", "REGISTER", "com")
	if err != nil || len(got) != 2 {
		t.Fatalf("%v, %v", got, err)
	}
	if got[0].Product != "com" || got[0].Yours != 10.87 || got[0].Duration != 1 {
		t.Errorf("price wrong: %+v", got[0])
	}
	if (*q).Get("ProductType") != "DOMAIN" || (*q).Get("ProductName") != "com" {
		t.Errorf("params: %v", *q)
	}
}

func TestAddresses(t *testing.T) {
	c, _ := serveFixtureCapture(t, "address_list.xml")
	got, err := c.ListAddresses(context.Background())
	if err != nil || len(got) != 2 || !got[0].Default || got[1].Name != "Office" {
		t.Errorf("%+v, %v", got, err)
	}

	c2, _ := serveFixtureCapture(t, "address_info.xml")
	addr, err := c2.GetAddress(context.Background(), "18827")
	if err != nil || addr.Name != "Home" || addr.Contact.FirstName != "Ada" || addr.Contact.PostalCode != "E1 6AN" {
		t.Errorf("%+v, %v", addr, err)
	}

	c3, q := serveFixtureCapture(t, "dns_ok.xml")
	err = c3.CreateAddress(context.Background(), account.Address{
		Name: "Home", Contact: testContact(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if (*q).Get("AddressName") != "Home" || (*q).Get("Zip") != "E1 6AN" || (*q).Get("EmailAddress") != "ada@example.org" {
		t.Errorf("params: %v", *q)
	}
}
