package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

const tldTTL = 24 * time.Hour

// invalidateDomains drops cached domain lists/info after a mutating call.
func (s *DomainService) invalidateDomains(ctx context.Context) {
	_ = s.cache.Invalidate(ctx, s.profile, "domains")
	_ = s.cache.Invalidate(ctx, s.profile, "domain-info")
}

// Register registers a new domain and invalidates the domain cache.
func (s *DomainService) Register(ctx context.Context, reg registrar.Registration) (registrar.RegistrationResult, error) {
	if err := reg.Validate(); err != nil {
		return registrar.RegistrationResult{}, err
	}
	res, err := s.api.RegisterDomain(ctx, reg)
	if err != nil {
		return registrar.RegistrationResult{}, err
	}
	s.invalidateDomains(ctx)
	return res, nil
}

// Renew extends a domain registration.
func (s *DomainService) Renew(ctx context.Context, domain string, years int) (registrar.RenewalResult, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return registrar.RenewalResult{}, err
	}
	res, err := s.api.RenewDomain(ctx, name, years)
	if err != nil {
		return registrar.RenewalResult{}, err
	}
	s.invalidateDomains(ctx)
	return res, nil
}

// Reactivate re-activates an expired domain.
func (s *DomainService) Reactivate(ctx context.Context, domain string) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	if err := s.api.ReactivateDomain(ctx, name); err != nil {
		return err
	}
	s.invalidateDomains(ctx)
	return nil
}

// Contacts returns the domain's WHOIS contacts.
func (s *DomainService) Contacts(ctx context.Context, domain string) (registrar.ContactSet, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return registrar.ContactSet{}, err
	}
	return s.api.GetContacts(ctx, name)
}

// SetContacts replaces the domain's WHOIS contacts.
func (s *DomainService) SetContacts(ctx context.Context, domain string, contacts registrar.ContactSet) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	if err := contacts.Validate(); err != nil {
		return err
	}
	return s.api.SetContacts(ctx, name, contacts)
}

// Lock reports the registrar-lock status.
func (s *DomainService) Lock(ctx context.Context, domain string) (bool, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return false, err
	}
	return s.api.LockStatus(ctx, name)
}

// SetLock toggles the registrar lock and invalidates the domain cache.
func (s *DomainService) SetLock(ctx context.Context, domain string, locked bool) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	if err := s.api.SetLock(ctx, name, locked); err != nil {
		return err
	}
	s.invalidateDomains(ctx)
	return nil
}

// TLDs lists offered TLDs, cached for a day.
func (s *DomainService) TLDs(ctx context.Context) ([]registrar.TLD, error) {
	if payload, ok, err := s.cache.Get(ctx, s.profile, "tlds", "list", tldTTL); err == nil && ok {
		var cached []registrar.TLD
		if json.Unmarshal(payload, &cached) == nil {
			return cached, nil
		}
	}
	tlds, err := s.api.TLDs(ctx)
	if err != nil {
		return nil, err
	}
	if payload, err := json.Marshal(tlds); err == nil {
		_ = s.cache.Put(ctx, s.profile, "tlds", "list", payload)
	}
	return tlds, nil
}

// TransferService manages inbound domain transfers.
type TransferService struct {
	api ports.TransferAPI
}

// NewTransferService builds a TransferService.
func NewTransferService(api ports.TransferAPI) *TransferService {
	return &TransferService{api: api}
}

// Create starts an inbound transfer with the given EPP code.
func (s *TransferService) Create(ctx context.Context, domain, eppCode string, years int) (registrar.Transfer, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return registrar.Transfer{}, err
	}
	if years < 1 {
		years = 1
	}
	return s.api.CreateTransfer(ctx, name, eppCode, years)
}

// Status fetches a transfer's current status.
func (s *TransferService) Status(ctx context.Context, id string) (registrar.Transfer, error) {
	return s.api.TransferStatus(ctx, id)
}

// List returns all transfers on the account.
func (s *TransferService) List(ctx context.Context) ([]registrar.Transfer, error) {
	return s.api.ListTransfers(ctx)
}

// Resubmit retries a failed transfer.
func (s *TransferService) Resubmit(ctx context.Context, id string) error {
	return s.api.ResubmitTransfer(ctx, id)
}
