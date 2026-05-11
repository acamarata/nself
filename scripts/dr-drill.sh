#!/usr/bin/env bash
# dr-drill.sh — operator-facing weekly DR drill wrapper (S9.T03 + S9.T15).
#
# Runs `nself backup drill --json`, parses the structured result, and posts a
# one-line summary to a notification webhook (Slack-compatible JSON, also works
# with Telegram / Mattermost / Discord webhook adapters).
#
# Intended for cron:
#   # Weekly drill, Sundays at 03:00 UTC, with notification on failure.
#   0 3 * * 0  cd /opt/nself && /opt/nself/cli/scripts/dr-drill.sh
#
# Environment:
#   NSELF_BIN              path to the nself binary (default: nself in PATH)
#   NOTIFY_WEBHOOK_URL     webhook URL; empty = no notification (still logs locally)
#   NOTIFY_ON              "always" | "failure" (default: failure)
#   DR_DRILL_RTO_HOURS     RTO target override; default 4
#   DR_DRILL_LOG           local log file (default: /var/log/nself/dr-drill.log)
#
# Bash 3.2 compatible (no associative arrays, no echo -e, no ${var,,}).

set -u

NSELF_BIN="${NSELF_BIN:-nself}"
NOTIFY_WEBHOOK_URL="${NOTIFY_WEBHOOK_URL:-}"
NOTIFY_ON="${NOTIFY_ON:-failure}"
DR_DRILL_RTO_HOURS="${DR_DRILL_RTO_HOURS:-4}"
DR_DRILL_LOG="${DR_DRILL_LOG:-/var/log/nself/dr-drill.log}"

ts() { date -u +"%Y-%m-%dT%H:%M:%SZ"; }

log_line() {
  local msg="$1"
  printf "[%s] %s\n" "$(ts)" "$msg"
  if [ -n "$DR_DRILL_LOG" ]; then
    mkdir -p "$(dirname "$DR_DRILL_LOG")" 2>/dev/null || true
    printf "[%s] %s\n" "$(ts)" "$msg" >>"$DR_DRILL_LOG" 2>/dev/null || true
  fi
}

# notify_webhook posts a JSON payload to NOTIFY_WEBHOOK_URL. Slack-style schema
# (text + attachment color) — Mattermost / Discord / Telegram bridges all
# accept this shape via small adapters.
notify_webhook() {
  local color="$1"
  local text="$2"

  if [ -z "$NOTIFY_WEBHOOK_URL" ]; then
    return 0
  fi

  # Escape inner quotes in text for JSON (basic — drill output is structured
  # and never contains untrusted user content).
  local escaped
  escaped=$(printf '%s' "$text" | sed 's/"/\\"/g')

  local payload
  payload=$(printf '{"text":"nself DR drill","attachments":[{"color":"%s","text":"%s"}]}' \
    "$color" "$escaped")

  # 5-second timeout — never block the cron beyond this if the webhook is down.
  curl -sS --max-time 5 -X POST -H "Content-Type: application/json" \
    -d "$payload" "$NOTIFY_WEBHOOK_URL" >/dev/null 2>&1 || \
    log_line "WARN webhook POST failed (continuing)"
}

main() {
  log_line "starting DR drill (rto-hours=$DR_DRILL_RTO_HOURS)"

  local out
  local rc
  out=$("$NSELF_BIN" backup drill --json --rto-hours "$DR_DRILL_RTO_HOURS" 2>&1)
  rc=$?

  # Try to extract a few fields without requiring jq — keep this script
  # dependency-free. The fields are stable per backup.DrillResult.
  local success duration_s rto_met
  success=$(printf '%s' "$out" | grep -E '"success"' | head -1 | sed -E 's/.*"success"[[:space:]]*:[[:space:]]*([^,}]*).*/\1/' | tr -d ' ')
  duration_s=$(printf '%s' "$out" | grep -E '"total_duration_ns"' | head -1 | sed -E 's/.*"total_duration_ns"[[:space:]]*:[[:space:]]*([0-9]+).*/\1/')
  rto_met=$(printf '%s' "$out" | grep -E '"rto_target_met"' | head -1 | sed -E 's/.*"rto_target_met"[[:space:]]*:[[:space:]]*([^,}]*).*/\1/' | tr -d ' ')

  # Convert nanoseconds → seconds (integer, truncated) without bc.
  local duration_h="?"
  if [ -n "$duration_s" ]; then
    duration_h=$(awk "BEGIN{printf \"%.2f\", $duration_s/3600/1000000000}")
  fi

  local status_text
  if [ "$rc" -eq 0 ] && [ "$success" = "true" ]; then
    status_text="DRILL PASS (duration=${duration_h}h, rto_target_met=${rto_met})"
    log_line "$status_text"
    if [ "$NOTIFY_ON" = "always" ]; then
      notify_webhook "good" "$status_text"
    fi
    return 0
  fi

  status_text="DRILL FAIL (rc=$rc, duration=${duration_h}h)"
  log_line "$status_text"
  log_line "raw output: $out"
  notify_webhook "danger" "$status_text"
  return 1
}

main "$@"
