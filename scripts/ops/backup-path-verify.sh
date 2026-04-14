#!/usr/bin/env bash
# backup-path-verify.sh
#
# Confirms backup artifacts land in Backblaze B2 under the expected
# daily path with full/ and WAL/ subdirs. Used by G-29-02 on nclaw-prod.
#
# Usage: backup-path-verify.sh [REMOTE] [BUCKET_ROOT] [YYYY] [MM]

set -euo pipefail

REMOTE="${1:-b2-nclaw}"
ROOT="${2:-nclaw-backups/nclaw-prod/prod}"
YEAR="${3:-$(date -u +%Y)}"
MONTH="${4:-$(date -u +%m)}"

log() { printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

path="${REMOTE}:${ROOT}/${YEAR}/${MONTH}/"
log "listing ${path}"

days="$(rclone lsf "$path" --dirs-only 2>/dev/null | sort || true)"
if [ -z "$days" ]; then
  log "ERROR: no day directories under ${path}"
  exit 2
fi

log "found day directories:"
printf '%s\n' "$days" | sed 's/^/  /'

today="$(date -u +%d)"
if ! printf '%s\n' "$days" | grep -q "^${today}/"; then
  log "WARN: today's directory (${today}) not yet present"
fi

missing_full=0
missing_wal=0
while IFS= read -r day; do
  [ -z "$day" ] && continue
  day_path="${path}${day}"
  subdirs="$(rclone lsf "$day_path" --dirs-only 2>/dev/null || true)"
  if ! printf '%s\n' "$subdirs" | grep -qE '^[0-9]{4}-full/$'; then
    log "missing HHMM-full/ under ${day_path}"
    missing_full=$((missing_full + 1))
  fi
  if ! printf '%s\n' "$subdirs" | grep -qE '^[0-9]{4}-wal/$'; then
    log "missing HHMM-wal/ under ${day_path}"
    missing_wal=$((missing_wal + 1))
  fi
done <<EOF
${days}
EOF

if [ "$missing_full" -gt 0 ] || [ "$missing_wal" -gt 0 ]; then
  log "FAIL: missing_full=${missing_full} missing_wal=${missing_wal}"
  exit 3
fi
log "PASS: every day has HHMM-full/ and HHMM-wal/"
