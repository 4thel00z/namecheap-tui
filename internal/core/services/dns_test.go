package services_test

import (
	"context"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type fakeDNS struct {
	records   []dns.HostRecord
	getCalls  int
	setCalls  int
	lastSet   []dns.HostRecord
	nsInfo    dns.NameserverInfo
	defaulted bool
	custom    []string
	forwards  []dns.EmailForward
}

func (f *fakeDNS) GetHosts(context.Context, registrar.DomainName) ([]dns.HostRecord, error) {
	f.getCalls++
	out := make([]dns.HostRecord, len(f.records))
	copy(out, f.records)
	return out, nil
}

func (f *fakeDNS) SetHosts(_ context.Context, _ registrar.DomainName, records []dns.HostRecord) error {
	f.setCalls++
	f.lastSet = records
	f.records = records
	return nil
}

func (f *fakeDNS) GetNameservers(context.Context, registrar.DomainName) (dns.NameserverInfo, error) {
	return f.nsInfo, nil
}

func (f *fakeDNS) SetDefaultNS(context.Context, registrar.DomainName) error {
	f.defaulted = true
	return nil
}

func (f *fakeDNS) SetCustomNS(_ context.Context, _ registrar.DomainName, ns []string) error {
	f.custom = ns
	return nil
}

func (f *fakeDNS) GetEmailForwarding(context.Context, registrar.DomainName) ([]dns.EmailForward, error) {
	return f.forwards, nil
}

func (f *fakeDNS) SetEmailForwarding(_ context.Context, _ registrar.DomainName, fwds []dns.EmailForward) error {
	f.forwards = fwds
	return nil
}

func aRec(name, value string) dns.HostRecord {
	return dns.HostRecord{Name: name, Type: dns.A, Value: value, TTL: 1799}
}

func newDNSService(api *fakeDNS) (*services.DNSService, *fakeCache) {
	cache := newFakeCache()
	return services.NewDNSService(api, cache, "p"), cache
}

func TestZoneCaches(t *testing.T) {
	api := &fakeDNS{records: []dns.HostRecord{aRec("@", "1.2.3.4")}}
	svc, _ := newDNSService(api)
	ctx := context.Background()

	z, err := svc.Zone(ctx, "alpha.com", false)
	if err != nil || len(z.Records) != 1 || z.Domain.String() != "alpha.com" {
		t.Fatalf("%+v, %v", z, err)
	}
	if _, err := svc.Zone(ctx, "alpha.com", false); err != nil {
		t.Fatal(err)
	}
	if api.getCalls != 1 {
		t.Errorf("getCalls = %d, want 1 (cached)", api.getCalls)
	}
}

func TestApplyMergesOntoFreshState(t *testing.T) {
	api := &fakeDNS{records: []dns.HostRecord{aRec("@", "1.2.3.4"), aRec("www", "1.2.3.5")}}
	svc, cache := newDNSService(api)
	ctx := context.Background()

	// Prime the cache, then mutate remote state behind the service's back.
	if _, err := svc.Zone(ctx, "alpha.com", false); err != nil {
		t.Fatal(err)
	}
	api.records = append(api.records, aRec("mail", "1.2.3.6"))

	z, err := svc.Apply(ctx, "alpha.com", dns.ChangeSet{Add: []dns.HostRecord{aRec("new", "9.9.9.9")}})
	if err != nil {
		t.Fatal(err)
	}
	// Merge must include the record added remotely after the cache was primed.
	if len(z.Records) != 4 || len(api.lastSet) != 4 {
		t.Errorf("merge lost remote state: applied %d records", len(api.lastSet))
	}
	// Cache refreshed with post-apply zone.
	if payload, ok, _ := cache.Get(ctx, "p", "zone", "alpha.com", 1); !ok || len(payload) == 0 {
		t.Error("cache not refreshed after apply")
	}
}

func TestApplyRejectsInvalidChange(t *testing.T) {
	api := &fakeDNS{records: []dns.HostRecord{aRec("@", "1.2.3.4")}}
	svc, _ := newDNSService(api)
	ctx := context.Background()

	// Removing a nonexistent record must fail without calling SetHosts.
	_, err := svc.Apply(ctx, "alpha.com", dns.ChangeSet{Remove: []dns.HostRecord{aRec("ghost", "0.0.0.0")}})
	if err == nil || api.setCalls != 0 {
		t.Errorf("err=%v setCalls=%d", err, api.setCalls)
	}
	// Invalid record in Add must fail validation.
	_, err = svc.Apply(ctx, "alpha.com", dns.ChangeSet{Add: []dns.HostRecord{{Name: "x"}}})
	if err == nil || api.setCalls != 0 {
		t.Errorf("invalid add accepted: err=%v setCalls=%d", err, api.setCalls)
	}
}

func TestAddRemoveHelpers(t *testing.T) {
	api := &fakeDNS{records: []dns.HostRecord{aRec("@", "1.2.3.4")}}
	svc, _ := newDNSService(api)
	ctx := context.Background()

	if _, err := svc.Add(ctx, "alpha.com", aRec("www", "5.6.7.8")); err != nil {
		t.Fatal(err)
	}
	if len(api.records) != 2 {
		t.Fatalf("add did not apply: %+v", api.records)
	}
	if _, err := svc.Remove(ctx, "alpha.com", "www", "A", "5.6.7.8"); err != nil {
		t.Fatal(err)
	}
	if len(api.records) != 1 {
		t.Errorf("remove did not apply: %+v", api.records)
	}
	// Remove without value works when exactly one record matches name+type.
	if _, err := svc.Remove(ctx, "alpha.com", "@", "A", ""); err != nil {
		t.Fatal(err)
	}
	if len(api.records) != 0 {
		t.Errorf("valueless remove did not apply: %+v", api.records)
	}
}

func TestRemoveAmbiguousNeedsValue(t *testing.T) {
	api := &fakeDNS{records: []dns.HostRecord{aRec("@", "1.1.1.1"), aRec("@", "2.2.2.2")}}
	svc, _ := newDNSService(api)
	if _, err := svc.Remove(context.Background(), "alpha.com", "@", "A", ""); err == nil {
		t.Error("ambiguous remove accepted")
	}
}

func TestSetReplacesZone(t *testing.T) {
	api := &fakeDNS{records: []dns.HostRecord{aRec("@", "1.2.3.4")}}
	svc, _ := newDNSService(api)
	replacement := []dns.HostRecord{aRec("new", "9.9.9.9")}
	z, err := svc.Set(context.Background(), "alpha.com", replacement)
	if err != nil || len(z.Records) != 1 || api.records[0].Name != "new" {
		t.Errorf("%+v, %v", z, err)
	}
}

func TestNameserverOps(t *testing.T) {
	api := &fakeDNS{nsInfo: dns.NameserverInfo{UsingOurDNS: true}}
	svc, cache := newDNSService(api)
	ctx := context.Background()

	info, err := svc.Nameservers(ctx, "alpha.com")
	if err != nil || !info.UsingOurDNS {
		t.Errorf("%+v, %v", info, err)
	}
	_ = cache.Put(ctx, "p", "zone", "alpha.com", []byte("stale"))
	// Namecheap requires at least two nameservers.
	if err := svc.UseCustom(ctx, "alpha.com", []string{"ns1.x.com"}); err == nil {
		t.Error("single nameserver accepted")
	}
	if err := svc.UseCustom(ctx, "alpha.com", []string{"ns1.x.com", "ns2.x.com"}); err != nil {
		t.Fatal(err)
	}
	if len(api.custom) != 2 {
		t.Error("custom NS not set")
	}
	if _, ok, _ := cache.Get(ctx, "p", "zone", "alpha.com", 1<<40); ok {
		t.Error("zone cache not invalidated after NS switch")
	}
	if err := svc.UseDefault(ctx, "alpha.com"); err != nil || !api.defaulted {
		t.Errorf("default NS: %v", err)
	}
}

func TestEmailForwards(t *testing.T) {
	api := &fakeDNS{forwards: []dns.EmailForward{{Mailbox: "info", ForwardTo: "a@b.c"}}}
	svc, _ := newDNSService(api)
	ctx := context.Background()

	fwds, err := svc.EmailForwards(ctx, "alpha.com")
	if err != nil || len(fwds) != 1 {
		t.Errorf("%+v, %v", fwds, err)
	}
	err = svc.SetEmailForwards(ctx, "alpha.com", []dns.EmailForward{
		{Mailbox: "x", ForwardTo: "y@z.dev"},
	})
	if err != nil || api.forwards[0].Mailbox != "x" {
		t.Errorf("%+v, %v", api.forwards, err)
	}
}
