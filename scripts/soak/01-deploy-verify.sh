#!/usr/bin/env bash
# T01 — verify every prod surface is serving the P93 build.
# Exits 0 if every endpoint returns 200 and reports the expected version.

set -euo pipefail

TARGET_VERSION="${TARGET_VERSION:-1.0.9}"

# URL / health-endpoint / expected-version-substring, one per line.
# Kept as parallel strings (no associative arrays — Bash 3.2 compat for plugin scripts).
ENDPOINTS="
nself.org|https://nself.org/api/version|${TARGET_VERSION}
docs.nself.org|https://docs.nself.org/api/version|${TARGET_VERSION}
chat.nself.org|https://chat.nself.org/api/version|${TARGET_VERSION}
claw.nself.org|https://claw.nself.org/api/version|${TARGET_VERSION}
task.nself.org|https://task.nself.org/api/version|${TARGET_VERSION}
cloud.nself.org|https://cloud.nself.org/api/version|${TARGET_VERSION}
install.nself.org|https://install.nself.org/api/version|${TARGET_VERSION}
base.nself.org|https://base.nself.org/api/version|${TARGET_VERSION}
ntv.nself.org|https://ntv.nself.org/api/version|${TARGET_VERSION}
api.nself.org|https://api.nself.org/healthz|ok
ping.nself.org|https://ping.nself.org/health|ok
"

fail=0
printf "T01 deploy verify — target=%s\n" "$TARGET_VERSION"

printf '%s\n' "$ENDPOINTS" | while IFS='|' read -r name url expect; do
  [ -z "$name" ] && continue
  body="$(curl -fsSL --max-time 10 "$url" 2>/dev/null || echo FAIL)"
  if printf '%s' "$body" | grep -q "$expect"; then
    printf "  OK   %-20s %s\n" "$name" "$url"
  else
    printf "  FAIL %-20s %s got=%s expected=%s\n" "$name" "$url" "$body" "$expect"
    fail=1
  fi
done

exit "$fail"
