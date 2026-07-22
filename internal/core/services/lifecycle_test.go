package services_test

import (
	"context"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type lifecycleRegistrar struct {
	ports.RegistrarAPI
	renewed   int
	locked    *bool
	tldsCalls int
}

func (f *lifecycleRegistrar) RenewDomain(_ context.Context, _ registrar.DomainName, years int) (registrar.RenewalResult, error) {
	f.renewed = years
	return registrar.RenewalResult{ChargedAmount: 14.2}, nil
}

func (f *lifecycleRegistrar) SetLock(_ context.Context, _ registrar.DomainName, locked bool) error {
	f.locked = &locked
	return nil
}

func (f *lifecycleRegistrar) TLDs(context.Context) ([]registrar.TLD, error) {
	f.tldsCalls++
	return []registrar.TLD{{Name: "com", MinYears: 1, MaxYears: 10, Registerable: true}}, nil
}

func TestRenewInvalidatesDomainsCache(t *testing.T) {
	api := &lifecycleRegistrar{}
	cache := newFakeCache()
	svc := services.NewDomainService(api, cache, "p")
	ctx := context.Background()

	_ = cache.Put(ctx, "p", "domains", "list", []byte("stale"))
	res, err := svc.Renew(ctx, "alpha.com", 2)
	if err != nil || res.ChargedAmount != 14.2 || api.renewed != 2 {
		t.Fatalf("%+v, %v", res, err)
	}
	if _, ok, _ := cache.Get(ctx, "p", "domains", "list", 1<<40); ok {
		t.Error("domains cache not invalidated after renew")
	}
}

func TestSetLockInvalidates(t *testing.T) {
	api := &lifecycleRegistrar{}
	cache := newFakeCache()
	svc := services.NewDomainService(api, cache, "p")
	ctx := context.Background()

	_ = cache.Put(ctx, "p", "domain-info", "alpha.com", []byte("stale"))
	if err := svc.SetLock(ctx, "alpha.com", true); err != nil {
		t.Fatal(err)
	}
	if api.locked == nil || !*api.locked {
		t.Error("lock not passed through")
	}
	if _, ok, _ := cache.Get(ctx, "p", "domain-info", "alpha.com", 1<<40); ok {
		t.Error("domain-info cache not invalidated")
	}
}

func TestTLDsCached(t *testing.T) {
	api := &lifecycleRegistrar{}
	svc := services.NewDomainService(api, newFakeCache(), "p")
	ctx := context.Background()

	if _, err := svc.TLDs(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.TLDs(ctx); err != nil {
		t.Fatal(err)
	}
	if api.tldsCalls != 1 {
		t.Errorf("tldsCalls = %d, want 1 (cached)", api.tldsCalls)
	}
}

func TestRegisterValidates(t *testing.T) {
	svc := services.NewDomainService(&lifecycleRegistrar{}, newFakeCache(), "p")
	name, _ := registrar.Parse("x.com")
	if _, err := svc.Register(context.Background(), registrar.Registration{Name: name, Years: 1}); err == nil {
		t.Error("registration without contacts accepted")
	}
}

type fakeTransfer struct {
	created string
	resub   string
}

func (f *fakeTransfer) CreateTransfer(_ context.Context, name registrar.DomainName, epp string, years int) (registrar.Transfer, error) {
	f.created = name.String() + "/" + epp
	return registrar.Transfer{ID: "1", Name: name.String()}, nil
}

func (f *fakeTransfer) TransferStatus(_ context.Context, id string) (registrar.Transfer, error) {
	return registrar.Transfer{ID: id, Status: "INPROGRESS"}, nil
}

func (f *fakeTransfer) ListTransfers(context.Context) ([]registrar.Transfer, error) {
	return []registrar.Transfer{{ID: "1"}}, nil
}

func (f *fakeTransfer) ResubmitTransfer(_ context.Context, id string) error {
	f.resub = id
	return nil
}

func TestTransferService(t *testing.T) {
	api := &fakeTransfer{}
	svc := services.NewTransferService(api)
	ctx := context.Background()

	tr, err := svc.Create(ctx, "moving.com", "EPP123", 0)
	if err != nil || tr.ID != "1" || api.created != "moving.com/EPP123" {
		t.Errorf("%+v, %v, %q", tr, err, api.created)
	}
	if st, err := svc.Status(ctx, "1"); err != nil || st.Status != "INPROGRESS" {
		t.Errorf("%+v, %v", st, err)
	}
	if list, err := svc.List(ctx); err != nil || len(list) != 1 {
		t.Errorf("%+v, %v", list, err)
	}
	if err := svc.Resubmit(ctx, "1"); err != nil || api.resub != "1" {
		t.Errorf("%v, %q", err, api.resub)
	}
}
