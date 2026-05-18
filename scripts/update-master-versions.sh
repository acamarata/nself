#!/usr/bin/env bash
# update-master-versions.sh — refresh MASTER-VERSIONS.md version fields after a release
#
# Usage:
#   ./scripts/update-master-versions.sh <version>
#
# Example (called by release.yml after successful tag push):
#   ./scripts/update-master-versions.sh 1.2.0
#
# What it updates:
#   - CLI version row (from .github/VERSION)
#   - homebrew-nself formula version + SHA256 (SHA is "HELD — binary-dependent" pre-release)
#   - Last-refreshed timestamp
#
# Held items (require binary to exist before population):
#   - homebrew sha256: populated by homebrew-nself formula update step in release.yml
#     (after goreleaser builds the tarball, the sha can be computed)
#   - ping_api LATEST_CLI_VERSION: updated by Downloads agent post-deploy (UA-02)
#
# This script is SAFE to call pre-release for dry-run verification.
# It does NOT push, tag, or publish anything.
set -euo pipefail

VERSION="${1:?Usage: $0 <version>  (e.g. 1.2.0)}"
MASTER_VERSIONS_PATH="${MASTER_VERSIONS_PATH:-$(dirname "$0")/../../.claude/docs/MASTER-VERSIONS.md}"

if [ ! -f "$MASTER_VERSIONS_PATH" ]; then
  echo "ERROR: MASTER-VERSIONS.md not found at $MASTER_VERSIONS_PATH"
  exit 1
fi

TODAY=$(date -u +%Y-%m-%d)
TIMESTAMP=$(date -u +%Y-%m-%dT%H:%M:%SZ)

echo "=== MASTER-VERSIONS auto-update (v$VERSION, $TODAY) ==="

# --- 1. CLI version row ---
echo "  Updating CLI version to v${VERSION}..."
sed -i.bak \
  "s/| CLI | \*\*v[0-9.]*\*\*/| CLI | **v${VERSION}**/g" \
  "$MASTER_VERSIONS_PATH"

# --- 2. homebrew-nself formula version ---
echo "  Updating homebrew-nself formula version..."
sed -i.bak \
  "s/| homebrew-nself formula | \*\*v[0-9.]*\*\*/| homebrew-nself formula | **v${VERSION}**/g" \
  "$MASTER_VERSIONS_PATH"

# --- 3. admin Docker image version (lockstep) ---
echo "  Updating admin Docker image version (lockstep with CLI)..."
sed -i.bak \
  "s/| admin Docker \`nself\/nself-admin\` | \*\*v[0-9.]*\*\*/| admin Docker \`nself\/nself-admin\` | **v${VERSION}**/g" \
  "$MASTER_VERSIONS_PATH"

# --- 4. Last-refreshed timestamp ---
echo "  Updating Last refreshed timestamp..."
sed -i.bak \
  "s/^\*\*Last refreshed:\*\* .*/**Last refreshed:** ${TODAY} (auto-update from update-master-versions.sh v${VERSION})/g" \
  "$MASTER_VERSIONS_PATH"

# --- 5. Cleanup backup files created by sed -i.bak ---
rm -f "${MASTER_VERSIONS_PATH}.bak"

echo ""
echo "=== Held items (require binary before population) ==="
echo "  homebrew sha256: HELD — computed after goreleaser tarball exists."
echo "    Command: sha256sum nself-${VERSION}-darwin-amd64.tar.gz"
echo "    Then: sed -i 's/sha256 \".*\"/sha256 \"<hash>\"/' Formula/nself.rb"
echo "  ping_api LATEST_CLI_VERSION: HELD — Downloads agent updates post-deploy (UA-02)."
echo ""
echo "=== MASTER-VERSIONS update complete (v${VERSION}) ==="
echo "  File: $MASTER_VERSIONS_PATH"
echo "  Review and commit as part of the release PR."
