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
