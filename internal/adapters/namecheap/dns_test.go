package namecheap

import (
	"context"
	"net/http"
	"net/url"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

var alpha = registrar.DomainName{SLD: "alpha", TLD: "com"}

// serveFixtureCapture serves one fixture and captures the request query.
func serveFixtureCapture(t *testing.T, fixture string) (*Client, *url.Values) {
	t.Helper()
	var captured url.Values
	c := serveFixtures(t, func(r *http.Request) string {
		captured = r.URL.Query()
		return fixture
	})
	return c, &captured
}

func TestGetHosts(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_gethosts.xml")
	got, err := c.GetHosts(context.Background(), alpha)
	if err != nil {
		t.Fatal(err)
	}
	if (*q).Get("SLD") != "alpha" || (*q).Get("TLD") != "com" {
		t.Errorf("SLD/TLD params wrong: %v", *q)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if got[0].Name != "@" || got[0].Type != dns.A || got[0].Value != "1.2.3.4" || got[0].TTL != 1800 {
		t.Errorf("record 0 wrong: %+v", got[0])
	}
	if got[2].Type != dns.MX || got[2].MXPref != 20 {
		t.Errorf("MX record wrong: %+v", got[2])
	}
}

func TestSetHostsSendsIndexedParams(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_sethosts.xml")
	records := []dns.HostRecord{
		{Name: "@", Type: dns.A, Value: "1.2.3.4", TTL: 1800},
		{Name: "@", Type: dns.MX, Value: "mail.alpha.com", MXPref: 20},
	}
	if err := c.SetHosts(context.Background(), alpha, records); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"SLD": "alpha", "TLD": "com",
		"HostName1": "@", "RecordType1": "A", "Address1": "1.2.3.4", "TTL1": "1800",
		"HostName2": "@", "RecordType2": "MX", "Address2": "mail.alpha.com", "MXPref2": "20",
		"EmailType": "MX",
	} {
		if got := (*q).Get(k); got != want {
			t.Errorf("param %s = %q, want %q", k, got, want)
		}
	}
	// Default TTL fills in when unset.
	if got := (*q).Get("TTL2"); got != "1799" {
		t.Errorf("TTL2 = %q, want 1799", got)
	}
}

func TestGetNameservers(t *testing.T) {
	c, _ := serveFixtureCapture(t, "dns_getlist.xml")
	got, err := c.GetNameservers(context.Background(), alpha)
	if err != nil {
		t.Fatal(err)
	}
	if got.UsingOurDNS || len(got.Nameservers) != 2 || got.Nameservers[0] != "dns1.example.net" {
		t.Errorf("nameserver info wrong: %+v", got)
	}
}

func TestSetCustomNS(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_ok.xml")
	if err := c.SetCustomNS(context.Background(), alpha, []string{"ns1.x.com", "ns2.x.com"}); err != nil {
		t.Fatal(err)
	}
	if got := (*q).Get("Nameservers"); got != "ns1.x.com,ns2.x.com" {
		t.Errorf("Nameservers = %q", got)
	}
}

func TestSetDefaultNS(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_ok.xml")
	if err := c.SetDefaultNS(context.Background(), alpha); err != nil {
		t.Fatal(err)
	}
	if got := (*q).Get("Command"); got != "namecheap.domains.dns.setDefault" {
		t.Errorf("Command = %q", got)
	}
}

func TestEmailForwarding(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_getemailfwd.xml")
	got, err := c.GetEmailForwarding(context.Background(), alpha)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Mailbox != "info" || got[0].ForwardTo != "team@example.org" {
		t.Errorf("forwards wrong: %+v", got)
	}
	if (*q).Get("DomainName") != "alpha.com" {
		t.Errorf("DomainName param missing: %v", *q)
	}

	c2, q2 := serveFixtureCapture(t, "dns_ok.xml")
	err = c2.SetEmailForwarding(context.Background(), alpha, []dns.EmailForward{
		{Mailbox: "info", ForwardTo: "team@example.org"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if (*q2).Get("MailBox1") != "info" || (*q2).Get("ForwardTo1") != "team@example.org" {
		t.Errorf("set params wrong: %v", *q2)
	}
}

func TestNSLifecycle(t *testing.T) {
	c, q := serveFixtureCapture(t, "dns_ok.xml")
	if err := c.CreateNS(context.Background(), alpha, "ns1.alpha.com", "203.0.113.10"); err != nil {
		t.Fatal(err)
	}
	if (*q).Get("Nameserver") != "ns1.alpha.com" || (*q).Get("IP") != "203.0.113.10" {
		t.Errorf("create params wrong: %v", *q)
	}

	c2, q2 := serveFixtureCapture(t, "dns_ok.xml")
	if err := c2.UpdateNS(context.Background(), alpha, "ns1.alpha.com", "203.0.113.10", "203.0.113.11"); err != nil {
		t.Fatal(err)
	}
	if (*q2).Get("OldIP") != "203.0.113.10" || (*q2).Get("IP") != "203.0.113.11" {
		t.Errorf("update params wrong: %v", *q2)
	}

	c3, q3 := serveFixtureCapture(t, "ns_getinfo.xml")
	info, err := c3.NSInfo(context.Background(), alpha, "ns1.alpha.com")
	if err != nil {
		t.Fatal(err)
	}
	if info.IP != "203.0.113.10" || len(info.Statuses) != 2 || info.Statuses[0] != "OK" {
		t.Errorf("ns info wrong: %+v", info)
	}
	_ = q3

	c4, q4 := serveFixtureCapture(t, "dns_ok.xml")
	if err := c4.DeleteNS(context.Background(), alpha, "ns1.alpha.com"); err != nil {
		t.Fatal(err)
	}
	if (*q4).Get("Command") != "namecheap.domains.ns.delete" {
		t.Errorf("delete command wrong: %v", *q4)
	}
}
