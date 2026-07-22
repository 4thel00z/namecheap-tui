package turso_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/adapters/turso"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

func testStore(t *testing.T) *turso.Store {
	t.Helper()
	s, err := turso.Open(context.Background(), filepath.Join(t.TempDir(), "ncp.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func creds(name string) account.Credentials {
	return account.Credentials{
		Profile: account.Profile{
			Name: name, APIUser: "au", Username: "un",
			ClientIP: "203.0.113.7", Endpoint: account.EndpointSandbox,
		},
		APIKey: "secret-" + name,
	}
}

func TestProfileRoundtrip(t *testing.T) {
	ctx := context.Background()
	repo := testStore(t).Profiles()

	if err := repo.Save(ctx, creds("a")); err != nil {
		t.Fatal(err)
	}
	got, err := repo.Get(ctx, "a")
	if err != nil {
		t.Fatal(err)
	}
	if got.APIKey != "secret-a" || got.Username != "un" || got.Endpoint != account.EndpointSandbox {
		t.Errorf("roundtrip mismatch: %+v", got)
	}

	// Upsert overwrites.
	c2 := creds("a")
	c2.APIKey = "rotated"
	if err := repo.Save(ctx, c2); err != nil {
		t.Fatal(err)
	}
	got, _ = repo.Get(ctx, "a")
	if got.APIKey != "rotated" {
		t.Errorf("upsert did not rotate key: %+v", got)
	}
}

func TestProfileDefaultAndList(t *testing.T) {
	ctx := context.Background()
	repo := testStore(t).Profiles()

	// First saved profile becomes default automatically.
	if err := repo.Save(ctx, creds("a")); err != nil {
		t.Fatal(err)
	}
	if err := repo.Save(ctx, creds("b")); err != nil {
		t.Fatal(err)
	}
	d, err := repo.Default(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if d.Name != "a" {
		t.Errorf("default = %q, want a", d.Name)
	}

	if err := repo.SetDefault(ctx, "b"); err != nil {
		t.Fatal(err)
	}
	d, _ = repo.Default(ctx)
	if d.Name != "b" {
		t.Errorf("default after SetDefault = %q, want b", d.Name)
	}

	list, err := repo.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}
}

func TestProfileDeleteAndMissing(t *testing.T) {
	ctx := context.Background()
	repo := testStore(t).Profiles()

	if _, err := repo.Get(ctx, "ghost"); !errors.Is(err, ports.ErrProfileNotFound) {
		t.Errorf("Get missing: err = %v, want ErrProfileNotFound", err)
	}
	if _, err := repo.Default(ctx); !errors.Is(err, ports.ErrNoDefaultProfile) {
		t.Errorf("Default on empty: err = %v, want ErrNoDefaultProfile", err)
	}
	if err := repo.Save(ctx, creds("a")); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, "a"); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Get(ctx, "a"); !errors.Is(err, ports.ErrProfileNotFound) {
		t.Errorf("Get deleted: err = %v, want ErrProfileNotFound", err)
	}
}
