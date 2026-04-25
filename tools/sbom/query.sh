#!/bin/sh
# tools/sbom/query.sh — query a nSelf CLI SBOM (Q04)
#
# Usage:
#   ./query.sh <version> --pkg <name>
#     Search for a package by name in the release SBOM.
#
#   ./query.sh <version> --cve <CVE-ID>
#     List packages that a given CVE affects (requires grype).
#
#   ./query.sh --local <sbom-file> --pkg <name>
#     Query a locally-downloaded SBOM.
#
#   ./query.sh --local <sbom-file> --cve <CVE-ID>
#     CVE query against a local SBOM.
#
# Dependencies: python3 (stdlib only), curl; grype (for --cve)
#
# Examples:
#   ./query.sh v1.0.9 --pkg cobra
#   ./query.sh v1.0.9 --cve CVE-2024-45337
#   ./query.sh --local sbom.cdx.json --pkg spf13

set -eu

REPO="nself-org/cli"

usage() {
    printf 'Usage:\n'
    printf '  %s <version> --pkg <name>\n' "$0"
    printf '  %s <version> --cve <CVE-ID>\n' "$0"
    printf '  %s --local <sbom-file> --pkg <name>\n' "$0"
    printf '  %s --local <sbom-file> --cve <CVE-ID>\n' "$0"
    exit 1
}

if [ $# -lt 3 ]; then
    usage
fi

SBOM_FILE=""
VERSION=""
FETCH=1

if [ "$1" = "--local" ]; then
    SBOM_FILE="$2"
    FETCH=0
    shift 2
else
    VERSION="$1"
    shift
fi

MODE=""
QUERY=""

while [ $# -gt 0 ]; do
    case "$1" in
        --pkg)
            MODE="pkg"
            QUERY="$2"
            shift 2
            ;;
        --cve)
            MODE="cve"
            QUERY="$2"
            shift 2
            ;;
        *)
            usage
            ;;
    esac
done

if [ -z "$MODE" ] || [ -z "$QUERY" ]; then
    usage
fi

TMPDIR_USED=0
if [ "$FETCH" = "1" ]; then
    WORK=$(mktemp -d)
    TMPDIR_USED=1
    SBOM_FILE="${WORK}/sbom.cdx.json"

    printf 'Fetching SBOM for %s...\n' "$VERSION"
    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"
    if ! curl -fsSL "${BASE_URL}/sbom.cdx.json" -o "$SBOM_FILE"; then
        printf 'ERROR: could not download sbom.cdx.json for %s\n' "$VERSION"
        rm -rf "$WORK"
        exit 1
    fi
    printf 'Downloaded: sbom.cdx.json (%d bytes)\n\n' "$(wc -c < "$SBOM_FILE")"
fi

if [ "$MODE" = "pkg" ]; then
    printf 'Searching for packages matching "%s" in %s...\n\n' "$QUERY" "$SBOM_FILE"
    python3 - "$SBOM_FILE" "$QUERY" <<'PYEOF'
import json, sys, re

sbom_path = sys.argv[1]
query = sys.argv[2].lower()

with open(sbom_path) as f:
    doc = json.load(f)

components = doc.get('components', [])
matches = []
for c in components:
    name = c.get('name', '')
    purl = c.get('purl', '')
    version = c.get('version', 'unknown')
    if query in name.lower() or query in purl.lower():
        matches.append((name, version, purl))

if not matches:
    print(f'No components matching "{query}" found.')
    sys.exit(0)

print(f'Found {len(matches)} matching component(s):')
print()
for name, version, purl in sorted(matches):
    print(f'  {name} @ {version}')
    if purl:
        print(f'    purl: {purl}')
PYEOF

elif [ "$MODE" = "cve" ]; then
    if ! command -v grype >/dev/null 2>&1; then
        printf 'ERROR: grype not found. Install: https://github.com/anchore/grype\n'
        if [ "$TMPDIR_USED" = "1" ]; then rm -rf "$WORK"; fi
        exit 1
    fi

    printf 'Scanning SBOM for %s...\n\n' "$QUERY"
    grype "sbom:${SBOM_FILE}" --output table 2>/dev/null | grep -i "$QUERY" || {
        printf 'No matches for %s in SBOM vulnerability scan.\n' "$QUERY"
        printf '(This does not mean the CVE is irrelevant — check grype directly for full output.)\n'
    }
fi

if [ "$TMPDIR_USED" = "1" ]; then
    rm -rf "$WORK"
fi
