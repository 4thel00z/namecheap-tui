package iputil_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/adapters/iputil"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.IPResolver = (*iputil.Resolver)(nil)

func TestPublicIPTrimsAndReturns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("203.0.113.9\n"))
	}))
	defer srv.Close()
	r := iputil.New(iputil.WithURLs(srv.URL))
	ip, err := r.PublicIP(context.Background())
	if err != nil || ip != "203.0.113.9" {
		t.Errorf("PublicIP = %q, %v", ip, err)
	}
}

func TestPublicIPFallsBack(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("198.51.100.4"))
	}))
	defer good.Close()
	r := iputil.New(iputil.WithURLs(bad.URL, good.URL))
	ip, err := r.PublicIP(context.Background())
	if err != nil || ip != "198.51.100.4" {
		t.Errorf("PublicIP = %q, %v", ip, err)
	}
}

func TestPublicIPRejectsGarbage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>nope</html>"))
	}))
	defer srv.Close()
	r := iputil.New(iputil.WithURLs(srv.URL))
	if _, err := r.PublicIP(context.Background()); err == nil {
		t.Error("want error on non-IP body")
	}
}
