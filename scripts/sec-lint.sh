#!/usr/bin/env bash
# scripts/sec-lint.sh — security lint for cli Go source
#
# Fails (exit 1) if any of the following forbidden patterns are found in
# non-test production Go source under cli/internal/:
#
#   1. "--admin-secret=" or "--admin-secret" as exec argv element in production
#      source (CWE-214 process table exposure).
#      Allowed: env injection via cmd.Env, bare -e VAR_NAME, comments, tests.
#
#   2. "--password=", "--token=", "--secret=" patterns constructing exec argv
#      with a literal credential concatenated in (passing secrets via argv).
#      Allowed: cmdlog redaction list, test fixtures, comments.
#
#   3. exec.Command / exec.CommandContext calls where one of the argv strings
#      contains "curl ... | sh" or "wget ... | sh" — supply-chain attack.
#      Does NOT flag: template strings, error message strings, comments.
#
# Intentionally excluded from Rules 1+2:
#   - *_test.go files (assertions checking a flag is absent are safe)
#   - internal/cmdlog/ (defines secretFlags list for redaction, not exec argv)
#   - Comment-only lines (// prefix)
#
# Run: bash scripts/sec-lint.sh [path]
# Default path: <repo-root>/internal

set -euo pipefail

ROOT="${1:-$(cd "$(dirname "$0")/.." && pwd)/internal}"
FAIL=0

if [ -t 1 ]; then
  RED='\033[0;31m'
  NC='\033[0m'
else
  RED=''
  NC=''
fi

fail() {
  printf "%b[SEC-LINT FAIL]%b %s\n" "$RED" "$NC" "$1" >&2
  FAIL=1
}

# Enumerate production (non-test) Go files, excluding cmdlog/.
prod_go_files() {
  find "$ROOT" -name "*.go" \
    ! -name "*_test.go" \
    ! -path "*/cmdlog/*" \
    -print
}

# ── Rule 1: --admin-secret in exec argv ────────────────────────────────────
# Flag: any non-test, non-cmdlog Go file containing the string "--admin-secret"
# followed by = or " (indicating a value concatenation or standalone flag).
# Comment lines (grep for // before the token) are excluded.

while IFS= read -r gofile; do
  hits=$(command grep -n '"--admin-secret[="]' "$gofile" \
    | command grep -v '^\s*//' \
    | command grep -v '//.*--admin-secret' \
    || true)
  if [ -n "$hits" ]; then
    while IFS= read -r hit; do
      fail "Rule 1 (CWE-214 --admin-secret argv): ${gofile}:${hit}"
    done <<EOF
$hits
EOF
  fi
done < <(prod_go_files)

# ── Rule 2: --password= / --token= / --secret= with a value in exec argv ──
# These indicate a credential is being passed directly as an argv element.
# Redaction-list style definitions (secretFlags = []string{"--password"...})
# are already excluded by the cmdlog/ path exclusion above.

while IFS= read -r gofile; do
  hits=$(command grep -n '"--password=\|"--token=\|"--secret=' "$gofile" \
    | command grep -v '^\s*//' \
    | command grep -v '//.*--\(password\|token\|secret\)' \
    | command grep -v 'REDACTED\|redact\|want.*REDACTED\|got.*REDACTED' \
    || true)
  if [ -n "$hits" ]; then
    while IFS= read -r hit; do
      fail "Rule 2 (credential-value argv): ${gofile}:${hit}"
    done <<EOF
$hits
EOF
  fi
done < <(prod_go_files)

# ── Rule 3: exec.Command*() with a curl/wget|sh argv ───────────────────────
# Looks for exec.Command or exec.CommandContext call sites where one of the
# string arguments contains "| sh" combined with curl or wget.
# Exempts: template string literals (backtick multiline, cloud-init yaml),
#          error message strings (fmt.Errorf / errors.New context lines),
#          comment lines.
#
# Strategy: grep for lines containing both an exec.Command invocation token
# AND a pipe-to-shell pattern on the same source line.

while IFS= read -r gofile; do
  hits=$(command grep -n 'exec\.Command' "$gofile" \
    | command grep -E '(curl|wget)[^"]*\|\s*(ba)?sh' \
    | command grep -v '^\s*//' \
    | command grep -v '//.*\(curl\|wget\)' \
    || true)
  if [ -n "$hits" ]; then
    while IFS= read -r hit; do
      fail "Rule 3 (pipe-to-shell in exec): ${gofile}:${hit}"
    done <<EOF
$hits
EOF
  fi
done < <(find "$ROOT" -name "*.go" -print)

# ── Rule 4: SQL string interpolation with data values in tenant/ (SEC-SQL-02) ─
# Flag fmt.Sprintf calls in internal/tenant/ that embed a SQL keyword alongside
# a string-value verb (%s, %q, %v) — the precise pattern of data-value injection.
# Allowed: fmt.Sprintf with only integer verbs (%d, %[N]d) building $N indices.
# Allowed (Path B docker-exec belt-and-suspenders): any fmt.Sprintf wrapping sanitize().
#
# Rule 4 catches both the old-style '%s' literal and the more general %s/%q/%v
# pattern so that regressions introduced in future patches are caught at lint time.

TENANT_DIR="$ROOT/tenant"
if [ -d "$TENANT_DIR" ]; then
  while IFS= read -r gofile; do
    # Match: fmt.Sprintf on a line that also contains a SQL keyword AND a string
    # verb (%s, %q, %v, %[N]s etc.) — but NOT solely integer-positional verbs.
    # Exclude lines that are comment-only or wrap sanitize() (Path B intent).
    hits=$(command grep -n "fmt\.Sprintf" "$gofile" \
      | command grep -E '(SELECT|INSERT|UPDATE|DELETE|WHERE|FROM|JOIN|SET|INTO|VALUES|RETURNING)' \
      | command grep -E '%(\[[0-9]+\])?[sqv]' \
      | command grep -v "sanitize(" \
      | awk -F: '{rest=$0; sub(/^[0-9]+:/, "", rest); if (rest !~ /^[[:space:]]*\/\//) print}' \
      || true)
    if [ -n "$hits" ]; then
      while IFS= read -r hit; do
        fail "Rule 4 (SEC-SQL-02 data-value interpolation in SQL — use \$N bound params): ${gofile}:${hit}"
      done <<EOF
$hits
EOF
    fi
  done < <(find "$TENANT_DIR" -name "*.go" ! -name "*_test.go" -print)
fi

# ── Rule 5: plain &http.Client{} in webhook package (SEC-SSRF-01) ─────────
# All outbound HTTP in internal/webhook/ must use NewWebhookClient() (SSRF-safe
# transport). A bare &http.Client{} bypasses the safeDialContext guard.

WEBHOOK_DIR="$ROOT/webhook"
if [ -d "$WEBHOOK_DIR" ]; then
  while IFS= read -r gofile; do
    # ssrf.go is the SSRF guard implementation — it is allowed to construct
    # &http.Client{} inside NewWebhookClient(). All other webhook files must
    # use NewWebhookClient() instead.
    if [[ "$gofile" == */ssrf.go ]]; then
      continue
    fi
    hits=$(command grep -n '&http\.Client{' "$gofile" \
      | awk -F: '{rest=$0; sub(/^[0-9]+:/, "", rest); if (rest !~ /^[[:space:]]*\/\//) print}' \
      || true)
    if [ -n "$hits" ]; then
      while IFS= read -r hit; do
        fail "Rule 5 (SEC-SSRF-01 plain http.Client in webhook/): ${gofile}:${hit}"
      done <<EOF
$hits
EOF
    fi
  done < <(find "$WEBHOOK_DIR" -name "*.go" ! -name "*_test.go" -print)
fi

# ── Result ──────────────────────────────────────────────────────────────────

if [ "$FAIL" -eq 0 ]; then
  printf "sec-lint: all checks passed (Go source under %s)\n" "$ROOT"
  exit 0
fi

printf "\nsec-lint: violations found — fix before merging\n" >&2
exit 1
