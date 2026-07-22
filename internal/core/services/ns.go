package services

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// NSService manages personal (glue-record) nameservers.
type NSService struct {
	api ports.NSAPI
}

// NewNSService builds an NSService.
func NewNSService(api ports.NSAPI) *NSService {
	return &NSService{api: api}
}

// Create registers host (e.g. "ns1.example.com") with the given IP.
func (s *NSService) Create(ctx context.Context, domain, host, ip string) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	return s.api.CreateNS(ctx, name, host, ip)
}

// Update changes the nameserver's glue IP.
func (s *NSService) Update(ctx context.Context, domain, host, oldIP, newIP string) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	return s.api.UpdateNS(ctx, name, host, oldIP, newIP)
}

// Delete removes the personal nameserver.
func (s *NSService) Delete(ctx context.Context, domain, host string) error {
	name, err := registrar.Parse(domain)
	if err != nil {
		return err
	}
	return s.api.DeleteNS(ctx, name, host)
}

// Info fetches the personal nameserver's details.
func (s *NSService) Info(ctx context.Context, domain, host string) (dns.RegisteredNS, error) {
	name, err := registrar.Parse(domain)
	if err != nil {
		return dns.RegisteredNS{}, err
	}
	return s.api.NSInfo(ctx, name, host)
}
