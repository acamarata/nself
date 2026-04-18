#!/usr/bin/env bash
# P93 S49 soak — run all 10 gate checks.
# Returns 0 only if every gate passes. Called every 5 min by the CI monitor.

set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FAILED=""

gates="01-deploy-verify.sh 02-clock.sh:status 03-critical-events.sh 04-mux-classification.sh 05-null-topics.sh 06-nself-ai-commands.sh 07-pool-keys.sh 08-oauth-reauth.sh 09-fresh-install.sh 10-ollama-fallback.sh"

# T09 fresh-install is too slow for every 5 min — gate it behind SOAK_RUN_T09=1.
# CI workflow runs T09 once per 6h.

printf "=== P93 S49 soak run-all @ %s ===\n" "$(date -u +%FT%TZ)"

for g in $gates; do
  script="${g%%:*}"
  arg="${g##*:}"
  [ "$arg" = "$script" ] && arg=""

  # T09 skip logic
  if [ "$script" = "09-fresh-install.sh" ] && [ "${SOAK_RUN_T09:-0}" != "1" ]; then
    printf "SKIP %s (set SOAK_RUN_T09=1 to run)\n" "$script"
    continue
  fi

  path="$SCRIPT_DIR/$script"
  if [ ! -x "$path" ]; then
    chmod +x "$path" 2>/dev/null || true
  fi

  if "$path" $arg; then
    :
  else
    FAILED="$FAILED $script"
  fi
  printf -- "---\n"
done

if [ -n "$FAILED" ]; then
  printf "FAIL: gates failed:%s\n" "$FAILED"
  exit 1
fi
printf "PASS: all soak gates green\n"
