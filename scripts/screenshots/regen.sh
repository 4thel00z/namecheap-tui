#!/usr/bin/env zsh
# Regenerate the README screenshots in assets/.
#
# Self-contained: builds ncp, starts a local fake Namecheap API
# (scripts/screenshots/fakeapi), seeds a throwaway profile in a temp HOME,
# captures CLI output and TUI screens (via tmux), and renders everything
# with charmbracelet/freeze.
#
# Requires: go, tmux, freeze (go install github.com/charmbracelet/freeze@latest)
set -euo pipefail

REPO=${0:a:h:h:h}
OUT=$REPO/assets
WORK=$(mktemp -d)
ANSI=$WORK/ansi
FAKE_ADDR="127.0.0.1:8811"
mkdir -p "$OUT" "$ANSI"

cleanup() {
  [[ -n "${FAKE_PID:-}" ]] && kill "$FAKE_PID" 2>/dev/null || true
  tmux -L ncpshoot kill-server 2>/dev/null || true
  rm -rf "$WORK"
}
trap cleanup EXIT

command -v freeze >/dev/null || { echo "freeze not found — go install github.com/charmbracelet/freeze@latest" >&2; exit 1; }
command -v tmux   >/dev/null || { echo "tmux not found" >&2; exit 1; }

# ── Build + fake API ─────────────────────────────────────────────────────
make -C "$REPO" build
NCP=$REPO/bin/ncp
( cd "$REPO/scripts/screenshots/fakeapi" && go build -o "$WORK/fakeapi" . )
"$WORK/fakeapi" & FAKE_PID=$!
sleep 1

# ── Throwaway HOME with a seeded demo profile ────────────────────────────
export HOME=$WORK/home
export NCP_BASE_URL="http://$FAKE_ADDR/xml.response"
export CLICOLOR_FORCE=1
mkdir -p "$HOME"
"$NCP" profile list >/dev/null 2>&1 || true   # create + migrate the DB
DB=$HOME/Library/Application\ Support/ncp/ncp.db
[[ -f "$DB" ]] || DB=$HOME/.config/ncp/ncp.db  # linux
sqlite3 "$DB" "INSERT INTO profiles (name, api_user, username, client_ip, endpoint, is_default)
  VALUES ('personal','4thel00z','4thel00z','203.0.113.7','https://api.namecheap.com/xml.response',1);
  INSERT INTO secrets (profile_id, api_key) SELECT id, 'demo-key' FROM profiles WHERE name='personal';"

PROMPT=$'\e[35m❯\e[0m'
T() { tmux -L ncpshoot "$@"; }

shot() { # shot <name> <display-cmd> <actual-cmd...>
  local name=$1 display=$2; shift 2
  { printf '%s \e[1m%s\e[0m\n' "$PROMPT" "$display"; "$@" 2>&1; } > "$ANSI/$name.ansi"
}

grab() { T capture-pane -e -p -t shoot | sed -e 's/[[:space:]]*$//' > "$ANSI/$1.ansi"; }

# ── CLI shots ────────────────────────────────────────────────────────────
shot domains-list  "ncp domains list"                            "$NCP" domains list
shot domains-check "ncp domains check ravenhold.{com,io,sh,ai}"  "$NCP" domains check ravenhold.com ravenhold.io ravenhold.sh ravenhold.ai
shot dns-get       "ncp dns get ravenhold.dev"                   "$NCP" dns get ravenhold.dev
shot ssl-list      "ncp ssl list"                                "$NCP" ssl list
shot pricing       "ncp account pricing DOMAIN"                  "$NCP" account pricing DOMAIN
shot balance       "ncp account balance"                         "$NCP" account balance

# help needs a PTY (tmux) so fang picks the dark color scheme
T kill-server 2>/dev/null || true
T -f /dev/null new-session -d -s shoot -x 110 -y 50 \
  "env HOME='$HOME' COLORTERM=truecolor '$NCP' --help; sleep 30"
sleep 3
{ printf '%s \e[1m%s\e[0m\n' "$PROMPT" "ncp --help"; T capture-pane -e -p -t shoot; } \
  | sed -e 's/[[:space:]]*$//' | awk 'NF {blank=0} !NF {blank++} blank<2' > "$ANSI/help.ansi"
T kill-server 2>/dev/null || true

# ── TUI: dashboard ───────────────────────────────────────────────────────
T -f /dev/null new-session -d -s shoot -x 100 -y 16 \
  "env HOME='$HOME' NCP_BASE_URL='$NCP_BASE_URL' COLORTERM=truecolor '$NCP'"
sleep 4
grab dashboard-domains
T send-keys -t shoot 2; sleep 2; grab dashboard-ssl
T send-keys -t shoot 3; sleep 2; grab dashboard-transfers
T send-keys -t shoot 4; sleep 2; grab dashboard-addresses
T send-keys -t shoot q; sleep 1
T kill-server 2>/dev/null || true

# ── TUI: zone editor (stage add + edit + delete, then confirm view) ──────
T -f /dev/null new-session -d -s shoot -x 100 -y 16 \
  "env HOME='$HOME' NCP_BASE_URL='$NCP_BASE_URL' COLORTERM=truecolor '$NCP' dns edit ravenhold.dev"
sleep 4
T send-keys -t shoot a; sleep 1                              # add: blog A 76.76.21.99
T send-keys -t shoot -l "blog"; sleep 0.3; T send-keys -t shoot Enter; sleep 0.5
T send-keys -t shoot Enter; sleep 0.5
T send-keys -t shoot -l "76.76.21.99"; sleep 0.3; T send-keys -t shoot Enter; sleep 0.5
T send-keys -t shoot -l "300"; sleep 0.3; T send-keys -t shoot Enter; sleep 0.5
T send-keys -t shoot Enter; sleep 1
T send-keys -t shoot Down Down; sleep 0.3                    # edit: api → new IP
T send-keys -t shoot e; sleep 1
T send-keys -t shoot Enter; sleep 0.4
T send-keys -t shoot Enter; sleep 0.4
T send-keys -t shoot C-u; sleep 0.2
T send-keys -t shoot -l "76.76.21.50"; sleep 0.3; T send-keys -t shoot Enter; sleep 0.4
T send-keys -t shoot Enter; sleep 0.4
T send-keys -t shoot Enter; sleep 1
T send-keys -t shoot Down Down Down; sleep 0.3               # delete: status
T send-keys -t shoot d; sleep 0.5
grab zone-editor
T send-keys -t shoot A; sleep 1
grab zone-editor-confirm
T kill-server 2>/dev/null || true

# ── Render with freeze ───────────────────────────────────────────────────
FREEZE=(--window -r 8 -p 24,32 -m 24 --shadow.blur 24 --shadow.y 12
        --background '#171717' --font.family Menlo)
for f in "$ANSI"/*.ansi; do
  # help is tall; render it smaller to stay under the pre-commit 500 KB cap
  size=14; [[ ${f:t} == help.ansi ]] && size=11
  freeze "$f" "${FREEZE[@]}" --font.size $size -o "$OUT/${${f:t}%.ansi}.png"
done
ls -la "$OUT"
