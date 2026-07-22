// Package turso implements the storage ports on libSQL (local file or
// Turso embedded replica).
package turso

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tursodatabase/go-libsql"
)

// Store owns the libSQL handle and hands out port implementations.
type Store struct {
	db        *sql.DB
	connector *libsql.Connector // nil for plain local files
}

// Open opens (creating if needed) a plain local libSQL database and
// applies migrations. The parent directory is created 0700, the file 0600.
func Open(ctx context.Context, path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	db, err := sql.Open("libsql", "file:"+path)
	if err != nil {
		return nil, fmt.Errorf("open libsql: %w", err)
	}
	return finishOpen(ctx, db, nil, path)
}

// OpenReplica opens an embedded replica that syncs to a Turso primary.
func OpenReplica(ctx context.Context, path, primaryURL, authToken string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, err
	}
	connector, err := libsql.NewEmbeddedReplicaConnector(path, primaryURL,
		libsql.WithAuthToken(authToken))
	if err != nil {
		return nil, fmt.Errorf("open embedded replica: %w", err)
	}
	return finishOpen(ctx, sql.OpenDB(connector), connector, path)
}

func finishOpen(ctx context.Context, db *sql.DB, connector *libsql.Connector, path string) (*Store, error) {
	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := migrate(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	_ = os.Chmod(path, 0o600)
	return &Store{db: db, connector: connector}, nil
}

// Sync pushes/pulls the embedded replica; no-op for local-only stores.
func (s *Store) Sync() error {
	if s.connector == nil {
		return nil
	}
	_, err := s.connector.Sync()
	return err
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }
