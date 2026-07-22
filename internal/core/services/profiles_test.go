package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

// ---- fakes shared across service tests ----

type fakeProfileRepo struct {
	byName  map[string]account.Credentials
	defName string
}

func newFakeProfileRepo() *fakeProfileRepo {
	return &fakeProfileRepo{byName: map[string]account.Credentials{}}
}

func (f *fakeProfileRepo) Save(_ context.Context, c account.Credentials) error {
	if len(f.byName) == 0 {
		f.defName = c.Name
	}
	f.byName[c.Name] = c
	return nil
}

func (f *fakeProfileRepo) Get(_ context.Context, name string) (account.Credentials, error) {
	c, ok := f.byName[name]
	if !ok {
		return account.Credentials{}, ports.ErrProfileNotFound
	}
	return c, nil
}

func (f *fakeProfileRepo) List(_ context.Context) ([]account.Profile, error) {
	var out []account.Profile
	for _, c := range f.byName {
		out = append(out, c.Profile)
	}
	return out, nil
}

func (f *fakeProfileRepo) Delete(_ context.Context, name string) error {
	if _, ok := f.byName[name]; !ok {
		return ports.ErrProfileNotFound
	}
	delete(f.byName, name)
	return nil
}

func (f *fakeProfileRepo) SetDefault(_ context.Context, name string) error {
	if _, ok := f.byName[name]; !ok {
		return ports.ErrProfileNotFound
	}
	f.defName = name
	return nil
}

func (f *fakeProfileRepo) Default(_ context.Context) (account.Credentials, error) {
	if f.defName == "" {
		return account.Credentials{}, ports.ErrNoDefaultProfile
	}
	return f.byName[f.defName], nil
}

type fakeIP struct{ ip string }

func (f fakeIP) PublicIP(context.Context) (string, error) { return f.ip, nil }

func testCreds(name string) account.Credentials {
	return account.Credentials{
		Profile: account.Profile{
			Name: name, APIUser: "au", Username: "un",
			ClientIP: "203.0.113.7", Endpoint: account.EndpointProduction,
		},
		APIKey: "k",
	}
}

// ---- tests ----

func TestAddValidatesAndVerifies(t *testing.T) {
	repo := newFakeProfileRepo()
	verified := false
	svc := services.NewProfileService(repo, fakeIP{"1.2.3.4"},
		func(context.Context, account.Credentials) error { verified = true; return nil })

	if err := svc.Add(context.Background(), testCreds("a")); err != nil {
		t.Fatal(err)
	}
	if !verified {
		t.Error("verify was not called")
	}
	bad := testCreds("b")
	bad.APIKey = ""
	if err := svc.Add(context.Background(), bad); err == nil {
		t.Error("invalid creds accepted")
	}
}

func TestAddRejectsWhenVerifyFails(t *testing.T) {
	repo := newFakeProfileRepo()
	svc := services.NewProfileService(repo, nil,
		func(context.Context, account.Credentials) error { return errors.New("nope") })
	if err := svc.Add(context.Background(), testCreds("a")); err == nil {
		t.Fatal("want verify error")
	}
	if len(repo.byName) != 0 {
		t.Error("profile saved despite failed verification")
	}
}

func TestCurrentPrecedence(t *testing.T) {
	repo := newFakeProfileRepo()
	svc := services.NewProfileService(repo, nil, nil)
	ctx := context.Background()
	_ = repo.Save(ctx, testCreds("def"))
	_ = repo.Save(ctx, testCreds("other"))

	// Isolate from the developer's real environment.
	for _, k := range []string{
		"NAMECHEAP_API_KEY", "NAMECHEAP_API_USER",
		"NAMECHEAP_USERNAME", "NAMECHEAP_CLIENT_IP", "NAMECHEAP_SANDBOX", "NCP_PROFILE",
	} {
		t.Setenv(k, "")
	}

	// default
	c, err := svc.Current(ctx, "")
	if err != nil || c.Name != "def" {
		t.Errorf("default: %q, %v", c.Name, err)
	}

	// NCP_PROFILE env
	t.Setenv("NCP_PROFILE", "other")
	c, _ = svc.Current(ctx, "")
	if c.Name != "other" {
		t.Errorf("NCP_PROFILE: %q", c.Name)
	}

	// env credentials beat NCP_PROFILE
	t.Setenv("NAMECHEAP_API_KEY", "envkey")
	t.Setenv("NAMECHEAP_API_USER", "envuser")
	t.Setenv("NAMECHEAP_CLIENT_IP", "198.51.100.1")
	c, err = svc.Current(ctx, "")
	if err != nil || c.APIKey != "envkey" || c.Username != "envuser" {
		t.Errorf("env creds: %+v, %v", c, err)
	}
	if c.Endpoint != account.EndpointProduction {
		t.Errorf("env endpoint: %v", c.Endpoint)
	}
	t.Setenv("NAMECHEAP_SANDBOX", "1")
	c, _ = svc.Current(ctx, "")
	if c.Endpoint != account.EndpointSandbox {
		t.Errorf("sandbox env ignored: %v", c.Endpoint)
	}

	// explicit override beats everything
	c, err = svc.Current(ctx, "def")
	if err != nil || c.Name != "def" {
		t.Errorf("override: %q, %v", c.Name, err)
	}
}
