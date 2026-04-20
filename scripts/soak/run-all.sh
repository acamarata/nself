#!/usr/bin/env bash
# Soak harness — run all 10 gate checks.
# Returns 0 only if every gate passes. Called every 5 min by the CI monitor.
#
# Usage:
#   run-all.sh [--dry-run] [--duration <hours>] [--gate <name>]
#
# Flags:
#   --dry-run           Print which gates would run without executing them.
#   --duration <hours>  Override TARGET_HOURS for the clock gate (default: 48).
#   --gate <name>       Run only the named gate script (e.g. 02-clock.sh).

set -u

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
FAILED=""
DRY_RUN=0
ONLY_GATE=""

# Arg parsing (Bash 3.2 compatible — no getopt, no ${var,,}).
while [ "$#" -gt 0 ]; do
  case "$1" in
    --dry-run)
      DRY_RUN=1
      shift
      ;;
    --duration)
      if [ -z "${2:-}" ]; then
        printf "ERROR: --duration requires a value\n" >&2
        exit 1
      fi
      export TARGET_HOURS="$2"
      shift 2
      ;;
    --gate)
      if [ -z "${2:-}" ]; then
        printf "ERROR: --gate requires a value\n" >&2
        exit 1
      fi
      ONLY_GATE="$2"
      shift 2
      ;;
    *)
      printf "ERROR: unknown argument: %s\n" "$1" >&2
      printf "Usage: %s [--dry-run] [--duration <hours>] [--gate <name>]\n" "$0" >&2
      exit 1
      ;;
  esac
done

gates="01-deploy-verify.sh 02-clock.sh:status 03-critical-events.sh 04-mux-classification.sh 05-null-topics.sh 06-nself-ai-commands.sh 07-pool-keys.sh 08-oauth-reauth.sh 09-fresh-install.sh 10-ollama-fallback.sh"

# T09 fresh-install is too slow for every 5 min — gate it behind SOAK_RUN_T09=1.
# CI workflow runs T09 once per 6h.

printf "=== soak run-all @ %s ===\n" "$(date -u +%FT%TZ)"
if [ "$DRY_RUN" = "1" ]; then
  printf "DRY-RUN mode — no gates will execute\n"
fi
if [ -n "$ONLY_GATE" ]; then
  printf "Single-gate mode — running only: %s\n" "$ONLY_GATE"
fi

for g in $gates; do
  script="${g%%:*}"
  arg="${g##*:}"
  [ "$arg" = "$script" ] && arg=""

  # Single-gate filter.
  if [ -n "$ONLY_GATE" ] && [ "$script" != "$ONLY_GATE" ]; then
    continue
  fi

  # T09 skip logic.
  if [ "$script" = "09-fresh-install.sh" ] && [ "${SOAK_RUN_T09:-0}" != "1" ]; then
    printf "SKIP %s (set SOAK_RUN_T09=1 to run)\n" "$script"
    continue
  fi

  if [ "$DRY_RUN" = "1" ]; then
    printf "WOULD RUN: %s %s\n" "$script" "$arg"
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

if [ "$DRY_RUN" = "1" ]; then
  printf "DRY-RUN complete.\n"
  exit 0
fi

if [ -n "$FAILED" ]; then
  printf "FAIL: gates failed:%s\n" "$FAILED"
  exit 1
fi
printf "PASS: all soak gates green\n"
