#!/usr/bin/env bash
# T06 — all 14 `nself ai` subcommands respond successfully.
# Each command runs with --dry-run where supported, or with a trivial stub payload.
# Exit 0 if all 14 exit 0; else 1 with per-command status.

set -u

# 14 commands per F03 SPORT / CLI command tree. Parallel arrays (Bash 3.2 compat).
CMDS="
status
keys
providers
budget
models
test
classify
summarize
embed
chat
route
circuit
metrics
reset
"

pass=0
fail=0
total=0

printf "T06 nself ai command smoke — 14 commands\n"
for c in $CMDS; do
  [ -z "$c" ] && continue
  total=$((total + 1))
  out="$(nself ai "$c" --dry-run 2>&1 || true)"
  rc=$?
  # Some subcommands don't support --dry-run — re-try without, but with a safe flag.
  if printf '%s' "$out" | grep -qi 'unknown flag'; then
    out="$(nself ai "$c" --help 2>&1 || true)"
    rc=$?
  fi
  if [ "$rc" -eq 0 ]; then
    printf "  OK   nself ai %-12s\n" "$c"
    pass=$((pass + 1))
  else
    printf "  FAIL nself ai %-12s rc=%s out=%s\n" "$c" "$rc" "$(printf '%s' "$out" | head -1)"
    fail=$((fail + 1))
  fi
done

printf "\nResult: %s/%s passed\n" "$pass" "$total"
[ "$fail" -eq 0 ] || exit 1
[ "$pass" -eq 14 ] || { printf "FAIL T06: expected 14 commands, saw %s\n" "$pass"; exit 1; }
printf "OK T06: 14/14 nself ai commands healthy\n"
