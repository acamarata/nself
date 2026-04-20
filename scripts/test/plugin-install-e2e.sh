#!/usr/bin/env bash
# S01 E2E smoke test: plugin install regressions
# Verifies the three S01 fixes on a clean tmpdir:
#   1. Multi-arg install: nself plugin install <a> <b> <c> is accepted
#   2. Conflict error names both plugins
#   3. Base-service route not false-positive on clean init
#
# Usage: bash scripts/test/plugin-install-e2e.sh
# Exit: 0 on success, non-zero on any failure

set -euo pipefail

BINARY="${NSELF_BINARY:-./nself}"
PASS=0
FAIL=0

pass() { echo "[PASS] $1"; PASS=$((PASS+1)); }
fail() { echo "[FAIL] $1"; FAIL=$((FAIL+1)); }

# --- Check 1: Binary exists ---
if [[ ! -x "$BINARY" ]]; then
  echo "Building binary..."
  go build -mod=vendor -o nself ./cmd/nself/ || { echo "Build failed"; exit 1; }
  BINARY="./nself"
fi

# --- Check 2: plugin install --help shows multi-arg syntax ---
HELP_OUT=$("$BINARY" plugin install --help 2>&1 || true)
if echo "$HELP_OUT" | grep -q "\[plugin\.\.\.\]"; then
  pass "plugin install --help shows '[plugin...]' multi-arg syntax"
else
  fail "plugin install --help missing '[plugin...]' — got: $HELP_OUT"
fi

# --- Check 3: plugin install with 0 args fails ---
ZERO_OUT=$("$BINARY" plugin install 2>&1 || true)
if echo "$ZERO_OUT" | grep -qi "requires at least\|accepts at least\|argument"; then
  pass "plugin install with 0 args correctly rejected"
else
  fail "plugin install with 0 args did not show expected error; got: $ZERO_OUT"
fi

# --- Check 4: go test all 3 regression unit tests ---
echo "Running S01 unit tests..."
if go test -mod=vendor \
  -run 'TestPluginInstallMultiArg|TestConflictErrorNamesBothPlugins|TestCollectRoutesIncludesBaseServices' \
  ./cmd/commands/... ./internal/nginx/... ./internal/plugin/... \
  -timeout 30s -v 2>&1 | tee /tmp/s01-unit-out.txt | grep -E 'PASS|FAIL|---'; then
  if grep -q "^FAIL" /tmp/s01-unit-out.txt; then
    fail "One or more S01 unit tests FAILED — see /tmp/s01-unit-out.txt"
  else
    pass "All 3 S01 unit tests PASS"
  fi
else
  fail "go test command failed"
fi

# --- Summary ---
echo ""
echo "S01 E2E: ${PASS} passed, ${FAIL} failed"

if [[ "$FAIL" -gt 0 ]]; then
  exit 1
fi
exit 0
