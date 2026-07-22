package cli_test

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

// in-memory ProfileRepo fake (mirrors the services test fake)
type memRepo struct {
	byName  map[string]account.Credentials
	defName string
}

func newMemRepo() *memRepo { return &memRepo{byName: map[string]account.Credentials{}} }

func (f *memRepo) Save(_ context.Context, c account.Credentials) error {
	if len(f.byName) == 0 {
		f.defName = c.Name
	}
	f.byName[c.Name] = c
	return nil
}

func (f *memRepo) Get(_ context.Context, n string) (account.Credentials, error) {
	c, ok := f.byName[n]
	if !ok {
		return account.Credentials{}, ports.ErrProfileNotFound
	}
	return c, nil
}

func (f *memRepo) List(_ context.Context) ([]account.Profile, error) {
	var out []account.Profile
	for _, c := range f.byName {
		p := c.Profile
		p.IsDefault = c.Name == f.defName
		out = append(out, p)
	}
	return out, nil
}

func (f *memRepo) Delete(_ context.Context, n string) error {
	if _, ok := f.byName[n]; !ok {
		return ports.ErrProfileNotFound
	}
	delete(f.byName, n)
	return nil
}

func (f *memRepo) SetDefault(_ context.Context, n string) error {
	if _, ok := f.byName[n]; !ok {
		return ports.ErrProfileNotFound
	}
	f.defName = n
	return nil
}

func (f *memRepo) Default(_ context.Context) (account.Credentials, error) {
	if f.defName == "" {
		return account.Credentials{}, ports.ErrNoDefaultProfile
	}
	return f.byName[f.defName], nil
}

func run(t *testing.T, app *cli.App, args ...string) (string, error) {
	t.Helper()
	root := cli.Root(app)
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(args)
	err := root.ExecuteContext(context.Background())
	return buf.String(), err
}

func testApp(repo *memRepo) *cli.App {
	return &cli.App{
		Profiles: services.NewProfileService(repo, nil, nil),
		Version:  "test",
	}
}

func TestProfileAddListRmViaFlags(t *testing.T) {
	repo := newMemRepo()
	app := testApp(repo)

	_, err := run(t, app, "profile", "add",
		"--name", "work", "--api-user", "au", "--username", "un",
		"--api-key", "k", "--client-ip", "203.0.113.7")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := repo.byName["work"]; !ok {
		t.Fatal("profile not saved")
	}

	out, err := run(t, app, "profile", "list")
	if err != nil || !strings.Contains(out, "work") {
		t.Errorf("list output %q, %v", out, err)
	}

	out, err = run(t, app, "profile", "list", "--json")
	if err != nil || !strings.Contains(out, `"name": "work"`) {
		t.Errorf("json list output %q, %v", out, err)
	}

	if _, err := run(t, app, "profile", "rm", "work"); err != nil {
		t.Fatal(err)
	}
	if len(repo.byName) != 0 {
		t.Error("profile not deleted")
	}
}

func TestProfileUse(t *testing.T) {
	repo := newMemRepo()
	app := testApp(repo)
	for _, n := range []string{"a", "b"} {
		if _, err := run(t, app, "profile", "add",
			"--name", n, "--api-user", "au", "--username", "un",
			"--api-key", "k", "--client-ip", "203.0.113.7"); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := run(t, app, "profile", "use", "b"); err != nil {
		t.Fatal(err)
	}
	if repo.defName != "b" {
		t.Errorf("default = %q, want b", repo.defName)
	}
}

func TestProfileAddSandboxFlag(t *testing.T) {
	repo := newMemRepo()
	app := testApp(repo)
	if _, err := run(t, app, "profile", "add",
		"--name", "sb", "--api-user", "au", "--username", "un",
		"--api-key", "k", "--client-ip", "203.0.113.7", "--sandbox"); err != nil {
		t.Fatal(err)
	}
	if repo.byName["sb"].Endpoint != account.EndpointSandbox {
		t.Errorf("endpoint = %v", repo.byName["sb"].Endpoint)
	}
}
