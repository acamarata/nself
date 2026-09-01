#!/bin/bash
# ci-masking-lint.sh — fails if any .github/workflows/*.yml has
# `continue-on-error: true` without an "# ACCEPTED:" comment within 5 lines
# above it. P6-E11-W2-S3-T15 (CI-masking sweep): the ACCEPTED/Expiry pattern
# documented in .claude/docs/doctrines/ci-cd-green.md is the one sanctioned
# way to use continue-on-error; this script is the regression check that
# keeps the sweep's zero-unjustified-hits result from drifting back.
#
# Usage: bash scripts/ci-masking-lint.sh [workflows-dir]
# Exit 0: every continue-on-error: true is preceded by an ACCEPTED comment.
# Exit 1: at least one is not — prints the offending file:line.

set -euo pipefail

WORKFLOWS_DIR="${1:-.github/workflows}"
FAILED=0

if [ ! -d "$WORKFLOWS_DIR" ]; then
  echo "ERROR: workflows dir not found: $WORKFLOWS_DIR" >&2
  exit 1
fi

for f in "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml; do
  [ -f "$f" ] || continue
  # Line numbers of every continue-on-error: true occurrence.
  while IFS=: read -r lineno _; do
    [ -z "$lineno" ] && continue
    start=$((lineno - 5))
    [ "$start" -lt 1 ] && start=1
    context=$(sed -n "${start},${lineno}p" "$f")
    if ! printf '%s' "$context" | grep -q '# ACCEPTED:'; then
      echo "FAIL: $f:$lineno — continue-on-error: true with no ACCEPTED comment within 5 lines above"
      FAILED=1
    fi
  done < <(grep -n 'continue-on-error: *true' "$f" || true)
done

if [ "$FAILED" -eq 1 ]; then
  echo ""
  echo "One or more continue-on-error: true steps lack the required justification."
  echo "Add a comment block directly above the line:"
  echo "  # ACCEPTED: <why this step's failure must not block the job>"
  echo "  # Expiry: permanent | <YYYY-MM-DD>"
  echo "See .claude/docs/doctrines/ci-cd-green.md § The ACCEPTED/Expiry Exception Pattern."
  exit 1
fi

echo "PASS: every continue-on-error: true in $WORKFLOWS_DIR carries an ACCEPTED justification"
