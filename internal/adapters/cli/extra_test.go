package cli_test

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type memSSL struct {
	activated *ssl.Activation
	revoked   string
}

func (f *memSSL) ListCertificates(context.Context) ([]ssl.Certificate, error) {
	return []ssl.Certificate{{
		ID: "52556", Host: "alpha.com", Type: "PositiveSSL", Status: "active",
		Expires: time.Date(2026, 12, 16, 0, 0, 0, 0, time.UTC),
	}}, nil
}

func (f *memSSL) PurchaseCertificate(_ context.Context, p ssl.Purchase) (ssl.Certificate, error) {
	return ssl.Certificate{ID: "52557", Type: p.Type, Years: p.Years, Status: "Newpurchase"}, nil
}

func (f *memSSL) ActivateCertificate(_ context.Context, a ssl.Activation) error {
	f.activated = &a
	return nil
}

func (f *memSSL) CertificateInfo(_ context.Context, id string) (ssl.Certificate, error) {
	return ssl.Certificate{ID: id, Status: "active"}, nil
}

func (f *memSSL) RenewCertificate(_ context.Context, id string, p ssl.Purchase) (ssl.Certificate, error) {
	return ssl.Certificate{ID: id, Type: p.Type, Years: p.Years}, nil
}

func (f *memSSL) ReissueCertificate(_ context.Context, a ssl.Activation) error {
	f.activated = &a
	return nil
}

func (f *memSSL) ApproverEmails(context.Context, string, string) ([]string, error) {
	return []string{"admin@alpha.com"}, nil
}

func (f *memSSL) ResendApproverEmail(context.Context, string) error { return nil }

func (f *memSSL) RevokeCertificate(_ context.Context, id, _ string) error {
	f.revoked = id
	return nil
}

type memPrivacy struct{ assigned string }

func (f *memPrivacy) ListPrivacy(context.Context) ([]account.PrivacySubscription, error) {
	return []account.PrivacySubscription{{ID: "7578", Domain: "alpha.com", Status: "ENABLED"}}, nil
}
func (f *memPrivacy) EnablePrivacy(context.Context, string, string) error { return nil }
func (f *memPrivacy) DisablePrivacy(context.Context, string) error        { return nil }
func (f *memPrivacy) RenewPrivacy(context.Context, string, int) error     { return nil }
func (f *memPrivacy) ChangePrivacyEmail(context.Context, string) error    { return nil }

func (f *memPrivacy) AssignPrivacy(_ context.Context, id, domain string) error {
	f.assigned = id + "=" + domain
	return nil
}
func (f *memPrivacy) UnassignPrivacy(context.Context, string) error { return nil }
func (f *memPrivacy) DiscardPrivacy(context.Context, string) error  { return nil }

type memAccount struct{ defaulted string }

func (f *memAccount) Balances(context.Context) (account.Balance, error) {
	return account.Balance{Currency: "USD", Available: 4932.96, Total: 4932.96}, nil
}

func (f *memAccount) Pricing(context.Context, string, string, string) ([]account.Price, error) {
	return []account.Price{{Product: "com", Category: "register", Duration: 1, DurationType: "YEAR", Regular: 12.98, Yours: 10.87, Currency: "USD"}}, nil
}

func (f *memAccount) ListAddresses(context.Context) ([]account.Address, error) {
	return []account.Address{{ID: "18827", Name: "Home", Default: true}}, nil
}

func (f *memAccount) GetAddress(_ context.Context, id string) (account.Address, error) {
	return account.Address{ID: id, Name: "Home"}, nil
}

func (f *memAccount) CreateAddress(context.Context, account.Address) error { return nil }
func (f *memAccount) UpdateAddress(context.Context, account.Address) error { return nil }
func (f *memAccount) DeleteAddress(context.Context, string) error          { return nil }

func (f *memAccount) SetDefaultAddress(_ context.Context, id string) error {
	f.defaulted = id
	return nil
}

func extraApp(sslAPI *memSSL, priv *memPrivacy, acct *memAccount) *cli.App {
	app := testApp(newMemRepo())
	app.SSL = func(context.Context, string, bool) (*services.SSLService, error) {
		return services.NewSSLService(sslAPI), nil
	}
	app.Privacy = func(context.Context, string, bool) (*services.PrivacyService, error) {
		return services.NewPrivacyService(priv), nil
	}
	app.Account = func(context.Context, string, bool) (*services.AccountService, error) {
		return services.NewAccountService(acct, noCache{}, "t"), nil
	}
	return app
}

func TestSSLCommands(t *testing.T) {
	sslAPI := &memSSL{}
	app := extraApp(sslAPI, &memPrivacy{}, &memAccount{})

	out, err := run(t, app, "ssl", "list")
	if err != nil || !strings.Contains(out, "alpha.com") {
		t.Errorf("list: %q, %v", out, err)
	}

	csr := filepath.Join(t.TempDir(), "req.csr")
	if err := os.WriteFile(csr, []byte("-----BEGIN CERTIFICATE REQUEST-----"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := run(t, app, "ssl", "activate", "52557", "--csr-file", csr, "--dv", "dns"); err != nil {
		t.Fatal(err)
	}
	if sslAPI.activated == nil || sslAPI.activated.DVMethod != ssl.DVDNS {
		t.Errorf("activation: %+v", sslAPI.activated)
	}
	// email DV without approver fails validation
	if _, err := run(t, app, "ssl", "activate", "52557", "--csr-file", csr); err == nil {
		t.Error("email DV without approver accepted")
	}
	if _, err := run(t, app, "ssl", "revoke", "52556", "PositiveSSL"); err != nil || sslAPI.revoked != "52556" {
		t.Errorf("revoke: %v, %q", err, sslAPI.revoked)
	}
}

func TestPrivacyCommands(t *testing.T) {
	priv := &memPrivacy{}
	app := extraApp(&memSSL{}, priv, &memAccount{})

	out, err := run(t, app, "privacy", "list", "--json")
	if err != nil || !strings.Contains(out, `"id": "7578"`) {
		t.Errorf("list: %q, %v", out, err)
	}
	if _, err := run(t, app, "privacy", "assign", "7579", "beta.com"); err != nil || priv.assigned != "7579=beta.com" {
		t.Errorf("assign: %v, %q", err, priv.assigned)
	}
}

func TestAccountCommands(t *testing.T) {
	acct := &memAccount{}
	app := extraApp(&memSSL{}, &memPrivacy{}, acct)

	out, err := run(t, app, "account", "balance")
	if err != nil || !strings.Contains(out, "4932.96") {
		t.Errorf("balance: %q, %v", out, err)
	}
	out, err = run(t, app, "account", "pricing", "DOMAIN", "--product", "com")
	if err != nil || !strings.Contains(out, "10.87") {
		t.Errorf("pricing: %q, %v", out, err)
	}
	out, err = run(t, app, "address", "list")
	if err != nil || !strings.Contains(out, "Home") {
		t.Errorf("address list: %q, %v", out, err)
	}
	if _, err := run(t, app, "address", "default", "18828"); err != nil || acct.defaulted != "18828" {
		t.Errorf("address default: %v, %q", err, acct.defaulted)
	}
}
