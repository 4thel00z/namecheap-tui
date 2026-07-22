package namecheap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

func serveFixtures(t *testing.T, pick func(r *http.Request) string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := os.ReadFile("testdata/" + pick(r))
		if err != nil {
			t.Errorf("fixture: %v", err)
			return
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
}

func TestListDomainsMergesPages(t *testing.T) {
	c := serveFixtures(t, func(r *http.Request) string {
		if r.URL.Query().Get("Page") == "2" {
			return "domains_getlist_p2.xml"
		}
		return "domains_getlist_p1.xml"
	})
	got, err := c.ListDomains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (paging merge)", len(got))
	}
	a := got[0]
	if a.Name.String() != "alpha.com" || !a.AutoRenew || a.Locked || !a.Privacy {
		t.Errorf("alpha.com parsed wrong: %+v", a)
	}
	if a.Expires != time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC) {
		t.Errorf("expires = %v", a.Expires)
	}
	if got[1].Name.TLD != "co.uk" {
		t.Errorf("multi-label TLD lost: %+v", got[1].Name)
	}
	if got[2].ID != "3" {
		t.Errorf("page 2 domain missing: %+v", got[2])
	}
}

func TestCheckDomains(t *testing.T) {
	c := serveFixtures(t, func(*http.Request) string { return "domains_check.xml" })
	names := []registrar.DomainName{
		{SLD: "free", TLD: "com"}, {SLD: "taken", TLD: "com"}, {SLD: "fancy", TLD: "io"},
	}
	got, err := c.CheckDomains(context.Background(), names)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if !got[0].Available || got[1].Available {
		t.Errorf("availability wrong: %+v", got[:2])
	}
	if !got[2].Premium || got[2].PremiumPrice != 1200.5 {
		t.Errorf("premium wrong: %+v", got[2])
	}
}

func TestDomainInfo(t *testing.T) {
	c := serveFixtures(t, func(*http.Request) string { return "domains_getinfo.xml" })
	got, err := c.DomainInfo(context.Background(), registrar.DomainName{SLD: "alpha", TLD: "com"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "42" || got.Status != "Ok" || !got.Privacy {
		t.Errorf("details wrong: %+v", got)
	}
	if got.DNSProvider != "CUSTOM" || len(got.Nameservers) != 2 || got.Nameservers[0] != "dns1.example.net" {
		t.Errorf("dns details wrong: %+v", got)
	}
	if got.Created != time.Date(2020, 2, 15, 0, 0, 0, 0, time.UTC) {
		t.Errorf("created = %v", got.Created)
	}
}
