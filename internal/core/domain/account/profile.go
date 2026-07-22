// Package account holds profiles, credentials, balances and address-book types.
package account

import (
	"errors"
	"fmt"
	"net/netip"
)

// Endpoint is a Namecheap API base URL.
type Endpoint string

const (
	EndpointProduction Endpoint = "https://api.namecheap.com/xml.response"
	EndpointSandbox    Endpoint = "https://api.sandbox.namecheap.com/xml.response"
)

// Profile is a named Namecheap account configuration (sans secret).
type Profile struct {
	Name      string
	APIUser   string
	Username  string
	ClientIP  string
	Endpoint  Endpoint
	IsDefault bool
}

// Validate checks all fields are present and well-formed.
func (p Profile) Validate() error {
	switch {
	case p.Name == "":
		return errors.New("profile name is required")
	case p.APIUser == "":
		return errors.New("api user is required")
	case p.Username == "":
		return errors.New("username is required")
	}
	addr, err := netip.ParseAddr(p.ClientIP)
	if err != nil || !addr.Is4() {
		return fmt.Errorf("client ip %q is not a valid IPv4 address (Namecheap whitelists IPv4 only)", p.ClientIP)
	}
	if p.Endpoint != EndpointProduction && p.Endpoint != EndpointSandbox {
		return fmt.Errorf("unknown endpoint %q", p.Endpoint)
	}
	return nil
}

// Credentials is a Profile plus its API key.
type Credentials struct {
	Profile
	APIKey string
}

// Validate checks the profile and the API key.
func (c Credentials) Validate() error {
	if err := c.Profile.Validate(); err != nil {
		return err
	}
	if c.APIKey == "" {
		return errors.New("api key is required")
	}
	return nil
}
