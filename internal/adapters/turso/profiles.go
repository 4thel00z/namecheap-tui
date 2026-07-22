package turso

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// Profiles returns the ProfileRepo backed by this store.
func (s *Store) Profiles() ports.ProfileRepo { return &profileRepo{db: s.db} }

type profileRepo struct{ db *sql.DB }

func (r *profileRepo) Save(ctx context.Context, creds account.Credentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var count int
	if err := tx.QueryRowContext(ctx, `SELECT COUNT(*) FROM profiles`).Scan(&count); err != nil {
		return err
	}
	isDefault := 0
	if count == 0 {
		isDefault = 1 // first profile becomes the default
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO profiles (name, api_user, username, client_ip, endpoint, is_default)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(name) DO UPDATE SET
			api_user = excluded.api_user, username = excluded.username,
			client_ip = excluded.client_ip, endpoint = excluded.endpoint`,
		creds.Name, creds.APIUser, creds.Username, creds.ClientIP, string(creds.Endpoint), isDefault); err != nil {
		return err
	}
	// LastInsertId is unreliable on upsert-update; resolve the id by name.
	var id int64
	if err := tx.QueryRowContext(ctx,
		`SELECT id FROM profiles WHERE name = ?`, creds.Name).Scan(&id); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO secrets (profile_id, api_key) VALUES (?, ?)
		ON CONFLICT(profile_id) DO UPDATE SET api_key = excluded.api_key`,
		id, creds.APIKey); err != nil {
		return err
	}
	return tx.Commit()
}

const credsQuery = `
	SELECT p.name, p.api_user, p.username, p.client_ip, p.endpoint, p.is_default, s.api_key
	FROM profiles p JOIN secrets s ON s.profile_id = p.id`

func scanCreds(row *sql.Row) (account.Credentials, error) {
	var c account.Credentials
	var isDefault int
	var endpoint string
	err := row.Scan(&c.Name, &c.APIUser, &c.Username, &c.ClientIP, &endpoint, &isDefault, &c.APIKey)
	if err != nil {
		return account.Credentials{}, err
	}
	c.Endpoint = account.Endpoint(endpoint)
	c.IsDefault = isDefault == 1
	return c, nil
}

func (r *profileRepo) Get(ctx context.Context, name string) (account.Credentials, error) {
	c, err := scanCreds(r.db.QueryRowContext(ctx, credsQuery+` WHERE p.name = ?`, name))
	if errors.Is(err, sql.ErrNoRows) {
		return account.Credentials{}, fmt.Errorf("%q: %w", name, ports.ErrProfileNotFound)
	}
	return c, err
}

func (r *profileRepo) Default(ctx context.Context) (account.Credentials, error) {
	c, err := scanCreds(r.db.QueryRowContext(ctx, credsQuery+` WHERE p.is_default = 1`))
	if errors.Is(err, sql.ErrNoRows) {
		return account.Credentials{}, ports.ErrNoDefaultProfile
	}
	return c, err
}

func (r *profileRepo) List(ctx context.Context) ([]account.Profile, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT name, api_user, username, client_ip, endpoint, is_default
		FROM profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []account.Profile
	for rows.Next() {
		var p account.Profile
		var isDefault int
		var endpoint string
		if err := rows.Scan(&p.Name, &p.APIUser, &p.Username, &p.ClientIP, &endpoint, &isDefault); err != nil {
			return nil, err
		}
		p.Endpoint = account.Endpoint(endpoint)
		p.IsDefault = isDefault == 1
		out = append(out, p)
	}
	return out, rows.Err()
}

func (r *profileRepo) Delete(ctx context.Context, name string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM profiles WHERE name = ?`, name)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%q: %w", name, ports.ErrProfileNotFound)
	}
	return nil
}

func (r *profileRepo) SetDefault(ctx context.Context, name string) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.ExecContext(ctx, `UPDATE profiles SET is_default = 1 WHERE name = ?`, name)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("%q: %w", name, ports.ErrProfileNotFound)
	}
	if _, err := tx.ExecContext(ctx, `UPDATE profiles SET is_default = 0 WHERE name != ?`, name); err != nil {
		return err
	}
	return tx.Commit()
}
