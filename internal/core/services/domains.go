package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// Cache TTLs per the design spec.
const (
	domainsListTTL = 10 * time.Minute
	domainInfoTTL  = 5 * time.Minute
)

// DomainService is the domains use-case facade for one profile.
type DomainService struct {
	api     ports.RegistrarAPI
	cache   ports.CacheRepo
	profile string
}

// NewDomainService builds a DomainService bound to a profile name (cache scope).
func NewDomainService(api ports.RegistrarAPI, cache ports.CacheRepo, profile string) *DomainService {
	return &DomainService{api: api, cache: cache, profile: profile}
}

// List returns all account domains, from cache unless refresh or stale.
func (s *DomainService) List(ctx context.Context, refresh bool) ([]registrar.Domain, error) {
	if !refresh {
		if payload, ok, err := s.cache.Get(ctx, s.profile, "domains", "list", domainsListTTL); err == nil && ok {
			var cached []registrar.Domain
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	domains, err := s.api.ListDomains(ctx)
	if err != nil {
		return nil, err
	}
	if payload, err := json.Marshal(domains); err == nil {
		_ = s.cache.Put(ctx, s.profile, "domains", "list", payload) // cache failure is non-fatal
	}
	return domains, nil
}

// Check parses raw names and checks availability (never cached).
func (s *DomainService) Check(ctx context.Context, raw []string) ([]registrar.Availability, error) {
	names := make([]registrar.DomainName, len(raw))
	for i, r := range raw {
		n, err := registrar.Parse(r)
		if err != nil {
			return nil, err
		}
		names[i] = n
	}
	return s.api.CheckDomains(ctx, names)
}

// Info returns full details for one domain, cached briefly.
func (s *DomainService) Info(ctx context.Context, raw string) (registrar.Details, error) {
	name, err := registrar.Parse(raw)
	if err != nil {
		return registrar.Details{}, err
	}
	if payload, ok, err := s.cache.Get(ctx, s.profile, "domain-info", name.String(), domainInfoTTL); err == nil && ok {
		var cached registrar.Details
		if json.Unmarshal(payload, &cached) == nil {
			return cached, nil
		}
	}
	details, err := s.api.DomainInfo(ctx, name)
	if err != nil {
		return registrar.Details{}, err
	}
	if payload, err := json.Marshal(details); err == nil {
		_ = s.cache.Put(ctx, s.profile, "domain-info", name.String(), payload)
	}
	return details, nil
}
