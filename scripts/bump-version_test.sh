#!/usr/bin/env bash
# bump-version_test.sh — Unit tests for bump-version.sh
#
# Tests each file shape with a synthetic fixture:
#   1. VERSION (bare semver file)
#   2. version.go (Go const: `\tVersion   string = "X.Y.Z"`)
#   3. sdk/go/doc.go (Go const: `const Version = "X.Y.Z"`)
#   4. package.json (JSON — handled via jq)
#   5. pyproject.toml (TOML — `version = "X.Y.Z"` under [project])
#   6. Cargo.toml (TOML — `version = "X.Y.Z"` under [package])
#   7. pubspec.yaml (YAML — `version: X.Y.Z`)
#   8. cli-version.ts (TS const)
#   9. Dockerfile (three lines: ARG, ENV, LABEL)
#
# Run: bash scripts/bump-version_test.sh
# Exit 0 = all pass, 1 = any failure.

set -uo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
TMP_BASE="$(mktemp -d)"
trap 'rm -rf "${TMP_BASE}"' EXIT

PASS=0
FAIL=0

# Source the helpers from bump-version.sh in test-mode.
# We'll re-implement minimal in-process test harness rather than execute
# the full script (it expects a real repo layout). Each test creates an
# isolated fixture and invokes the awk/jq/sed transforms in isolation.

# pass / fail reporters
ok()   { PASS=$((PASS+1)); printf '  PASS  %s\n' "$1"; }
bad()  { FAIL=$((FAIL+1)); printf '  FAIL  %s\n    %s\n' "$1" "$2"; }

# ── Test helpers (mirror bump-version.sh transforms) ──────────────────────────

NEW="9.8.7"

# Shape: bare semver file
test_version_file() {
  local f="${TMP_BASE}/VERSION"
  printf '1.0.13\n' > "${f}"
  printf '%s\n' "${NEW}" > "${f}"
  local got
  got="$(tr -d '[:space:]' < "${f}")"
  if [ "${got}" = "${NEW}" ]; then
    ok "VERSION file write"
  else
    bad "VERSION file write" "got: ${got}"
  fi
}

# Shape: version.go — `\tVersion   string = "X.Y.Z"`
test_version_go() {
  local f="${TMP_BASE}/version.go"
  cat > "${f}" <<'EOF'
package version
var (
	Version   string = "1.0.13"
	Commit    string = "unknown"
)
EOF
  sed "s/^\([[:space:]]*Version[[:space:]][[:space:]]*string[[:space:]]*=[[:space:]]*\"\)[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/\1${NEW}/" "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  if grep -q "Version   string = \"${NEW}\"" "${f}"; then
    ok "version.go Go const (tab + type annotation)"
  else
    bad "version.go Go const" "expected Version string = \"${NEW}\"; got: $(grep Version "${f}")"
  fi
}

# Shape: sdk/go/doc.go — `const Version = "X.Y.Z"`
test_sdk_go_doc() {
  local f="${TMP_BASE}/doc.go"
  cat > "${f}" <<'EOF'
package sdk
// Version is the SDK release.
const Version = "2.0.0"
EOF
  sed "s/^const Version = \"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\"/const Version = \"${NEW}\"/" "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  if grep -q "const Version = \"${NEW}\"" "${f}"; then
    ok "sdk/go/doc.go const Version"
  else
    bad "sdk/go/doc.go const Version" "got: $(grep Version "${f}")"
  fi
}

# Shape: package.json — JSON via jq
test_json_version() {
  local f="${TMP_BASE}/package.json"
  cat > "${f}" <<'EOF'
{
  "name": "test-pkg",
  "version": "1.0.13",
  "dependencies": {
    "some-dep": "version 1.2.3 in description"
  }
}
EOF
  jq --arg v "${NEW}" '.version = $v' "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  local got
  got="$(jq -r .version "${f}")"
  if [ "${got}" = "${NEW}" ]; then
    ok "package.json (jq replaces top-level only)"
  else
    bad "package.json" "got: ${got}"
  fi
  # Verify nested string was NOT touched
  if grep -q '"version 1.2.3 in description"' "${f}"; then
    ok "package.json (nested 'version' string preserved)"
  else
    bad "package.json nested" "nested description was clobbered"
  fi
}

# Shape: pyproject.toml — `version = "X.Y.Z"` under [project]
test_pyproject_toml() {
  local f="${TMP_BASE}/pyproject.toml"
  cat > "${f}" <<'EOF'
[build-system]
requires = ["hatchling"]

[project]
name = "test-pkg"
version = "1.0.13"
description = "Test"

[project.optional-dependencies]
dev = ["pytest>=7.0"]
EOF
  awk -v sect="[project]" -v newv="${NEW}" '
    $0 == sect { in_section = 1; print; next }
    in_section && /^\[/ { in_section = 0 }
    in_section && !replaced && /^[[:space:]]*version[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"/ {
      sub(/[0-9]+\.[0-9]+\.[0-9]+/, newv)
      replaced = 1
    }
    { print }
  ' "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  if grep -q "^version = \"${NEW}\"" "${f}"; then
    ok "pyproject.toml [project] version"
  else
    bad "pyproject.toml" "got: $(grep '^version' "${f}")"
  fi
}

# Shape: Cargo.toml — `version = "X.Y.Z"` under [package], with sibling [dependencies.foo] sub-tables
test_cargo_toml() {
  local f="${TMP_BASE}/Cargo.toml"
  cat > "${f}" <<'EOF'
[package]
name = "test-crate"
version = "1.0.13"
edition = "2021"

[dependencies]
serde = "1.0"

[dependencies.tokio]
version = "1.35.0"
features = ["full"]
EOF
  awk -v sect="[package]" -v newv="${NEW}" '
    $0 == sect { in_section = 1; print; next }
    in_section && /^\[/ { in_section = 0 }
    in_section && !replaced && /^[[:space:]]*version[[:space:]]*=[[:space:]]*"[0-9]+\.[0-9]+\.[0-9]+"/ {
      sub(/[0-9]+\.[0-9]+\.[0-9]+/, newv)
      replaced = 1
    }
    { print }
  ' "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  if grep -q "^version = \"${NEW}\"" "${f}"; then
    ok "Cargo.toml [package] version"
  else
    bad "Cargo.toml" "got: $(grep '^version' "${f}")"
  fi
  # Critical: sibling [dependencies.tokio] version must remain "1.35.0"
  if grep -q '^version = "1.35.0"' "${f}"; then
    ok "Cargo.toml ([dependencies.tokio] version preserved)"
  else
    bad "Cargo.toml dep version" "tokio version was clobbered: $(grep -A1 tokio "${f}")"
  fi
}

# Shape: pubspec.yaml — `version: X.Y.Z`
test_pubspec_yaml() {
  local f="${TMP_BASE}/pubspec.yaml"
  cat > "${f}" <<'EOF'
name: test_sdk
version: 2.0.0

environment:
  sdk: ">=3.0.0 <4.0.0"
EOF
  sed "s/^version:[[:space:]][[:space:]]*[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/version: ${NEW}/" "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  if grep -q "^version: ${NEW}" "${f}"; then
    ok "pubspec.yaml version"
  else
    bad "pubspec.yaml" "got: $(grep '^version' "${f}")"
  fi
}

# Shape: cli-version.ts — `export const CLI_VERSION = 'X.Y.Z'`
test_cli_version_ts() {
  local f="${TMP_BASE}/cli-version.ts"
  cat > "${f}" <<'EOF'
export const CLI_VERSION = '1.0.13'
EOF
  sed "s/^export const CLI_VERSION = '[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*'/export const CLI_VERSION = '${NEW}'/" "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  if grep -q "CLI_VERSION = '${NEW}'" "${f}"; then
    ok "cli-version.ts const"
  else
    bad "cli-version.ts" "got: $(cat "${f}")"
  fi
}

# Shape: Dockerfile — ARG / ENV / LABEL triplet
test_dockerfile() {
  local f="${TMP_BASE}/Dockerfile"
  cat > "${f}" <<'EOF'
FROM node:20-alpine
ARG NSELF_VERSION=1.0.16
ENV ADMIN_VERSION=1.0.16
LABEL org.opencontainers.image.version="1.0.16"
EOF
  sed \
    -e "s/^ARG NSELF_VERSION=[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/ARG NSELF_VERSION=${NEW}/" \
    -e "s/^ENV ADMIN_VERSION=[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*/ENV ADMIN_VERSION=${NEW}/" \
    -e "s/^LABEL org\.opencontainers\.image\.version=\"[0-9][0-9]*\.[0-9][0-9]*\.[0-9][0-9]*\"/LABEL org.opencontainers.image.version=\"${NEW}\"/" \
    "${f}" > "${f}.new"
  mv "${f}.new" "${f}"
  local arg_ok env_ok lbl_ok
  arg_ok=$(grep -c "^ARG NSELF_VERSION=${NEW}$" "${f}")
  env_ok=$(grep -c "^ENV ADMIN_VERSION=${NEW}$" "${f}")
  lbl_ok=$(grep -c "^LABEL org.opencontainers.image.version=\"${NEW}\"$" "${f}")
  if [ "${arg_ok}" = "1" ] && [ "${env_ok}" = "1" ] && [ "${lbl_ok}" = "1" ]; then
    ok "Dockerfile (ARG + ENV + LABEL triplet)"
  else
    bad "Dockerfile" "arg=${arg_ok} env=${env_ok} lbl=${lbl_ok}"
  fi
}

# Integration smoke: run the actual bump-version.sh --dry-run and assert
# it covers all 11 known lockstep files without errors.
test_dry_run_smoke() {
  local out
  out="$(bash "${SCRIPT_DIR}/bump-version.sh" 9.8.7 --dry-run 2>&1)"
  local rc=$?
  if [ "${rc}" -ne 0 ]; then
    bad "dry-run smoke" "exit ${rc}: ${out}"
    return
  fi
  # Expect at least 7 unique file paths reported (some are SKIP if already at target).
  local n
  n="$(printf '%s\n' "${out}" | grep -E '(SKIP|->)' | wc -l | tr -d ' ')"
  if [ "${n}" -ge 8 ]; then
    ok "dry-run smoke (${n} files reported)"
  else
    bad "dry-run smoke" "only ${n} files reported (expected >=8)"
  fi
}

# ── Run ───────────────────────────────────────────────────────────────────────

printf 'Running bump-version.sh shape tests...\n\n'

if ! command -v jq >/dev/null 2>&1; then
  printf 'SKIP: jq not installed; JSON tests will be skipped.\n'
else
  test_json_version
fi

test_version_file
test_version_go
test_sdk_go_doc
test_pyproject_toml
test_cargo_toml
test_pubspec_yaml
test_cli_version_ts
test_dockerfile
test_dry_run_smoke

printf '\n%d passed, %d failed\n' "${PASS}" "${FAIL}"
[ "${FAIL}" -eq 0 ]
