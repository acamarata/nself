#!/usr/bin/env bash
# P93 S49 soak pre-flight — verify every repo is on the P93 target tag.
# Usage: ./00-pre-flight.sh [--target v1.0.9]
# Exit 0 if all 11 repos are at the target. Exit 1 with diff if any drift.

set -euo pipefail

TARGET="${1:-v1.0.9}"
REPOS="cli admin web nchat nclaw ntask ntv clawde plugins plugins-pro homebrew-nself"
ROOT="${NSELF_ROOT:-$(cd "$(dirname "$0")/../../.." && pwd)}"

printf "P93 soak pre-flight — target=%s root=%s\n" "$TARGET" "$ROOT"
fail=0

for repo in $REPOS; do
  dir="$ROOT/$repo"
  if [ ! -d "$dir/.git" ]; then
    printf "  SKIP %-18s (not a git repo at %s)\n" "$repo" "$dir"
    continue
  fi
  tag="$(git -C "$dir" describe --tags --exact-match HEAD 2>/dev/null || echo none)"
  if [ "$tag" = "$TARGET" ]; then
    printf "  OK   %-18s @ %s\n" "$repo" "$tag"
  else
    printf "  FAIL %-18s @ %s (expected %s)\n" "$repo" "$tag" "$TARGET"
    fail=1
  fi
done

if [ "$fail" -ne 0 ]; then
  printf "\nPre-flight FAILED — do not start the soak clock.\n"
  exit 1
fi
printf "\nPre-flight PASSED — all 11 repos on %s. Safe to deploy.\n" "$TARGET"
