#!/usr/bin/env bash
# Purpose: enforce that every first-party Go file in the repo is gofmt-clean.
# Inputs:  optional list of directories to check (defaults to cmd internal tools).
# Outputs: exit 0 when clean; exit 1 listing the offending files otherwise.
# Constraints: bash 3.2 compatible; skips vendor/ and testdata/ (fixtures are
#              deliberately not required to be canonical Go formatting).
set -euo pipefail

DIRS=("$@")
if [ ${#DIRS[@]} -eq 0 ]; then
  DIRS=(cmd internal tools)
fi

EXISTING=()
for d in "${DIRS[@]}"; do
  [ -d "$d" ] && EXISTING+=("$d")
done

if [ ${#EXISTING[@]} -eq 0 ]; then
  echo "gofmt-check: no target directories found" >&2
  exit 1
fi

OUT=$(gofmt -l "${EXISTING[@]}" | grep -v '/testdata/' || true)

if [ -n "$OUT" ]; then
  echo "gofmt check FAILED. The following files are not formatted:" >&2
  echo "$OUT" >&2
  echo "" >&2
  echo "Fix with: make fmt" >&2
  exit 1
fi

echo "gofmt check passed (${EXISTING[*]})."
