package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type fakeRegistrar struct {
	listCalls int
	domains   []registrar.Domain
	checked   []registrar.DomainName
}

func (f *fakeRegistrar) ListDomains(context.Context) ([]registrar.Domain, error) {
	f.listCalls++
	return f.domains, nil
}

func (f *fakeRegistrar) CheckDomains(_ context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	f.checked = names
	out := make([]registrar.Availability, len(names))
	for i, n := range names {
		out[i] = registrar.Availability{Name: n, Available: true}
	}
	return out, nil
}

func (f *fakeRegistrar) DomainInfo(_ context.Context, name registrar.DomainName) (registrar.Details, error) {
	return registrar.Details{Domain: registrar.Domain{Name: name}, Status: "Ok"}, nil
}

type fakeCache struct{ m map[string][]byte }

func newFakeCache() *fakeCache { return &fakeCache{m: map[string][]byte{}} }

func (f *fakeCache) Get(_ context.Context, p, kind, key string, _ time.Duration) ([]byte, bool, error) {
	v, ok := f.m[p+"/"+kind+"/"+key]
	return v, ok, nil
}

func (f *fakeCache) Put(_ context.Context, p, kind, key string, payload []byte) error {
	f.m[p+"/"+kind+"/"+key] = payload
	return nil
}

func (f *fakeCache) Invalidate(_ context.Context, p, kind string) error { return nil }

func dom(name string) registrar.Domain {
	n, _ := registrar.Parse(name)
	return registrar.Domain{Name: n}
}

func TestListUsesCache(t *testing.T) {
	api := &fakeRegistrar{domains: []registrar.Domain{dom("a.com")}}
	svc := services.NewDomainService(api, newFakeCache(), "p")
	ctx := context.Background()

	got, err := svc.List(ctx, false)
	if err != nil || len(got) != 1 {
		t.Fatalf("first list: %v, %v", got, err)
	}
	if api.listCalls != 1 {
		t.Fatalf("listCalls = %d", api.listCalls)
	}
	// Second call served from cache.
	if _, err := svc.List(ctx, false); err != nil {
		t.Fatal(err)
	}
	if api.listCalls != 1 {
		t.Errorf("cache miss: listCalls = %d, want 1", api.listCalls)
	}
	// refresh bypasses cache.
	if _, err := svc.List(ctx, true); err != nil {
		t.Fatal(err)
	}
	if api.listCalls != 2 {
		t.Errorf("refresh ignored: listCalls = %d, want 2", api.listCalls)
	}
}

func TestCheckParsesNames(t *testing.T) {
	api := &fakeRegistrar{}
	svc := services.NewDomainService(api, newFakeCache(), "p")
	got, err := svc.Check(context.Background(), []string{"A.com", "b.co.uk"})
	if err != nil || len(got) != 2 {
		t.Fatalf("%v, %v", got, err)
	}
	if api.checked[0].SLD != "a" || api.checked[1].TLD != "co.uk" {
		t.Errorf("parsed names wrong: %+v", api.checked)
	}
	if _, err := svc.Check(context.Background(), []string{"garbage"}); err == nil {
		t.Error("invalid name accepted")
	}
}

func TestInfo(t *testing.T) {
	svc := services.NewDomainService(&fakeRegistrar{}, newFakeCache(), "p")
	got, err := svc.Info(context.Background(), "a.com")
	if err != nil || got.Status != "Ok" || got.Name.String() != "a.com" {
		t.Errorf("%+v, %v", got, err)
	}
}
