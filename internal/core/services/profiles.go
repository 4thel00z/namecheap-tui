// Package services contains the application use-cases driving the hexagon.
package services

import (
	"context"
	"fmt"
	"os"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// ProfileService manages named account profiles.
type ProfileService struct {
	repo   ports.ProfileRepo
	ip     ports.IPResolver
	verify func(context.Context, account.Credentials) error
}

// NewProfileService builds a ProfileService. verify (nil = skip) is called
// with candidate credentials before saving, e.g. a live domains.getList.
func NewProfileService(repo ports.ProfileRepo, ip ports.IPResolver,
	verify func(context.Context, account.Credentials) error,
) *ProfileService {
	return &ProfileService{repo: repo, ip: ip, verify: verify}
}

// Add validates, live-verifies, and persists a profile.
func (s *ProfileService) Add(ctx context.Context, creds account.Credentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	if s.verify != nil {
		if err := s.verify(ctx, creds); err != nil {
			return fmt.Errorf("credential verification failed: %w", err)
		}
	}
	return s.repo.Save(ctx, creds)
}

// List returns all stored profiles (without secrets).
func (s *ProfileService) List(ctx context.Context) ([]account.Profile, error) {
	return s.repo.List(ctx)
}

// Use marks the named profile as default.
func (s *ProfileService) Use(ctx context.Context, name string) error {
	return s.repo.SetDefault(ctx, name)
}

// Remove deletes the named profile and its secret.
func (s *ProfileService) Remove(ctx context.Context, name string) error {
	return s.repo.Delete(ctx, name)
}

// DetectIP returns the caller's current public IPv4.
func (s *ProfileService) DetectIP(ctx context.Context) (string, error) {
	return s.ip.PublicIP(ctx)
}

// Current resolves the active credentials.
// Precedence: explicit override > env credentials > NCP_PROFILE > stored default.
func (s *ProfileService) Current(ctx context.Context, override string) (account.Credentials, error) {
	if override != "" {
		return s.repo.Get(ctx, override)
	}
	if key := os.Getenv("NAMECHEAP_API_KEY"); key != "" {
		if user := os.Getenv("NAMECHEAP_API_USER"); user != "" {
			return credsFromEnv(key, user)
		}
	}
	if name := os.Getenv("NCP_PROFILE"); name != "" {
		return s.repo.Get(ctx, name)
	}
	return s.repo.Default(ctx)
}

func credsFromEnv(key, user string) (account.Credentials, error) {
	username := os.Getenv("NAMECHEAP_USERNAME")
	if username == "" {
		username = user
	}
	endpoint := account.EndpointProduction
	if os.Getenv("NAMECHEAP_SANDBOX") == "1" {
		endpoint = account.EndpointSandbox
	}
	c := account.Credentials{
		Profile: account.Profile{
			Name: "env", APIUser: user, Username: username,
			ClientIP: os.Getenv("NAMECHEAP_CLIENT_IP"), Endpoint: endpoint,
		},
		APIKey: key,
	}
	if err := c.Validate(); err != nil {
		return account.Credentials{}, fmt.Errorf("environment credentials incomplete: %w", err)
	}
	return c, nil
}
