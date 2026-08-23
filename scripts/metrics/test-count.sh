#!/usr/bin/env bash
# Purpose: repeatable test/package/assert counts so "tests must not go down"
#          can be verified rather than asserted.
# Inputs:  none (runs the full suite with -count=1).
# Outputs: "tests=<n> packages=<n> failures=<n>" on stdout; exit code of go test.
# Constraints: needs the Go toolchain; no Docker (integration tests gate on INTEGRATION=1).
set -uo pipefail
TMP=$(mktemp)
trap 'rm -f "$TMP"' EXIT
CGO_ENABLED=0 go test -mod=vendor -count=1 -json ./... > "$TMP" 2>/dev/null
RC=$?
TESTS=$(grep -c '"Action":"pass","Package":"[^"]*","Test":' "$TMP" || true)
FAILS=$(grep -c '"Action":"fail","Package":"[^"]*","Test":' "$TMP" || true)
PKGS=$(grep -o '"Package":"[^"]*"' "$TMP" | sort -u | wc -l | tr -d ' ')
echo "tests=$TESTS packages=$PKGS failures=$FAILS"
exit $RC
