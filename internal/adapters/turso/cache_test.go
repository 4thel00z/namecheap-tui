package turso_test

import (
	"context"
	"testing"
	"time"
)

func TestCache(t *testing.T) {
	ctx := context.Background()
	c := testStore(t).Cache()

	if _, ok, err := c.Get(ctx, "p", "domains", "list", time.Minute); ok || err != nil {
		t.Errorf("empty cache: ok=%v err=%v", ok, err)
	}
	if err := c.Put(ctx, "p", "domains", "list", []byte(`["a"]`)); err != nil {
		t.Fatal(err)
	}
	got, ok, err := c.Get(ctx, "p", "domains", "list", time.Minute)
	if err != nil || !ok || string(got) != `["a"]` {
		t.Errorf("fresh get = %q ok=%v err=%v", got, ok, err)
	}
	// maxAge 0 means everything is stale.
	if _, ok, _ := c.Get(ctx, "p", "domains", "list", 0); ok {
		t.Error("zero maxAge should miss")
	}
	if err := c.Invalidate(ctx, "p", "domains"); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := c.Get(ctx, "p", "domains", "list", time.Minute); ok {
		t.Error("invalidated entry should miss")
	}
}
