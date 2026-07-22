package services_test

import (
	"context"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type fakeAccountAPI struct {
	ports.AccountAPI
	balanceCalls int
	pricingCalls int
}

func (f *fakeAccountAPI) Balances(context.Context) (account.Balance, error) {
	f.balanceCalls++
	return account.Balance{Currency: "USD", Available: 42}, nil
}

func (f *fakeAccountAPI) Pricing(context.Context, string, string, string) ([]account.Price, error) {
	f.pricingCalls++
	return []account.Price{{Product: "com", Yours: 10.87}}, nil
}

func TestBalanceCached(t *testing.T) {
	api := &fakeAccountAPI{}
	svc := services.NewAccountService(api, newFakeCache(), "p")
	ctx := context.Background()

	if _, err := svc.Balance(ctx, false); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Balance(ctx, false); err != nil {
		t.Fatal(err)
	}
	if api.balanceCalls != 1 {
		t.Errorf("balanceCalls = %d, want 1", api.balanceCalls)
	}
	if _, err := svc.Balance(ctx, true); err != nil {
		t.Fatal(err)
	}
	if api.balanceCalls != 2 {
		t.Errorf("refresh ignored: %d", api.balanceCalls)
	}
}

func TestPricingCachedPerQuery(t *testing.T) {
	api := &fakeAccountAPI{}
	svc := services.NewAccountService(api, newFakeCache(), "p")
	ctx := context.Background()

	_, _ = svc.Pricing(ctx, "DOMAIN", "REGISTER", "com")
	_, _ = svc.Pricing(ctx, "DOMAIN", "REGISTER", "com")
	if api.pricingCalls != 1 {
		t.Errorf("same query not cached: %d", api.pricingCalls)
	}
	_, _ = svc.Pricing(ctx, "DOMAIN", "RENEW", "com")
	if api.pricingCalls != 2 {
		t.Errorf("different query served from cache: %d", api.pricingCalls)
	}
}

type fakeSSLAPI struct {
	ports.SSLAPI
	purchased *ssl.Purchase
}

func (f *fakeSSLAPI) PurchaseCertificate(_ context.Context, p ssl.Purchase) (ssl.Certificate, error) {
	f.purchased = &p
	return ssl.Certificate{ID: "1"}, nil
}

func TestSSLPurchaseValidates(t *testing.T) {
	api := &fakeSSLAPI{}
	svc := services.NewSSLService(api)
	if _, err := svc.Purchase(context.Background(), ssl.Purchase{Years: 1}); err == nil {
		t.Error("purchase without type accepted")
	}
	if _, err := svc.Purchase(context.Background(), ssl.Purchase{Type: "PositiveSSL", Years: 1}); err != nil {
		t.Fatal(err)
	}
	if api.purchased == nil {
		t.Error("purchase not passed through")
	}
}
