package dns_test

import (
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
)

func TestParseRecordType(t *testing.T) {
	for _, ok := range []string{"A", "a", "AAAA", "cname", "MX", "TXT", "URL301", "caa", "frame"} {
		if _, err := dns.ParseRecordType(ok); err != nil {
			t.Errorf("ParseRecordType(%q): %v", ok, err)
		}
	}
	for _, bad := range []string{"", "PTR", "SOA", "bogus"} {
		if _, err := dns.ParseRecordType(bad); err == nil {
			t.Errorf("ParseRecordType(%q): want error", bad)
		}
	}
}

func rec(name, typ, value string) dns.HostRecord {
	return dns.HostRecord{Name: name, Type: dns.RecordType(typ), Value: value, TTL: 1799}
}

func TestHostRecordValidate(t *testing.T) {
	if err := rec("@", "A", "1.2.3.4").Validate(); err != nil {
		t.Errorf("valid record rejected: %v", err)
	}
	mx := rec("@", "MX", "mail.example.com")
	mx.MXPref = 10
	if err := mx.Validate(); err != nil {
		t.Errorf("valid MX rejected: %v", err)
	}
	cases := map[string]dns.HostRecord{
		"empty name":  rec("", "A", "1.2.3.4"),
		"empty value": rec("@", "A", ""),
		"bad type":    rec("@", "PTR", "x"),
		"low ttl":     {Name: "@", Type: "A", Value: "1.2.3.4", TTL: 30},
		"high ttl":    {Name: "@", Type: "A", Value: "1.2.3.4", TTL: 99999},
	}
	for name, r := range cases {
		if err := r.Validate(); err == nil {
			t.Errorf("%s: want error", name)
		}
	}
	// TTL 0 is allowed (means provider default).
	if err := (dns.HostRecord{Name: "@", Type: "A", Value: "1.2.3.4"}).Validate(); err != nil {
		t.Errorf("zero TTL rejected: %v", err)
	}
}

func TestMergeAdd(t *testing.T) {
	current := []dns.HostRecord{rec("@", "A", "1.2.3.4")}
	got, err := dns.Merge(current, dns.ChangeSet{Add: []dns.HostRecord{rec("www", "CNAME", "example.com.")}})
	if err != nil || len(got) != 2 {
		t.Fatalf("%v, %v", got, err)
	}
	// duplicate add rejected
	if _, err := dns.Merge(current, dns.ChangeSet{Add: []dns.HostRecord{rec("@", "A", "1.2.3.4")}}); err == nil {
		t.Error("duplicate add accepted")
	}
}

func TestMergeRemove(t *testing.T) {
	current := []dns.HostRecord{rec("@", "A", "1.2.3.4"), rec("www", "A", "1.2.3.5")}
	got, err := dns.Merge(current, dns.ChangeSet{Remove: []dns.HostRecord{rec("www", "A", "1.2.3.5")}})
	if err != nil || len(got) != 1 || got[0].Name != "@" {
		t.Fatalf("%v, %v", got, err)
	}
	if _, err := dns.Merge(current, dns.ChangeSet{Remove: []dns.HostRecord{rec("ghost", "A", "9.9.9.9")}}); err == nil {
		t.Error("removing nonexistent record accepted")
	}
}

func TestMergeUpdate(t *testing.T) {
	current := []dns.HostRecord{rec("@", "A", "1.2.3.4")}
	updated := rec("@", "A", "5.6.7.8")
	got, err := dns.Merge(current, dns.ChangeSet{
		Update: []dns.RecordUpdate{{Old: rec("@", "A", "1.2.3.4"), New: updated}},
	})
	if err != nil || len(got) != 1 || got[0].Value != "5.6.7.8" {
		t.Fatalf("%v, %v", got, err)
	}
	if _, err := dns.Merge(current, dns.ChangeSet{
		Update: []dns.RecordUpdate{{Old: rec("ghost", "A", "0.0.0.0"), New: updated}},
	}); err == nil {
		t.Error("updating nonexistent record accepted")
	}
}

func TestMergeEmptyChangeSetIsNoop(t *testing.T) {
	current := []dns.HostRecord{rec("@", "A", "1.2.3.4")}
	got, err := dns.Merge(current, dns.ChangeSet{})
	if err != nil || len(got) != 1 {
		t.Fatalf("%v, %v", got, err)
	}
	if (dns.ChangeSet{}).Empty() != true {
		t.Error("Empty() = false for zero changeset")
	}
}
