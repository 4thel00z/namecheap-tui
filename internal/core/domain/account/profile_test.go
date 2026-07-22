package account_test

import (
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
)

func valid() account.Profile {
	return account.Profile{
		Name:     "default",
		APIUser:  "apiuser",
		Username: "user",
		ClientIP: "203.0.113.7",
		Endpoint: account.EndpointProduction,
	}
}

func TestProfileValidate(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	for name, mutate := range map[string]func(*account.Profile){
		"empty name":     func(p *account.Profile) { p.Name = "" },
		"empty apiuser":  func(p *account.Profile) { p.APIUser = "" },
		"empty username": func(p *account.Profile) { p.Username = "" },
		"bad ip":         func(p *account.Profile) { p.ClientIP = "nope" },
		"ipv6":           func(p *account.Profile) { p.ClientIP = "::1" },
		"bad endpoint":   func(p *account.Profile) { p.Endpoint = "https://evil.example" },
	} {
		p := valid()
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestCredentialsValidate(t *testing.T) {
	c := account.Credentials{Profile: valid(), APIKey: "k"}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}
	c.APIKey = ""
	if err := c.Validate(); err == nil {
		t.Error("empty api key: want error, got nil")
	}
}
