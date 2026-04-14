#!/usr/bin/env bash
# backup-age-decrypt-test.sh
#
# Verifies that the latest full backup in B2 is decryptable with the
# on-host age identity and that the tar contains the expected PostgreSQL
# directories. Used by G-29-03 on nclaw-prod.
#
# Exit codes:
#   0  decrypt + tar listing succeeded
#   2  rclone locate/copy failed
#   3  age decryption failed
#   4  tar listing missing required directories

set -euo pipefail

REMOTE="${1:-b2-nclaw}"
BUCKET_PATH="${2:-nclaw-backups/nclaw-prod/prod}"
AGE_KEY="${AGE_KEY:-/etc/nself/age-key.txt}"
AGE_KEY_V2="${AGE_KEY_V2:-/etc/nself/age-key-v2.txt}"
WORK_DIR="$(mktemp -d -t nself-backup-age-XXXXXX)"

cleanup() {
  rm -rf "$WORK_DIR"
}
trap cleanup EXIT

log() { printf '[%s] %s\n' "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$*"; }

# 1. Locate latest full backup artifact in B2.
log "locating latest full backup under ${REMOTE}:${BUCKET_PATH}"
latest="$(rclone lsf --files-only --recursive --include '*_full_*.tar.age' \
  "${REMOTE}:${BUCKET_PATH}" | sort | tail -n1 || true)"

if [ -z "$latest" ]; then
  log "ERROR: no full backup tar.age found in ${REMOTE}:${BUCKET_PATH}"
  exit 2
fi
log "latest artifact: ${latest}"

# 2. Copy locally.
local_path="${WORK_DIR}/$(basename "$latest")"
log "rclone copyto -> ${local_path}"
if ! rclone copyto "${REMOTE}:${BUCKET_PATH}/${latest}" "${local_path}"; then
  log "ERROR: rclone copyto failed"
  exit 2
fi

# 3. Decrypt with the primary age key; fall back to v2 if needed.
decrypted="${WORK_DIR}/decrypted.tar"
if age -d -i "$AGE_KEY" -o "$decrypted" "$local_path" 2>"${WORK_DIR}/age.err"; then
  log "decrypted with primary key"
elif [ -f "$AGE_KEY_V2" ] && age -d -i "$AGE_KEY_V2" -o "$decrypted" "$local_path" 2>"${WORK_DIR}/age.err"; then
  log "decrypted with v2 key"
else
  log "ERROR: age decryption failed"
  cat "${WORK_DIR}/age.err" >&2 || true
  exit 3
fi

# 4. Inspect tar listing for expected PG directories.
listing="$(tar tf "$decrypted" 2>"${WORK_DIR}/tar.err" | head -n 200 || true)"
if [ -z "$listing" ]; then
  log "ERROR: tar listing empty"
  cat "${WORK_DIR}/tar.err" >&2 || true
  exit 4
fi

missing=0
for required in base/ pg_wal/ pg_tblspc/; do
  if ! printf '%s\n' "$listing" | grep -q "^${required}"; then
    log "missing required directory: ${required}"
    missing=1
  else
    log "found: ${required}"
  fi
done

if [ "$missing" -ne 0 ]; then
  log "ERROR: tar listing did not contain all required PG directories"
  exit 4
fi

log "PASS: backup artifact decrypted and contains base/ pg_wal/ pg_tblspc/"
