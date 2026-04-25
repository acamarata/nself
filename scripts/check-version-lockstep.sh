#!/usr/bin/env bash
# check-version-lockstep.sh
#
# Verifies CLI version.go matches admin/package.json.
# Per nSelf PPI Hard Rule: CLI and Admin ship the same version number.
#
# Usage:
#   bash scripts/check-version-lockstep.sh
#
# Environment overrides:
#   CLI_VERSION_FILE   path to internal/version/version.go (default: relative to script)
#   ADMIN_PKG_JSON     path to admin's package.json (default: ../admin/package.json)
#
# Exit codes:
#   0  — versions match
#   1  — versions differ or a file is not found
#
# Wired as a pre-step in release.yml and release-dryrun.yml.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

CLI_VERSION_FILE="${CLI_VERSION_FILE:-${REPO_ROOT}/internal/version/version.go}"
ADMIN_PKG_JSON="${ADMIN_PKG_JSON:-${REPO_ROOT}/../admin/package.json}"

# ── Read CLI version ─────────────────────────────────────────────────────────
if [ ! -f "${CLI_VERSION_FILE}" ]; then
  echo "ERROR: CLI version file not found: ${CLI_VERSION_FILE}" >&2
  exit 1
fi

CLI_VERSION=$(grep -E '^\s+Version\s+string\s*=' "${CLI_VERSION_FILE}" \
  | grep -oE '"[0-9]+\.[0-9]+\.[0-9]+"' \
  | tr -d '"' \
  | head -1)

if [ -z "${CLI_VERSION}" ]; then
  echo "ERROR: Could not parse Version from ${CLI_VERSION_FILE}" >&2
  echo "  Expected a line like:  Version   string = \"1.0.11\"" >&2
  exit 1
fi

# ── Read Admin version ────────────────────────────────────────────────────────
if [ ! -f "${ADMIN_PKG_JSON}" ]; then
  echo "ERROR: Admin package.json not found: ${ADMIN_PKG_JSON}" >&2
  echo "  Set ADMIN_PKG_JSON env var to override the path." >&2
  exit 1
fi

# Use python or node if available; fall back to grep for environments with neither.
if command -v python3 &>/dev/null; then
  ADMIN_VERSION=$(python3 -c "import json,sys; d=json.load(open('${ADMIN_PKG_JSON}')); print(d['version'])")
elif command -v node &>/dev/null; then
  ADMIN_VERSION=$(node -e "const d=require('${ADMIN_PKG_JSON}'); console.log(d.version)")
else
  ADMIN_VERSION=$(grep -E '"version"\s*:' "${ADMIN_PKG_JSON}" \
    | grep -oE '"[0-9]+\.[0-9]+\.[0-9]+"' \
    | tr -d '"' \
    | head -1)
fi

if [ -z "${ADMIN_VERSION}" ]; then
  echo "ERROR: Could not parse version from ${ADMIN_PKG_JSON}" >&2
  exit 1
fi

# ── Compare ───────────────────────────────────────────────────────────────────
echo "CLI   version: ${CLI_VERSION}  (${CLI_VERSION_FILE})"
echo "Admin version: ${ADMIN_VERSION}  (${ADMIN_PKG_JSON})"

if [ "${CLI_VERSION}" != "${ADMIN_VERSION}" ]; then
  echo ""
  echo "VERSION LOCKSTEP FAILED"
  echo "  CLI is at v${CLI_VERSION} but Admin is at v${ADMIN_VERSION}"
  echo ""
  echo "  Fix: bump the lower one to match, then re-push."
  echo "  Canonical rule: CLI and Admin ship the same version (nSelf PPI, Product Strategy)"
  exit 1
fi

echo ""
echo "Version lockstep OK: CLI == Admin == v${CLI_VERSION}"
