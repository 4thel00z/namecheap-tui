package services

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/ssl"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

const (
	balanceTTL = 5 * time.Minute
	pricingTTL = 24 * time.Hour
)

// SSLService manages SSL certificates.
type SSLService struct {
	api ports.SSLAPI
}

// NewSSLService builds an SSLService.
func NewSSLService(api ports.SSLAPI) *SSLService {
	return &SSLService{api: api}
}

// List returns all certificates.
func (s *SSLService) List(ctx context.Context) ([]ssl.Certificate, error) {
	return s.api.ListCertificates(ctx)
}

// Purchase buys a new certificate.
func (s *SSLService) Purchase(ctx context.Context, p ssl.Purchase) (ssl.Certificate, error) {
	if err := p.Validate(); err != nil {
		return ssl.Certificate{}, err
	}
	return s.api.PurchaseCertificate(ctx, p)
}

// Activate submits a CSR for domain validation.
func (s *SSLService) Activate(ctx context.Context, a ssl.Activation) error {
	if err := a.Validate(); err != nil {
		return err
	}
	return s.api.ActivateCertificate(ctx, a)
}

// Info fetches one certificate's details.
func (s *SSLService) Info(ctx context.Context, id string) (ssl.Certificate, error) {
	return s.api.CertificateInfo(ctx, id)
}

// Renew renews a certificate.
func (s *SSLService) Renew(ctx context.Context, id string, p ssl.Purchase) (ssl.Certificate, error) {
	if err := p.Validate(); err != nil {
		return ssl.Certificate{}, err
	}
	return s.api.RenewCertificate(ctx, id, p)
}

// Reissue reissues a certificate with a new CSR.
func (s *SSLService) Reissue(ctx context.Context, a ssl.Activation) error {
	if err := a.Validate(); err != nil {
		return err
	}
	return s.api.ReissueCertificate(ctx, a)
}

// Approvers lists valid DV approver emails for a domain.
func (s *SSLService) Approvers(ctx context.Context, domain, certType string) ([]string, error) {
	return s.api.ApproverEmails(ctx, domain, certType)
}

// ResendApproverEmail resends the DV mail.
func (s *SSLService) ResendApproverEmail(ctx context.Context, id string) error {
	return s.api.ResendApproverEmail(ctx, id)
}

// Revoke revokes a certificate.
func (s *SSLService) Revoke(ctx context.Context, id, certType string) error {
	return s.api.RevokeCertificate(ctx, id, certType)
}

// PrivacyService manages domain-privacy subscriptions.
type PrivacyService struct {
	api ports.PrivacyAPI
}

// NewPrivacyService builds a PrivacyService.
func NewPrivacyService(api ports.PrivacyAPI) *PrivacyService {
	return &PrivacyService{api: api}
}

// List returns all privacy subscriptions.
func (s *PrivacyService) List(ctx context.Context) ([]account.PrivacySubscription, error) {
	return s.api.ListPrivacy(ctx)
}

// Enable turns privacy on for a subscription.
func (s *PrivacyService) Enable(ctx context.Context, id, forwardTo string) error {
	return s.api.EnablePrivacy(ctx, id, forwardTo)
}

// Disable turns privacy off.
func (s *PrivacyService) Disable(ctx context.Context, id string) error {
	return s.api.DisablePrivacy(ctx, id)
}

// Renew extends a subscription.
func (s *PrivacyService) Renew(ctx context.Context, id string, years int) error {
	if years < 1 {
		years = 1
	}
	return s.api.RenewPrivacy(ctx, id, years)
}

// ChangeEmail rotates the privacy contact address.
func (s *PrivacyService) ChangeEmail(ctx context.Context, id string) error {
	return s.api.ChangePrivacyEmail(ctx, id)
}

// Assign attaches a subscription to a domain.
func (s *PrivacyService) Assign(ctx context.Context, id, domain string) error {
	return s.api.AssignPrivacy(ctx, id, domain)
}

// Unassign detaches a subscription from its domain.
func (s *PrivacyService) Unassign(ctx context.Context, id string) error {
	return s.api.UnassignPrivacy(ctx, id)
}

// Discard drops an unused subscription.
func (s *PrivacyService) Discard(ctx context.Context, id string) error {
	return s.api.DiscardPrivacy(ctx, id)
}

// AccountService exposes balances, pricing, and the address book.
type AccountService struct {
	api     ports.AccountAPI
	cache   ports.CacheRepo
	profile string
}

// NewAccountService builds an AccountService bound to a profile (cache scope).
func NewAccountService(api ports.AccountAPI, cache ports.CacheRepo, profile string) *AccountService {
	return &AccountService{api: api, cache: cache, profile: profile}
}

// Balance returns account funds, cached briefly.
func (s *AccountService) Balance(ctx context.Context, refresh bool) (account.Balance, error) {
	if !refresh {
		if payload, ok, err := s.cache.Get(ctx, s.profile, "balance", "current", balanceTTL); err == nil && ok {
			var cached account.Balance
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	b, err := s.api.Balances(ctx)
	if err != nil {
		return account.Balance{}, err
	}
	if payload, err := json.Marshal(b); err == nil {
		_ = s.cache.Put(ctx, s.profile, "balance", "current", payload)
	}
	return b, nil
}

// Pricing returns price entries, cached per query for a day.
func (s *AccountService) Pricing(ctx context.Context, productType, category, product string) ([]account.Price, error) {
	key := strings.ToLower(productType + "/" + category + "/" + product)
	if payload, ok, err := s.cache.Get(ctx, s.profile, "pricing", key, pricingTTL); err == nil && ok {
		var cached []account.Price
		if json.Unmarshal(payload, &cached) == nil {
			return cached, nil
		}
	}
	prices, err := s.api.Pricing(ctx, productType, category, product)
	if err != nil {
		return nil, err
	}
	if payload, err := json.Marshal(prices); err == nil {
		_ = s.cache.Put(ctx, s.profile, "pricing", key, payload)
	}
	return prices, nil
}

// Addresses lists the address book.
func (s *AccountService) Addresses(ctx context.Context) ([]account.Address, error) {
	return s.api.ListAddresses(ctx)
}

// Address fetches one address-book entry.
func (s *AccountService) Address(ctx context.Context, id string) (account.Address, error) {
	return s.api.GetAddress(ctx, id)
}

// AddAddress creates an address-book entry.
func (s *AccountService) AddAddress(ctx context.Context, a account.Address) error {
	if err := a.Contact.Validate(); err != nil {
		return err
	}
	return s.api.CreateAddress(ctx, a)
}

// UpdateAddress edits an address-book entry.
func (s *AccountService) UpdateAddress(ctx context.Context, a account.Address) error {
	if err := a.Contact.Validate(); err != nil {
		return err
	}
	return s.api.UpdateAddress(ctx, a)
}

// RemoveAddress deletes an entry.
func (s *AccountService) RemoveAddress(ctx context.Context, id string) error {
	return s.api.DeleteAddress(ctx, id)
}

// SetDefaultAddress marks an entry default.
func (s *AccountService) SetDefaultAddress(ctx context.Context, id string) error {
	return s.api.SetDefaultAddress(ctx, id)
}
