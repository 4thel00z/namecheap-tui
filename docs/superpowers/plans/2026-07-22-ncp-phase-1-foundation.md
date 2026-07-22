# ncp Phase 1 — Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Bootstrap the ncp repo (tooling, CI/CD, hexagon skeleton) through working `ncp profile *`, `ncp domains list|check|info`, and a TUI dashboard shell.

**Architecture:** One hexagon: pure `core/domain` + `core/ports` interfaces + `core/services` use-cases; driven adapters `namecheap` (XML over lambda pipelines), `turso` (libSQL embedded replica), `iputil`; driving adapters `cli` (cobra/fang) and `tui` (bubbletea). Spec: `docs/superpowers/specs/2026-07-22-ncp-namecheap-tui-design.md`.

**Tech Stack:** Go 1.24+ (CGo), cobra + charmbracelet/fang, bubbletea/bubbles/lipgloss/huh, github.com/4thel00z/lambda/v2, tursodatabase/go-libsql, golang.org/x/time/rate.

## Global Constraints

- Module path: `github.com/4thel00z/namecheap-tui`; binary `ncp` from `cmd/ncp`.
- Go ≥ 1.24, `CGO_ENABLED=1` everywhere (go-libsql).
- `core/` imports only stdlib + other `core/` packages. Adapters import `core` only, never each other.
- All commits: conventional commits (`feat:`, `fix:`, `ci:`, `docs:`, `test:`, `chore:`).
- Endpoints: prod `https://api.namecheap.com/xml.response`, sandbox `https://api.sandbox.namecheap.com/xml.response`.
- Rate limits (client-side): 20/min, 700/hr, 8000/day.
- Env overrides: `NAMECHEAP_API_KEY`, `NAMECHEAP_API_USER`, `NAMECHEAP_USERNAME`, `NAMECHEAP_CLIENT_IP`, `NAMECHEAP_SANDBOX=1`, `NCP_PROFILE`.
- DB path: `os.UserConfigDir()/ncp/ncp.db`, dir 0700, file 0600.
- lambda v2 Option API: `.Get() (T, error)`, `λ.Client(h).Get(url).Do(ctx).Slurp()` → `Bytes`.

---

### Task 1: Module bootstrap + Makefile + version package

**Files:**
- Create: `go.mod` (via `go mod init`), `cmd/ncp/main.go`, `internal/version/version.go`, `Makefile`, `.gitignore`

**Interfaces:**
- Produces: `version.Version` (string const), `make build|test|lint|fmt|hooks` targets, `bin/ncp` binary.

- [ ] **Step 1: Init module and write files**

```bash
go mod init github.com/4thel00z/namecheap-tui
```

`internal/version/version.go`:
```go
// Package version holds the ncp version, stamped by release-please.
package version

// Version is the current release version.
const Version = "0.1.0" // x-release-please-version
```

`cmd/ncp/main.go` (stub, replaced in Task 16):
```go
package main

import (
	"fmt"

	"github.com/4thel00z/namecheap-tui/internal/version"
)

func main() {
	fmt.Println("ncp", version.Version)
}
```

`.gitignore`:
```
bin/
*.db
*.db-shm
*.db-wal
.DS_Store
coverage.out
```

`Makefile`:
```makefile
.PHONY: build test lint fmt hooks run clean

build:
	CGO_ENABLED=1 go build -o bin/ncp ./cmd/ncp

test:
	CGO_ENABLED=1 go test -race ./...

lint:
	golangci-lint run

fmt:
	gofumpt -l -w .

hooks:
	pre-commit install --install-hooks
	pre-commit install --hook-type commit-msg

run: build
	./bin/ncp

clean:
	rm -rf bin
```

- [ ] **Step 2: Verify build**

Run: `make build && ./bin/ncp`
Expected: prints `ncp 0.1.0`

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: bootstrap ncp module, version package, and Makefile"
```

---

### Task 2: Lint config, pre-commit hooks, CI, release-please

**Files:**
- Create: `.golangci.yml`, `.pre-commit-config.yaml`, `.github/workflows/ci.yml`, `.github/workflows/release-please.yml`, `release-please-config.json`, `.release-please-manifest.json`, `CHANGELOG.md`

- [ ] **Step 1: Write configs**

`.golangci.yml`:
```yaml
version: "2"
formatters:
  enable:
    - gofumpt
```

`.pre-commit-config.yaml`:
```yaml
repos:
  - repo: https://github.com/pre-commit/pre-commit-hooks
    rev: v5.0.0
    hooks:
      - id: trailing-whitespace
      - id: end-of-file-fixer
      - id: check-yaml
      - id: check-added-large-files
      - id: check-merge-conflict
  - repo: https://github.com/compilerla/conventional-pre-commit
    rev: v4.0.0
    hooks:
      - id: conventional-pre-commit
        stages: [commit-msg]
  - repo: local
    hooks:
      - id: gofumpt
        name: gofumpt
        entry: gofumpt -l -w
        language: system
        types: [go]
      - id: go-vet
        name: go vet
        entry: bash -c 'go vet ./...'
        language: system
        types: [go]
        pass_filenames: false
      - id: go-mod-tidy
        name: go mod tidy
        entry: bash -c 'go mod tidy'
        language: system
        pass_filenames: false
        files: go\.(mod|sum)$
```

`.github/workflows/ci.yml`:
```yaml
name: CI

on:
  push:
    branches: [main, master]
  pull_request:

jobs:
  fmt:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: go install mvdan.cc/gofumpt@latest
      - run: test -z "$(gofumpt -l .)"

  lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - uses: golangci/golangci-lint-action@v8

  test:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: CGO_ENABLED=1 go test -race ./...

  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - run: CGO_ENABLED=1 go build -o bin/ncp ./cmd/ncp
```

`.github/workflows/release-please.yml`:
```yaml
name: release-please

on:
  push:
    branches: [main, master]
  workflow_dispatch:

permissions:
  contents: write
  pull-requests: write

jobs:
  release-please:
    runs-on: ubuntu-latest
    outputs:
      release_created: ${{ steps.rp.outputs.release_created }}
      tag_name: ${{ steps.rp.outputs.tag_name }}
    steps:
      - uses: googleapis/release-please-action@v4
        id: rp
        with:
          config-file: release-please-config.json
          manifest-file: .release-please-manifest.json

  # Chained like konan's PyPI publish: tags created by release-please's
  # GITHUB_TOKEN do not trigger `on: push: tags` workflows.
  binaries:
    needs: release-please
    if: needs.release-please.outputs.release_created == 'true'
    strategy:
      matrix:
        include:
          - { runner: ubuntu-latest,   goos: linux,  goarch: amd64 }
          - { runner: ubuntu-24.04-arm, goos: linux, goarch: arm64 }
          - { runner: macos-latest,    goos: darwin, goarch: arm64 }
          - { runner: macos-13,        goos: darwin, goarch: amd64 }
    runs-on: ${{ matrix.runner }}
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with: { go-version: "1.24" }
      - name: Build (CGo, native runner)
        run: |
          CGO_ENABLED=1 GOOS=${{ matrix.goos }} GOARCH=${{ matrix.goarch }} \
            go build -trimpath -ldflags="-s -w" -o dist/ncp-${{ matrix.goos }}-${{ matrix.goarch }} ./cmd/ncp
          shasum -a 256 dist/ncp-${{ matrix.goos }}-${{ matrix.goarch }} | tee dist/ncp-${{ matrix.goos }}-${{ matrix.goarch }}.sha256
      - uses: softprops/action-gh-release@v2
        with:
          tag_name: ${{ needs.release-please.outputs.tag_name }}
          files: dist/*
```

`release-please-config.json`:
```json
{
  "$schema": "https://raw.githubusercontent.com/googleapis/release-please/main/schemas/config.json",
  "packages": {
    ".": {
      "release-type": "go",
      "package-name": "ncp",
      "include-component-in-tag": false,
      "extra-files": ["internal/version/version.go"]
    }
  }
}
```

`.release-please-manifest.json`:
```json
{
  ".": "0.1.0"
}
```

`CHANGELOG.md`:
```markdown
# Changelog
```

- [ ] **Step 2: Verify tooling** (install if missing: `go install mvdan.cc/gofumpt@latest`, `brew install golangci-lint pre-commit` or equivalents)

Run: `make lint && pre-commit run --all-files`
Expected: both exit 0 (hygiene hooks may auto-fix line endings on first run — re-run until clean)

- [ ] **Step 3: Verify commit-msg hook rejects non-conventional messages**

Run: `make hooks && git commit --allow-empty -m "bad message"`
Expected: FAIL (conventional-pre-commit rejects). Then verify a good message passes and drop it:
`git commit --allow-empty -m "chore: hook check" && git reset --soft HEAD~1`

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "ci: add golangci-lint, pre-commit hooks, CI and release-please pipelines"
```

---

### Task 3: registrar domain types (DomainName VO, Domain, Availability, Details)

**Files:**
- Create: `internal/core/domain/registrar/domain.go`
- Test: `internal/core/domain/registrar/domain_test.go`

**Interfaces:**
- Produces:
  - `func Parse(name string) (DomainName, error)` — lowercases, validates `sld.tld` (tld may be multi-label, e.g. `co.uk`)
  - `type DomainName struct { SLD, TLD string }`, `func (d DomainName) String() string`
  - `type Domain struct { ID string; Name DomainName; Owner string; Created, Expires time.Time; AutoRenew, Locked, Privacy, Expired bool }`
  - `type Availability struct { Name DomainName; Available, Premium bool; PremiumPrice float64 }`
  - `type Details struct { Domain; Status string; DNSProvider string; Nameservers []string }`

- [ ] **Step 1: Write the failing test**

`internal/core/domain/registrar/domain_test.go`:
```go
package registrar_test

import (
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

func TestParse(t *testing.T) {
	cases := []struct {
		in      string
		sld     string
		tld     string
		wantErr bool
	}{
		{"example.com", "example", "com", false},
		{"EXAMPLE.COM", "example", "com", false},
		{"foo.co.uk", "foo", "co.uk", false},
		{"  example.com ", "example", "com", false},
		{"example", "", "", true},
		{"", "", "", true},
		{".com", "", "", true},
		{"exa mple.com", "", "", true},
	}
	for _, c := range cases {
		got, err := registrar.Parse(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("Parse(%q): want error, got %+v", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("Parse(%q): unexpected error %v", c.in, err)
			continue
		}
		if got.SLD != c.sld || got.TLD != c.tld {
			t.Errorf("Parse(%q) = %q/%q, want %q/%q", c.in, got.SLD, got.TLD, c.sld, c.tld)
		}
	}
}

func TestDomainNameString(t *testing.T) {
	d, err := registrar.Parse("foo.co.uk")
	if err != nil {
		t.Fatal(err)
	}
	if d.String() != "foo.co.uk" {
		t.Errorf("String() = %q, want %q", d.String(), "foo.co.uk")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/domain/registrar/`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Write minimal implementation**

`internal/core/domain/registrar/domain.go`:
```go
// Package registrar holds the domain-registration aggregate: registered
// domains, availability results, and the DomainName value object.
package registrar

import (
	"fmt"
	"strings"
	"time"
)

// DomainName is a validated second-level + top-level domain pair.
type DomainName struct {
	SLD string
	TLD string
}

// Parse validates and normalizes a domain name into a DomainName.
// The TLD may be multi-label (e.g. "co.uk"): everything after the first dot.
func Parse(name string) (DomainName, error) {
	n := strings.ToLower(strings.TrimSpace(name))
	sld, tld, ok := strings.Cut(n, ".")
	if !ok || sld == "" || tld == "" || strings.ContainsAny(n, " \t") {
		return DomainName{}, fmt.Errorf("invalid domain name %q", name)
	}
	return DomainName{SLD: sld, TLD: tld}, nil
}

func (d DomainName) String() string { return d.SLD + "." + d.TLD }

// Domain is a registered domain as listed by the registrar.
type Domain struct {
	ID        string
	Name      DomainName
	Owner     string
	Created   time.Time
	Expires   time.Time
	AutoRenew bool
	Locked    bool
	Privacy   bool
	Expired   bool
}

// Availability is the result of a domain availability check.
type Availability struct {
	Name         DomainName
	Available    bool
	Premium      bool
	PremiumPrice float64
}

// Details is the full per-domain info from the registrar.
type Details struct {
	Domain
	Status      string
	DNSProvider string
	Nameservers []string
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/domain/registrar/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/domain/registrar && git commit -m "feat: add registrar domain types with DomainName parsing"
```

---

### Task 4: account domain types (Profile, Credentials, Endpoint)

**Files:**
- Create: `internal/core/domain/account/profile.go`
- Test: `internal/core/domain/account/profile_test.go`

**Interfaces:**
- Produces:
  - `type Endpoint string`; consts `EndpointProduction`, `EndpointSandbox` (exact URLs from Global Constraints)
  - `type Profile struct { Name, APIUser, Username, ClientIP string; Endpoint Endpoint; IsDefault bool }`
  - `func (p Profile) Validate() error` — all fields non-empty, ClientIP parses as IPv4, Endpoint one of the two consts
  - `type Credentials struct { Profile; APIKey string }`; `func (c Credentials) Validate() error` — Profile valid + APIKey non-empty

- [ ] **Step 1: Write the failing test**

`internal/core/domain/account/profile_test.go`:
```go
package account_test

import (
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
)

func valid() account.Profile {
	return account.Profile{
		Name:     "default",
		APIUser:  "apiuser",
		Username: "user",
		ClientIP: "203.0.113.7",
		Endpoint: account.EndpointProduction,
	}
}

func TestProfileValidate(t *testing.T) {
	if err := valid().Validate(); err != nil {
		t.Fatalf("valid profile rejected: %v", err)
	}
	for name, mutate := range map[string]func(*account.Profile){
		"empty name":     func(p *account.Profile) { p.Name = "" },
		"empty apiuser":  func(p *account.Profile) { p.APIUser = "" },
		"empty username": func(p *account.Profile) { p.Username = "" },
		"bad ip":         func(p *account.Profile) { p.ClientIP = "nope" },
		"ipv6":           func(p *account.Profile) { p.ClientIP = "::1" },
		"bad endpoint":   func(p *account.Profile) { p.Endpoint = "https://evil.example" },
	} {
		p := valid()
		mutate(&p)
		if err := p.Validate(); err == nil {
			t.Errorf("%s: want error, got nil", name)
		}
	}
}

func TestCredentialsValidate(t *testing.T) {
	c := account.Credentials{Profile: valid(), APIKey: "k"}
	if err := c.Validate(); err != nil {
		t.Fatalf("valid credentials rejected: %v", err)
	}
	c.APIKey = ""
	if err := c.Validate(); err == nil {
		t.Error("empty api key: want error, got nil")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/domain/account/`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Write minimal implementation**

`internal/core/domain/account/profile.go`:
```go
// Package account holds profiles, credentials, balances and address-book types.
package account

import (
	"errors"
	"fmt"
	"net/netip"
)

// Endpoint is a Namecheap API base URL.
type Endpoint string

const (
	EndpointProduction Endpoint = "https://api.namecheap.com/xml.response"
	EndpointSandbox    Endpoint = "https://api.sandbox.namecheap.com/xml.response"
)

// Profile is a named Namecheap account configuration (sans secret).
type Profile struct {
	Name      string
	APIUser   string
	Username  string
	ClientIP  string
	Endpoint  Endpoint
	IsDefault bool
}

// Validate checks all fields are present and well-formed.
func (p Profile) Validate() error {
	switch {
	case p.Name == "":
		return errors.New("profile name is required")
	case p.APIUser == "":
		return errors.New("api user is required")
	case p.Username == "":
		return errors.New("username is required")
	}
	addr, err := netip.ParseAddr(p.ClientIP)
	if err != nil || !addr.Is4() {
		return fmt.Errorf("client ip %q is not a valid IPv4 address (Namecheap whitelists IPv4 only)", p.ClientIP)
	}
	if p.Endpoint != EndpointProduction && p.Endpoint != EndpointSandbox {
		return fmt.Errorf("unknown endpoint %q", p.Endpoint)
	}
	return nil
}

// Credentials is a Profile plus its API key.
type Credentials struct {
	Profile
	APIKey string
}

// Validate checks the profile and the API key.
func (c Credentials) Validate() error {
	if err := c.Profile.Validate(); err != nil {
		return err
	}
	if c.APIKey == "" {
		return errors.New("api key is required")
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/domain/account/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/domain/account && git commit -m "feat: add account profile and credentials types"
```

---

### Task 5: Ports and typed errors

**Files:**
- Create: `internal/core/ports/api.go`, `internal/core/ports/store.go`, `internal/core/ports/errors.go`
- Test: `internal/core/ports/errors_test.go`

**Interfaces:**
- Produces (consumed by every later task):

```go
// api.go
type RegistrarAPI interface {
	ListDomains(ctx context.Context) ([]registrar.Domain, error)
	CheckDomains(ctx context.Context, names []registrar.DomainName) ([]registrar.Availability, error)
	DomainInfo(ctx context.Context, name registrar.DomainName) (registrar.Details, error)
}

// store.go
type ProfileRepo interface {
	Save(ctx context.Context, creds account.Credentials) error // upsert by name
	Get(ctx context.Context, name string) (account.Credentials, error)
	List(ctx context.Context) ([]account.Profile, error)
	Delete(ctx context.Context, name string) error
	SetDefault(ctx context.Context, name string) error
	Default(ctx context.Context) (account.Credentials, error)
}
type SettingsRepo interface {
	Get(ctx context.Context, key string) (string, error) // "" if unset
	Set(ctx context.Context, key, value string) error
}
type CacheRepo interface {
	Get(ctx context.Context, profile, kind, key string, maxAge time.Duration) (payload []byte, ok bool, err error)
	Put(ctx context.Context, profile, kind, key string, payload []byte) error
	Invalidate(ctx context.Context, profile, kind string) error
}
type IPResolver interface {
	PublicIP(ctx context.Context) (string, error)
}

// errors.go
var ErrProfileNotFound = errors.New("profile not found")
var ErrNoDefaultProfile = errors.New("no default profile configured")
var ErrIPNotWhitelisted = errors.New("client IP is not whitelisted for API access")
var ErrAuth = errors.New("authentication failed")
var ErrRateLimited = errors.New("rate limited by namecheap API")
type APIError struct{ Number int; Message string }
func (e *APIError) Error() string
```

- [ ] **Step 1: Write the failing test**

`internal/core/ports/errors_test.go`:
```go
package ports_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

func TestAPIError(t *testing.T) {
	err := &ports.APIError{Number: 1011147, Message: "Invalid request IP"}
	want := "namecheap api error 1011147: Invalid request IP"
	if err.Error() != want {
		t.Errorf("Error() = %q, want %q", err.Error(), want)
	}
	wrapped := fmt.Errorf("call failed: %w", err)
	var target *ports.APIError
	if !errors.As(wrapped, &target) || target.Number != 1011147 {
		t.Error("errors.As failed to unwrap APIError")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/ports/`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Write implementation**

`internal/core/ports/api.go`:
```go
// Package ports defines the hexagon's driven interfaces and the error
// contract adapters must honor.
package ports

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

// RegistrarAPI is the domain-registration capability of the registrar.
type RegistrarAPI interface {
	// ListDomains returns all domains in the account (all pages merged).
	ListDomains(ctx context.Context) ([]registrar.Domain, error)
	// CheckDomains checks availability for up to 50 names per call.
	CheckDomains(ctx context.Context, names []registrar.DomainName) ([]registrar.Availability, error)
	// DomainInfo returns full details for one domain.
	DomainInfo(ctx context.Context, name registrar.DomainName) (registrar.Details, error)
}
```

`internal/core/ports/store.go`:
```go
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
```

`internal/core/ports/errors.go`:
```go
package ports

import (
	"errors"
	"fmt"
)

var (
	// ErrProfileNotFound is returned when a named profile does not exist.
	ErrProfileNotFound = errors.New("profile not found")
	// ErrNoDefaultProfile is returned when no profile is marked default.
	ErrNoDefaultProfile = errors.New("no default profile configured")
	// ErrIPNotWhitelisted maps Namecheap error 1011147.
	ErrIPNotWhitelisted = errors.New("client IP is not whitelisted for API access")
	// ErrAuth maps Namecheap credential errors (e.g. 1011102, 1010104).
	ErrAuth = errors.New("authentication failed")
	// ErrRateLimited maps Namecheap request-throttling errors.
	ErrRateLimited = errors.New("rate limited by namecheap API")
)

// APIError is a Namecheap API error that has no dedicated sentinel.
type APIError struct {
	Number  int
	Message string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("namecheap api error %d: %s", e.Number, e.Message)
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/ports/ && go vet ./...`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/ports && git commit -m "feat: add hexagon ports and typed API errors"
```

---

### Task 6: Turso store — open, migrations, ProfileRepo

**Files:**
- Create: `internal/adapters/turso/store.go`, `internal/adapters/turso/migrations.go`, `internal/adapters/turso/migrations/0001_init.sql`, `internal/adapters/turso/profiles.go`
- Test: `internal/adapters/turso/profiles_test.go`

**Interfaces:**
- Consumes: `ports.ProfileRepo`, `account.Credentials`, `ports.ErrProfileNotFound`, `ports.ErrNoDefaultProfile`
- Produces:
  - `func Open(ctx context.Context, path string) (*Store, error)` — local libSQL file, applies migrations
  - `func OpenReplica(ctx context.Context, path, primaryURL, authToken string) (*Store, error)` — embedded replica
  - `func (s *Store) Close() error`, `func (s *Store) Sync() error` (no-op locally)
  - `func (s *Store) Profiles() ports.ProfileRepo`
  - Task 7 adds: `func (s *Store) Settings() ports.SettingsRepo`, `func (s *Store) Cache() ports.CacheRepo`

- [ ] **Step 1: Get the dependency**

Run: `go get github.com/tursodatabase/go-libsql@latest && go mod tidy`

- [ ] **Step 2: Write the failing test**

`internal/adapters/turso/profiles_test.go`:
```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/turso/`
Expected: FAIL (package does not exist)

- [ ] **Step 4: Write implementation**

`internal/adapters/turso/migrations/0001_init.sql`:
```sql
CREATE TABLE IF NOT EXISTS profiles (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT NOT NULL UNIQUE,
    api_user   TEXT NOT NULL,
    username   TEXT NOT NULL,
    client_ip  TEXT NOT NULL,
    endpoint   TEXT NOT NULL,
    is_default INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS secrets (
    profile_id INTEGER PRIMARY KEY REFERENCES profiles(id) ON DELETE CASCADE,
    api_key    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS cache (
    profile    TEXT NOT NULL,
    kind       TEXT NOT NULL,
    key        TEXT NOT NULL,
    payload    BLOB NOT NULL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (profile, kind, key)
);
```

`internal/adapters/turso/migrations.go`:
```go
package turso

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

func migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version TEXT PRIMARY KEY)`); err != nil {
		return fmt.Errorf("create schema_migrations: %w", err)
	}
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	sort.Strings(names)
	for _, name := range names {
		var done int
		err := db.QueryRowContext(ctx,
			`SELECT COUNT(*) FROM schema_migrations WHERE version = ?`, name).Scan(&done)
		if err != nil {
			return err
		}
		if done > 0 {
			continue
		}
		body, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		// Split on ";" — the driver may not accept multi-statement execs.
		// Our migration SQL never embeds semicolons in string literals.
		for _, stmt := range strings.Split(string(body), ";") {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if _, err := db.ExecContext(ctx, stmt); err != nil {
				return fmt.Errorf("apply %s: %w", name, err)
			}
		}
		if _, err := db.ExecContext(ctx,
			`INSERT INTO schema_migrations (version) VALUES (?)`, name); err != nil {
			return err
		}
	}
	return nil
}
```

`internal/adapters/turso/store.go`:
```go
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
// NOTE: go-libsql's Connector.Sync signature has changed across versions
// (error vs (Replicated, error)) — adjust to the installed version.
func (s *Store) Sync() error {
	if s.connector == nil {
		return nil
	}
	_, err := s.connector.Sync()
	return err
}

// Close closes the underlying database.
func (s *Store) Close() error { return s.db.Close() }
```

`internal/adapters/turso/profiles.go`:
```go
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
	if errors.Is(err, sql.ErrNoRows) {
		return account.Credentials{}, err
	}
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
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/adapters/turso/`
Expected: PASS (CGo build of go-libsql may take a while on first run)

- [ ] **Step 6: Commit**

```bash
git add internal/adapters/turso go.mod go.sum && git commit -m "feat: add turso store with migrations and profile repository"
```

---

### Task 7: Turso SettingsRepo + CacheRepo

**Files:**
- Create: `internal/adapters/turso/settings.go`, `internal/adapters/turso/cache.go`
- Test: `internal/adapters/turso/settings_test.go`, `internal/adapters/turso/cache_test.go`

**Interfaces:**
- Produces: `(*Store).Settings() ports.SettingsRepo`, `(*Store).Cache() ports.CacheRepo` (contracts from Task 5)

- [ ] **Step 1: Write the failing tests**

`internal/adapters/turso/settings_test.go`:
```go
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
```

`internal/adapters/turso/cache_test.go`:
```go
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/adapters/turso/`
Expected: FAIL (Settings/Cache undefined)

- [ ] **Step 3: Write implementation**

`internal/adapters/turso/settings.go`:
```go
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
```

`internal/adapters/turso/cache.go`:
```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/adapters/turso/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/turso && git commit -m "feat: add turso settings and cache repositories"
```

---

### Task 8: Namecheap XML client core (auth, throttle, envelope, error mapping)

**Files:**
- Create: `internal/adapters/namecheap/client.go`
- Test: `internal/adapters/namecheap/client_test.go`

**Interfaces:**
- Consumes: `account.Credentials`, `ports` errors, `github.com/4thel00z/lambda/v2`
- Produces:
  - `func New(creds account.Credentials, opts ...Option) *Client`
  - `func WithHTTPClient(h *http.Client) Option`, `func WithBaseURL(u string) Option` (tests), `func WithNoThrottle() Option` (tests)
  - unexported `(c *Client) call(ctx context.Context, command string, params url.Values, out any) error` — used by every namespace file (Task 9+)

- [ ] **Step 1: Get dependencies**

Run: `go get github.com/4thel00z/lambda/v2@latest golang.org/x/time@latest && go mod tidy`

- [ ] **Step 2: Write the failing test**

`internal/adapters/namecheap/client_test.go`:
```go
package namecheap

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

func testCreds() account.Credentials {
	return account.Credentials{
		Profile: account.Profile{
			Name: "t", APIUser: "apiu", Username: "usern",
			ClientIP: "203.0.113.7", Endpoint: account.EndpointSandbox,
		},
		APIKey: "key123",
	}
}

const okBody = `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.check">
    <DomainCheckResult Domain="example.com" Available="true" />
  </CommandResponse>
</ApiResponse>`

func TestCallSendsAuthParams(t *testing.T) {
	var gotQuery url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query()
		_, _ = w.Write([]byte(okBody))
	}))
	defer srv.Close()

	c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
	var out struct {
		XMLName xml.Name `xml:"ApiResponse"`
	}
	params := url.Values{}
	params.Set("DomainList", "example.com")
	if err := c.call(context.Background(), "namecheap.domains.check", params, &out); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"ApiUser":    "apiu",
		"ApiKey":     "key123",
		"UserName":   "usern",
		"ClientIp":   "203.0.113.7",
		"Command":    "namecheap.domains.check",
		"DomainList": "example.com",
	} {
		if got := gotQuery.Get(k); got != want {
			t.Errorf("query %s = %q, want %q", k, got, want)
		}
	}
}

func errBody(number, msg string) string {
	return `<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="ERROR" xmlns="http://api.namecheap.com/xml.response">
  <Errors><Error Number="` + number + `">` + msg + `</Error></Errors>
</ApiResponse>`
}

func TestCallMapsErrors(t *testing.T) {
	cases := []struct {
		number string
		want   error
	}{
		{"1011147", ports.ErrIPNotWhitelisted},
		{"1011102", ports.ErrAuth},
		{"1010104", ports.ErrAuth},
		{"500000", ports.ErrRateLimited},
	}
	for _, tc := range cases {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte(errBody(tc.number, "boom")))
		}))
		c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
		err := c.call(context.Background(), "cmd", nil, &struct{}{})
		if !errors.Is(err, tc.want) {
			t.Errorf("number %s: err = %v, want %v", tc.number, err, tc.want)
		}
		srv.Close()
	}
}

func TestCallUnknownErrorIsAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(errBody("2030280", "TLD is not supported")))
	}))
	defer srv.Close()
	c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
	err := c.call(context.Background(), "cmd", nil, &struct{}{})
	var apiErr *ports.APIError
	if !errors.As(err, &apiErr) || apiErr.Number != 2030280 {
		t.Errorf("err = %v, want APIError 2030280", err)
	}
}

func TestCallHTTPStatusError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()
	c := New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
	if err := c.call(context.Background(), "cmd", nil, &struct{}{}); err == nil {
		t.Error("want error on HTTP 502, got nil")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/namecheap/`
Expected: FAIL (package does not exist)

- [ ] **Step 4: Write implementation**

`internal/adapters/namecheap/client.go`:
```go
// Package namecheap implements the API ports against the Namecheap XML API.
package namecheap

import (
	"context"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"time"

	λ "github.com/4thel00z/lambda/v2"
	"golang.org/x/time/rate"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// Client talks to the Namecheap XML API for one account profile.
type Client struct {
	creds    account.Credentials
	http     *http.Client
	baseURL  string
	limiters []*rate.Limiter
}

// Option configures a Client.
type Option func(*Client)

// WithHTTPClient replaces the underlying HTTP client.
func WithHTTPClient(h *http.Client) Option { return func(c *Client) { c.http = h } }

// WithBaseURL overrides the endpoint (tests).
func WithBaseURL(u string) Option { return func(c *Client) { c.baseURL = u } }

// WithNoThrottle disables client-side rate limiting (tests).
func WithNoThrottle() Option { return func(c *Client) { c.limiters = nil } }

// New builds a Client bound to the given credentials.
func New(creds account.Credentials, opts ...Option) *Client {
	c := &Client{
		creds:   creds,
		http:    &http.Client{Timeout: 30 * time.Second},
		baseURL: string(creds.Endpoint),
		// Namecheap's published limits: 20/min, 700/hr, 8000/day.
		limiters: []*rate.Limiter{
			rate.NewLimiter(rate.Every(time.Minute/20), 1),
			rate.NewLimiter(rate.Every(time.Hour/700), 5),
			rate.NewLimiter(rate.Every(24*time.Hour/8000), 20),
		},
	}
	for _, o := range opts {
		o(c)
	}
	return c
}

type envelope struct {
	XMLName xml.Name    `xml:"ApiResponse"`
	Status  string      `xml:"Status,attr"`
	Errors  []xmlAPIErr `xml:"Errors>Error"`
}

type xmlAPIErr struct {
	Number  int    `xml:"Number,attr"`
	Message string `xml:",chardata"`
}

// call executes one API command and unmarshals the full response into out.
func (c *Client) call(ctx context.Context, command string, params url.Values, out any) error {
	for _, l := range c.limiters {
		if err := l.Wait(ctx); err != nil {
			return err
		}
	}
	q := url.Values{}
	q.Set("ApiUser", c.creds.APIUser)
	q.Set("ApiKey", c.creds.APIKey)
	q.Set("UserName", c.creds.Username)
	q.Set("ClientIp", c.creds.ClientIP)
	q.Set("Command", command)
	for k, vs := range params {
		for _, v := range vs {
			q.Add(k, v)
		}
	}

	resp := λ.Client(c.http).Get(c.baseURL + "?" + q.Encode()).Do(ctx)
	status, err := resp.StatusCode().Get()
	if err != nil {
		return fmt.Errorf("%s: %w", command, err)
	}
	body, err := resp.Slurp().Get()
	if err != nil {
		return fmt.Errorf("%s: read body: %w", command, err)
	}
	if status != http.StatusOK {
		return fmt.Errorf("%s: unexpected HTTP status %d", command, status)
	}

	var env envelope
	if err := xml.Unmarshal(body, &env); err != nil {
		return fmt.Errorf("%s: parse response: %w", command, err)
	}
	if env.Status != "OK" {
		return fmt.Errorf("%s: %w", command, mapAPIError(env.Errors))
	}
	if err := xml.Unmarshal(body, out); err != nil {
		return fmt.Errorf("%s: parse command response: %w", command, err)
	}
	return nil
}

// mapAPIError converts Namecheap error numbers into the port error contract.
func mapAPIError(errs []xmlAPIErr) error {
	if len(errs) == 0 {
		return &ports.APIError{Number: -1, Message: "unknown API error"}
	}
	e := errs[0]
	switch e.Number {
	case 1011147:
		return fmt.Errorf("%s: %w", e.Message, ports.ErrIPNotWhitelisted)
	case 1011102, 1010104:
		return fmt.Errorf("%s: %w", e.Message, ports.ErrAuth)
	case 500000:
		return fmt.Errorf("%s: %w", e.Message, ports.ErrRateLimited)
	default:
		return &ports.APIError{Number: e.Number, Message: e.Message}
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/adapters/namecheap/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/adapters/namecheap go.mod go.sum && git commit -m "feat: add namecheap XML client core with throttling and error mapping"
```

---

### Task 9: Namecheap domains adapter (ListDomains, CheckDomains, DomainInfo)

**Files:**
- Create: `internal/adapters/namecheap/domains.go`, `internal/adapters/namecheap/testdata/domains_getlist_p1.xml`, `internal/adapters/namecheap/testdata/domains_getlist_p2.xml`, `internal/adapters/namecheap/testdata/domains_check.xml`, `internal/adapters/namecheap/testdata/domains_getinfo.xml`
- Test: `internal/adapters/namecheap/domains_test.go`

**Interfaces:**
- Consumes: `(c *Client) call`, `registrar` types
- Produces: `*Client` satisfies `ports.RegistrarAPI` (compile-checked with `var _ ports.RegistrarAPI = (*Client)(nil)`)

- [ ] **Step 1: Write fixtures**

`testdata/domains_getlist_p1.xml` (PageSize=20 with TotalItems=21 forces a second page; fixture uses 2 rows for brevity — the paging loop only reads `Paging`):
```xml
<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>
      <Domain ID="1" Name="alpha.com" User="owner" Created="02/15/2020" Expires="02/15/2027" IsExpired="false" IsLocked="false" AutoRenew="true" WhoisGuard="ENABLED" IsPremium="false" IsOurDNS="true" />
      <Domain ID="2" Name="beta.co.uk" User="owner" Created="03/01/2021" Expires="03/01/2026" IsExpired="false" IsLocked="true" AutoRenew="false" WhoisGuard="NOTPRESENT" IsPremium="false" IsOurDNS="false" />
    </DomainGetListResult>
    <Paging>
      <TotalItems>3</TotalItems>
      <CurrentPage>1</CurrentPage>
      <PageSize>2</PageSize>
    </Paging>
  </CommandResponse>
</ApiResponse>
```

`testdata/domains_getlist_p2.xml`:
```xml
<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getList">
    <DomainGetListResult>
      <Domain ID="3" Name="gamma.dev" User="owner" Created="01/10/2026" Expires="01/10/2028" IsExpired="false" IsLocked="false" AutoRenew="true" WhoisGuard="ENABLED" IsPremium="false" IsOurDNS="true" />
    </DomainGetListResult>
    <Paging>
      <TotalItems>3</TotalItems>
      <CurrentPage>2</CurrentPage>
      <PageSize>2</PageSize>
    </Paging>
  </CommandResponse>
</ApiResponse>
```

`testdata/domains_check.xml`:
```xml
<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.check">
    <DomainCheckResult Domain="free.com" Available="true" IsPremiumName="false" PremiumRegistrationPrice="0" />
    <DomainCheckResult Domain="taken.com" Available="false" IsPremiumName="false" PremiumRegistrationPrice="0" />
    <DomainCheckResult Domain="fancy.io" Available="true" IsPremiumName="true" PremiumRegistrationPrice="1200.5" />
  </CommandResponse>
</ApiResponse>
```

`testdata/domains_getinfo.xml`:
```xml
<?xml version="1.0" encoding="utf-8"?>
<ApiResponse Status="OK" xmlns="http://api.namecheap.com/xml.response">
  <Errors />
  <CommandResponse Type="namecheap.domains.getInfo">
    <DomainGetInfoResult Status="Ok" ID="42" DomainName="alpha.com" OwnerName="owner" IsOwner="true" IsPremium="false">
      <DomainDetails>
        <CreatedDate>02/15/2020</CreatedDate>
        <ExpiredDate>02/15/2027</ExpiredDate>
        <NumYears>1</NumYears>
      </DomainDetails>
      <Whoisguard Enabled="True">
        <ID>7</ID>
      </Whoisguard>
      <DnsDetails ProviderType="CUSTOM" IsUsingOurDNS="false">
        <Nameserver>dns1.example.net</Nameserver>
        <Nameserver>dns2.example.net</Nameserver>
      </DnsDetails>
    </DomainGetInfoResult>
  </CommandResponse>
</ApiResponse>
```

- [ ] **Step 2: Write the failing test**

`internal/adapters/namecheap/domains_test.go`:
```go
package namecheap

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
)

func serveFixtures(t *testing.T, pick func(r *http.Request) string) *Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := os.ReadFile("testdata/" + pick(r))
		if err != nil {
			t.Fatalf("fixture: %v", err)
		}
		_, _ = w.Write(body)
	}))
	t.Cleanup(srv.Close)
	return New(testCreds(), WithBaseURL(srv.URL), WithNoThrottle())
}

func TestListDomainsMergesPages(t *testing.T) {
	c := serveFixtures(t, func(r *http.Request) string {
		if r.URL.Query().Get("Page") == "2" {
			return "domains_getlist_p2.xml"
		}
		return "domains_getlist_p1.xml"
	})
	got, err := c.ListDomains(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3 (paging merge)", len(got))
	}
	a := got[0]
	if a.Name.String() != "alpha.com" || !a.AutoRenew || a.Locked || !a.Privacy {
		t.Errorf("alpha.com parsed wrong: %+v", a)
	}
	if a.Expires != time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC) {
		t.Errorf("expires = %v", a.Expires)
	}
	if got[1].Name.TLD != "co.uk" {
		t.Errorf("multi-label TLD lost: %+v", got[1].Name)
	}
	if got[2].ID != "3" {
		t.Errorf("page 2 domain missing: %+v", got[2])
	}
}

func TestCheckDomains(t *testing.T) {
	c := serveFixtures(t, func(*http.Request) string { return "domains_check.xml" })
	names := []registrar.DomainName{
		{SLD: "free", TLD: "com"}, {SLD: "taken", TLD: "com"}, {SLD: "fancy", TLD: "io"},
	}
	got, err := c.CheckDomains(context.Background(), names)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("len = %d, want 3", len(got))
	}
	if !got[0].Available || got[1].Available {
		t.Errorf("availability wrong: %+v", got[:2])
	}
	if !got[2].Premium || got[2].PremiumPrice != 1200.5 {
		t.Errorf("premium wrong: %+v", got[2])
	}
}

func TestDomainInfo(t *testing.T) {
	c := serveFixtures(t, func(*http.Request) string { return "domains_getinfo.xml" })
	got, err := c.DomainInfo(context.Background(), registrar.DomainName{SLD: "alpha", TLD: "com"})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "42" || got.Status != "Ok" || !got.Privacy {
		t.Errorf("details wrong: %+v", got)
	}
	if got.DNSProvider != "CUSTOM" || len(got.Nameservers) != 2 || got.Nameservers[0] != "dns1.example.net" {
		t.Errorf("dns details wrong: %+v", got)
	}
	if got.Created != time.Date(2020, 2, 15, 0, 0, 0, 0, time.UTC) {
		t.Errorf("created = %v", got.Created)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/namecheap/`
Expected: FAIL (ListDomains undefined)

- [ ] **Step 4: Write implementation**

`internal/adapters/namecheap/domains.go`:
```go
package namecheap

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.RegistrarAPI = (*Client)(nil)

// dateLayout is Namecheap's MM/DD/YYYY date format.
const dateLayout = "01/02/2006"

func parseDate(s string) time.Time {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

type xmlDomain struct {
	ID         string `xml:"ID,attr"`
	Name       string `xml:"Name,attr"`
	User       string `xml:"User,attr"`
	Created    string `xml:"Created,attr"`
	Expires    string `xml:"Expires,attr"`
	IsExpired  bool   `xml:"IsExpired,attr"`
	IsLocked   bool   `xml:"IsLocked,attr"`
	AutoRenew  bool   `xml:"AutoRenew,attr"`
	WhoisGuard string `xml:"WhoisGuard,attr"`
}

type getListResponse struct {
	Domains []xmlDomain `xml:"CommandResponse>DomainGetListResult>Domain"`
	Paging  struct {
		TotalItems  int `xml:"TotalItems"`
		CurrentPage int `xml:"CurrentPage"`
		PageSize    int `xml:"PageSize"`
	} `xml:"CommandResponse>Paging"`
}

// ListDomains fetches every page of namecheap.domains.getList.
func (c *Client) ListDomains(ctx context.Context) ([]registrar.Domain, error) {
	var out []registrar.Domain
	for page := 1; ; page++ {
		params := url.Values{}
		params.Set("Page", strconv.Itoa(page))
		params.Set("PageSize", "100")
		var resp getListResponse
		if err := c.call(ctx, "namecheap.domains.getList", params, &resp); err != nil {
			return nil, err
		}
		for _, d := range resp.Domains {
			name, err := registrar.Parse(d.Name)
			if err != nil {
				return nil, fmt.Errorf("api returned invalid domain %q: %w", d.Name, err)
			}
			out = append(out, registrar.Domain{
				ID: d.ID, Name: name, Owner: d.User,
				Created: parseDate(d.Created), Expires: parseDate(d.Expires),
				AutoRenew: d.AutoRenew, Locked: d.IsLocked,
				Privacy: d.WhoisGuard == "ENABLED", Expired: d.IsExpired,
			})
		}
		fetched := resp.Paging.CurrentPage * resp.Paging.PageSize
		if resp.Paging.PageSize == 0 || fetched >= resp.Paging.TotalItems {
			return out, nil
		}
	}
}

type checkResponse struct {
	Results []struct {
		Domain       string  `xml:"Domain,attr"`
		Available    bool    `xml:"Available,attr"`
		IsPremium    bool    `xml:"IsPremiumName,attr"`
		PremiumPrice float64 `xml:"PremiumRegistrationPrice,attr"`
	} `xml:"CommandResponse>DomainCheckResult"`
}

// CheckDomains checks availability via namecheap.domains.check.
func (c *Client) CheckDomains(ctx context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	strs := make([]string, len(names))
	for i, n := range names {
		strs[i] = n.String()
	}
	params := url.Values{}
	params.Set("DomainList", strings.Join(strs, ","))
	var resp checkResponse
	if err := c.call(ctx, "namecheap.domains.check", params, &resp); err != nil {
		return nil, err
	}
	out := make([]registrar.Availability, 0, len(resp.Results))
	for _, r := range resp.Results {
		name, err := registrar.Parse(r.Domain)
		if err != nil {
			return nil, fmt.Errorf("api returned invalid domain %q: %w", r.Domain, err)
		}
		out = append(out, registrar.Availability{
			Name: name, Available: r.Available,
			Premium: r.IsPremium, PremiumPrice: r.PremiumPrice,
		})
	}
	return out, nil
}

type getInfoResponse struct {
	Result struct {
		Status     string `xml:"Status,attr"`
		ID         string `xml:"ID,attr"`
		DomainName string `xml:"DomainName,attr"`
		OwnerName  string `xml:"OwnerName,attr"`
		Details    struct {
			CreatedDate string `xml:"CreatedDate"`
			ExpiredDate string `xml:"ExpiredDate"`
		} `xml:"DomainDetails"`
		Whoisguard struct {
			Enabled string `xml:"Enabled,attr"`
		} `xml:"Whoisguard"`
		DNS struct {
			ProviderType string   `xml:"ProviderType,attr"`
			Nameservers  []string `xml:"Nameserver"`
		} `xml:"DnsDetails"`
	} `xml:"CommandResponse>DomainGetInfoResult"`
}

// DomainInfo fetches namecheap.domains.getInfo for one domain.
func (c *Client) DomainInfo(ctx context.Context, name registrar.DomainName) (registrar.Details, error) {
	params := url.Values{}
	params.Set("DomainName", name.String())
	var resp getInfoResponse
	if err := c.call(ctx, "namecheap.domains.getInfo", params, &resp); err != nil {
		return registrar.Details{}, err
	}
	r := resp.Result
	return registrar.Details{
		Domain: registrar.Domain{
			ID: r.ID, Name: name, Owner: r.OwnerName,
			Created: parseDate(r.Details.CreatedDate),
			Expires: parseDate(r.Details.ExpiredDate),
			Privacy: strings.EqualFold(r.Whoisguard.Enabled, "true"),
		},
		Status:      r.Status,
		DNSProvider: r.DNS.ProviderType,
		Nameservers: r.DNS.Nameservers,
	}, nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/adapters/namecheap/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/adapters/namecheap && git commit -m "feat: implement RegistrarAPI on namecheap client (list, check, info)"
```

---

### Task 10: iputil public IP resolver

**Files:**
- Create: `internal/adapters/iputil/resolver.go`
- Test: `internal/adapters/iputil/resolver_test.go`

**Interfaces:**
- Produces: `func New(opts ...Option) *Resolver` satisfying `ports.IPResolver`; `func WithURLs(urls ...string) Option` (tests/fallback order)

- [ ] **Step 1: Write the failing test**

`internal/adapters/iputil/resolver_test.go`:
```go
package iputil_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/4thel00z/namecheap-tui/internal/adapters/iputil"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

var _ ports.IPResolver = (*iputil.Resolver)(nil)

func TestPublicIPTrimsAndReturns(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("203.0.113.9\n"))
	}))
	defer srv.Close()
	r := iputil.New(iputil.WithURLs(srv.URL))
	ip, err := r.PublicIP(context.Background())
	if err != nil || ip != "203.0.113.9" {
		t.Errorf("PublicIP = %q, %v", ip, err)
	}
}

func TestPublicIPFallsBack(t *testing.T) {
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer bad.Close()
	good := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("198.51.100.4"))
	}))
	defer good.Close()
	r := iputil.New(iputil.WithURLs(bad.URL, good.URL))
	ip, err := r.PublicIP(context.Background())
	if err != nil || ip != "198.51.100.4" {
		t.Errorf("PublicIP = %q, %v", ip, err)
	}
}

func TestPublicIPRejectsGarbage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<html>nope</html>"))
	}))
	defer srv.Close()
	r := iputil.New(iputil.WithURLs(srv.URL))
	if _, err := r.PublicIP(context.Background()); err == nil {
		t.Error("want error on non-IP body")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/iputil/`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Write implementation**

`internal/adapters/iputil/resolver.go`:
```go
// Package iputil detects the caller's public IPv4 address.
package iputil

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strings"
	"time"

	λ "github.com/4thel00z/lambda/v2"
)

// Resolver queries well-known what's-my-ip services in order.
type Resolver struct {
	urls []string
	http *http.Client
}

// Option configures a Resolver.
type Option func(*Resolver)

// WithURLs overrides the service URLs (first success wins).
func WithURLs(urls ...string) Option { return func(r *Resolver) { r.urls = urls } }

// New builds a Resolver with icanhazip + ipify defaults.
func New(opts ...Option) *Resolver {
	r := &Resolver{
		urls: []string{"https://ipv4.icanhazip.com", "https://api.ipify.org"},
		http: &http.Client{Timeout: 10 * time.Second},
	}
	for _, o := range opts {
		o(r)
	}
	return r
}

// PublicIP returns the first valid IPv4 address any service reports.
func (r *Resolver) PublicIP(ctx context.Context) (string, error) {
	var errs []error
	for _, u := range r.urls {
		body, err := λ.Client(r.http).Get(u).Do(ctx).Slurp().Get()
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", u, err))
			continue
		}
		ip := strings.TrimSpace(string(body))
		addr, err := netip.ParseAddr(ip)
		if err != nil || !addr.Is4() {
			errs = append(errs, fmt.Errorf("%s: invalid response %q", u, ip))
			continue
		}
		return ip, nil
	}
	return "", fmt.Errorf("public ip detection failed: %w", errors.Join(errs...))
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapters/iputil/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/iputil && git commit -m "feat: add public IP resolver with service fallback"
```

---

### Task 11: ProfileService

**Files:**
- Create: `internal/core/services/profiles.go`
- Test: `internal/core/services/profiles_test.go` (includes in-memory fakes shared by Task 12 tests)

**Interfaces:**
- Consumes: `ports.ProfileRepo`, `ports.IPResolver`
- Produces:
  - `func NewProfileService(repo ports.ProfileRepo, ip ports.IPResolver, verify func(context.Context, account.Credentials) error) *ProfileService` (`verify` nil ⇒ skip live check)
  - `Add(ctx, creds account.Credentials) error`, `List(ctx) ([]account.Profile, error)`, `Use(ctx, name string) error`, `Remove(ctx, name string) error`, `DetectIP(ctx) (string, error)`
  - `Current(ctx, override string) (account.Credentials, error)` — precedence: `override` flag > env credentials (`NAMECHEAP_API_KEY`+`NAMECHEAP_API_USER`) > `NCP_PROFILE` env > repo default

- [ ] **Step 1: Write the failing test**

`internal/core/services/profiles_test.go`:
```go
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
	for _, k := range []string{"NAMECHEAP_API_KEY", "NAMECHEAP_API_USER",
		"NAMECHEAP_USERNAME", "NAMECHEAP_CLIENT_IP", "NAMECHEAP_SANDBOX", "NCP_PROFILE"} {
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/services/`
Expected: FAIL (package does not exist)

- [ ] **Step 3: Write implementation**

`internal/core/services/profiles.go`:
```go
// Package services contains the application use-cases driving the hexagon.
package services

import (
	"context"
	"fmt"
	"os"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// ProfileService manages named account profiles.
type ProfileService struct {
	repo   ports.ProfileRepo
	ip     ports.IPResolver
	verify func(context.Context, account.Credentials) error
}

// NewProfileService builds a ProfileService. verify (nil = skip) is called
// with candidate credentials before saving, e.g. a live domains.getList.
func NewProfileService(repo ports.ProfileRepo, ip ports.IPResolver,
	verify func(context.Context, account.Credentials) error,
) *ProfileService {
	return &ProfileService{repo: repo, ip: ip, verify: verify}
}

// Add validates, live-verifies, and persists a profile.
func (s *ProfileService) Add(ctx context.Context, creds account.Credentials) error {
	if err := creds.Validate(); err != nil {
		return err
	}
	if s.verify != nil {
		if err := s.verify(ctx, creds); err != nil {
			return fmt.Errorf("credential verification failed: %w", err)
		}
	}
	return s.repo.Save(ctx, creds)
}

// List returns all stored profiles (without secrets).
func (s *ProfileService) List(ctx context.Context) ([]account.Profile, error) {
	return s.repo.List(ctx)
}

// Use marks the named profile as default.
func (s *ProfileService) Use(ctx context.Context, name string) error {
	return s.repo.SetDefault(ctx, name)
}

// Remove deletes the named profile and its secret.
func (s *ProfileService) Remove(ctx context.Context, name string) error {
	return s.repo.Delete(ctx, name)
}

// DetectIP returns the caller's current public IPv4.
func (s *ProfileService) DetectIP(ctx context.Context) (string, error) {
	return s.ip.PublicIP(ctx)
}

// Current resolves the active credentials.
// Precedence: explicit override > env credentials > NCP_PROFILE > stored default.
func (s *ProfileService) Current(ctx context.Context, override string) (account.Credentials, error) {
	if override != "" {
		return s.repo.Get(ctx, override)
	}
	if key := os.Getenv("NAMECHEAP_API_KEY"); key != "" {
		if user := os.Getenv("NAMECHEAP_API_USER"); user != "" {
			return credsFromEnv(key, user)
		}
	}
	if name := os.Getenv("NCP_PROFILE"); name != "" {
		return s.repo.Get(ctx, name)
	}
	return s.repo.Default(ctx)
}

func credsFromEnv(key, user string) (account.Credentials, error) {
	username := os.Getenv("NAMECHEAP_USERNAME")
	if username == "" {
		username = user
	}
	endpoint := account.EndpointProduction
	if os.Getenv("NAMECHEAP_SANDBOX") == "1" {
		endpoint = account.EndpointSandbox
	}
	c := account.Credentials{
		Profile: account.Profile{
			Name: "env", APIUser: user, Username: username,
			ClientIP: os.Getenv("NAMECHEAP_CLIENT_IP"), Endpoint: endpoint,
		},
		APIKey: key,
	}
	if err := c.Validate(); err != nil {
		return account.Credentials{}, fmt.Errorf("environment credentials incomplete: %w", err)
	}
	return c, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/services/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/services && git commit -m "feat: add profile service with env-override resolution"
```

---

### Task 12: DomainService with cache

**Files:**
- Create: `internal/core/services/domains.go`
- Test: `internal/core/services/domains_test.go`

**Interfaces:**
- Consumes: `ports.RegistrarAPI`, `ports.CacheRepo`, fakes from Task 11's test file
- Produces:
  - `func NewDomainService(api ports.RegistrarAPI, cache ports.CacheRepo, profile string) *DomainService`
  - `List(ctx, refresh bool) ([]registrar.Domain, error)` — cache kind `"domains"` key `"list"` TTL 10min
  - `Check(ctx, raw []string) ([]registrar.Availability, error)` — no cache
  - `Info(ctx, raw string) (registrar.Details, error)` — cache kind `"domain-info"` key = domain, TTL 5min

- [ ] **Step 1: Write the failing test**

`internal/core/services/domains_test.go`:
```go
package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type fakeRegistrar struct {
	listCalls int
	domains   []registrar.Domain
	checked   []registrar.DomainName
}

func (f *fakeRegistrar) ListDomains(context.Context) ([]registrar.Domain, error) {
	f.listCalls++
	return f.domains, nil
}

func (f *fakeRegistrar) CheckDomains(_ context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	f.checked = names
	out := make([]registrar.Availability, len(names))
	for i, n := range names {
		out[i] = registrar.Availability{Name: n, Available: true}
	}
	return out, nil
}

func (f *fakeRegistrar) DomainInfo(_ context.Context, name registrar.DomainName) (registrar.Details, error) {
	return registrar.Details{Domain: registrar.Domain{Name: name}, Status: "Ok"}, nil
}

type fakeCache struct{ m map[string][]byte }

func newFakeCache() *fakeCache { return &fakeCache{m: map[string][]byte{}} }

func (f *fakeCache) Get(_ context.Context, p, kind, key string, _ time.Duration) ([]byte, bool, error) {
	v, ok := f.m[p+"/"+kind+"/"+key]
	return v, ok, nil
}

func (f *fakeCache) Put(_ context.Context, p, kind, key string, payload []byte) error {
	f.m[p+"/"+kind+"/"+key] = payload
	return nil
}

func (f *fakeCache) Invalidate(_ context.Context, p, kind string) error { return nil }

func dom(name string) registrar.Domain {
	n, _ := registrar.Parse(name)
	return registrar.Domain{Name: n}
}

func TestListUsesCache(t *testing.T) {
	api := &fakeRegistrar{domains: []registrar.Domain{dom("a.com")}}
	svc := services.NewDomainService(api, newFakeCache(), "p")
	ctx := context.Background()

	got, err := svc.List(ctx, false)
	if err != nil || len(got) != 1 {
		t.Fatalf("first list: %v, %v", got, err)
	}
	if api.listCalls != 1 {
		t.Fatalf("listCalls = %d", api.listCalls)
	}
	// Second call served from cache.
	if _, err := svc.List(ctx, false); err != nil {
		t.Fatal(err)
	}
	if api.listCalls != 1 {
		t.Errorf("cache miss: listCalls = %d, want 1", api.listCalls)
	}
	// refresh bypasses cache.
	if _, err := svc.List(ctx, true); err != nil {
		t.Fatal(err)
	}
	if api.listCalls != 2 {
		t.Errorf("refresh ignored: listCalls = %d, want 2", api.listCalls)
	}
}

func TestCheckParsesNames(t *testing.T) {
	api := &fakeRegistrar{}
	svc := services.NewDomainService(api, newFakeCache(), "p")
	got, err := svc.Check(context.Background(), []string{"A.com", "b.co.uk"})
	if err != nil || len(got) != 2 {
		t.Fatalf("%v, %v", got, err)
	}
	if api.checked[0].SLD != "a" || api.checked[1].TLD != "co.uk" {
		t.Errorf("parsed names wrong: %+v", api.checked)
	}
	if _, err := svc.Check(context.Background(), []string{"garbage"}); err == nil {
		t.Error("invalid name accepted")
	}
}

func TestInfo(t *testing.T) {
	svc := services.NewDomainService(&fakeRegistrar{}, newFakeCache(), "p")
	got, err := svc.Info(context.Background(), "a.com")
	if err != nil || got.Status != "Ok" || got.Name.String() != "a.com" {
		t.Errorf("%+v, %v", got, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/core/services/`
Expected: FAIL (NewDomainService undefined)

- [ ] **Step 3: Write implementation**

`internal/core/services/domains.go`:
```go
package services

import (
	"context"
	"encoding/json"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/ports"
)

// Cache TTLs per the design spec.
const (
	domainsListTTL = 10 * time.Minute
	domainInfoTTL  = 5 * time.Minute
)

// DomainService is the domains use-case facade for one profile.
type DomainService struct {
	api     ports.RegistrarAPI
	cache   ports.CacheRepo
	profile string
}

// NewDomainService builds a DomainService bound to a profile name (cache scope).
func NewDomainService(api ports.RegistrarAPI, cache ports.CacheRepo, profile string) *DomainService {
	return &DomainService{api: api, cache: cache, profile: profile}
}

// List returns all account domains, from cache unless refresh or stale.
func (s *DomainService) List(ctx context.Context, refresh bool) ([]registrar.Domain, error) {
	if !refresh {
		if payload, ok, err := s.cache.Get(ctx, s.profile, "domains", "list", domainsListTTL); err == nil && ok {
			var cached []registrar.Domain
			if json.Unmarshal(payload, &cached) == nil {
				return cached, nil
			}
		}
	}
	domains, err := s.api.ListDomains(ctx)
	if err != nil {
		return nil, err
	}
	if payload, err := json.Marshal(domains); err == nil {
		_ = s.cache.Put(ctx, s.profile, "domains", "list", payload) // cache failure is non-fatal
	}
	return domains, nil
}

// Check parses raw names and checks availability (never cached).
func (s *DomainService) Check(ctx context.Context, raw []string) ([]registrar.Availability, error) {
	names := make([]registrar.DomainName, len(raw))
	for i, r := range raw {
		n, err := registrar.Parse(r)
		if err != nil {
			return nil, err
		}
		names[i] = n
	}
	return s.api.CheckDomains(ctx, names)
}

// Info returns full details for one domain, cached briefly.
func (s *DomainService) Info(ctx context.Context, raw string) (registrar.Details, error) {
	name, err := registrar.Parse(raw)
	if err != nil {
		return registrar.Details{}, err
	}
	if payload, ok, err := s.cache.Get(ctx, s.profile, "domain-info", name.String(), domainInfoTTL); err == nil && ok {
		var cached registrar.Details
		if json.Unmarshal(payload, &cached) == nil {
			return cached, nil
		}
	}
	details, err := s.api.DomainInfo(ctx, name)
	if err != nil {
		return registrar.Details{}, err
	}
	if payload, err := json.Marshal(details); err == nil {
		_ = s.cache.Put(ctx, s.profile, "domain-info", name.String(), payload)
	}
	return details, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/core/services/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/core/services && git commit -m "feat: add domain service with turso-backed caching"
```

---

### Task 13: CLI root + profile commands

**Files:**
- Create: `internal/adapters/cli/app.go`, `internal/adapters/cli/root.go`, `internal/adapters/cli/render.go`, `internal/adapters/cli/profile.go`
- Test: `internal/adapters/cli/profile_test.go`

**Interfaces:**
- Consumes: `services.ProfileService`, `services.DomainService`, `account` types
- Produces:
  - `type App struct { Profiles *services.ProfileService; Domains func(ctx context.Context, profile string, sandbox bool) (*services.DomainService, error); RunTUI func(ctx context.Context, profile string, sandbox bool) error; Version string }`
  - `func Root(app *App) *cobra.Command` — persistent flags `--profile`, `--json`, `--no-cache`, `--sandbox`; bare invocation calls `app.RunTUI`
  - helpers `renderJSON(w io.Writer, v any) error`, `renderTable(w io.Writer, headers []string, rows [][]string)`

- [ ] **Step 1: Get dependencies**

Run: `go get github.com/spf13/cobra@latest github.com/charmbracelet/fang@latest github.com/charmbracelet/lipgloss@latest github.com/charmbracelet/huh@latest && go mod tidy`

- [ ] **Step 2: Write the failing test**

`internal/adapters/cli/profile_test.go`:
```go
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
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/cli/`
Expected: FAIL (package does not exist)

- [ ] **Step 4: Write implementation**

`internal/adapters/cli/app.go`:
```go
// Package cli is the cobra-based driving adapter.
package cli

import (
	"context"

	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

// App carries the wired services into the command tree. The composition
// root (cmd/ncp) fills the factories so cli never imports other adapters.
type App struct {
	Profiles *services.ProfileService
	Domains  func(ctx context.Context, profile string, sandbox bool) (*services.DomainService, error)
	RunTUI   func(ctx context.Context, profile string, sandbox bool) error
	Version  string
}
```

`internal/adapters/cli/root.go`:
```go
package cli

import (
	"errors"

	"github.com/spf13/cobra"
)

// Root builds the ncp command tree.
func Root(app *App) *cobra.Command {
	root := &cobra.Command{
		Use:           "ncp",
		Short:         "A fast TUI and CLI for the Namecheap API",
		Version:       app.Version,
		SilenceUsage:  true,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if app.RunTUI == nil {
				return errors.New("TUI not wired; use a subcommand (see ncp --help)")
			}
			profile, _ := cmd.Flags().GetString("profile")
			sandbox, _ := cmd.Flags().GetBool("sandbox")
			return app.RunTUI(cmd.Context(), profile, sandbox)
		},
	}
	pf := root.PersistentFlags()
	pf.String("profile", "", "profile to use (default: the stored default)")
	pf.Bool("json", false, "output JSON instead of tables")
	pf.Bool("no-cache", false, "bypass the local cache")
	pf.Bool("sandbox", false, "use the Namecheap sandbox endpoint")

	root.AddCommand(profileCmd(app))
	return root
}
```

`internal/adapters/cli/render.go`:
```go
package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

var headerStyle = lipgloss.NewStyle().Bold(true)

// renderJSON writes v as indented JSON.
func renderJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// renderTable writes a lipgloss-styled table.
func renderTable(w io.Writer, headers []string, rows [][]string) {
	t := table.New().
		Headers(headers...).
		Rows(rows...).
		StyleFunc(func(row, _ int) lipgloss.Style {
			if row == table.HeaderRow {
				return headerStyle
			}
			return lipgloss.NewStyle().Padding(0, 1)
		})
	fmt.Fprintln(w, t)
}
```

`internal/adapters/cli/profile.go`:
```go
package cli

import (
	"fmt"
	"strconv"

	"github.com/charmbracelet/huh"
	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
)

func profileCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "profile",
		Short: "Manage Namecheap account profiles",
	}
	cmd.AddCommand(profileAddCmd(app), profileListCmd(app), profileUseCmd(app), profileRmCmd(app))
	return cmd
}

func profileAddCmd(app *App) *cobra.Command {
	var name, apiUser, username, apiKey, clientIP string
	var sandbox bool
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a profile (interactive form when flags are omitted)",
		RunE: func(cmd *cobra.Command, args []string) error {
			// --sandbox is the root's persistent flag; don't redefine it locally.
			sandbox, _ = cmd.Flags().GetBool("sandbox")
			interactive := name == "" && apiUser == "" && apiKey == ""
			if interactive {
				if err := profileForm(cmd, app, &name, &apiUser, &username, &apiKey, &clientIP, &sandbox); err != nil {
					return err
				}
			}
			if username == "" {
				username = apiUser
			}
			endpoint := account.EndpointProduction
			if sandbox {
				endpoint = account.EndpointSandbox
			}
			creds := account.Credentials{
				Profile: account.Profile{
					Name: name, APIUser: apiUser, Username: username,
					ClientIP: clientIP, Endpoint: endpoint,
				},
				APIKey: apiKey,
			}
			if err := app.Profiles.Add(cmd.Context(), creds); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "profile %q saved\n", name)
			return nil
		},
	}
	f := cmd.Flags()
	f.StringVar(&name, "name", "", "profile name")
	f.StringVar(&apiUser, "api-user", "", "Namecheap API user")
	f.StringVar(&username, "username", "", "Namecheap username (default: api-user)")
	f.StringVar(&apiKey, "api-key", "", "Namecheap API key")
	f.StringVar(&clientIP, "client-ip", "", "whitelisted client IPv4 (empty = auto-detect)")
	return cmd
}

// profileForm collects profile fields interactively, pre-filling the
// detected public IP.
func profileForm(cmd *cobra.Command, app *App,
	name, apiUser, username, apiKey, clientIP *string, sandbox *bool,
) error {
	if *clientIP == "" {
		if ip, err := app.Profiles.DetectIP(cmd.Context()); err == nil {
			*clientIP = ip
		}
	}
	return huh.NewForm(huh.NewGroup(
		huh.NewInput().Title("Profile name").Value(name),
		huh.NewInput().Title("API user").Value(apiUser),
		huh.NewInput().Title("Username (empty = API user)").Value(username),
		huh.NewInput().Title("API key").EchoMode(huh.EchoModePassword).Value(apiKey),
		huh.NewInput().Title("Whitelisted client IP").Value(clientIP),
		huh.NewConfirm().Title("Use sandbox endpoint?").Value(sandbox),
	)).Run()
}

func profileListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List profiles",
		RunE: func(cmd *cobra.Command, args []string) error {
			profiles, err := app.Profiles.List(cmd.Context())
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					Name     string `json:"name"`
					APIUser  string `json:"api_user"`
					Username string `json:"username"`
					ClientIP string `json:"client_ip"`
					Endpoint string `json:"endpoint"`
					Default  bool   `json:"default"`
				}
				out := make([]row, len(profiles))
				for i, p := range profiles {
					out[i] = row{p.Name, p.APIUser, p.Username, p.ClientIP, string(p.Endpoint), p.IsDefault}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(profiles))
			for i, p := range profiles {
				rows[i] = []string{p.Name, p.APIUser, p.Username, p.ClientIP,
					string(p.Endpoint), strconv.FormatBool(p.IsDefault)}
			}
			renderTable(cmd.OutOrStdout(),
				[]string{"NAME", "API USER", "USERNAME", "CLIENT IP", "ENDPOINT", "DEFAULT"}, rows)
			return nil
		},
	}
}

func profileUseCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the default profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Profiles.Use(cmd.Context(), args[0])
		},
	}
}

func profileRmCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "rm <name>",
		Short: "Delete a profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.Profiles.Remove(cmd.Context(), args[0])
		},
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/adapters/cli/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/adapters/cli go.mod go.sum && git commit -m "feat: add cobra root and profile commands"
```

---

### Task 14: CLI domains commands

**Files:**
- Create: `internal/adapters/cli/domains.go`
- Modify: `internal/adapters/cli/root.go` (add `root.AddCommand(domainsCmd(app))`)
- Test: `internal/adapters/cli/domains_test.go`

**Interfaces:**
- Consumes: `App.Domains` factory, `renderJSON`/`renderTable`, `registrar` types

- [ ] **Step 1: Write the failing test**

`internal/adapters/cli/domains_test.go`:
```go
package cli_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type memRegistrar struct{ domains []registrar.Domain }

func (f *memRegistrar) ListDomains(context.Context) ([]registrar.Domain, error) {
	return f.domains, nil
}

func (f *memRegistrar) CheckDomains(_ context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	out := make([]registrar.Availability, len(names))
	for i, n := range names {
		out[i] = registrar.Availability{Name: n, Available: n.SLD == "free"}
	}
	return out, nil
}

func (f *memRegistrar) DomainInfo(_ context.Context, name registrar.DomainName) (registrar.Details, error) {
	return registrar.Details{
		Domain: registrar.Domain{Name: name, Expires: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)},
		Status: "Ok", DNSProvider: "CUSTOM",
		Nameservers: []string{"ns1.example.net"},
	}, nil
}

type noCache struct{}

func (noCache) Get(context.Context, string, string, string, time.Duration) ([]byte, bool, error) {
	return nil, false, nil
}
func (noCache) Put(context.Context, string, string, string, []byte) error { return nil }
func (noCache) Invalidate(context.Context, string, string) error          { return nil }

func domainsApp(reg *memRegistrar) *cli.App {
	app := testApp(newMemRepo())
	app.Domains = func(context.Context, string, bool) (*services.DomainService, error) {
		return services.NewDomainService(reg, noCache{}, "t"), nil
	}
	return app
}

func TestDomainsList(t *testing.T) {
	n, _ := registrar.Parse("alpha.com")
	reg := &memRegistrar{domains: []registrar.Domain{{
		Name: n, Expires: time.Date(2027, 2, 15, 0, 0, 0, 0, time.UTC), AutoRenew: true,
	}}}
	out, err := run(t, domainsApp(reg), "domains", "list")
	if err != nil || !strings.Contains(out, "alpha.com") {
		t.Errorf("output %q, %v", out, err)
	}
	out, err = run(t, domainsApp(reg), "domains", "list", "--json")
	if err != nil || !strings.Contains(out, `"name": "alpha.com"`) {
		t.Errorf("json output %q, %v", out, err)
	}
}

func TestDomainsCheck(t *testing.T) {
	out, err := run(t, domainsApp(&memRegistrar{}), "domains", "check", "free.com", "taken.com")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "free.com") || !strings.Contains(out, "taken.com") {
		t.Errorf("output %q", out)
	}
}

func TestDomainsInfo(t *testing.T) {
	out, err := run(t, domainsApp(&memRegistrar{}), "domains", "info", "alpha.com", "--json")
	if err != nil || !strings.Contains(out, `"dns_provider": "CUSTOM"`) {
		t.Errorf("output %q, %v", out, err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/adapters/cli/`
Expected: FAIL (unknown command "domains")

- [ ] **Step 3: Write implementation**

`internal/adapters/cli/domains.go`:
```go
package cli

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/cobra"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

func domainsCmd(app *App) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "domains",
		Short: "Manage registered domains",
	}
	cmd.AddCommand(domainsListCmd(app), domainsCheckCmd(app), domainsInfoCmd(app))
	return cmd
}

// domainsService builds the domain service from the persistent flags.
func domainsService(cmd *cobra.Command, app *App) (*services.DomainService, error) {
	if app.Domains == nil {
		return nil, fmt.Errorf("domains service not wired")
	}
	profile, _ := cmd.Flags().GetString("profile")
	sandbox, _ := cmd.Flags().GetBool("sandbox")
	return app.Domains(cmd.Context(), profile, sandbox)
}

type domainJSON struct {
	Name      string `json:"name"`
	Expires   string `json:"expires"`
	AutoRenew bool   `json:"auto_renew"`
	Privacy   bool   `json:"privacy"`
	Locked    bool   `json:"locked"`
	Expired   bool   `json:"expired"`
}

func toDomainJSON(d registrar.Domain) domainJSON {
	return domainJSON{
		Name: d.Name.String(), Expires: d.Expires.Format(time.DateOnly),
		AutoRenew: d.AutoRenew, Privacy: d.Privacy, Locked: d.Locked, Expired: d.Expired,
	}
}

func domainsListCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all domains in the account",
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			noCache, _ := cmd.Flags().GetBool("no-cache")
			domains, err := svc.List(cmd.Context(), noCache)
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				out := make([]domainJSON, len(domains))
				for i, d := range domains {
					out[i] = toDomainJSON(d)
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(domains))
			for i, d := range domains {
				days := strconv.Itoa(int(time.Until(d.Expires).Hours() / 24))
				rows[i] = []string{d.Name.String(), d.Expires.Format(time.DateOnly), days,
					boolMark(d.AutoRenew), boolMark(d.Privacy), boolMark(d.Locked)}
			}
			renderTable(cmd.OutOrStdout(),
				[]string{"DOMAIN", "EXPIRES", "DAYS", "AUTORENEW", "PRIVACY", "LOCKED"}, rows)
			return nil
		},
	}
}

func boolMark(b bool) string {
	if b {
		return "✓"
	}
	return "-"
}

func domainsCheckCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "check <domain>...",
		Short: "Check domain availability",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			results, err := svc.Check(cmd.Context(), args)
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type row struct {
					Name         string  `json:"name"`
					Available    bool    `json:"available"`
					Premium      bool    `json:"premium"`
					PremiumPrice float64 `json:"premium_price,omitempty"`
				}
				out := make([]row, len(results))
				for i, r := range results {
					out[i] = row{r.Name.String(), r.Available, r.Premium, r.PremiumPrice}
				}
				return renderJSON(cmd.OutOrStdout(), out)
			}
			rows := make([][]string, len(results))
			for i, r := range results {
				price := "-"
				if r.Premium {
					price = strconv.FormatFloat(r.PremiumPrice, 'f', 2, 64)
				}
				rows[i] = []string{r.Name.String(), boolMark(r.Available), boolMark(r.Premium), price}
			}
			renderTable(cmd.OutOrStdout(),
				[]string{"DOMAIN", "AVAILABLE", "PREMIUM", "PREMIUM PRICE"}, rows)
			return nil
		},
	}
}

func domainsInfoCmd(app *App) *cobra.Command {
	return &cobra.Command{
		Use:   "info <domain>",
		Short: "Show full details for a domain",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc, err := domainsService(cmd, app)
			if err != nil {
				return err
			}
			details, err := svc.Info(cmd.Context(), args[0])
			if err != nil {
				return err
			}
			if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
				type infoJSON struct {
					domainJSON
					Status      string   `json:"status"`
					DNSProvider string   `json:"dns_provider"`
					Nameservers []string `json:"nameservers"`
				}
				return renderJSON(cmd.OutOrStdout(), infoJSON{
					domainJSON: toDomainJSON(details.Domain),
					Status:     details.Status, DNSProvider: details.DNSProvider,
					Nameservers: details.Nameservers,
				})
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Domain:       %s\n", details.Name)
			fmt.Fprintf(w, "Status:       %s\n", details.Status)
			fmt.Fprintf(w, "Created:      %s\n", details.Created.Format(time.DateOnly))
			fmt.Fprintf(w, "Expires:      %s\n", details.Expires.Format(time.DateOnly))
			fmt.Fprintf(w, "Privacy:      %s\n", boolMark(details.Privacy))
			fmt.Fprintf(w, "DNS provider: %s\n", details.DNSProvider)
			for _, ns := range details.Nameservers {
				fmt.Fprintf(w, "Nameserver:   %s\n", ns)
			}
			return nil
		},
	}
}
```

In `root.go`, change the AddCommand line to:
```go
	root.AddCommand(profileCmd(app), domainsCmd(app))
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/adapters/cli/`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/adapters/cli && git commit -m "feat: add domains list/check/info commands"
```

---

### Task 15: TUI dashboard shell

**Files:**
- Create: `internal/adapters/tui/dashboard.go`
- Test: `internal/adapters/tui/dashboard_test.go`

**Interfaces:**
- Consumes: `services.DomainService`, bubbles table/spinner, lipgloss
- Produces:
  - `func NewDashboard(svc *services.DomainService, profileName, version string) Dashboard`
  - `func (Dashboard) Init() tea.Cmd`, `Update`, `View` (tea.Model)
  - `func Run(m tea.Model) error` — alt-screen program

- [ ] **Step 1: Get dependencies**

Run: `go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/bubbles@latest github.com/charmbracelet/x/exp/teatest@latest && go mod tidy`

- [ ] **Step 2: Write the failing test**

`internal/adapters/tui/dashboard_test.go`:
```go
package tui_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"

	"github.com/4thel00z/namecheap-tui/internal/adapters/tui"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

type fakeRegistrar struct{}

func (fakeRegistrar) ListDomains(context.Context) ([]registrar.Domain, error) {
	n, _ := registrar.Parse("alpha.com")
	return []registrar.Domain{{
		Name: n, Expires: time.Now().AddDate(1, 0, 0), AutoRenew: true, Privacy: true,
	}}, nil
}

func (fakeRegistrar) CheckDomains(_ context.Context, names []registrar.DomainName) ([]registrar.Availability, error) {
	return nil, nil
}

func (fakeRegistrar) DomainInfo(_ context.Context, name registrar.DomainName) (registrar.Details, error) {
	return registrar.Details{}, nil
}

type nilCache struct{}

func (nilCache) Get(context.Context, string, string, string, time.Duration) ([]byte, bool, error) {
	return nil, false, nil
}
func (nilCache) Put(context.Context, string, string, string, []byte) error { return nil }
func (nilCache) Invalidate(context.Context, string, string) error          { return nil }

func TestDashboardShowsDomains(t *testing.T) {
	svc := services.NewDomainService(fakeRegistrar{}, nilCache{}, "test")
	m := tui.NewDashboard(svc, "test", "0.0.0")
	tm := teatest.NewTestModel(t, m, teatest.WithInitialTermSize(100, 30))

	teatest.WaitFor(t, tm.Output(), func(b []byte) bool {
		return bytes.Contains(b, []byte("alpha.com"))
	}, teatest.WithDuration(3*time.Second))

	tm.Send(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	tm.WaitFinished(t, teatest.WithFinalTimeout(2*time.Second))
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/adapters/tui/`
Expected: FAIL (package does not exist)

- [ ] **Step 4: Write implementation**

`internal/adapters/tui/dashboard.go`:
```go
// Package tui is the bubbletea driving adapter.
package tui

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/4thel00z/namecheap-tui/internal/core/domain/registrar"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
)

var (
	headerStyle = lipgloss.NewStyle().Bold(true).Padding(0, 1)
	errStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Padding(0, 1)
	helpStyle   = lipgloss.NewStyle().Faint(true).Padding(0, 1)
	warnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("11"))
	dangerStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
)

type domainsMsg struct {
	domains   []registrar.Domain
	fromCache bool
}

type errMsg struct{ err error }

// Dashboard is the top-level TUI model.
type Dashboard struct {
	svc     *services.DomainService
	profile string
	version string

	table   table.Model
	spinner spinner.Model
	loading bool
	err     error
	width   int
}

// NewDashboard builds the dashboard model.
func NewDashboard(svc *services.DomainService, profileName, version string) Dashboard {
	t := table.New(
		table.WithColumns([]table.Column{
			{Title: "DOMAIN", Width: 32},
			{Title: "EXPIRES", Width: 12},
			{Title: "DAYS", Width: 6},
			{Title: "AUTORENEW", Width: 10},
			{Title: "PRIVACY", Width: 8},
			{Title: "LOCKED", Width: 7},
		}),
		table.WithFocused(true),
	)
	s := spinner.New(spinner.WithSpinner(spinner.Dot))
	return Dashboard{svc: svc, profile: profileName, version: version, table: t, spinner: s, loading: true}
}

func (m Dashboard) load(refresh bool) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		domains, err := m.svc.List(ctx, refresh)
		if err != nil {
			return errMsg{err}
		}
		return domainsMsg{domains: domains, fromCache: !refresh}
	}
}

// Init loads cached data instantly, then refreshes in the background.
func (m Dashboard) Init() tea.Cmd {
	return tea.Batch(m.spinner.Tick, m.load(false))
}

// Update handles messages.
func (m Dashboard) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.table.SetHeight(msg.Height - 4)
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "r":
			m.loading = true
			return m, tea.Batch(m.spinner.Tick, m.load(true))
		}
	case domainsMsg:
		m.err = nil
		m.table.SetRows(toRows(msg.domains))
		if msg.fromCache {
			// stale-while-revalidate: kick a background refresh
			return m, m.load(true)
		}
		m.loading = false
		return m, nil
	case errMsg:
		m.err = msg.err
		m.loading = false
		return m, nil
	case spinner.TickMsg:
		if !m.loading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.table, cmd = m.table.Update(msg)
	return m, cmd
}

func toRows(domains []registrar.Domain) []table.Row {
	rows := make([]table.Row, len(domains))
	for i, d := range domains {
		days := int(time.Until(d.Expires).Hours() / 24)
		daysStr := strconv.Itoa(days)
		switch {
		case days < 30:
			daysStr = dangerStyle.Render(daysStr)
		case days < 90:
			daysStr = warnStyle.Render(daysStr)
		}
		rows[i] = table.Row{
			d.Name.String(), d.Expires.Format(time.DateOnly), daysStr,
			mark(d.AutoRenew), mark(d.Privacy), mark(d.Locked),
		}
	}
	return rows
}

func mark(b bool) string {
	if b {
		return "✓"
	}
	return "-"
}

// View renders the dashboard.
func (m Dashboard) View() string {
	head := fmt.Sprintf("ncp %s — profile %s", m.version, m.profile)
	if m.loading {
		head += "  " + m.spinner.View()
	}
	out := headerStyle.Render(head) + "\n"
	if m.err != nil {
		out += errStyle.Render("error: "+m.err.Error()) + "\n"
	}
	out += m.table.View() + "\n"
	out += helpStyle.Render("q quit · r refresh · ↑/↓ navigate")
	return out
}

// Run starts the program in the alternate screen.
func Run(m tea.Model) error {
	_, err := tea.NewProgram(m, tea.WithAltScreen()).Run()
	return err
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/adapters/tui/`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/adapters/tui go.mod go.sum && git commit -m "feat: add TUI dashboard with stale-while-revalidate domain table"
```

---

### Task 16: Composition root (main.go), fang wiring, README

**Files:**
- Modify: `cmd/ncp/main.go` (replace stub)
- Create: `README.md`

**Interfaces:**
- Consumes: everything above; only `cmd/ncp` may import multiple adapters.

- [ ] **Step 1: Write main.go**

```go
// Command ncp is a TUI and CLI for the Namecheap API.
package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/fang"

	"github.com/4thel00z/namecheap-tui/internal/adapters/cli"
	"github.com/4thel00z/namecheap-tui/internal/adapters/iputil"
	"github.com/4thel00z/namecheap-tui/internal/adapters/namecheap"
	"github.com/4thel00z/namecheap-tui/internal/adapters/turso"
	"github.com/4thel00z/namecheap-tui/internal/adapters/tui"
	"github.com/4thel00z/namecheap-tui/internal/core/domain/account"
	"github.com/4thel00z/namecheap-tui/internal/core/services"
	"github.com/4thel00z/namecheap-tui/internal/version"
)

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	store, err := openStore(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = store.Close() }()

	verify := func(ctx context.Context, creds account.Credentials) error {
		_, err := namecheap.New(creds).ListDomains(ctx)
		return err
	}
	profiles := services.NewProfileService(store.Profiles(), iputil.New(), verify)

	domainsFactory := func(ctx context.Context, profile string, sandbox bool) (*services.DomainService, error) {
		creds, err := profiles.Current(ctx, profile)
		if err != nil {
			return nil, err
		}
		if sandbox {
			creds.Endpoint = account.EndpointSandbox
		}
		return services.NewDomainService(namecheap.New(creds), store.Cache(), creds.Name), nil
	}

	app := &cli.App{
		Profiles: profiles,
		Domains:  domainsFactory,
		RunTUI: func(ctx context.Context, profile string, sandbox bool) error {
			svc, err := domainsFactory(ctx, profile, sandbox)
			if err != nil {
				return err
			}
			creds, err := profiles.Current(ctx, profile)
			if err != nil {
				return err
			}
			return tui.Run(tui.NewDashboard(svc, creds.Name, version.Version))
		},
		Version: version.Version,
	}
	return fang.Execute(ctx, cli.Root(app), fang.WithVersion(version.Version))
}

// openStore opens the config DB — an embedded Turso replica when
// TURSO_DATABASE_URL is set, a plain local file otherwise.
func openStore(ctx context.Context) (*turso.Store, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "ncp", "ncp.db")
	if primary := os.Getenv("TURSO_DATABASE_URL"); primary != "" {
		return turso.OpenReplica(ctx, path, primary, os.Getenv("TURSO_AUTH_TOKEN"))
	}
	return turso.Open(ctx, path)
}
```

- [ ] **Step 2: Write README.md**

```markdown
# ncp

A fast TUI and CLI for the Namecheap API. Hexagonal architecture, Turso-backed
config and cache, charmbracelet UI.

## Install

    make build   # produces bin/ncp (CGo required)

## Quick start

    ncp profile add        # interactive; auto-detects your public IP
    ncp                    # full-screen dashboard
    ncp domains list       # scriptable table
    ncp domains list --json | jq .

## Configuration

- Profiles and cache live in `~/.config/ncp/ncp.db` (libSQL).
- Set `TURSO_DATABASE_URL` + `TURSO_AUTH_TOKEN` to sync via a Turso embedded replica.
- Env overrides: `NAMECHEAP_API_KEY`, `NAMECHEAP_API_USER`, `NAMECHEAP_USERNAME`,
  `NAMECHEAP_CLIENT_IP`, `NAMECHEAP_SANDBOX=1`, `NCP_PROFILE`.

## Development

    make test lint fmt hooks

Conventional commits enforced via pre-commit; releases via release-please.
```

- [ ] **Step 3: Verify the full build**

Run: `make build && make test && make lint && ./bin/ncp --help`
Expected: build/test/lint pass; help shows `profile` and `domains` subcommands, fang-styled.

- [ ] **Step 4: Commit**

```bash
git add cmd/ncp README.md && git commit -m "feat: wire composition root with fang, turso, and TUI dashboard"
```

---

## Phase-1 exit criteria

- `make build && make test && make lint` all green, `pre-commit run --all-files` clean.
- `ncp profile add --name t --api-user u --username u --api-key k --client-ip 1.2.3.4` fails only at live verification (expected without real creds) — validates the whole wiring.
- Phases 2–5 (DNS + zone editor, registrar lifecycle, SSL/privacy/account, polish) get their own plan documents following this same structure and the interfaces produced here.

