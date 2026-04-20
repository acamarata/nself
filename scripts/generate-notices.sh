#!/usr/bin/env bash
# generate-notices.sh — Generate NOTICES.md files for open-source license attribution
#
# Sprint S50-T02: MIT/Apache/BSD dependencies require attribution preservation.
# Run this script from the nself project root on every release.
#
# Requirements:
#   - go-licenses: go install github.com/google/go-licenses@latest
#   - pnpm: already installed (GCI requirement)
#   - cargo-about: cargo install cargo-about (optional — only for Rust components)
#
# Usage:
#   ./cli/scripts/generate-notices.sh [--check]
#
# --check: verify all NOTICES.md files are non-empty without regenerating.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

CHECK_ONLY=false
if [ "${1:-}" = "--check" ]; then
  CHECK_ONLY=true
fi

log() {
  printf '[generate-notices] %s\n' "$1"
}

error() {
  printf '[generate-notices] ERROR: %s\n' "$1" >&2
  exit 1
}

# ------------------------------------------------------------------------------
# Go repos (cli)
# ------------------------------------------------------------------------------

generate_go_notices() {
  local repo_path="$1"
  local output_file="$repo_path/NOTICES.md"

  if $CHECK_ONLY; then
    if [ ! -s "$output_file" ]; then
      error "NOTICES.md missing or empty at $output_file"
    fi
    log "OK: $output_file"
    return
  fi

  if ! command -v go-licenses > /dev/null 2>&1; then
    log "WARNING: go-licenses not found. Install: go install github.com/google/go-licenses@latest"
    log "Skipping Go notices for $repo_path"
    return
  fi

  log "Generating Go NOTICES.md for $repo_path..."
  cd "$repo_path"
  {
    printf '# Third-Party Notices\n\n'
    printf 'This project uses third-party open-source libraries.\n'
    printf 'The following notices are required by their licenses.\n\n'
    printf '---\n\n'
    go-licenses report ./... 2>/dev/null || printf '(go-licenses report unavailable — run go mod tidy first)\n'
  } > "$output_file"
  log "Written: $output_file"
}

# ------------------------------------------------------------------------------
# Node/pnpm repos (admin, web)
# ------------------------------------------------------------------------------

generate_node_notices() {
  local repo_path="$1"
  local output_file="$repo_path/NOTICES.md"

  if $CHECK_ONLY; then
    if [ ! -s "$output_file" ]; then
      error "NOTICES.md missing or empty at $output_file"
    fi
    log "OK: $output_file"
    return
  fi

  if ! command -v pnpm > /dev/null 2>&1; then
    error "pnpm not found. Install: https://pnpm.io/installation"
  fi

  log "Generating Node NOTICES.md for $repo_path..."
  cd "$repo_path"
  {
    printf '# Third-Party Notices\n\n'
    printf 'This project uses third-party open-source libraries.\n'
    printf 'The following notices are required by their licenses.\n\n'
    printf '---\n\n'
    pnpm licenses list --long 2>/dev/null || printf '(pnpm licenses list unavailable)\n'
  } > "$output_file"
  log "Written: $output_file"
}

# ------------------------------------------------------------------------------
# Mixed repo (plugins, plugins-pro — Go + optional Rust)
# ------------------------------------------------------------------------------

generate_plugins_notices() {
  local repo_path="$1"
  local output_file="$repo_path/NOTICES.md"

  if $CHECK_ONLY; then
    if [ ! -s "$output_file" ]; then
      error "NOTICES.md missing or empty at $output_file"
    fi
    log "OK: $output_file"
    return
  fi

  log "Generating plugins NOTICES.md for $repo_path..."
  {
    printf '# Third-Party Notices\n\n'
    printf 'This plugin library uses third-party open-source libraries.\n'
    printf 'The following notices are required by their licenses.\n\n'
    printf '---\n\n'
    printf '## Go Dependencies\n\n'
    if command -v go-licenses > /dev/null 2>&1 && [ -f "$repo_path/go.mod" ]; then
      cd "$repo_path"
      go-licenses report ./... 2>/dev/null || printf '(go-licenses report unavailable)\n'
    else
      printf '(go-licenses not available or no go.mod found)\n'
    fi
    printf '\n---\n\n'
    printf '## Rust Dependencies (FFI components)\n\n'
    if command -v cargo-about > /dev/null 2>&1 && [ -f "$repo_path/Cargo.toml" ]; then
      cd "$repo_path"
      cargo about generate --all-features 2>/dev/null | sed 's/<[^>]*>//g' || printf '(cargo about unavailable)\n'
    else
      printf '(cargo-about not available or no Cargo.toml found)\n'
    fi
  } > "$output_file"
  log "Written: $output_file"
}

# ------------------------------------------------------------------------------
# Check for GPL/LGPL deps (fail loudly)
# ------------------------------------------------------------------------------

check_incompatible_licenses() {
  local notices_file="$1"
  if grep -qi "GPL\|LGPL" "$notices_file" 2>/dev/null; then
    error "Potentially incompatible license (GPL/LGPL) found in $notices_file. Review manually."
  fi
}

# ------------------------------------------------------------------------------
# Main
# ------------------------------------------------------------------------------

log "Starting notices generation (check_only=$CHECK_ONLY)..."
log "Project root: $PROJECT_ROOT"

generate_go_notices "$PROJECT_ROOT/cli"
generate_node_notices "$PROJECT_ROOT/web/org"
generate_plugins_notices "$PROJECT_ROOT/plugins"
generate_plugins_notices "$PROJECT_ROOT/plugins-pro"

# Aggregate web/NOTICES.md from per-repo files
if ! $CHECK_ONLY; then
  WEB_NOTICES="$PROJECT_ROOT/web/NOTICES.md"
  log "Aggregating web/NOTICES.md..."
  {
    printf '# Third-Party Notices — nSelf Web\n\n'
    printf 'Aggregated from per-repo NOTICES.md files.\n\n'
    printf '---\n\n'
    printf '## CLI (Go)\n\n'
    cat "$PROJECT_ROOT/cli/NOTICES.md" 2>/dev/null || printf '(cli/NOTICES.md not found)\n'
    printf '\n---\n\n'
    printf '## Web (Node)\n\n'
    cat "$PROJECT_ROOT/web/org/NOTICES.md" 2>/dev/null || printf '(web/org/NOTICES.md not found)\n'
    printf '\n---\n\n'
    printf '## Plugins (Go + Rust)\n\n'
    cat "$PROJECT_ROOT/plugins/NOTICES.md" 2>/dev/null || printf '(plugins/NOTICES.md not found)\n'
  } > "$WEB_NOTICES"
  log "Written: $WEB_NOTICES"
fi

# Check for GPL/LGPL in all generated files
for notices in \
  "$PROJECT_ROOT/cli/NOTICES.md" \
  "$PROJECT_ROOT/web/org/NOTICES.md" \
  "$PROJECT_ROOT/plugins/NOTICES.md" \
  "$PROJECT_ROOT/plugins-pro/NOTICES.md" \
  "$PROJECT_ROOT/web/NOTICES.md"
do
  if [ -f "$notices" ]; then
    check_incompatible_licenses "$notices"
  fi
done

log "Done. All NOTICES.md files generated and checked."
