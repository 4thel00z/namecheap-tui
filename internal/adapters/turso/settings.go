package turso

import (
	"context"
	"database/sql"
	"errors"

	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// Settings returns the SettingsRepo backed by this store.
func (s *Store) Settings() ports.SettingsRepo { return &settingsRepo{db: s.db} }

type settingsRepo struct{ db *sql.DB }

func (r *settingsRepo) Get(ctx context.Context, key string) (string, error) {
	var v string
	err := r.db.QueryRowContext(ctx, `SELECT value FROM settings WHERE key = ?`, key).Scan(&v)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return v, err
}

func (r *settingsRepo) Set(ctx context.Context, key, value string) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
