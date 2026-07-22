package ports

import (
	"context"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
)

// ProfileRepo persists named account profiles and their secrets.
type ProfileRepo interface {
	Save(ctx context.Context, creds account.Credentials) error
	Get(ctx context.Context, name string) (account.Credentials, error)
	List(ctx context.Context) ([]account.Profile, error)
	Delete(ctx context.Context, name string) error
	SetDefault(ctx context.Context, name string) error
	Default(ctx context.Context) (account.Credentials, error)
}

// SettingsRepo is a string key/value store. Get returns "" for unset keys.
type SettingsRepo interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key, value string) error
}

// CacheRepo stores API payloads per profile+kind+key with staleness control.
type CacheRepo interface {
	Get(ctx context.Context, profile, kind, key string, maxAge time.Duration) (payload []byte, ok bool, err error)
	Put(ctx context.Context, profile, kind, key string, payload []byte) error
	Invalidate(ctx context.Context, profile, kind string) error
}

// IPResolver detects the caller's public IPv4 address.
type IPResolver interface {
	PublicIP(ctx context.Context) (string, error)
}
