package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

const zoneTTL = 5 * time.Minute

// DNSService is the zone-management use-case facade for one profile.
type DNSService struct {
	api     ports.DNSAPI
	cache   ports.CacheRepo
	profile string
}

// NewDNSService builds a DNSService bound to a profile name (cache scope).
func NewDNSService(api ports.DNSAPI, cache ports.CacheRepo, profile string) *DNSService {
	return &DNSService{api: api, cache: cache, profile: profile}
}

// Zone returns the domain's record set, from cache unless refresh or stale.
func (s *DNSService) Zone(ctx context.Context, domain string, refresh bool) (dns.Zone, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return dns.Zone{}, err
	}
	if !refresh {
		if payload, ok, err := s.cache.Get(ctx, s.profile, "zone", name.String(), zoneTTL); err == nil && ok {
			var records []dns.HostRecord
			if json.Unmarshal(payload, &records) == nil {
				return dns.Zone{Domain: name, Records: records}, nil
			}
		}
	}
	records, err := s.api.GetHosts(ctx, name)
	if err != nil {
		return dns.Zone{}, err
	}
	s.cacheZone(ctx, name, records)
	return dns.Zone{Domain: name, Records: records}, nil
}

func (s *DNSService) cacheZone(ctx context.Context, name registrar.DomainName, records []dns.HostRecord) {
	if payload, err := json.Marshal(records); err == nil {
		_ = s.cache.Put(ctx, s.profile, "zone", name.String(), payload) // cache failure is non-fatal
	}
}

// Apply merges cs onto the CURRENT remote zone (never the cache — setHosts
// replaces everything, so a stale base would silently destroy records),
// validates the result, and replaces the zone.
func (s *DNSService) Apply(ctx context.Context, domain string, cs dns.ChangeSet) (dns.Zone, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return dns.Zone{}, err
	}
	current, err := s.api.GetHosts(ctx, name)
	if err != nil {
		return dns.Zone{}, fmt.Errorf("fetch current zone: %w", err)
	}
	merged, err := dns.Merge(current, cs)
	if err != nil {
		return dns.Zone{}, err
	}
	for _, r := range merged {
		if err := r.Validate(); err != nil {
			return dns.Zone{}, fmt.Errorf("record %s %s: %w", r.Type, r.Name, err)
		}
	}
	if err := s.api.SetHosts(ctx, name, merged); err != nil {
		return dns.Zone{}, err
	}
	s.cacheZone(ctx, name, merged)
	return dns.Zone{Domain: name, Records: merged}, nil
}

// Add stages and applies a single record addition.
func (s *DNSService) Add(ctx context.Context, domain string, rec dns.HostRecord) (dns.Zone, error) {
	if err := rec.Validate(); err != nil {
		return dns.Zone{}, err
	}
	return s.Apply(ctx, domain, dns.ChangeSet{Add: []dns.HostRecord{rec}})
}

// Remove deletes one record matched by name+type (+value when ambiguous).
func (s *DNSService) Remove(ctx context.Context, domain, recName, recType, value string) (dns.Zone, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return dns.Zone{}, err
	}
	rt, err := dns.ParseRecordType(recType)
	if err != nil {
		return dns.Zone{}, err
	}
	current, err := s.api.GetHosts(ctx, name)
	if err != nil {
		return dns.Zone{}, err
	}
	var matches []dns.HostRecord
	for _, r := range current {
		if r.Name == recName && r.Type == rt && (value == "" || r.Value == value) {
			matches = append(matches, r)
		}
	}
	switch len(matches) {
	case 0:
		return dns.Zone{}, fmt.Errorf("no %s record named %q%s", rt, recName, valueHint(value))
	case 1:
		return s.Apply(ctx, domain, dns.ChangeSet{Remove: matches})
	default:
		return dns.Zone{}, fmt.Errorf("%d %s records named %q — pass the value to disambiguate", len(matches), rt, recName)
	}
}

func valueHint(value string) string {
	if value == "" {
		return ""
	}
	return fmt.Sprintf(" with value %q", value)
}

// Set REPLACES the domain's entire record set after validating it.
func (s *DNSService) Set(ctx context.Context, domain string, records []dns.HostRecord) (dns.Zone, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return dns.Zone{}, err
	}
	for _, r := range records {
		if err := r.Validate(); err != nil {
			return dns.Zone{}, fmt.Errorf("record %s %s: %w", r.Type, r.Name, err)
		}
	}
	if err := s.api.SetHosts(ctx, name, records); err != nil {
		return dns.Zone{}, err
	}
	s.cacheZone(ctx, name, records)
	return dns.Zone{Domain: name, Records: records}, nil
}

// Nameservers reports the domain's nameserver configuration.
func (s *DNSService) Nameservers(ctx context.Context, domain string) (dns.NameserverInfo, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return dns.NameserverInfo{}, err
	}
	return s.api.GetNameservers(ctx, name)
}

// UseDefault switches to Namecheap's default nameservers.
func (s *DNSService) UseDefault(ctx context.Context, domain string) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	if err := s.api.SetDefaultNS(ctx, name); err != nil {
		return err
	}
	return s.cache.Invalidate(ctx, s.profile, "zone")
}

// UseCustom switches to custom nameservers.
func (s *DNSService) UseCustom(ctx context.Context, domain string, ns []string) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	if len(ns) < 2 {
		return fmt.Errorf("namecheap requires at least 2 nameservers, got %d", len(ns))
	}
	if err := s.api.SetCustomNS(ctx, name, ns); err != nil {
		return err
	}
	return s.cache.Invalidate(ctx, s.profile, "zone")
}

// EmailForwards returns the domain's mailbox forwards.
func (s *DNSService) EmailForwards(ctx context.Context, domain string) ([]dns.EmailForward, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return nil, err
	}
	return s.api.GetEmailForwarding(ctx, name)
}

// SetEmailForwards REPLACES the domain's mailbox forwards.
func (s *DNSService) SetEmailForwards(ctx context.Context, domain string, fwds []dns.EmailForward) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	return s.api.SetEmailForwarding(ctx, name, fwds)
}
