#!/usr/bin/env bash
# govuln-gate.sh — run govulncheck and fail only on advisories that are not
# explicitly accepted in .github/govulncheck-allow.txt.
#
# Why this exists: govulncheck's own exit code is "found anything at all", which
# cannot distinguish a newly published advisory (must break the build) from one
# we have already reviewed and accepted because upstream ships no fix. A bare
# `govulncheck ./...` therefore either blocks every build forever or gets
# disabled wholesale. This narrows the gate instead of removing it.
#
# Single source of truth for every workflow that scans Go vulns:
#   govuln.yml · security-scan.yml · deep-qa-block.yml · govulncheck-nightly.yml
#
# Usage:   bash .github/scripts/govuln-gate.sh [packages]      (default ./...)
# Exit:    0 = only accepted advisories (or none)
#          1 = at least one advisory not in the allowlist
#
# Constraints: bash 3.2 compatible (no mapfile/associative arrays); writes only
# under ${RUNNER_TEMP:-/tmp} so it can never trip the clean-root gate.

set -uo pipefail

PKGS="${1:-./...}"
REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ALLOW_FILE="${REPO_ROOT}/.github/govulncheck-allow.txt"
TMPDIR_="${RUNNER_TEMP:-/tmp}"
REPORT="${TMPDIR_}/govuln-report.txt"

if [ ! -f "${ALLOW_FILE}" ]; then
  printf '::error::missing allowlist: %s\n' "${ALLOW_FILE}"
  exit 1
fi

# Non-zero just means "found something"; the allowlist decides pass/fail.
GOFLAGS="${GOFLAGS:--mod=vendor}" govulncheck "${PKGS}" > "${REPORT}" 2>&1 || true
cat "${REPORT}"

# Strip trailing comments so an ID mentioned in prose cannot silently allow itself.
sed 's/#.*//' "${ALLOW_FILE}" \
  | grep -oE 'GO-[0-9]{4}-[0-9]+' | sort -u > "${TMPDIR_}/govuln-allow.ids"
grep -oE 'GO-[0-9]{4}-[0-9]+' "${REPORT}" | sort -u > "${TMPDIR_}/govuln-found.ids"

comm -23 "${TMPDIR_}/govuln-found.ids" "${TMPDIR_}/govuln-allow.ids" \
  > "${TMPDIR_}/govuln-unexpected.ids"
comm -13 "${TMPDIR_}/govuln-found.ids" "${TMPDIR_}/govuln-allow.ids" \
  > "${TMPDIR_}/govuln-stale.ids"

FOUND=$(grep -c . "${TMPDIR_}/govuln-found.ids" || true)
UNEXPECTED=$(grep -c . "${TMPDIR_}/govuln-unexpected.ids" || true)
STALE=$(grep -c . "${TMPDIR_}/govuln-stale.ids" || true)

if [ "${STALE}" -gt 0 ]; then
  # Keeps the allowlist from rotting into a blanket suppression.
  printf '::warning::allowlist entries no longer reported, remove them: %s\n' \
    "$(tr '\n' ' ' < "${TMPDIR_}/govuln-stale.ids")"
fi

if [ "${UNEXPECTED}" -gt 0 ]; then
  printf '::error::govulncheck reported advisories not in .github/govulncheck-allow.txt:\n'
  while IFS= read -r id; do
    [ -n "${id}" ] && printf '::error::  %s (https://pkg.go.dev/vuln/%s)\n' "${id}" "${id}"
  done < "${TMPDIR_}/govuln-unexpected.ids"
  printf '::error::Fix the dependency, or add the ID with a reason and tracking issue.\n'
  exit 1
fi

printf 'govuln gate passed: %s advisory/ies reported, all accepted, 0 unexpected\n' "${FOUND}"
