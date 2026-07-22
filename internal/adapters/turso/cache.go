package turso

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// Cache returns the CacheRepo backed by this store.
func (s *Store) Cache() ports.CacheRepo { return &cacheRepo{db: s.db} }

type cacheRepo struct{ db *sql.DB }

func (r *cacheRepo) Get(ctx context.Context, profile, kind, key string, maxAge time.Duration) ([]byte, bool, error) {
	var payload []byte
	var fetchedAt int64
	err := r.db.QueryRowContext(ctx, `
		SELECT payload, fetched_at FROM cache
		WHERE profile = ? AND kind = ? AND key = ?`, profile, kind, key).
		Scan(&payload, &fetchedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if time.Since(time.Unix(fetchedAt, 0)) >= maxAge {
		return nil, false, nil
	}
	return payload, true, nil
}

func (r *cacheRepo) Put(ctx context.Context, profile, kind, key string, payload []byte) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO cache (profile, kind, key, payload, fetched_at) VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(profile, kind, key) DO UPDATE SET
			payload = excluded.payload, fetched_at = excluded.fetched_at`,
		profile, kind, key, payload, time.Now().Unix())
	return err
}

func (r *cacheRepo) Invalidate(ctx context.Context, profile, kind string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM cache WHERE profile = ? AND kind = ?`, profile, kind)
	return err
}
