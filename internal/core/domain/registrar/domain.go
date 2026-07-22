// Package registrar holds the domain-registration aggregate: registered
// domains, availability results, and the DomainName value object.
package registrar

import (
	"fmt"
	"strings"
	"time"
)

// DomainName is a validated second-level + top-level domain pair.
type DomainName struct {
	SLD string
	TLD string
}

// Parse validates and normalizes a domain name into a DomainName.
// The TLD may be multi-label (e.g. "co.uk"): everything after the first dot.
func Parse(name string) (DomainName, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	sld, tld, ok := strings.Cut(n, ".")
	if !ok || sld == "" || tld == "" || strings.ContainsAny(n, " \t") {
		return DomainName{}, fmt.Errorf("invalid domain name %q", name)
	}
	return DomainName{SLD: sld, TLD: tld}, nil
}

func (d DomainName) String() string { return d.SLD + "." + d.TLD }

// Domain is a registered domain as listed by the registrar.
type Domain struct {
	ID        string
	Name      DomainName
	Owner     string
	Created   time.Time
	Expires   time.Time
	AutoRenew bool
	Locked    bool
	Privacy   bool
	Expired   bool
}

// Availability is the result of a domain availability check.
type Availability struct {
	Name         DomainName
	Available    bool
	Premium      bool
	PremiumPrice float64
}

// Details is the full per-domain info from the registrar.
type Details struct {
	Domain
	Status      string
	DNSProvider string
	Nameservers []string
}
