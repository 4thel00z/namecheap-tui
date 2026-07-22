package namecheap

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

func testCreds() account.Credentials {
	return account.Credentials{
		Profile: account.Profile{
			Name: "t", APIUser: "apiu", Username: "usern",
			ClientIP: "203.0.113.7", Endpoint: account.EndpointSandbox,
		},
		APIKey: "key123",
	}
}

const okBody = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.check">
    <DomainCheckResult Domain="example.com" Available="true" />
  </CommandResponse>
</ApiResponse>`

func TestCallSendsAuthParams(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(okBody))
	}))
	defer srv.Close()

	c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
	var out struct {
		XMLName xml.Name `xml:"ApiResponse"`
	}
	params := url.Values{}
	params.Set("DomainList", "example.com")
	if err := c.call(context.Background(), "namecheap.domains.check", params, &out); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"ApiUser":    "apiu",
		"ApiKey":     "key123",
		"UserName":   "usern",
		"ClientIp":   "203.0.113.7",
		"Command":    "namecheap.domains.check",
		"DomainList": "example.com",
	} {
		if got := gotQuery.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
}

func errBody(number, msg string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="ERROR" xmlns="http://api.namecheap.com/xml.response">
  <Errors><Error Number="` + number + `">` + msg + `</Error></Errors>
</ApiResponse>`
}

func TestCallMapsErrors(t *testing.T) {
	cases := []struct {
		number string
		want   error
	}{
		{"1011147", ports.ErrIPNotWhitelisted},
		{"1011102", ports.ErrAuth},
		{"1010104", ports.ErrAuth},
		{"500000", ports.ErrRateLimited},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(errBody(tc.number, "boom")))
		}))
		c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
		err := c.call(context.Background(), "cmd", nil, &struct{}{})
		if !errors.Is(err, tc.want) {
			t.Errorf("number %s: err = %v, want %v", tc.number, err, tc.want)
		}
		srv.Close()
	}
}

func TestCallUnknownErrorIsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(errBody("2030280", "TLD is not supported")))
	}))
	defer srv.Close()
	c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
	err := c.call(context.Background(), "cmd", nil, &struct{}{})
	var apiErr *ports.APIError
	if !errors.As(err, &apiErr) || apiErr.Number != 2030280 {
		t.Errorf("err = %v, want APIError 2030280", err)
	}
}

func TestCallHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
	if err := c.call(context.Background(), "cmd", nil, &struct{}{}); err == nil {
		t.Error("want error on HTTP 502, got nil")
	}
}
