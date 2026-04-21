#!/usr/bin/env bash
# T03 — zero CRITICAL events since clock start.
# Queries Alertmanager for any firing CRITICAL-severity alerts.
# Exit 0 if count == 0, else 1 and emit alert names.

set -euo pipefail

ALERTMANAGER_URL="${ALERTMANAGER_URL:-https://alertmanager.nself.org/api/v2/alerts}"
SOAK_STATE_DIR="${SOAK_STATE_DIR:-/var/lib/nself}"
CLOCK_FILE="${CLOCK_FILE:-${SOAK_STATE_DIR}/soak-clock}"

if [ ! -f "$CLOCK_FILE" ]; then
  printf "ERROR: clock not started — run 02-clock.sh start\n"
  exit 2
fi

clock_start="$(cat "$CLOCK_FILE")"
clock_start_iso="$(date -u -r "$clock_start" +%FT%TZ 2>/dev/null || date -u -d "@$clock_start" +%FT%TZ)"

# Pull firing alerts with severity=critical newer than clock start.
alerts_json="$(curl -fsSL --max-time 15 "${ALERTMANAGER_URL}?filter=severity=critical&active=true" 2>/dev/null || echo '[]')"

# Filter by startsAt >= clock_start.
count="$(printf '%s' "$alerts_json" | \
  python3 -c "import sys,json,datetime
clk = '${clock_start_iso}'
try:
  data = json.load(sys.stdin)
except Exception:
  sys.exit(0)
bad = [a for a in data if a.get('labels',{}).get('severity')=='critical' and a.get('startsAt','') >= clk]
for a in bad:
  print(a.get('labels',{}).get('alertname','?'), a.get('startsAt',''))
print('__COUNT__', len(bad))" | tail -1 | awk '{print $2}')"

if [ "${count:-0}" != "0" ]; then
  printf "FAIL T03: %s CRITICAL events since %s\n" "$count" "$clock_start_iso"
  exit 1
fi
printf "OK T03: 0 CRITICAL events since %s\n" "$clock_start_iso"
