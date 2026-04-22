#!/usr/bin/env bash
# verify-tarball-arch.sh — asserts that each tarball's embedded binary matches
# the architecture declared in the tarball filename.
#
# Authority: S49-T05, STORM-CROSS-PLATFORM
#
# Usage:
#   ./scripts/verify-tarball-arch.sh [dist-dir]
#   Default dist-dir: ./dist
#
# Naming convention expected:
#   <name>-<os>-<arch>.tar.gz     e.g. nself-linux-arm64.tar.gz
#   <name>-<os>-<arch>-<tag>.tar.gz  e.g. auth-linux-amd64-v1.1.1-noarch.tar.gz
#
# Exit codes:
#   0 — all tarballs pass
#   1 — one or more tarballs fail (mismatch or inspection error)
#
# Requirements:
#   - `file` command (available on macOS and Linux — no GNU-specific flags used)
#   - GNU tar or BSD tar (both support -t flag)
#
# Notes on `file` output across platforms:
#   Linux binary (amd64):   "ELF 64-bit LSB executable, x86-64, ..."
#   Linux binary (arm64):   "ELF 64-bit LSB executable, ARM aarch64, ..."
#   macOS binary (amd64):   "Mach-O 64-bit executable x86_64"
#   macOS binary (arm64):   "Mach-O 64-bit executable arm64"
#   Fat binary (universal): "Mach-O universal binary with 2 architectures: ..."

set -euo pipefail

DIST_DIR="${1:-./dist}"
FAIL=0
PASS=0
SKIP=0

if [ ! -d "$DIST_DIR" ]; then
  echo "ERROR: dist directory not found: $DIST_DIR"
  exit 1
fi

# Map tarball arch token → expected `file` output substring
# These patterns work on both macOS BSD file and Linux file.
arch_to_pattern() {
  case "$1" in
    amd64|x86_64) echo "x86-64\|x86_64" ;;
    arm64|aarch64) echo "aarch64\|arm64" ;;
    arm|armv7)     echo "ARM, EABI\|armv7" ;;
    *) echo "" ;;
  esac
}

verify_tarball() {
  local tb="$1"
  local name
  name="$(basename "$tb" .tar.gz)"

  # Extract os/arch from filename: <prefix>-<os>-<arch>[-<extra>]
  # Examples: nself-linux-arm64 / auth-linux-amd64-v1.1.1-noarch
  local os_token arch_token
  os_token=$(echo "$name" | grep -oE '-(linux|darwin|windows)-' | tr -d '-' | head -1 || echo "")
  arch_token=$(echo "$name" | grep -oE '-(amd64|arm64|arm|x86_64|aarch64)-?' | tr -d '-' | head -1 || echo "")

  if [ -z "$os_token" ] || [ -z "$arch_token" ]; then
    printf "[SKIP]  %s — cannot parse os/arch from filename\n" "$(basename "$tb")"
    SKIP=$((SKIP + 1))
    return
  fi

  # Noarch tarballs contain source only (no binary) — skip binary inspection
  if echo "$name" | grep -q "noarch"; then
    printf "[SKIP]  %s — noarch tarball; no binary to inspect\n" "$(basename "$tb")"
    SKIP=$((SKIP + 1))
    return
  fi

  local pattern
  pattern=$(arch_to_pattern "$arch_token")
  if [ -z "$pattern" ]; then
    printf "[SKIP]  %s — no pattern for arch '%s'\n" "$(basename "$tb")" "$arch_token"
    SKIP=$((SKIP + 1))
    return
  fi

  # Extract the first binary-like entry from the tarball for inspection
  local tmpdir
  tmpdir=$(mktemp -d)
  # shellcheck disable=SC2064
  trap "rm -rf '$tmpdir'" RETURN

  # List tarball contents, find the first non-directory non-.txt non-.json entry
  local binary_path
  binary_path=$(tar -tf "$tb" 2>/dev/null \
    | grep -v '/$' \
    | grep -vE '\.(txt|json|md|yml|yaml|sha256|sig|sbom)$' \
    | head -1 || echo "")

  if [ -z "$binary_path" ]; then
    printf "[FAIL]  %s — no binary found inside tarball\n" "$(basename "$tb")"
    FAIL=$((FAIL + 1))
    return
  fi

  # Extract that specific file
  if ! tar -xf "$tb" -C "$tmpdir" "$binary_path" 2>/dev/null; then
    printf "[FAIL]  %s — could not extract '%s'\n" "$(basename "$tb")" "$binary_path"
    FAIL=$((FAIL + 1))
    return
  fi

  local extracted_file="${tmpdir}/${binary_path}"
  if [ ! -f "$extracted_file" ]; then
    printf "[FAIL]  %s — extracted file not found at expected path\n" "$(basename "$tb")"
    FAIL=$((FAIL + 1))
    return
  fi

  local file_output
  file_output=$(file "$extracted_file" 2>/dev/null || echo "unknown")

  # Check pattern against file output
  if echo "$file_output" | grep -qE "$pattern"; then
    printf "[PASS]  %s — %s/%s confirmed (%s)\n" \
      "$(basename "$tb")" "$os_token" "$arch_token" \
      "$(echo "$file_output" | head -1 | cut -c1-60)"
    PASS=$((PASS + 1))
  else
    printf "[FAIL]  %s\n" "$(basename "$tb")"
    printf "        Expected arch pattern: %s\n" "$pattern"
    printf "        Got from file:         %s\n" "$(echo "$file_output" | head -1)"
    printf "        Tarball declares:      %s/%s\n" "$os_token" "$arch_token"
    FAIL=$((FAIL + 1))
  fi
}

echo "=== tarball arch verification ==="
echo "Dist dir: $DIST_DIR"
echo ""

mapfile -t TARBALLS < <(find "$DIST_DIR" -maxdepth 1 -name "*.tar.gz" | sort)

if [ "${#TARBALLS[@]}" -eq 0 ]; then
  echo "No .tar.gz files found in $DIST_DIR"
  exit 0
fi

for tb in "${TARBALLS[@]}"; do
  verify_tarball "$tb"
done

echo ""
echo "=== Summary ==="
echo "PASS: $PASS  SKIP: $SKIP  FAIL: $FAIL"

if [ "$FAIL" -gt 0 ]; then
  echo ""
  echo "VERIFICATION FAILED: $FAIL tarball(s) contain binary with wrong architecture."
  echo "Check your cross-compile GOOS/GOARCH settings and re-run 'make dist'."
  exit 1
fi

echo ""
echo "VERIFICATION PASSED: all tarballs contain the expected architecture."
exit 0
