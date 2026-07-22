// Package namecheap implements the API ports against the Namecheap XML API.
package namecheap

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"time"

	λ "github.com/4thel00z/lambda/v2"
	"golang.org/x/time/rate"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// Client talks to the Namecheap XML API for one account profile.
type Client struct {
	creds    account.Credentials
	http     *http.Client
	baseURL  string
	limiters []*rate.Limiter
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the underlying HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithBaseURL overrides the endpoint (tests).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithNoThrottle disables client-side rate limiting (tests).
func WithNoThrottle() Option { return func(c *Client) { c.limiters = nil } }

// New builds a Client bound to the given credentials.
func New(creds account.Credentials, opts ...Option) *Client {
	c := &Client{
		creds:   creds,
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: string(creds.Endpoint),
		// Namecheap's published limits: 20/min, 700/hr, 8000/day.
		limiters: []*rate.Limiter{
			rate.NewLimiter(rate.Every(time.Minute/20), 1),
			rate.NewLimiter(rate.Every(time.Hour/700), 5),
			rate.NewLimiter(rate.Every(24*time.Hour/8000), 20),
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

type envelope struct {
	XMLName xml.Name    `xml:"ApiResponse"`
	Status  string      `xml:"Status,attr"`
	Errors  []xmlAPIErr `xml:"Errors>Error"`
}

type xmlAPIErr struct {
	Number  int    `xml:"Number,attr"`
	Message string `xml:",chardata"`
}

// call executes one API command and unmarshals the full response into out.
func (c *Client) call(ctx context.Context, command string, params url.Values, out any) error {
	for _, l := range c.limiters {
		if err := l.Wait(ctx); err != nil {
			return err
		}
	}
	q := url.Values{}
	q.Set("ApiUser", c.creds.APIUser)
	q.Set("ApiKey", c.creds.APIKey)
	q.Set("UserName", c.creds.Username)
	q.Set("ClientIp", c.creds.ClientIP)
	q.Set("Command", command)
	for k, vs := range params {
		for _, v := range vs {
			q.Add(k, v)
		}
	}

	resp := λ.Client(c.http).Get(c.baseURL + "?" + q.Encode()).Do(ctx)
	status, err := resp.StatusCode().Get()
	if err != nil {
		return fmt.Errorf("%s: %w", command, err)
	}
	body, err := resp.Slurp().Get()
	if err != nil {
		return fmt.Errorf("%s: read body: %w", command, err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("%s: unexpected HTTP status %d", command, status)
	}

	var env envelope
	if err := xml.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("%s: parse response: %w", command, err)
	}
	if env.Status != "OK" {
		return fmt.Errorf("%s: %w", command, mapAPIError(env.Errors))
	}
	if err := xml.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: parse command response: %w", command, err)
	}
	return nil
}

// mapAPIError converts Namecheap error numbers into the port error contract.
func mapAPIError(errs []xmlAPIErr) error {
	if len(errs) == 0 {
		return &ports.APIError{Number: -1, Message: "unknown API error"}
	}
	e := errs[0]
	switch e.Number {
	case 1011147:
		return fmt.Errorf("%s: %w", e.Message, ports.ErrIPNotWhitelisted)
	case 1011102, 1010104:
		return fmt.Errorf("%s: %w", e.Message, ports.ErrAuth)
	case 500000:
		return fmt.Errorf("%s: %w", e.Message, ports.ErrRateLimited)
	default:
		return &ports.APIError{Number: e.Number, Message: e.Message}
	}
}
