#!/usr/bin/env bash
# T02 — 48h clock manager. Starts, shows, resets the soak clock.
# Clock state lives in $CLOCK_FILE (default /var/lib/nself/soak-clock).
# Usage:
#   02-clock.sh start       -> write now() to clock file; fails if already running
#   02-clock.sh status      -> print hours elapsed, exit 0 if ≥48h, 1 if still running
#   02-clock.sh reset "why" -> reset clock, append reason to event log
#   02-clock.sh elapsed     -> print hours elapsed as float

set -euo pipefail

CLOCK_FILE="${CLOCK_FILE:-/var/lib/nself/soak-clock}"
EVENT_LOG="${EVENT_LOG:-/var/log/nself/soak-events.jsonl}"
TARGET_HOURS="${TARGET_HOURS:-48}"

mkdir -p "$(dirname "$CLOCK_FILE")" "$(dirname "$EVENT_LOG")" 2>/dev/null || true

cmd="${1:-status}"

now_epoch() { date +%s; }

elapsed_hours() {
  if [ ! -f "$CLOCK_FILE" ]; then
    echo "-1"
    return
  fi
  start="$(cat "$CLOCK_FILE")"
  now="$(now_epoch)"
  awk -v s="$start" -v n="$now" 'BEGIN { printf "%.2f", (n - s) / 3600.0 }'
}

case "$cmd" in
  start)
    if [ -f "$CLOCK_FILE" ]; then
      printf "ERROR: clock already running since %s\n" "$(date -r "$(cat "$CLOCK_FILE")" 2>/dev/null || cat "$CLOCK_FILE")"
      exit 2
    fi
    now_epoch > "$CLOCK_FILE"
    printf '{"ts":"%s","event":"clock_start","target_hours":%s}\n' "$(date -u +%FT%TZ)" "$TARGET_HOURS" >> "$EVENT_LOG"
    printf "OK: clock started at %s (target %sh)\n" "$(date -u +%FT%TZ)" "$TARGET_HOURS"
    ;;
  status)
    h="$(elapsed_hours)"
    if [ "$h" = "-1" ]; then
      printf "state=not_started\n"
      exit 3
    fi
    printf "state=running elapsed_hours=%s target_hours=%s\n" "$h" "$TARGET_HOURS"
    awk -v h="$h" -v t="$TARGET_HOURS" 'BEGIN { exit (h + 0 >= t + 0) ? 0 : 1 }'
    ;;
  reset)
    reason="${2:-unspecified}"
    prev="-"
    [ -f "$CLOCK_FILE" ] && prev="$(cat "$CLOCK_FILE")"
    now_epoch > "$CLOCK_FILE"
    printf '{"ts":"%s","event":"clock_reset","reason":"%s","previous_start":"%s"}\n' \
      "$(date -u +%FT%TZ)" "$reason" "$prev" >> "$EVENT_LOG"
    printf "RESET: reason=%s previous_start=%s\n" "$reason" "$prev"
    ;;
  elapsed)
    elapsed_hours
    ;;
  *)
    printf "Usage: %s {start|status|reset <reason>|elapsed}\n" "$0"
    exit 64
    ;;
esac
