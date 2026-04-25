#!/usr/bin/env bash
# dev-watch.sh — hot-reload a Go plugin during development.
#
# Usage:
#   dev-watch.sh                 # auto-detects ./cmd entrypoint
#   dev-watch.sh ./cmd/mywidget  # explicit entrypoint
#
# Prefers `air` (https://github.com/air-verse/air). Falls back to a polling
# loop with `fswatch` → `go run`. Designed to pair with `nself dev` so
# changes to plugin source rebuild + restart the container-less dev process.
set -euo pipefail

ENTRY="${1:-./cmd}"

if [ ! -d "$ENTRY" ] && [ ! -f "$ENTRY" ]; then
  printf 'dev-watch: entrypoint %s not found\n' "$ENTRY" >&2
  exit 1
fi

if command -v air >/dev/null 2>&1; then
  if [ ! -f ./.air.toml ]; then
    cat >./.air.toml <<'TOML'
root = "."
tmp_dir = "tmp"
[build]
  cmd = "go build -o ./tmp/plugin ./cmd"
  bin = "tmp/plugin"
  delay = 500
  include_ext = ["go", "yaml", "yml"]
  exclude_dir = ["tmp", "vendor", ".git"]
[log]
  time = true
[color]
  app = "magenta"
TOML
  fi
  exec air
fi

if command -v fswatch >/dev/null 2>&1; then
  printf 'dev-watch: air not installed; falling back to fswatch poll (install air for better UX)\n' >&2
  while true; do
    (go run "$ENTRY" &)
    pid=$!
    fswatch -1 -e ".*" -i "\\.go$" -i "\\.ya?ml$" . >/dev/null
    kill "$pid" 2>/dev/null || true
    wait "$pid" 2>/dev/null || true
    sleep 0.5
  done
fi

printf 'dev-watch: neither air nor fswatch is installed.\n' >&2
printf 'install one of:\n  go install github.com/air-verse/air@latest\n  brew install fswatch\n' >&2
exit 1
