# ncp — Namecheap TUI/CLI: Design Spec

**Date:** 2026-07-22
**Status:** Approved (design), pending implementation plan
**Repo:** namecheap-tui · **Binary:** `ncp` · **Module:** `github.com/4thel00z/namecheap-tui`

## Goal

A feature-complete Go TUI/CLI for the Namecheap XML API. Hybrid interaction model:
bare `ncp` opens a full-screen bubbletea dashboard; subcommands provide scriptable
output (`--json` on every read command, lipgloss tables by default); inherently
interactive flows (`dns edit`, `profile add`, `domains register`) open focused TUI
views or huh forms.

## Decisions (locked)

| Decision | Choice |
|---|---|
| CLI ↔ TUI | Hybrid: dashboard + scriptable subcommands + focused TUI editors |
| API scope | All namespaces (domains, dns, ns, transfer, ssl, whoisguard, users, users.address) |
| Secrets | Plaintext in local DB, file mode 0600; env vars override |
| Storage | Turso embedded replica (`tursodatabase/go-libsql`, CGo). Local file always works offline; syncs to Turso cloud only when `TURSO_DATABASE_URL` + `TURSO_AUTH_TOKEN` are set |
| Binary name | `ncp` |
| Architecture | One hexagon, subdomain packages in core, segregated ports |
| CLI framework | cobra wrapped in charmbracelet/fang |
| TUI | bubbletea + bubbles + lipgloss + huh (forms) |
| HTTP | 4thel00z/lambda/v2 fluent pipelines |

## Stack

- Go 1.24+, CGo required (go-libsql)
- `spf13/cobra` + `charmbracelet/fang` — command tree, styled help/errors, completions
- `charmbracelet/bubbletea`, `bubbles` (table, list, textinput, spinner, help, viewport), `lipgloss`, `huh`
- `github.com/4thel00z/lambda/v2` — HTTP pipelines
- `github.com/tursodatabase/go-libsql` — embedded replica libSQL
- `encoding/xml` — Namecheap response parsing

## Command surface

```
ncp                                  → TUI dashboard
ncp domains  list|check|register|renew|reactivate|info|tlds
ncp domains  lock get|on|off
ncp domains  contacts get|set
ncp dns      get|add|rm|set|export|import        # export/import: YAML zone snapshot
ncp dns      edit <domain>                       → focused TUI zone editor
ncp dns      use-default|use-custom <domain> [ns...]
ncp dns      emailfwd get|set
ncp ns       create|update|delete|info
ncp transfer list|create|status
ncp ssl      list|create|activate|info|renew|reissue|approvers|resend-approver|revoke
ncp privacy  list|enable|disable|renew|change-email|assign|unassign|discard
ncp account  balance|pricing
ncp address  list|add|edit|rm|default|info
ncp profile  add|list|use|rm
ncp config   get|set|sync
```

Global flags: `--profile <name>`, `--json`, `--no-cache`, `--sandbox`.
`ncp profile add` = huh form; auto-detects public IP for the whitelist field,
validates credentials with a live `domains.getList` before saving.

## API method → command mapping

| Namecheap command | ncp |
|---|---|
| domains.getList | `domains list` |
| domains.check | `domains check` |
| domains.create | `domains register` |
| domains.renew | `domains renew` |
| domains.reactivate | `domains reactivate` |
| domains.getInfo | `domains info` |
| domains.getTldList | `domains tlds` |
| domains.getRegistrarLock / setRegistrarLock | `domains lock get|on|off` |
| domains.getContacts / setContacts | `domains contacts get|set` |
| domains.dns.getHosts / setHosts | `dns get|add|rm|set|edit|import|export` |
| domains.dns.getList | `dns get --ns` (nameserver info) |
| domains.dns.setDefault / setCustom | `dns use-default|use-custom` |
| domains.dns.getEmailForwarding / setEmailForwarding | `dns emailfwd get|set` |
| domains.ns.create/update/delete/getInfo | `ns create|update|delete|info` |
| domains.transfer.create/getStatus/getList | `transfer create|status|list` |
| domains.transfer.updateStatus | `transfer status --resubmit` |
| ssl.getList/create/activate/getInfo/renew/reissue | `ssl list|create|activate|info|renew|reissue` |
| ssl.getApproverEmailList / resendApproverEmail | `ssl approvers` / `ssl resend-approver` |
| ssl.revokecertificate | `ssl revoke` |
| ssl.parseCSR | used internally by `ssl activate` |
| whoisguard.getList/enable/disable/renew | `privacy list|enable|disable|renew` |
| whoisguard.changeemailaddress | `privacy change-email` |
| whoisguard.allot/unallot/discard | `privacy assign|unassign|discard` |
| users.getBalances | `account balance` |
| users.getPricing | `account pricing` |
| users.address.* | `address *` |

**Out of scope:** reseller-only methods (`users.create`, `users.login`,
`users.resetPassword`, `users.changePassword`, `users.update`,
`users.createaddfundsrequest`, `users.getAddFundsStatus`,
`ssl.resendfulfillmentemail`, `ssl.purchasemoresans`, `ssl.editDNSDVOrder`) —
they require a reseller account and cannot be exercised or tested with a normal
API account. The port interfaces leave room to add them later.

Exact request parameter signatures are verified against the official docs
per-method during implementation (docs block scrapers; verify manually).

## Architecture — one hexagon

```
cmd/ncp/main.go                 # wiring: build adapters, inject into services, fang.Execute
internal/
├─ core/
│  ├─ domain/                   # pure Go, zero external deps
│  │  ├─ registrar/             # Domain, Contact, Transfer, TLD; DomainName VO (SLD/TLD)
│  │  ├─ dns/                   # Zone aggregate, HostRecord, ChangeSet, RecordType, TTL, MXPref, EmailForward
│  │  ├─ ssl/                   # Certificate, CSR, ApproverEmail
│  │  └─ account/               # Profile, Balance, Pricing, Address, PrivacySubscription
│  ├─ ports/                    # interfaces only
│  │  ├─ api.go                 # RegistrarAPI, DNSAPI, NSAPI, TransferAPI, SSLAPI, PrivacyAPI, AccountAPI
│  │  └─ store.go               # ProfileRepo, CacheRepo, SettingsRepo, IPResolver
│  └─ services/                 # use-cases; the only callers of ports
│     ├─ domains.go  dns.go  ssl.go  transfer.go  privacy.go  account.go  profile.go
└─ adapters/
   ├─ namecheap/                # XML API client (driven adapter)
   │  ├─ client.go              # lambda pipeline, auth params, throttle, error mapping
   │  ├─ domains.go dns.go ns.go transfer.go ssl.go privacy.go users.go address.go
   │  └─ xml_types.go           # response DTOs
   ├─ turso/                    # ProfileRepo/CacheRepo/SettingsRepo on go-libsql
   ├─ iputil/                   # IPResolver (https://icanhazip.com, fallback ipify)
   ├─ cli/                      # cobra commands; rendering only
   └─ tui/                      # bubbletea app; models call services
```

Rules:
- `core/` imports nothing outside `core/` and stdlib.
- Adapters never import each other; only `core`.
- CLI/TUI contain zero business logic — services own all orchestration.

### Domain invariants worth encoding

- **Zone staging:** `Zone` mutations (`Add/Update/Remove`) accumulate a `ChangeSet`;
  nothing touches the network. `DNSService.Apply(ctx, domain, changeset)` performs
  fetch → merge → validate → `setHosts`. Namecheap's `setHosts` **replaces the whole
  zone**, so single-record ops must never round-trip a partial zone. This is the
  central correctness property of the app.
- `DomainName` parses/validates SLD+TLD once; everything downstream uses the VO.
- `RecordType` whitelists Namecheap-supported types: A, AAAA, ALIAS, CAA, CNAME,
  MX, MXE, NS, TXT, URL, URL301, FRAME.

## Namecheap XML client

- Endpoints: prod `https://api.namecheap.com/xml.response`, sandbox
  `https://api.sandbox.namecheap.com/xml.response` (per profile, `--sandbox` override).
- Every call: `ApiUser`, `ApiKey`, `UserName`, `ClientIp`, `Command` + method params,
  built as query params via lambda: `λ.Get(u).Do(ctx).Slurp()` → `xml.Unmarshal` →
  DTO → domain mapping.
- `ApiResponse Status="ERROR"` → typed errors carrying the Namecheap error number:
  `ErrIPNotWhitelisted` (1011102/1011147 — message includes the caller's current
  public IP and the whitelist dashboard URL), `ErrRateLimited`, `ErrAuth`,
  `ErrDomainNotFound`, generic `APIError{Number, Message}`.
- Client-side token bucket: 20/min, 700/hr, 8000/day (Namecheap's published limits);
  waits with context cancellation rather than failing when briefly over.
- Timeouts: 30s per request via context.

## Storage (Turso embedded replica)

Path: `os.UserConfigDir()/ncp/ncp.db`, created 0700 dir / 0600 file.

```sql
profiles (id, name UNIQUE, api_user, username, client_ip, endpoint, is_default)
secrets  (profile_id → profiles.id, api_key)
settings (key PRIMARY KEY, value)
cache    (profile_id, kind, key, payload_json, fetched_at)   -- TTL per kind
schema_migrations (version)
```

- Migrations: embedded `.sql` files applied at startup.
- Sync: when `TURSO_DATABASE_URL`+`TURSO_AUTH_TOKEN` present, the DB opens as an
  embedded replica and `ncp config sync` (plus auto-sync after writes) pushes/pulls.
  Unset → plain local file, zero network.
- Cache TTLs: domains list 10m, zone 5m, pricing 24h, balances 5m. `--no-cache` bypasses.
- Env overrides (win over DB): `NAMECHEAP_API_KEY`, `NAMECHEAP_API_USER`,
  `NAMECHEAP_USERNAME`, `NAMECHEAP_CLIENT_IP`, `NAMECHEAP_SANDBOX=1`, `NCP_PROFILE`.

## TUI

- **Dashboard:** header (profile, balance, sync status) + tabs: Domains / SSL /
  Transfers / Addresses. Domains table: name, expiry (lipgloss color ramp by
  urgency: <30d red, <90d yellow), autorenew, privacy, lock. `/` filter, `enter`
  detail, `p` profile switcher overlay, `?` help, `q` quit.
- **Domain detail:** tabs Overview | DNS | Nameservers | Privacy.
- **Zone editor** (`ncp dns edit`, also reachable from detail): bubbles table of
  records; `a`dd / `e`dit (huh form) / `d`elete stage changes rendered as
  `+`/`~`/`-` diff rows; `A`pply shows full diff + confirm, then calls
  `DNSService.Apply`; `esc` discards.
- **Stale-while-revalidate everywhere:** render from cache instantly, background
  `tea.Cmd` refresh, spinner in header while fetching, swap on arrival. The TUI
  never blocks on the API.
- Wizards (`domains register`, `ssl activate`, `profile add`) are huh forms usable
  both standalone (CLI) and embedded in the TUI.

## Error handling

- Services return domain errors; CLI renders them via fang's styled error output
  with remediation hints; TUI shows a dismissible error banner.
- IP-not-whitelisted is the flagship UX case: show detected public IP + exact fix.
- Partial failures (e.g. `setHosts` succeeded but cache write failed) log a warning,
  never fail the command.

## Testing

- `core/services`: table-driven tests against in-memory fakes of all ports.
- `core/domain`: pure unit tests (zone merge/changeset logic gets the densest coverage).
- `adapters/namecheap`: `httptest.Server` + golden XML fixtures (recorded from sandbox).
- `adapters/turso`: real temp-file DB per test.
- `adapters/tui`: `charmbracelet/x/exp/teatest` snapshot flows for dashboard + zone editor.
- CI: `go vet`, `golangci-lint run`, `go test ./...` (CGO_ENABLED=1).

## Build phases

1. **Skeleton:** hexagon layout, Turso adapter + migrations, profile service +
   `profile add|list|use|rm`, XML client core (auth/throttle/errors),
   `domains list|check|info`, TUI dashboard shell with domains tab.
2. **DNS:** zone aggregate + changeset, full `dns`/`ns` commands, zone editor TUI,
   emailfwd, import/export.
3. **Registrar lifecycle:** register/renew/reactivate wizard, contacts, lock, transfers.
4. **SSL + privacy + account:** full `ssl`, `privacy`, `account`, `address` trees.
5. **Polish:** completions, `--json` coverage audit, Turso sync UX, README, goreleaser.

Each phase ends green (vet + lint + tests) and usable.
