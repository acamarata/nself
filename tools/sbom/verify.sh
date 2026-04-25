#!/bin/sh
# tools/sbom/verify.sh — verify a signed nSelf CLI SBOM (Q04)
#
# Usage:
#   ./verify.sh <version>
#     Downloads sbom.cdx.json + sbom.cdx.json.bundle for the given release tag
#     and runs cosign verify-blob against the Sigstore Rekor transparency log.
#
#   ./verify.sh <version> --local <sbom-file> <bundle-file>
#     Verifies a locally-downloaded SBOM without fetching from GitHub.
#
# Dependencies: cosign v2+, curl, gh (optional)
#
# Example:
#   ./verify.sh v1.0.9
#   ./verify.sh v1.0.9 --local ./sbom.cdx.json ./sbom.cdx.json.bundle

set -eu

REPO="nself-org/cli"
OIDC_ISSUER="https://token.actions.githubusercontent.com"
IDENTITY_REGEXP="^https://github.com/${REPO}/.github/workflows/release\\.yml@refs/tags/v"
REKOR_URL="https://rekor.sigstore.dev"

usage() {
    printf 'Usage:\n'
    printf '  %s <version>\n' "$0"
    printf '  %s <version> --local <sbom-file> <bundle-file>\n' "$0"
    printf '\nExamples:\n'
    printf '  %s v1.0.9\n' "$0"
    printf '  %s v1.0.9 --local sbom.cdx.json sbom.cdx.json.bundle\n' "$0"
    exit 1
}

if [ $# -lt 1 ]; then
    usage
fi

VERSION="$1"
shift

if [ $# -ge 3 ] && [ "$1" = "--local" ]; then
    SBOM_FILE="$2"
    BUNDLE_FILE="$3"
    FETCH=0
else
    FETCH=1
fi

if ! command -v cosign >/dev/null 2>&1; then
    printf 'ERROR: cosign not found.\n'
    printf 'Install: https://docs.sigstore.dev/cosign/installation/\n'
    exit 1
fi

TMPDIR_USED=0
if [ "$FETCH" = "1" ]; then
    WORK=$(mktemp -d)
    TMPDIR_USED=1
    SBOM_FILE="${WORK}/sbom.cdx.json"
    BUNDLE_FILE="${WORK}/sbom.cdx.json.bundle"

    printf 'Fetching SBOM assets for %s...\n' "$VERSION"
    BASE_URL="https://github.com/${REPO}/releases/download/${VERSION}"

    if ! curl -fsSL "${BASE_URL}/sbom.cdx.json" -o "$SBOM_FILE"; then
        printf 'ERROR: could not download sbom.cdx.json for %s\n' "$VERSION"
        rm -rf "$WORK"
        exit 1
    fi

    if ! curl -fsSL "${BASE_URL}/sbom.cdx.json.bundle" -o "$BUNDLE_FILE"; then
        printf 'ERROR: could not download sbom.cdx.json.bundle for %s\n' "$VERSION"
        rm -rf "$WORK"
        exit 1
    fi
    printf 'Downloaded: sbom.cdx.json (%d bytes), sbom.cdx.json.bundle (%d bytes)\n' \
        "$(wc -c < "$SBOM_FILE")" "$(wc -c < "$BUNDLE_FILE")"
fi

printf '\nVerifying SBOM signature via Sigstore Rekor...\n'
printf '  SBOM:    %s\n' "$SBOM_FILE"
printf '  Bundle:  %s\n' "$BUNDLE_FILE"
printf '  OIDC:    %s\n' "$OIDC_ISSUER"
printf '  Rekor:   %s\n' "$REKOR_URL"
printf '\n'

if cosign verify-blob "$SBOM_FILE" \
    --bundle "$BUNDLE_FILE" \
    --certificate-identity-regexp "$IDENTITY_REGEXP" \
    --certificate-oidc-issuer "$OIDC_ISSUER" \
    --rekor-url "$REKOR_URL"; then
    printf '\nVERIFIED OK — SBOM signature is valid and recorded in Rekor.\n'
else
    printf '\nVERIFICATION FAILED — SBOM may be tampered or was not signed by nSelf CI.\n'
    if [ "$TMPDIR_USED" = "1" ]; then
        rm -rf "$WORK"
    fi
    exit 1
fi

if [ "$TMPDIR_USED" = "1" ]; then
    rm -rf "$WORK"
fi
