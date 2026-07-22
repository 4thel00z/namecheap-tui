# ncp Phase 2 — DNS Implementation Plan

> Executed inline (executing-plans). Follows the interfaces and patterns
> established by Phase 1 (`2026-07-22-ncp-phase-1-foundation.md`); code style,
> test approach (fakes for core, httptest+fixtures for adapters, teatest for
> TUI), and commit discipline are identical, so tasks list files, interfaces,
> and test intentions rather than duplicating full code.

**Goal:** Full DNS management: zone read/edit with staged changes (the
setHosts-replaces-everything footgun neutralized), nameserver switching,
email forwarding, registered nameservers, YAML import/export, and the
interactive zone editor TUI.

## Task 1: dns domain package

- Create `internal/core/domain/dns/` — `RecordType` (A, AAAA, ALIAS, CAA,
  CNAME, MX, MXE, NS, TXT, URL, URL301, FRAME) with `ParseRecordType`;
  `HostRecord{ID, Name, Type, Value, TTL, MXPref}` with `Validate()`
  (name non-empty, type known, value non-empty, TTL 60–60000 or 0=default
  1799, MXPref only meaningful for MX); `Zone{Domain, Records}`;
  `EmailForward{Mailbox, ForwardTo}`.
- `ChangeSet{Add, Remove, Update []…}` + `Merge(current []HostRecord, cs
  ChangeSet) ([]HostRecord, error)` — Remove/Update must match an existing
  record by (Name, Type, Value); Add rejects exact duplicates. This is the
  central correctness function; densest test coverage here.
- Tests: table-driven for ParseRecordType, Validate, Merge (add, remove,
  update, missing target, duplicate add, empty changeset no-op).

## Task 2: DNSAPI + NSAPI ports

- Extend `internal/core/ports/api.go`:
  - `DNSAPI`: `GetHosts(ctx, name) ([]dns.HostRecord, error)`,
    `SetHosts(ctx, name, records) error`,
    `GetNameservers(ctx, name) (dns.NameserverInfo, error)` (`IsUsingOurDNS bool`, `Nameservers []string`),
    `SetDefaultNS(ctx, name) error`, `SetCustomNS(ctx, name, ns []string) error`,
    `GetEmailForwarding(ctx, name) ([]dns.EmailForward, error)`,
    `SetEmailForwarding(ctx, name, fwds []dns.EmailForward) error`
  - `NSAPI`: `CreateNS(ctx, domain, host, ip) error`, `UpdateNS(ctx, domain,
    host, oldIP, newIP) error`, `DeleteNS(ctx, domain, host) error`,
    `NSInfo(ctx, domain, host) (dns.RegisteredNS, error)`
- `dns.NameserverInfo` and `dns.RegisteredNS{Host, IP, Statuses}` live in the
  dns domain package.

## Task 3: namecheap dns adapter

- `internal/adapters/namecheap/dns.go` implementing DNSAPI:
  - `namecheap.domains.dns.getHosts` (SLD/TLD params; host attrs HostId,
    Name, Type, Address, MXPref, TTL)
  - `namecheap.domains.dns.setHosts` (HostName1…N, RecordType1…N, Address1…N,
    MXPref1…N, TTL1…N)
  - `namecheap.domains.dns.getList` / `setDefault` / `setCustom`
  - `namecheap.domains.dns.getEmailForwarding` / `setEmailForwarding`
- `internal/adapters/namecheap/ns.go` implementing NSAPI
  (`domains.ns.create|update|delete|getInfo`).
- Golden fixtures per command; tests assert request params (incl. indexed
  setHosts params) and response mapping. `var _ ports.DNSAPI = (*Client)(nil)`.

## Task 4: DNSService + NSService

- `internal/core/services/dns.go`:
  - `Zone(ctx, domain, refresh)` — cache kind "zone", TTL 5m
  - `Apply(ctx, domain, cs)` — **always** fetch fresh, `dns.Merge`, validate
    all, `SetHosts`, then re-cache; single-record helpers `Add`, `Remove`,
    `Set` (full replace) built on it. Cache invalidated on every write.
  - `Nameservers/UseDefault/UseCustom`, `EmailForwards/SetEmailForwards`
    (NS switch invalidates zone cache).
- `internal/core/services/ns.go`: thin passthrough NSService.
- Tests with fake DNSAPI: Apply merges onto *fresh* remote state (not the
  cache), rm-nonexistent errors, add-duplicate errors, cache invalidation.

## Task 5: CLI dns + ns commands

- `internal/adapters/cli/dns.go`: `dns get|add|rm|set|export|import|
  use-default|use-custom|emailfwd get|set|edit` — YAML zone format
  (`domain:` + `records:[{name,type,value,ttl,mx_pref}]`) for export/import
  (gopkg.in/yaml.v3); `--json` everywhere; `dns edit` calls `App.RunZoneEditor`.
- `internal/adapters/cli/ns.go`: `ns create|update|delete|info`.
- `App` gains `DNS func(ctx, profile, sandbox) (*services.DNSService, error)`,
  `NS func(…) (*services.NSService, error)`, `RunZoneEditor func(ctx, profile
  string, sandbox bool, domain string) error`.
- Tests: fake services via App factories; assert table + JSON output, YAML
  roundtrip (export → import).

## Task 6: TUI zone editor

- `internal/adapters/tui/zoneeditor.go`: bubbles table of records with staged
  diff marks (+/~/-, lipgloss colors); keys: `a` add (huh form overlay),
  `e` edit selected, `d` stage delete, `u` unstage, `A` apply (confirm view
  with diff summary), `esc` quit (discard). Calls `DNSService.Zone` +
  `DNSService.Apply`.
- Reachable as `ncp dns edit <domain>`; wired via composition root.
- teatest: load fake zone, stage a delete + add, apply, assert fake DNSAPI
  received merged record set.

## Task 7: wiring + verification

- main.go: build DNSService/NSService factories, RunZoneEditor.
- `make build test lint`, `pre-commit run --all-files`, merge to master.

Each task: failing test → implementation → green → conventional commit.
