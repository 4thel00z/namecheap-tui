# ncp

A fast TUI and CLI for the Namecheap API, in Go. Hexagonal architecture,
Turso/libSQL-backed config and cache, charmbracelet UI (bubbletea, lipgloss,
huh, fang).

## Install

    make build   # produces bin/ncp (CGo required for libSQL)

## Quick start

    ncp profile add        # interactive; auto-detects your public IP
    ncp                    # full-screen dashboard (domains, ssl, transfers, addresses)
    ncp domains list       # scriptable table
    ncp domains list --json | jq .
    ncp dns edit example.com   # interactive zone editor with staged changes

Namecheap requires API access to be enabled on your account and your public
IP whitelisted (Profile → Tools → API Access). `ncp profile add` detects the
IP for you and live-verifies the credentials before saving.

## Commands

| Tree | Commands |
|---|---|
| `profile` | `add` `list` `use` `rm` |
| `domains` | `list` `check` `info` `register` `renew` `reactivate` `tlds` `lock get\|on\|off` `contacts get\|set` |
| `dns` | `get` `add` `rm` `import`(alias `set`) `export` `use-default` `use-custom` `emailfwd get\|set` `edit` |
| `ns` | `create` `update` `delete` `info` |
| `transfer` | `create` `status [--resubmit]` `list` |
| `ssl` | `list` `create` `activate` `info` `renew` `reissue` `approvers` `resend-approver` `revoke` |
| `privacy` | `list` `enable` `disable` `renew` `change-email` `assign` `unassign` `discard` |
| `account` | `balance` `pricing` |
| `address` | `list` `info` `add` `edit` `rm` `default` |
| `config` | `get` `set` `sync` |

Global flags: `--profile <name>`, `--json` (all read commands), `--no-cache`,
`--sandbox`. Shell completions: `ncp completion <shell>`.

### The zone editor

`ncp dns edit <domain>` opens a staged-changes editor: `a`dd, `e`dit,
`d`elete, `u`ndo, then `A`pply shows the full diff before anything is sent.
Namecheap's `setHosts` replaces the entire zone, so ncp always merges your
changes onto a freshly fetched zone — single-record edits can never wipe
unrelated records.

### Zone snapshots (DNS as code)

    ncp dns export example.com > zone.yaml   # snapshot
    ncp dns import example.com -f zone.yaml  # replace zone from snapshot

## Configuration

- Profiles, settings, and the API cache live in `~/.config/ncp/ncp.db`
  (libSQL, file mode 0600).
- **Turso sync:** set `TURSO_DATABASE_URL` + `TURSO_AUTH_TOKEN` and the DB
  opens as an embedded replica; `ncp config sync` pushes/pulls.
- Env overrides (beat the stored profiles): `NAMECHEAP_API_KEY`,
  `NAMECHEAP_API_USER`, `NAMECHEAP_USERNAME`, `NAMECHEAP_CLIENT_IP`,
  `NAMECHEAP_SANDBOX=1`, `NCP_PROFILE`.

## Architecture

```
cmd/ncp            composition root (fang + wiring)
internal/core      pure domain + ports + services (no I/O)
internal/adapters  namecheap XML client · turso store · iputil · cli · tui
```

Rules: `core` imports only stdlib; adapters import `core` only, never each
other. All API calls are throttled client-side to Namecheap's published
limits (20/min, 700/hr, 8000/day) and read paths are cached
(stale-while-revalidate in the TUI).

## Development

    make test lint fmt   # CGO_ENABLED=1 go test -race, golangci-lint, gofumpt
    make hooks           # install pre-commit + conventional-commit msg hook

Conventional commits drive [release-please](https://github.com/googleapis/release-please);
releases build CGo binaries on native runners (linux/amd64+arm64,
darwin/amd64+arm64) and attach them to the GitHub release.

Design docs: `docs/superpowers/specs/`, plans: `docs/superpowers/plans/`.
