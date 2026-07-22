// Package ports defines the hexagon's driven interfaces and the error
// contract adapters must honor.
package ports

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/dns"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

// RegistrarAPI is the domain-registration capability of the registrar.
type RegistrarAPI interface {
	// ListDomains returns all domains in the account (all pages merged).
	ListDomains(ctx context.Context) ([]registrar.Domain, error)
	// CheckDomains checks availability for up to 50 names per call.
	CheckDomains(ctx context.Context, names []registrar.DomainName) ([]registrar.Availability, error)
	// DomainInfo returns full details for one domain.
	DomainInfo(ctx context.Context, name registrar.DomainName) (registrar.Details, error)
}

// DNSAPI is the zone-management capability of the registrar.
type DNSAPI interface {
	// GetHosts returns all host records of a domain using Namecheap DNS.
	GetHosts(ctx context.Context, name registrar.DomainName) ([]dns.HostRecord, error)
	// SetHosts REPLACES the domain's entire record set.
	SetHosts(ctx context.Context, name registrar.DomainName, records []dns.HostRecord) error
	// GetNameservers reports which nameservers the domain uses.
	GetNameservers(ctx context.Context, name registrar.DomainName) (dns.NameserverInfo, error)
	// SetDefaultNS switches the domain to Namecheap's default nameservers.
	SetDefaultNS(ctx context.Context, name registrar.DomainName) error
	// SetCustomNS switches the domain to the given custom nameservers.
	SetCustomNS(ctx context.Context, name registrar.DomainName, ns []string) error
	// GetEmailForwarding returns the domain's mailbox forwards.
	GetEmailForwarding(ctx context.Context, name registrar.DomainName) ([]dns.EmailForward, error)
	// SetEmailForwarding REPLACES the domain's mailbox forwards.
	SetEmailForwarding(ctx context.Context, name registrar.DomainName, fwds []dns.EmailForward) error
}

// NSAPI manages personal (glue-record) nameservers under a domain.
type NSAPI interface {
	CreateNS(ctx context.Context, domain registrar.DomainName, host, ip string) error
	UpdateNS(ctx context.Context, domain registrar.DomainName, host, oldIP, newIP string) error
	DeleteNS(ctx context.Context, domain registrar.DomainName, host string) error
	NSInfo(ctx context.Context, domain registrar.DomainName, host string) (dns.RegisteredNS, error)
}
