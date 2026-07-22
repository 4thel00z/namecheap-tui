package turso_test

import (
	"context"
	"testing"
)

func TestSettings(t *testing.T) {
	ctx := context.Background()
	s := testStore(t).Settings()

	v, err := s.Get(ctx, "missing")
	if err != nil || v != "" {
		t.Errorf("Get missing = %q, %v; want \"\", nil", v, err)
	}
	if err := s.Set(ctx, "k", "v1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Set(ctx, "k", "v2"); err != nil {
		t.Fatal(err)
	}
	v, _ = s.Get(ctx, "k")
	if v != "v2" {
		t.Errorf("Get = %q, want v2", v)
	}
}
