// Package iputil detects the caller's public IPv4 address.
package iputil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	λ "github.com/4thel00z/lambda/v2"
)

// Resolver queries well-known what's-my-ip services in order.
type Resolver struct {
	urls []string
	http *http.Client
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithURLs overrides the service URLs (first success wins).
func WithURLs(urls ...string) Option { return func(r *Resolver) { r.urls = urls } }

// New builds a Resolver with icanhazip + ipify defaults.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		urls: []string{"https://ipv4.icanhazip.com", "https://api.ipify.org"},
		http: &http.Client{Timeout: 10 * time.Second},
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// PublicIP returns the first valid IPv4 address any service reports.
func (r *Resolver) PublicIP(ctx context.Context) (string, error) {
	var errs []error
	for _, u := range r.urls {
		body, err := λ.Client(r.http).Get(u).Do(ctx).Slurp().Get()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", u, err))
			continue
		}
		ip := strings.TrimSpace(string(body))
		addr, err := netip.ParseAddr(ip)
		if err != nil || !addr.Is4() {
			errs = append(errs, fmt.Errorf("%s: invalid response %q", u, ip))
			continue
		}
		return ip, nil
	}
	return "", fmt.Errorf("public ip detection failed: %w", errors.Join(errs...))
}
