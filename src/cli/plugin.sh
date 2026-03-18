#!/usr/bin/env bash
# plugin.sh - Plugin management for nself v0.4.8
# Install, manage, and use nself plugins

set -o pipefail

set -euo pipefail


# ============================================================================
# INITIALIZATION
# ============================================================================

CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/platform-compat.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/plugin/core.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/plugin/registry.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/plugin/dependencies.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/plugin/runtime.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/plugin/licensing.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/plugin/schema-isolation.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/cli-output.sh" 2>/dev/null || true

# Fallbacks if display.sh didn't load
if ! declare -f log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_warning >/dev/null 2>&1; then
  log_warning() { printf "\033[0;33m[WARNING]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi

# ============================================================================
# CONSTANTS
# ============================================================================

PLUGIN_DIR="${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}"
PLUGIN_CACHE_DIR="${NSELF_PLUGIN_CACHE:-$HOME/.nself/cache/plugins}"
PLUGIN_REGISTRY_URL="${NSELF_PLUGIN_REGISTRY:-https://plugins.nself.org}"
PLUGIN_REGISTRY_FALLBACK="https://raw.githubusercontent.com/nself-org/plugins/main/registry.json"
PLUGIN_REPO_URL="https://github.com/nself-org/plugins"
NSELF_API_DOWNLOAD_URL="${NSELF_PING_API_URL:-${NSELF_PING_URL:-https://ping.nself.org}}/plugins"

# Known Rust/Docker pro plugins distributed as pre-built GHCR images.
# These are pulled from ghcr.io/nself-org/nself-<name>:<version> at install time.
# plugin_is_rust() checks plugin.json first; this list is a fallback for pre-install detection.
NSELF_RUST_PLUGINS="ai mux claw voice browser notify cron google"
NSELF_GHCR_BASE="ghcr.io/nself-org"

# ============================================================================
# REGISTRY FORMAT HELPERS
# ============================================================================

# _registry_plugin_names <registry_json>
# Extracts plugin names from either format:
#   - Object format (v0.9.7):  { "plugins": { "name": { ... }, ... } }
#   - Array format  (v0.9.8+): { "plugins": [ { "name": "...", ... }, ... ] }
# Prints one name per line.
_registry_plugin_names() {
  local reg="$1"
  if command -v jq >/dev/null 2>&1; then
    # jq .plugins[] iterates values for both objects and arrays
    printf '%s' "$reg" | jq -r '.plugins | if type == "array" then .[].name else keys[] end' 2>/dev/null
    return
  fi
  # grep fallback: try object format first ("name":{ ), then array format ("name":"value")
  local names
  names=$(printf '%s' "$reg" | grep -o '"[a-z][a-z0-9-]*"[[:space:]]*:{' | sed 's/"//g;s/[[:space:]]*:{//')
  if [[ -n "$names" ]]; then
    printf '%s\n' "$names"
    return
  fi
  # Array format: extract "name" field values, deduplicate
  printf '%s' "$reg" | grep -oE '"name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/' | awk '!seen[$0]++'
}

# _registry_plugin_field <registry_json> <plugin_name> <field>
# Extracts a specific field value for a named plugin from either registry format.
# Prints the field value (empty string if not found).
_registry_plugin_field() {
  local reg="$1"
  local pname="$2"
  local field="$3"
  if command -v jq >/dev/null 2>&1; then
    printf '%s' "$reg" | jq -r \
      "(.plugins | if type == \"array\" then .[] else .[] end | select(.name==\"$pname\") | .$field) // empty" 2>/dev/null
    return
  fi
  # grep fallback: python3 preferred for reliable JSON field extraction
  if command -v python3 >/dev/null 2>&1; then
    printf '%s' "$reg" | python3 -c "
import json,sys
try:
    d=json.load(sys.stdin)
    p=d.get('plugins',{})
    if isinstance(p,list):
        for e in p:
            if e.get('name')==sys.argv[1]:
                v=e.get(sys.argv[2],'')
                if v: print(v)
                break
    elif isinstance(p,dict):
        e=p.get(sys.argv[1],{})
        v=e.get(sys.argv[2],'')
        if v: print(v)
except: pass
" "$pname" "$field" 2>/dev/null
    return
  fi
  # Last resort: grep on multi-line JSON (may be inaccurate for single-line array format)
  printf '%s' "$reg" | grep -A10 "\"$pname\"" | grep "\"$field\"" | head -1 | sed 's/.*"'"$field"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/'
}

# ============================================================================
# PLUGIN MANAGEMENT
# ============================================================================

# List available plugins
cmd_list() {
  local show_installed_only=false
  local filter_category=""
  local show_detailed=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --help | -h)
        printf "Usage: nself plugin list [options]\n\n"
        printf "List available plugins.\n\n"
        printf "Options:\n"
        printf "  --installed, -i         Show only installed plugins\n"
        printf "  --detailed, -d          Show detailed status (with --installed)\n"
        printf "  --category, -c <cat>    Filter by category (billing, ecommerce, devops)\n"
        printf "  --help, -h              Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin list\n"
        printf "  nself plugin list --installed\n"
        printf "  nself plugin list --installed --detailed\n"
        return 0
        ;;
      --installed | -i)
        show_installed_only=true
        shift
        ;;
      --detailed | -d)
        show_detailed=true
        shift
        ;;
      --category | -c)
        filter_category="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done

  # If showing installed with detailed flag, use new list_all_plugins
  if [[ "$show_installed_only" == "true" ]] && [[ "$show_detailed" == "true" ]]; then
    list_all_plugins
    return 0
  fi

  # Fetch free registry
  local registry
  registry=$(fetch_registry)

  if [[ -z "$registry" ]]; then
    log_error "Failed to fetch plugin registry"
    return 1
  fi

  local count=0

  # ── Installed-only view ────────────────────────────────────────────────────
  if [[ "$show_installed_only" == "true" ]]; then
    printf "\n=== Installed Plugins ===\n\n"
    printf "%-20s %-10s %-12s %-6s %-30s\n" "NAME" "VERSION" "CATEGORY" "TIER" "DESCRIPTION"
    printf "%-20s %-10s %-12s %-6s %-30s\n" "----" "-------" "--------" "----" "-----------"

    for plugin_dir in "$PLUGIN_DIR"/*/; do
      [[ -d "$plugin_dir" ]] || continue
      local plugin
      plugin=$(basename "$plugin_dir")
      [[ "$plugin" == "_shared" ]] && continue
      [[ -f "$plugin_dir/plugin.json" ]] || continue

      local version description category
      if command -v jq >/dev/null 2>&1; then
        version=$(jq -r '.version // "1.0.0"' "$plugin_dir/plugin.json" 2>/dev/null)
        description=$(jq -r '.description // ""' "$plugin_dir/plugin.json" 2>/dev/null)
        category=$(jq -r '.category // "general"' "$plugin_dir/plugin.json" 2>/dev/null)
      else
        version=$(grep '"version"' "$plugin_dir/plugin.json" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
        description=$(grep '"description"' "$plugin_dir/plugin.json" | head -1 | sed 's/.*"description"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
        category=$(grep '"category"' "$plugin_dir/plugin.json" | head -1 | sed 's/.*"category"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
      fi
      version="${version:-1.0.0}"
      category="${category:-general}"

      if [[ -n "$filter_category" ]] && [[ "$category" != "$filter_category" ]]; then
        continue
      fi

      local tier="FREE"
      if declare -f license_is_paid_plugin >/dev/null 2>&1 && license_is_paid_plugin "$plugin"; then
        tier="PRO"
      fi

      printf "%-20s %-10s %-12s %-6s %-30s\n" "$plugin" "$version" "$category" "$tier" "${description:0:30}"
      count=$((count + 1))
    done

    if [[ $count -eq 0 ]]; then
      log_info "No plugins installed"
    fi
    printf "\nDetailed status: nself plugin list --installed --detailed\n"
    return 0
  fi

  # ── Available view (free + pro) ────────────────────────────────────────────
  local free_count=0
  local pro_count=0
  if declare -f license_is_paid_plugin >/dev/null 2>&1; then
    pro_count=$(printf '%s' "$NSELF_PRO_PLUGINS" | wc -w | tr -d ' ')
  fi
  local free_plugins
  free_plugins=$(_registry_plugin_names "$registry")
  for _p in $free_plugins; do free_count=$((free_count + 1)); done

  printf "\n=== Available Plugins (%d free, %d pro) ===\n\n" "$free_count" "$pro_count"
  printf "%-20s %-10s %-12s %-6s %-30s\n" "NAME" "VERSION" "CATEGORY" "TIER" "DESCRIPTION"
  printf "%-20s %-10s %-12s %-6s %-30s\n" "----" "-------" "--------" "----" "-----------"

  # Free plugins from registry
  for plugin in $free_plugins; do
    local version description category
    version=$(_registry_plugin_field "$registry" "$plugin" "version")
    description=$(_registry_plugin_field "$registry" "$plugin" "description")
    category=$(_registry_plugin_field "$registry" "$plugin" "category")

    if [[ -n "$filter_category" ]] && [[ "$category" != "$filter_category" ]]; then
      continue
    fi

    local suffix=""
    if is_plugin_installed "$plugin"; then suffix=" *"; fi

    printf "%-20s %-10s %-12s %-6s %-30s%s\n" "$plugin" "${version:-1.0.0}" "${category:-general}" "FREE" "${description:0:30}" "$suffix"
    count=$((count + 1))
  done

  # Pro plugins from hardcoded list (private registry — names only)
  if declare -f license_is_paid_plugin >/dev/null 2>&1; then
    local has_license=false
    if license_get_key >/dev/null 2>&1; then has_license=true; fi

    printf "\n  --- Pro Plugins (license required — %s) ---\n\n" "${NSELF_PRICING_URL:-https://nself.org/pricing}"

    for plugin in $NSELF_PRO_PLUGINS; do
      if [[ -n "$filter_category" ]]; then
        # Without full metadata, skip category filter for pro plugins
        continue
      fi

      local suffix=""
      if is_plugin_installed "$plugin"; then suffix=" *"; fi

      if [[ "$has_license" == "true" ]]; then
        printf "%-20s %-10s %-12s %-6s %-30s%s\n" "$plugin" "1.0.0" "-" "PRO" "Pro Plugin" "$suffix"
      else
        printf "%-20s %-10s %-12s %-6s %-30s%s\n" "$plugin" "1.0.0" "-" "PRO" "License required" "$suffix"
      fi
      count=$((count + 1))
    done
  fi

  if [[ $count -eq 0 ]] && [[ -n "$filter_category" ]]; then
    log_info "No free plugins found in category: $filter_category"
  fi

  printf "\n  * = installed\n"
  printf "\nInstall free:  nself plugin install <name>\n"
  printf "Install pro:   nself plugin license set <key> && nself plugin install <name>\n"
  printf "Get a license: %s\n" "${NSELF_PRICING_URL:-https://nself.org/pricing}"
}

# Return space-separated direct dependencies for a plugin (hardcoded for offline/Bash 3.2).
# Also checks installed plugin.json when available for dynamically-installed plugins.
_plugin_deps() {
  local name="$1"
  local nself_home="${NSELF_HOME:-$HOME/.nself}"
  # Check installed plugin manifest first
  local manifest="${PLUGIN_DIR:-$nself_home/plugins}/$name/plugin.json"
  # Also check plugin-registry (used by test fixtures)
  local registry_manifest="$nself_home/plugin-registry/${name}.json"
  local found_manifest=""
  [[ -f "$manifest" ]] && found_manifest="$manifest"
  [[ -z "$found_manifest" && -f "$registry_manifest" ]] && found_manifest="$registry_manifest"
  if [[ -n "$found_manifest" ]]; then
    # Extract "dependencies": [...] from plugin.json (Bash 3.2 compatible)
    # Handles both inline arrays and multi-line arrays
    local in_deps=false result="" dep inner
    while IFS= read -r line; do
      case "$line" in
        *'"dependencies"'*)
          in_deps=true
          case "$line" in
            *"]"*)
              # Inline array: "dependencies": ["a", "b"]
              inner="${line#*\[}"
              inner="${inner%%\]*}"
              while [[ "$inner" == *'"'* ]]; do
                dep="${inner#*\"}"
                dep="${dep%%\"*}"
                [[ -n "$dep" ]] && result="$result $dep"
                inner="${inner#*\"}"
                inner="${inner#*\"}"
              done
              in_deps=false
              ;;
          esac
          continue
          ;;
        *"]"*)
          [[ "$in_deps" == "true" ]] && { in_deps=false; break; }
          ;;
        *'"'*)
          if [[ "$in_deps" == "true" ]]; then
            dep="${line#*\"}"
            dep="${dep%%\"*}"
            [[ -n "$dep" ]] && result="$result $dep"
          fi
          ;;
      esac
    done < "$found_manifest"
    printf '%s' "$result"
    return
  fi
  # Fallback: static map for known pro plugins (covers CI / dry-run with no installed plugins)
  case "$name" in
    claw)    printf "ai mux notify cron voice browser" ;;
    mux)     printf "ai google" ;;
    voice)   printf "ai" ;;
    browser) printf "ai" ;;
    ai|google|notify|cron|livekit|stripe|cms|moderation|bots|chat|realtime|entitlements|recording|*)
             printf "" ;;
  esac
}

# Topological sort: given space-separated plugin names, outputs install order (deps first).
# Outputs one line per plugin as "nself-<name>".  Detects cycles.
_plugin_topo_sort() {
  local requested="$*"
  local visited=""   # space-separated names already output
  local in_stack=""  # cycle detection
  local nself_home="${NSELF_HOME:-$HOME/.nself}"

  _visit() {
    local p="$1"
    # Skip if already output in this run
    case " $visited " in *" $p "*) return 0 ;; esac
    # Cycle detection
    case " $in_stack " in
      *" $p "*)
        printf "Error: circular dependency detected involving '%s'\n" "$p" >&2
        return 1
        ;;
    esac
    in_stack="$in_stack $p"
    local deps dep
    deps=$(_plugin_deps "$p")
    for dep in $deps; do
      _visit "$dep" || return 1
    done
    in_stack="${in_stack% $p}"
    in_stack="${in_stack#$p }"
    # Remove p from anywhere in in_stack (handles edge cases)
    in_stack=$(printf '%s' "$in_stack" | sed "s/ $p / /g;s/^$p //;s/ $p\$//;s/^$p\$//")
    visited="$visited $p"
    # Check if plugin is already installed
    if [[ -f "$nself_home/plugins/installed/nself-$p" ]]; then
      printf 'nself-%s (already installed)\n' "$p"
    else
      printf 'nself-%s\n' "$p"
    fi
  }

  local name
  for name in $requested; do
    _visit "$name" || return 1
  done
}

# Install a plugin
cmd_install() {
  local plugin_name=""
  local dry_run=false
  local order_only=false
  local extra_names=""
  local pin_version=""
  local install_channel="stable"
  local no_docker=false
  local _skip_next=false

  # Parse flags before any other logic (Bash 3.2-compatible)
  for arg in "$@"; do
    if [[ "$_skip_next" == "true" ]]; then
      _skip_next=false
      continue
    fi
    case "$arg" in
      --help|-h)
        printf "Usage: nself plugin install <name> [options]\n\n"
        printf "Install a plugin by name.\n\n"
        printf "Options:\n"
        printf "  --dry-run               Show what would be installed without making changes\n"
        printf "  --order-only            With --dry-run: print dependency install order only\n"
        printf "  --version <ver>         Pin a specific image version (Rust/GHCR plugins only)\n"
        printf "  --channel <channel>     Release channel: stable (default), canary, beta\n"
        printf "  --no-docker             Install static musl binary instead of Docker image\n"
        printf "                          (Linux only; requires systemd; no Docker needed)\n"
        printf "  --help, -h              Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin install notify\n"
        printf "  nself plugin install ai --dry-run\n"
        printf "  nself plugin install ai --version 0.1.0\n"
        printf "  nself plugin install ai --channel canary\n"
        printf "  nself plugin install ai --no-docker\n"
        printf "  nself plugin install --dry-run --order-only claw\n"
        return 0
        ;;
      --dry-run)
        dry_run=true
        ;;
      --order-only)
        order_only=true
        ;;
      --version)
        # Next positional arg is the version value — handled in the positional branch below
        # by setting a flag; we consume it in the next iteration.
        # Because Bash 3.2 for-in loops can't peek ahead, we use a sentinel approach:
        # store the fact that the next non-flag arg is a version, not a plugin name.
        # We handle this via the _skip_next trick with a separate pass.
        ;;
      --channel)
        # Value consumed in second pass below
        ;;
      --no-docker)
        no_docker=true
        ;;
      -*)
        ;;
      *)
        if [[ -z "$plugin_name" ]]; then
          plugin_name="$arg"
        else
          extra_names="$extra_names $arg"
        fi
        ;;
    esac
  done

  # Second pass: extract --version <value> and --channel <value>
  # (Bash 3.2-compatible: no arrays, use set)
  local _prev=""
  for arg in "$@"; do
    if [[ "$_prev" == "--version" ]]; then
      pin_version="$arg"
    elif [[ "$_prev" == "--channel" ]]; then
      install_channel="$arg"
    fi
    _prev="$arg"
  done
  unset _prev _skip_next

  # Validate channel value
  case "$install_channel" in
    stable|canary|beta) ;;
    *)
      log_error "Invalid channel: $install_channel. Valid channels: stable, canary, beta"
      return 1
      ;;
  esac

  if [[ -z "$plugin_name" ]]; then
    log_error "Plugin name required"
    printf "\nUsage: nself plugin install <name>\n"
    return 1
  fi

  if [[ "$dry_run" == "true" ]]; then
    # Always run topo sort in dry-run to validate deps and detect cycles.
    # Use || to prevent set -e from exiting on non-zero before we can check status.
    local topo_output="" topo_status=0
    topo_output=$(_plugin_topo_sort "$plugin_name" $extra_names 2>&1) || topo_status=$?
    if [[ $topo_status -ne 0 ]]; then
      printf '%s\n' "$topo_output" >&2
      return $topo_status
    fi
    if [[ "$order_only" == "true" ]]; then
      printf '%s\n' "$topo_output"
      return 0
    fi
    printf "DRY RUN: would install plugin '%s'\n" "$plugin_name"
    return 0
  fi

  # Check if it's a local path
  if [[ -d "$plugin_name" ]]; then
    install_local_plugin "$plugin_name"
    return $?
  fi

  # Parse version if specified (plugin@version) — overrides --version flag
  local version=""
  if [[ "$plugin_name" == *"@"* ]]; then
    version="${plugin_name#*@}"
    plugin_name="${plugin_name%@*}"
  elif [[ -n "$pin_version" ]]; then
    version="$pin_version"
  fi

  # Normalize plugin name: strip 'nself-' or 'plugin-' prefix if present.
  # Allows: nself plugin install plugin-mux → mux; nself plugin install nself-ai → ai
  case "$plugin_name" in
    plugin-*) plugin_name="${plugin_name#plugin-}" ;;
    nself-*)  plugin_name="${plugin_name#nself-}" ;;
  esac

  log_info "Installing plugin: $plugin_name"

  # Check if already installed
  if is_plugin_installed "$plugin_name"; then
    log_warning "Plugin '$plugin_name' is already installed"
    printf "Use 'nself plugin update %s' to update\n" "$plugin_name"
    return 0
  fi

  # Check plugin prerequisites (hard dependencies)
  if ! _check_plugin_prerequisites "$plugin_name"; then
    return 1
  fi

  # Check for port conflicts before downloading
  _check_port_conflicts "$plugin_name" || return 1

  # Fetch registry
  local registry
  registry=$(fetch_registry "$plugin_name")

  # Determine if this is a pro plugin — if so, skip the free registry check
  local is_pro=false
  if declare -f license_is_paid_plugin >/dev/null 2>&1 && license_is_paid_plugin "$plugin_name"; then
    is_pro=true
  fi

  # For free plugins, verify existence in registry
  if [[ "$is_pro" == "false" ]]; then
    if [[ -z "$registry" ]]; then
      log_error "Registry format error — cannot verify plugin '$plugin_name'."
      printf "Try: nself update && nself plugin install %s\n" "$plugin_name"
      printf "For offline installs: https://nself.org/docs/plugins/offline\n"
      return 1
    fi
    if ! printf '%s' "$registry" | grep -qE "\"$plugin_name\"[[:space:]]*:|\"name\"[[:space:]]*:[[:space:]]*\"$plugin_name\""; then
      log_error "Plugin '$plugin_name' not found in registry"
      printf "\nRun 'nself plugin list' to see all available plugins.\n"
      return 1
    fi
  fi

  # Check license entitlement before downloading
  if declare -f license_check_entitlement >/dev/null 2>&1; then
    if ! license_check_entitlement "$plugin_name"; then
      return 1
    fi
  fi

  # Check tier entitlement — shows Max-tier upgrade prompt when needed
  if declare -f license_check_tier_entitlement >/dev/null 2>&1; then
    if ! license_check_tier_entitlement "$plugin_name"; then
      return 1
    fi
  fi

  # Rust/Docker plugins: pull pre-built GHCR image instead of (or in addition to) downloading source.
  # The tarball download still runs for plugin.json + config assets.
  # The GHCR pull writes .ghcr-image so plugin-services.sh generates image: instead of build:.
  local is_rust=false
  if _plugin_is_rust_known "$plugin_name"; then
    is_rust=true
  fi

  # Download and install (tarball for plugin.json + config assets)
  download_plugin "$plugin_name" "$version"

  # For Rust/Docker plugins that are now confirmed installed (plugin.json exists):
  # pull the GHCR image and record it for compose generation, OR install the
  # static musl binary when --no-docker was requested.
  if plugin_is_rust "$plugin_name" 2>/dev/null || [[ "$is_rust" == "true" ]]; then
    if [[ "$no_docker" == "true" ]]; then
      _install_plugin_binary "$plugin_name" "$version" || true
    else
      _pull_plugin_docker_image "$plugin_name" "$version" "$install_channel" || true
    fi
  fi

  # Record the release channel — written after image pull so it always reflects the
  # channel actually used (even if the pull failed and fell back to local build).
  mkdir -p "$PLUGIN_DIR/$plugin_name"
  printf '%s\n' "$install_channel" > "$PLUGIN_DIR/$plugin_name/.channel"

  # Create plugin PG schema + role (schema isolation — idempotent, non-blocking)
  if declare -f create_plugin_schema >/dev/null 2>&1; then
    create_plugin_schema "$plugin_name" || true
  fi

  # Run install script
  run_plugin_installer "$plugin_name"

  # Install plugin-to-plugin dependencies declared in plugin.json
  if declare -f plugin_install_dependencies >/dev/null 2>&1; then
    plugin_install_dependencies "$plugin_name" || true
  fi

  # Sync TypeScript source to project services directory (no-op for Rust plugins)
  export NSELF_PROJECT_DIR="$(pwd)"
  sync_plugin_source "$plugin_name"

  # Symlink plugin-provided CLI binaries (e.g. nclaw) into ~/.nself/bin/
  # core.sh always defines plugin_symlink_bins; guard via || true for safety (Bash 3.2 compatible)
  plugin_symlink_bins "$plugin_name" 2>/dev/null || true

  log_success "Plugin '$plugin_name' installed successfully!"

  # T-0256: Run first-run wizard for the AI plugin if not yet configured
  if [ "$plugin_name" = "ai" ]; then
    _run_ai_setup_wizard
  fi

  # Record in project plugin list if running inside a project directory
  if [[ -f ".env" ]]; then
    mkdir -p ".nself"
    local project_list=".nself/plugins"
    if ! grep -qx "$plugin_name" "$project_list" 2>/dev/null; then
      printf '%s\n' "$plugin_name" >> "$project_list"
      log_info "Registered in project plugin list: $project_list"
    fi
  fi

  # Check for system dependencies
  if declare -f check_plugin_dependencies >/dev/null 2>&1; then
    printf "\n"
    if ! check_plugin_dependencies "$plugin_name" 2>/dev/null; then
      printf "Note: This plugin has system dependencies\n"
      printf "Run: ${CLI_CYAN}nself plugin check-deps %s${CLI_RESET}\n" "$plugin_name"
    fi
  fi

  # For GHCR-installed Rust plugins: rebuild compose and bring the service up.
  if [[ -f "$PLUGIN_DIR/$plugin_name/.ghcr-image" ]]; then
    _plugin_activate_docker_service "$plugin_name"
  fi

  if [[ -f "$PLUGIN_DIR/$plugin_name/.ghcr-image" ]]; then
    printf "\nPlugin running as Docker service nself-%s. Check status: docker ps\n" "$plugin_name"
  else
    printf "\nConfigure in .env and run: nself plugin %s sync\n" "$plugin_name"
  fi
}

# Remove a plugin
cmd_remove() {
  local plugin_name="" delete_data="" dry_run=false

  # Parse arguments
  for _arg in "$@"; do
    case "$_arg" in
      --help|-h)
        printf "Usage: nself plugin remove <name> [options]\n\n"
        printf "Remove an installed plugin.\n\n"
        printf "Options:\n"
        printf "  --dry-run      Show what would be removed without making changes\n"
        printf "  --delete-data  Also delete plugin database tables\n"
        printf "  --help, -h     Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin remove notify\n"
        printf "  nself plugin remove stripe --delete-data\n"
        return 0
        ;;
      --dry-run)  dry_run=true ;;
      --delete-data) delete_data="--delete-data" ;;
      -*) ;;
      *) [[ -z "$plugin_name" ]] && plugin_name="$_arg" ;;
    esac
  done
  unset _arg

  if [[ -z "$plugin_name" ]]; then
    log_error "Plugin name required"
    return 1
  fi

  # Dry-run mode: print intent and exit 0 if plugin is installed OR is a known plugin.
  # Reject completely unknown plugin names even in dry-run.
  if [[ "$dry_run" == "true" ]]; then
    if is_plugin_installed "$plugin_name"; then
      printf "DRY RUN: would remove plugin '%s'\n" "$plugin_name"
      return 0
    fi
    # Check if it's a known free plugin (hardcoded offline list)
    case "$plugin_name" in
      content-acquisition|content-progress|feature-flags|github|github-runner|\
      invitations|jobs|link-preview|mdns|notifications|search|subtitle-manager|\
      tokens|torrent-manager|vpn|webhooks)
        printf "DRY RUN: would remove plugin '%s' (not currently installed)\n" "$plugin_name"
        return 0
        ;;
    esac
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  if ! is_plugin_installed "$plugin_name"; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  log_info "Removing plugin: $plugin_name"

  local plugin_dir="$PLUGIN_DIR/$plugin_name"

  # Run uninstall script
  if [[ -f "$plugin_dir/uninstall.sh" ]]; then
    local uninstall_args=""
    [[ "$delete_data" != "--delete-data" ]] && uninstall_args="--keep-data"
    bash "$plugin_dir/uninstall.sh" $uninstall_args
  fi

  # Remove plugin directory
  rm -rf "$plugin_dir"

  log_success "Plugin '$plugin_name' removed"
}

# Update a plugin
cmd_update() {
  local plugin_name=""
  local update_all=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --help | -h)
        printf "Usage: nself plugin update [name] [options]\n\n"
        printf "Update a plugin to the latest version.\n\n"
        printf "Options:\n"
        printf "  --all, -a    Update all installed plugins\n"
        printf "  --help, -h   Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin update notify\n"
        printf "  nself plugin update --all\n"
        return 0
        ;;
      --all | -a)
        update_all=true
        shift
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        plugin_name="$1"
        shift
        ;;
    esac
  done

  # Default to all if no plugin specified
  if [[ -z "$plugin_name" ]] || [[ "$update_all" == "true" ]]; then
    log_info "Updating all plugins..."
    local found=0
    for plugin_dir in "$PLUGIN_DIR"/*/; do
      if [[ -f "$plugin_dir/plugin.json" ]]; then
        local name
        name=$(basename "$plugin_dir")
        update_single_plugin "$name"
        found=$((found + 1))
      fi
    done
    if [[ $found -eq 0 ]]; then
      log_info "No plugins installed"
    fi
  else
    update_single_plugin "$plugin_name"
  fi
}

update_single_plugin() {
  local plugin_name="$1"

  if ! is_plugin_installed "$plugin_name"; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  log_info "Updating: $plugin_name"

  # Get current version
  local current_version
  current_version=$(get_installed_version "$plugin_name")

  # Fetch registry for latest version
  local registry
  registry=$(fetch_registry)

  local latest_version
  latest_version=$(printf '%s' "$registry" | grep -A5 "\"$plugin_name\"" | grep '"version"' | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

  if [[ "$current_version" == "$latest_version" ]]; then
    log_info "$plugin_name is already at latest version ($current_version)"
    return 0
  fi

  log_info "Updating $plugin_name: $current_version -> $latest_version"

  # Download new version
  download_plugin "$plugin_name" "$latest_version"

  # Sync TypeScript source to project services directory (no-op for Rust plugins)
  export NSELF_PROJECT_DIR="$(pwd)"
  sync_plugin_source "$plugin_name"

  log_success "Updated $plugin_name to $latest_version"
}

# Show plugin status
cmd_status() {
  case "${1:-}" in
    --help|-h)
      printf "Usage: nself plugin status [name]\n\n"
      printf "Show plugin installation status and health.\n\n"
      printf "Arguments:\n"
      printf "  name    Optional plugin name for detailed status\n\n"
      printf "Options:\n"
      printf "  --help, -h  Show this help text\n\n"
      printf "Examples:\n"
      printf "  nself plugin status\n"
      printf "  nself plugin status notify\n"
      return 0
      ;;
  esac

  local plugin_name="${1:-}"

  printf "\n=== Installed Plugins ===\n\n"

  if [[ -n "$plugin_name" ]]; then
    show_plugin_status "$plugin_name"
  else
    local found=0
    for plugin_dir in "$PLUGIN_DIR"/*/; do
      if [[ -f "$plugin_dir/plugin.json" ]]; then
        local name
        name=$(basename "$plugin_dir")
        show_plugin_status "$name"
        found=$((found + 1))
      fi
    done

    if [[ $found -eq 0 ]]; then
      log_info "No plugins installed"
      printf "\nInstall with: nself plugin install <name>\n"
    else
      # Check for updates
      echo ""
      if declare -f registry_check_updates_formatted >/dev/null 2>&1; then
        registry_check_updates_formatted
      fi
    fi
  fi
}

show_plugin_status() {
  local plugin_name="$1"
  local plugin_dir="$PLUGIN_DIR/$plugin_name"

  if [[ ! -d "$plugin_dir" ]]; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  local manifest="$plugin_dir/plugin.json"
  if [[ ! -f "$manifest" ]]; then
    log_error "Invalid plugin: missing plugin.json"
    return 1
  fi

  local version description
  version=$(grep '"version"' "$manifest" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
  description=$(grep '"description"' "$manifest" | head -1 | sed 's/.*"description"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

  printf "Plugin: %s\n" "$plugin_name"
  printf "  Version: %s\n" "$version"
  printf "  Description: %s\n" "$description"
  printf "  Path: %s\n" "$plugin_dir"

  # Check required env vars
  local required_vars
  required_vars=$(grep -A10 '"required"' "$manifest" | grep -o '"[A-Z_]*"' | tr -d '"' || true)

  if [[ -n "$required_vars" ]]; then
    printf "  Environment:\n"
    for var in $required_vars; do
      if [[ -n "${!var:-}" ]]; then
        printf "    %s: configured\n" "$var"
      else
        printf "    %s: NOT SET\n" "$var"
      fi
    done
  fi

  # Check system dependencies (if dependencies.sh is loaded)
  if declare -f check_plugin_dependencies >/dev/null 2>&1; then
    printf "  Dependencies:\n"

    # Parse and show quick dependency status
    local required_deps=$(parse_system_dependencies "$manifest" "required" 2>/dev/null || echo "")

    if [[ -n "$required_deps" ]]; then
      local dep_count=0
      local dep_ok=0

      while IFS= read -r line; do
        if [[ "$line" =~ \"name\" ]]; then
          local name=$(echo "$line" | sed 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
          dep_count=$((dep_count + 1))

          # Get verify command from next few lines
          local verify_cmd=$(echo "$required_deps" | grep -A5 "\"name\"[[:space:]]*:[[:space:]]*\"$name\"" | grep '"verify"' | head -1 | sed 's/.*"verify"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

          if verify_dependency "$name" "$verify_cmd" 2>/dev/null; then
            dep_ok=$((dep_ok + 1))
          fi
        fi
      done <<< "$required_deps"

      if [[ $dep_ok -eq $dep_count ]]; then
        printf "    ✓ All %d dependencies satisfied\n" "$dep_count"
      else
        printf "    ⚠ %d/%d dependencies installed\n" "$dep_ok" "$dep_count"
        printf "    Run: nself plugin check-deps %s\n" "$plugin_name"
      fi
    else
      printf "    (none required)\n"
    fi
  fi

  printf "\n"
}

# ============================================================================
# PLUGIN ACTION DISPATCH
# ============================================================================

# Run plugin action
cmd_run_action() {
  local plugin_name="$1"
  local action="$2"
  shift 2 || true

  if [[ -z "$plugin_name" ]]; then
    show_help
    return 1
  fi

  if ! is_plugin_installed "$plugin_name"; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  local plugin_dir="$PLUGIN_DIR/$plugin_name"

  if [[ -z "$action" ]]; then
    show_plugin_help "$plugin_name"
    return 0
  fi

  # Export plugin context
  export NSELF_PLUGIN_PATH="$plugin_dir"
  export NSELF_PROJECT_DIR="$(pwd)"

  # 1. Check for shell script action (original behavior)
  local action_script="$plugin_dir/actions/${action}.sh"
  if [[ -f "$action_script" ]]; then
    bash "$action_script" "$@"
    return $?
  fi

  # 2. Check for built-in actions
  case "$action" in
    init)
      run_builtin_init "$plugin_name" "$@"
      return $?
      ;;
    integrate)
      run_builtin_integrate "$plugin_name" "$@"
      return $?
      ;;
  esac

  # 3. Check if action is defined in plugin manifest
  local manifest="$plugin_dir/plugin.json"
  if [[ -f "$manifest" ]]; then
    local action_exists=false
    local action_desc=""

    if command -v jq >/dev/null 2>&1; then
      if jq -e ".actions[\"$action\"]" "$manifest" >/dev/null 2>&1; then
        action_exists=true
        action_desc=$(jq -r ".actions[\"$action\"].description // \"\"" "$manifest" 2>/dev/null)
      fi
    else
      if grep -q "\"$action\"" "$manifest"; then
        action_exists=true
        action_desc=$(grep -A2 "\"$action\"" "$manifest" | grep '"description"' | head -1 | sed 's/.*"description"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || true)
      fi
    fi

    if [[ "$action_exists" == "true" ]]; then
      log_info "Action '$action' is defined in plugin manifest"
      [[ -n "$action_desc" ]] && printf "  Description: %s\n" "$action_desc"
      printf "\nThis action requires the plugin to be running as a service.\n"
      printf "To set up the plugin service:\n\n"
      printf "  nself plugin %s integrate    # Show CS_N configuration\n" "$plugin_name"
      printf "  # Add the configuration to your .env file\n"
      printf "  nself build && nself restart  # Start the service\n\n"
      printf "Once running, plugin actions are available via its API.\n"
      return 0
    fi
  fi

  # 4. Action not found
  log_error "Unknown action: $action"
  show_plugin_help "$plugin_name"
  return 1
}

# ============================================================================
# BUILT-IN PLUGIN ACTIONS
# ============================================================================

# Built-in init action: apply database schema
run_builtin_init() {
  local plugin_name="$1"
  shift || true
  local plugin_dir="$PLUGIN_DIR/$plugin_name"
  local manifest="$plugin_dir/plugin.json"

  log_info "Initializing plugin: $plugin_name"

  # Load project environment
  if declare -f plugin_load_env >/dev/null 2>&1; then
    plugin_load_env "$plugin_name"
  fi

  # Look for schema SQL files
  local schema_applied=false

  # Check for specific schema files (in priority order)
  for sql_file in "$plugin_dir/schema/tables.sql" "$plugin_dir/schema/schema.sql" "$plugin_dir/schema/init.sql"; do
    if [[ -f "$sql_file" ]]; then
      log_info "Applying schema: $(basename "$sql_file")"
      if declare -f plugin_db_exec >/dev/null 2>&1 && plugin_db_exec "$(cat "$sql_file")"; then
        log_success "Schema applied successfully"
        schema_applied=true
      else
        log_error "Failed to apply schema. Is the database running?"
        printf "\nEnsure services are running: nself start\n"
        return 1
      fi
      break
    fi
  done

  # Apply all SQL files in schema directory
  if [[ "$schema_applied" == "false" ]] && [[ -d "$plugin_dir/schema" ]]; then
    for sql_file in "$plugin_dir"/schema/*.sql; do
      if [[ -f "$sql_file" ]]; then
        log_info "Applying: $(basename "$sql_file")"
        if declare -f plugin_db_exec >/dev/null 2>&1 && plugin_db_exec "$(cat "$sql_file")"; then
          schema_applied=true
        else
          log_error "Failed to apply: $(basename "$sql_file")"
        fi
      fi
    done
    if [[ "$schema_applied" == "true" ]]; then
      log_success "Schema applied successfully"
    fi
  fi

  if [[ "$schema_applied" == "true" ]]; then
    log_success "Plugin '$plugin_name' initialized"
    return 0
  fi

  # No schema files found - show table info from manifest
  log_info "No SQL schema files found in plugin directory"

  local tables=""
  if command -v jq >/dev/null 2>&1; then
    tables=$(jq -r '.tables[]? // empty' "$manifest" 2>/dev/null)
  else
    tables=$(grep -o '"np_[a-z_]*"' "$manifest" | tr -d '"' || true)
  fi

  if [[ -n "$tables" ]]; then
    printf "\nPlugin expects these database tables:\n"
    printf '%s\n' "$tables" | while read -r table; do
      [[ -n "$table" ]] && printf "  - %s\n" "$table"
    done
    printf "\nThe plugin will create these tables when started as a service.\n"
  fi

  printf "\nTo run the plugin as a service:\n"
  printf "  nself plugin %s integrate    # Show configuration\n" "$plugin_name"
  printf "  # Add configuration to .env\n"
  printf "  nself build && nself restart  # Start services\n"

  return 0
}

# Built-in integrate action: generate CS_N configuration
run_builtin_integrate() {
  local plugin_name="$1"
  shift || true
  local plugin_dir="$PLUGIN_DIR/$plugin_name"
  local manifest="$plugin_dir/plugin.json"

  # Get plugin config
  local port="3000"
  local description=""

  if command -v jq >/dev/null 2>&1; then
    port=$(jq -r '.port // .config.port // 3000' "$manifest" 2>/dev/null)
    description=$(jq -r '.description // ""' "$manifest" 2>/dev/null)
  else
    port=$(grep '"port"' "$manifest" | head -1 | sed 's/[^0-9]//g')
    description=$(grep '"description"' "$manifest" | head -1 | sed 's/.*"description"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
  fi

  port="${port:-3000}"

  # Load project env to find next CS_N slot
  if declare -f plugin_load_env >/dev/null 2>&1; then
    plugin_load_env "$plugin_name"
  fi

  local next_cs=1
  for i in $(seq 1 10); do
    local cs_var="CS_${i}"
    if [[ -z "${!cs_var:-}" ]]; then
      next_cs=$i
      break
    fi
  done

  # Determine plugin type based on source
  local plugin_type="custom"
  if [[ -f "$plugin_dir/ts/package.json" ]] || [[ -f "$plugin_dir/package.json" ]]; then
    plugin_type="express-js"
  elif [[ -f "$plugin_dir/requirements.txt" ]] || [[ -f "$plugin_dir/setup.py" ]]; then
    plugin_type="fastapi"
  elif [[ -f "$plugin_dir/go.mod" ]]; then
    plugin_type="gin"
  fi

  local upper_name
  upper_name=$(printf '%s' "$plugin_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')

  printf "\n=== Plugin Integration: %s ===\n\n" "$plugin_name"
  [[ -n "$description" ]] && printf "%s\n\n" "$description"

  printf "Add the following to your .env file:\n\n"

  printf "  # %s Plugin\n" "$plugin_name"
  printf "  CS_%d=%s:%s:%s\n" "$next_cs" "$plugin_name" "$plugin_type" "$port"
  printf "  %s_PLUGIN_ENABLED=true\n" "$upper_name"
  printf "  %s_PLUGIN_PORT=%s\n" "$upper_name" "$port"

  # Show required env vars
  if command -v jq >/dev/null 2>&1; then
    local required_vars
    required_vars=$(jq -r '.envVars.required[]? // empty' "$manifest" 2>/dev/null)
    if [[ -n "$required_vars" ]]; then
      printf "\n  # Required environment variables\n"
      printf '%s\n' "$required_vars" | while read -r var; do
        [[ -n "$var" ]] && printf "  %s=\n" "$var"
      done
    fi

    local optional_vars
    optional_vars=$(jq -r '.envVars.optional[]? // empty' "$manifest" 2>/dev/null)
    if [[ -n "$optional_vars" ]]; then
      printf "\n  # Optional environment variables\n"
      printf '%s\n' "$optional_vars" | while read -r var; do
        [[ -n "$var" ]] && printf "  # %s=\n" "$var"
      done
    fi
  fi

  printf "\nThen run:\n"
  printf "  nself build      # Generate docker-compose config\n"
  printf "  nself restart     # Start/restart services\n\n"
  printf "The plugin will be available at:\n"
  printf "  https://%s.{BASE_DOMAIN}\n\n" "$plugin_name"

  # Show tables
  local tables=""
  if command -v jq >/dev/null 2>&1; then
    tables=$(jq -r '.tables[]? // empty' "$manifest" 2>/dev/null)
  fi
  if [[ -n "$tables" ]]; then
    printf "Database tables (auto-created on startup):\n"
    printf '%s\n' "$tables" | while read -r table; do
      [[ -n "$table" ]] && printf "  - %s\n" "$table"
    done
    printf "\n"
  fi
}

# ============================================================================
# PLUGIN HELP
# ============================================================================

show_plugin_help() {
  local plugin_name="$1"
  local plugin_dir="$PLUGIN_DIR/$plugin_name"
  local manifest="$plugin_dir/plugin.json"

  printf "\nUsage: nself plugin %s <action> [args]\n\n" "$plugin_name"

  # Show built-in actions
  printf "Built-in Actions:\n"
  printf "  %-20s %s\n" "init" "Initialize database schema"
  printf "  %-20s %s\n" "integrate" "Show CS_N service configuration"

  # List shell script actions
  local has_scripts=false
  if [[ -d "$plugin_dir/actions" ]]; then
    for action in "$plugin_dir"/actions/*.sh; do
      if [[ -f "$action" ]]; then
        if [[ "$has_scripts" == "false" ]]; then
          printf "\nScript Actions:\n"
          has_scripts=true
        fi
        local action_name
        action_name=$(basename "$action" .sh)
        local action_desc=""
        if [[ -f "$manifest" ]]; then
          if command -v jq >/dev/null 2>&1; then
            action_desc=$(jq -r ".actions[\"$action_name\"].description // \"\"" "$manifest" 2>/dev/null)
          else
            action_desc=$(grep -A2 "\"$action_name\"" "$manifest" | grep '"description"' | head -1 | sed 's/.*"description"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || true)
          fi
        fi
        printf "  %-20s %s\n" "$action_name" "$action_desc"
      fi
    done
  fi

  # List manifest-defined actions (service actions)
  if [[ -f "$manifest" ]]; then
    local manifest_actions=""
    if command -v jq >/dev/null 2>&1; then
      manifest_actions=$(jq -r '.actions // {} | keys[]' "$manifest" 2>/dev/null)
    fi

    if [[ -n "$manifest_actions" ]]; then
      local has_service_actions=false
      while IFS= read -r action_name; do
        [[ -z "$action_name" ]] && continue
        # Skip if shell script exists or is a built-in
        [[ -f "$plugin_dir/actions/${action_name}.sh" ]] && continue
        [[ "$action_name" == "init" || "$action_name" == "integrate" ]] && continue

        if [[ "$has_service_actions" == "false" ]]; then
          printf "\nService Actions (requires running service):\n"
          has_service_actions=true
        fi

        local desc=""
        desc=$(jq -r ".actions[\"$action_name\"].description // \"\"" "$manifest" 2>/dev/null)
        printf "  %-20s %s\n" "$action_name" "$desc"
      done <<< "$manifest_actions"
    fi
  fi

  printf "\n"
}

# ============================================================================
# HELPER FUNCTIONS
# ============================================================================

fetch_registry() {
  # Check cache first
  local cache_file="$PLUGIN_CACHE_DIR/registry.json"
  local cache_age=3600 # 1 hour

  mkdir -p "$PLUGIN_CACHE_DIR"

  if [[ -f "$cache_file" ]]; then
    local file_time current_time
    current_time=$(date +%s)

    if [[ "$OSTYPE" == "darwin"* ]]; then
      file_time=$(stat -f %m "$cache_file" 2>/dev/null || echo 0)
    else
      file_time=$(stat -c %Y "$cache_file" 2>/dev/null || echo 0)
    fi

    if ((current_time - file_time < cache_age)); then
      cat "$cache_file"
      return 0
    fi
  fi

  # Fetch fresh registry — try primary URL then GitHub fallback
  local registry
  local ua="nself-cli/${NSELF_VERSION:-0.9.9}"
  registry=$(curl -sf --connect-timeout 10 --max-time 15 \
    -H "User-Agent: $ua" \
    "$PLUGIN_REGISTRY_URL/registry.json" 2>/dev/null)

  # Validate: must contain JSON structure (not a CF challenge HTML page)
  if [[ -z "$registry" ]] || ! printf '%s' "$registry" | grep -q '"name"\|"plugins"\|"version"'; then
    # Primary failed or returned non-JSON — try GitHub fallback
    registry=$(curl -sf --connect-timeout 10 --max-time 15 \
      -H "User-Agent: $ua" \
      "$PLUGIN_REGISTRY_FALLBACK" 2>/dev/null)
  fi

  if [[ -n "$registry" ]] && printf '%s' "$registry" | grep -q '"name"\|"plugins"\|"version"'; then
    printf '%s' "$registry" >"$cache_file"
    printf '%s' "$registry"
  elif [[ -f "$cache_file" ]]; then
    # Return stale cache if all fetches failed
    cat "$cache_file"
  else
    # No cache and both fetches failed or returned unexpected format — print diagnostic to stderr
    printf '\033[0;31m[ERROR]\033[0m Registry format error or network unavailable.\n' >&2
    printf 'Try: nself update && nself plugin install %s\n' "${1:-<name>}" >&2
    printf 'For offline installs, see: https://nself.org/docs/plugins/offline\n' >&2
  fi
}

is_plugin_installed() {
  local plugin_name="$1"
  [[ -f "$PLUGIN_DIR/$plugin_name/plugin.json" ]]
}

get_installed_version() {
  local plugin_name="$1"
  local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"

  if [[ -f "$manifest" ]]; then
    grep '"version"' "$manifest" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/'
  fi
}

# ---------------------------------------------------------------------------
# _plugin_is_rust_known <plugin_name>
# Fast offline check: returns 0 if the plugin is in the known Rust plugin list.
# This is used before plugin.json exists (pre-install detection).
# After install, plugin_is_rust() checks plugin.json directly — always prefer that.
# ---------------------------------------------------------------------------
_plugin_is_rust_known() {
  local name="$1"
  local entry
  for entry in $NSELF_RUST_PLUGINS; do
    if [[ "$entry" == "$name" ]]; then
      return 0
    fi
  done
  return 1
}

# ---------------------------------------------------------------------------
# _pull_plugin_docker_image <plugin_name> [version] [channel]
# Pulls the pre-built GHCR image for a Rust/Docker plugin and writes the image
# reference to ~/.nself/plugins/<name>/.ghcr-image.
#
# Image reference format: ghcr.io/nself-org/nself-<name>:<tag>
# Tag resolution order:
#   1. channel=canary  → always use :canary tag (ignores version)
#   2. channel=beta    → always use :beta tag (ignores version)
#   3. Explicit version argument (from --version flag or plugin@version)
#   4. PLUGIN_<UPPER_NAME>_VERSION env var
#   5. version field in plugin.json (if already downloaded)
#   6. "latest"
#
# Requires Docker. Authenticates to GHCR if GHCR_TOKEN or GITHUB_TOKEN is set.
# Returns 0 on success, 1 on failure (non-fatal — install continues without GHCR).
# ---------------------------------------------------------------------------
_pull_plugin_docker_image() {
  local plugin_name="$1"
  local explicit_version="${2:-}"
  local channel="${3:-stable}"

  if ! command -v docker >/dev/null 2>&1; then
    log_warning "docker not found — skipping GHCR image pull for $plugin_name"
    return 0
  fi

  # Resolve image tag — canary/beta channels always override the version
  local img_version=""
  case "$channel" in
    canary|beta)
      img_version="$channel"
      ;;
    *)
      # stable channel: use explicit version, env var, plugin.json, or "latest"
      img_version="$explicit_version"

      # Check PLUGIN_<NAME>_VERSION env var (e.g. PLUGIN_AI_VERSION)
      if [[ -z "$img_version" ]]; then
        local env_var_name
        env_var_name="PLUGIN_$(printf '%s' "$plugin_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')_VERSION"
        # Dereference the env var name portably (Bash 3.2 compatible)
        local env_var_val
        env_var_val=$(eval "printf '%s' \"\${${env_var_name}:-}\"" 2>/dev/null || true)
        [[ -n "$env_var_val" ]] && img_version="$env_var_val"
      fi

      # Check plugin.json version field (plugin already downloaded by download_plugin)
      if [[ -z "$img_version" ]]; then
        local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
        if [[ -f "$manifest" ]]; then
          img_version=$(grep '"version"' "$manifest" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || true)
        fi
      fi

      # Default to latest
      [[ -z "$img_version" ]] && img_version="latest"
      ;;
  esac

  local image_ref="${NSELF_GHCR_BASE}/nself-${plugin_name}:${img_version}"

  # Authenticate to GHCR if a token is available (needed for private images)
  local ghcr_token="${GHCR_TOKEN:-${GITHUB_TOKEN:-}}"
  if [[ -n "$ghcr_token" ]]; then
    printf '%s' "$ghcr_token" | docker login ghcr.io -u nself-bot --password-stdin >/dev/null 2>&1 || true
  fi

  log_info "Pulling Docker image: $image_ref"
  if ! docker pull "$image_ref" 2>/dev/null; then
    log_warning "Could not pull GHCR image $image_ref — service will use local build if available"
    return 1
  fi

  # Record the image reference for compose generation
  mkdir -p "$PLUGIN_DIR/$plugin_name"
  printf '%s\n' "$image_ref" > "$PLUGIN_DIR/$plugin_name/.ghcr-image"

  # Track version history so `nself plugin rollback` always has a previous version
  # Extract version tag from image_ref (everything after the last colon)
  local pulled_version="${image_ref##*:}"
  if declare -f _plugin_update_version_files >/dev/null 2>&1; then
    _plugin_update_version_files "$plugin_name" "$pulled_version"
  fi

  log_success "GHCR image pulled: $image_ref"
  return 0
}

# ---------------------------------------------------------------------------
# _install_plugin_binary <plugin_name> [version]
# Downloads a static musl binary for a Rust plugin from GitHub Releases and
# installs it as a systemd service.  Used by `nself plugin install --no-docker`
# for Docker-free Linux environments.
#
# Binary release asset naming convention:
#   nself-{plugin}-{version}-{arch}-linux-musl.tar.gz
#
# Supported architectures (uname -m → asset suffix):
#   x86_64  → x86_64
#   aarch64 → aarch64
#   armv7l  → armv7
#
# Install layout:
#   /usr/local/lib/nself/plugins/nself-{plugin}/
#     nself-{plugin}          ← binary
#     plugin.json             ← (already written by download_plugin)
#   /etc/systemd/system/nself-{plugin}.service
#
# Requires: curl or wget, tar, systemctl (on Linux with systemd).
# Writes a .no-docker marker to $PLUGIN_DIR/{plugin}/ so that plugin-services.sh
# knows to skip docker compose generation for this plugin.
# ---------------------------------------------------------------------------
_install_plugin_binary() {
  local plugin_name="$1"
  local explicit_version="${2:-}"

  # Linux-only
  local os_name
  os_name=$(uname -s)
  if [[ "$os_name" != "Linux" ]]; then
    log_error "--no-docker binary install is only supported on Linux (detected: $os_name)"
    return 1
  fi

  # Detect architecture
  local arch
  arch=$(uname -m)
  local asset_arch=""
  case "$arch" in
    x86_64)  asset_arch="x86_64" ;;
    aarch64) asset_arch="aarch64" ;;
    armv7l)  asset_arch="armv7" ;;
    *)
      log_error "Unsupported architecture for binary install: $arch"
      log_error "Supported: x86_64, aarch64, armv7l"
      return 1
      ;;
  esac

  # Resolve version: explicit > plugin.json > "latest" via GitHub API
  local version="$explicit_version"
  if [[ -z "$version" ]]; then
    local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
    if [[ -f "$manifest" ]]; then
      version=$(grep '"version"' "$manifest" | head -1 | \
        sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/' || true)
    fi
  fi

  # Fetch latest release tag from GitHub API if version still unknown
  local api_base="https://api.github.com/repos/nself-org/nself-${plugin_name}"
  if [[ -z "$version" ]]; then
    log_info "Resolving latest release for nself-${plugin_name}..."
    local rel_json=""
    if command -v curl >/dev/null 2>&1; then
      rel_json=$(curl -fsSL "${api_base}/releases/latest" 2>/dev/null || true)
    elif command -v wget >/dev/null 2>&1; then
      rel_json=$(wget -qO- "${api_base}/releases/latest" 2>/dev/null || true)
    fi
    if [[ -n "$rel_json" ]]; then
      version=$(printf '%s' "$rel_json" | grep '"tag_name"' | head -1 | \
        sed 's/.*"tag_name"[[:space:]]*:[[:space:]]*"v\?\([^"]*\)".*/\1/' || true)
    fi
  fi

  if [[ -z "$version" ]]; then
    log_error "Could not determine version for nself-${plugin_name}"
    return 1
  fi

  local asset_name="nself-${plugin_name}-${version}-${asset_arch}-linux-musl.tar.gz"
  local download_url="https://github.com/nself-org/nself-${plugin_name}/releases/download/v${version}/${asset_name}"

  log_info "Downloading binary: $asset_name"

  local tmp_dir
  tmp_dir=$(mktemp -d)
  local tarball="$tmp_dir/$asset_name"

  # Download
  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$tarball" "$download_url" || {
      log_error "Download failed: $download_url"
      rm -rf "$tmp_dir"
      return 1
    }
  elif command -v wget >/dev/null 2>&1; then
    wget -qO "$tarball" "$download_url" || {
      log_error "Download failed: $download_url"
      rm -rf "$tmp_dir"
      return 1
    }
  else
    log_error "curl or wget is required for binary install"
    rm -rf "$tmp_dir"
    return 1
  fi

  # Extract
  tar -xzf "$tarball" -C "$tmp_dir" || {
    log_error "Failed to extract $asset_name"
    rm -rf "$tmp_dir"
    return 1
  }

  # Install binary to /usr/local/lib/nself/plugins/nself-{plugin}/
  local install_dir="/usr/local/lib/nself/plugins/nself-${plugin_name}"
  mkdir -p "$install_dir"

  local binary_src="$tmp_dir/nself-${plugin_name}"
  if [[ ! -f "$binary_src" ]]; then
    # Some archives nest under a directory
    binary_src=$(find "$tmp_dir" -name "nself-${plugin_name}" -type f | head -1)
  fi
  if [[ -z "$binary_src" ]]; then
    log_error "Binary nself-${plugin_name} not found in archive"
    rm -rf "$tmp_dir"
    return 1
  fi

  cp "$binary_src" "$install_dir/nself-${plugin_name}"
  chmod 755 "$install_dir/nself-${plugin_name}"

  # Copy plugin.json from download_plugin output if available
  local plugin_json_src="$PLUGIN_DIR/$plugin_name/plugin.json"
  if [[ -f "$plugin_json_src" ]]; then
    cp "$plugin_json_src" "$install_dir/plugin.json"
  fi

  rm -rf "$tmp_dir"

  # Write .no-docker marker — tells plugin-services.sh to skip compose generation
  mkdir -p "$PLUGIN_DIR/$plugin_name"
  printf 'binary\n' > "$PLUGIN_DIR/$plugin_name/.no-docker"

  # Create systemd unit file
  if command -v systemctl >/dev/null 2>&1; then
    local service_file="/etc/systemd/system/nself-${plugin_name}.service"
    # Determine default port from plugin.json if available
    local svc_port=""
    if [[ -f "$install_dir/plugin.json" ]]; then
      svc_port=$(grep '"port"' "$install_dir/plugin.json" | head -1 | \
        sed 's/.*"port"[[:space:]]*:[[:space:]]*\([0-9]*\).*/\1/' || true)
    fi

    # Build env block for systemd unit
    local env_line=""
    if [[ -n "$svc_port" ]]; then
      env_line="Environment=PORT=${svc_port}"
    fi

    cat > "$service_file" <<UNIT
[Unit]
Description=nself plugin: nself-${plugin_name}
After=network.target

[Service]
Type=simple
ExecStart=${install_dir}/nself-${plugin_name}
Restart=on-failure
RestartSec=5
${env_line}
StandardOutput=journal
StandardError=journal
SyslogIdentifier=nself-${plugin_name}

[Install]
WantedBy=multi-user.target
UNIT

    systemctl daemon-reload 2>/dev/null || true
    systemctl enable "nself-${plugin_name}.service" 2>/dev/null || \
      log_warning "Could not enable nself-${plugin_name} service (run as root?)"
    systemctl start "nself-${plugin_name}.service" 2>/dev/null || \
      log_warning "Could not start nself-${plugin_name} service (run as root?)"
  else
    log_warning "systemd not available — binary installed but service not registered"
    log_warning "Start manually: $install_dir/nself-${plugin_name}"
  fi

  log_success "Binary installed: $install_dir/nself-${plugin_name}"
  log_success "Plugin $plugin_name installed via binary (no Docker required)"
  return 0
}

# ---------------------------------------------------------------------------
# _plugin_activate_docker_service <plugin_name>
# Regenerates docker-compose.yml via `nself build` (if available and inside a
# project directory with a .env) then brings the plugin service up.
# Runs as a best-effort step — failures are logged but do not abort install.
# ---------------------------------------------------------------------------
_plugin_activate_docker_service() {
  local plugin_name="$1"

  # Only run inside a project directory
  if [[ ! -f ".env" ]]; then
    log_info "Not in a project directory — skipping docker compose up for $plugin_name"
    log_info "Run 'nself build && docker compose up -d nself-${plugin_name}' to start the service"
    return 0
  fi

  if ! command -v docker >/dev/null 2>&1; then
    log_info "Run 'nself build && docker compose up -d nself-${plugin_name}' to start the service"
    return 0
  fi

  # Rebuild compose file to include the new plugin service
  if command -v nself >/dev/null 2>&1; then
    log_info "Regenerating docker-compose.yml..."
    nself build >/dev/null 2>&1 || {
      log_warning "nself build failed — run 'nself build' manually then start the service"
      return 0
    }
  fi

  # Start the plugin service
  log_info "Starting nself-${plugin_name}..."
  if docker compose up -d "nself-${plugin_name}" 2>/dev/null; then
    log_success "Service nself-${plugin_name} started"
  else
    log_warning "Could not start nself-${plugin_name} — run: docker compose up -d nself-${plugin_name}"
  fi

  return 0
}

download_plugin() {
  local plugin_name="$1"
  local version="${2:-main}"

  mkdir -p "$PLUGIN_DIR"

  local temp_dir
  temp_dir=$(mktemp -d)

  # Paid plugins are served from the API via a signed download URL.
  # Free plugins come straight from the public GitHub repo.
  # Use license_get_key() which checks both NSELF_PLUGIN_LICENSE_KEY env var
  # and the persisted ~/.nself/license/key file (set via `nself plugin license set`).
  local license_key=""
  if declare -f license_get_key >/dev/null 2>&1; then
    license_key=$(license_get_key 2>/dev/null || true)
  else
    license_key="${NSELF_PLUGIN_LICENSE_KEY:-}"
  fi
  local use_signed_url=false
  if declare -f license_is_paid_plugin >/dev/null 2>&1; then
    if license_is_paid_plugin "$plugin_name" && [[ -n "$license_key" ]]; then
      use_signed_url=true
    fi
  fi

  if [[ "$use_signed_url" == "true" ]]; then
    log_info "Downloading $plugin_name (pro)..."
    if ! _download_plugin_signed "$plugin_name" "$license_key" "$temp_dir"; then
      rm -rf "$temp_dir"
      return 1
    fi
    # For Rust binary plugins, also download and install the pre-compiled binary.
    # The tarball step above installs plugin.json and any TS/config assets;
    # the binary step installs the executable into $PLUGIN_DIR/<name>/bin/.
    if plugin_is_rust "$plugin_name"; then
      log_info "Downloading Rust binary for $plugin_name..."
      local rust_temp_dir
      rust_temp_dir=$(mktemp -d)
      if ! _download_plugin_rust "$plugin_name" "$license_key" "$rust_temp_dir"; then
        rm -rf "$rust_temp_dir"
        rm -rf "$temp_dir"
        return 1
      fi
      rm -rf "$rust_temp_dir"
      # Generate systemd unit file on Linux (no-op on macOS / no systemd)
      _generate_systemd_unit "$plugin_name"
    fi
  else
    log_info "Downloading $plugin_name..."
    local tarball_url="${PLUGIN_REPO_URL}/archive/refs/heads/main.tar.gz"
    if ! curl -sL "$tarball_url" | tar -xz -C "$temp_dir" 2>/dev/null; then
      log_error "Failed to download plugin"
      rm -rf "$temp_dir"
      return 1
    fi

    local plugin_src="$temp_dir/nself-plugins-main/plugins/$plugin_name"
    if [[ ! -d "$plugin_src" ]]; then
      log_error "Plugin '$plugin_name' not found in repository"
      rm -rf "$temp_dir"
      return 1
    fi

    rm -rf "$PLUGIN_DIR/$plugin_name"
    cp -r "$plugin_src" "$PLUGIN_DIR/$plugin_name"

    if [[ -d "$temp_dir/nself-plugins-main/shared" ]]; then
      mkdir -p "$PLUGIN_DIR/_shared"
      cp -r "$temp_dir/nself-plugins-main/shared/"* "$PLUGIN_DIR/_shared/"
    fi
  fi

  rm -rf "$temp_dir"
  log_success "Downloaded $plugin_name"
}

# Fetch a paid plugin via a server-issued signed URL.
# The API validates the license, generates a time-limited token, and the CLI
# redeems the token in a second request — keeping the license key out of
# download logs and caches.
_download_plugin_signed() {
  local plugin_name="$1"
  local license_key="$2"
  local temp_dir="$3"

  local url_endpoint="${NSELF_API_DOWNLOAD_URL}/${plugin_name}/download-url"

  # Step 1: request a signed download URL from the API
  local response
  response=$(curl -sf \
    -H "X-License-Key: ${license_key}" \
    -H "X-Domain: ${NSELF_DOMAIN:-}" \
    "$url_endpoint" 2>/dev/null)

  if [[ -z "$response" ]]; then
    log_error "Failed to reach plugin distribution service"
    printf "Check your internet connection or run: nself plugin license validate\n"
    return 1
  fi

  # Parse HTTP-level errors returned as JSON { "error": "..." }
  local api_error
  api_error=$(printf '%s' "$response" | grep -o '"error"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"error"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
  if [[ -n "$api_error" ]]; then
    log_error "Download authorization failed: $api_error"
    if [[ "$api_error" == *"expired"* ]] || [[ "$api_error" == *"Invalid license"* ]]; then
      printf "Renew your license at: https://nself.org/commercial\n"
    fi
    return 1
  fi

  # Extract the signed URL from the JSON response
  local signed_url
  signed_url=$(printf '%s' "$response" | grep -o '"url"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"url"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
  if [[ -z "$signed_url" ]]; then
    log_error "Unexpected response from plugin distribution service"
    return 1
  fi

  # Step 2: download the tarball using the signed URL (no license key needed here)
  local tarball="$temp_dir/${plugin_name}.tar.gz"
  if ! curl -sL -o "$tarball" "$signed_url" 2>/dev/null; then
    log_error "Failed to download plugin tarball"
    return 1
  fi

  if ! tar -xz -C "$temp_dir" -f "$tarball" 2>/dev/null; then
    log_error "Failed to extract plugin tarball"
    return 1
  fi

  # Pro plugins unpack as plugins-pro-main/paid/<plugin_name>/
  local plugin_src="$temp_dir/plugins-pro-main/paid/$plugin_name"
  if [[ ! -d "$plugin_src" ]]; then
    log_error "Plugin '$plugin_name' not found in downloaded archive"
    return 1
  fi

  rm -rf "$PLUGIN_DIR/$plugin_name"
  cp -r "$plugin_src" "$PLUGIN_DIR/$plugin_name"

  # Copy shared utilities bundled with the pro repo (if any)
  if [[ -d "$temp_dir/plugins-pro-main/shared" ]]; then
    mkdir -p "$PLUGIN_DIR/_shared"
    cp -r "$temp_dir/plugins-pro-main/shared/"* "$PLUGIN_DIR/_shared/"
  fi

  return 0
}

# Check whether a plugin is a Rust binary plugin by inspecting its registry entry.
# Returns 0 if rust, 1 otherwise.
# Falls back to checking the locally installed plugin.json when the registry is
# unavailable (e.g. offline install or after the plugin is already downloaded).
plugin_is_rust() {
  local plugin_name="$1"

  # Primary: check locally installed plugin.json (fast, no network)
  local local_manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
  if [[ -f "$local_manifest" ]]; then
    if grep -q '"language"[[:space:]]*:[[:space:]]*"rust"' "$local_manifest"; then
      return 0
    fi
    # Manifest exists but language is not rust — no need to hit registry
    return 1
  fi

  # Secondary: check registry (plugin not yet downloaded)
  local registry
  registry=$(fetch_registry 2>/dev/null || true)
  if [[ -n "$registry" ]]; then
    printf '%s' "$registry" | grep -A 20 "\"name\"[[:space:]]*:[[:space:]]*\"${plugin_name}\"" \
      | grep -q '"language"[[:space:]]*:[[:space:]]*"rust"'
    return $?
  fi

  return 1
}

# Download a pre-compiled Rust binary for the current architecture.
# Requests a signed URL from ping_api with an arch parameter, verifies SHA-256,
# and installs the binary to $PLUGIN_DIR/<name>/bin/<name>.
_download_plugin_rust() {
  local plugin_name="$1"
  local key="$2"
  local temp_dir="$3"

  # Detect current platform
  local os arch uname_m
  os=$(uname -s | tr '[:upper:]' '[:lower:]')
  uname_m=$(uname -m)
  case "$uname_m" in
    x86_64)        arch="x86_64" ;;
    arm64|aarch64) arch="arm64" ;;
    *)
      log_error "Unsupported architecture: $uname_m"
      return 1
      ;;
  esac
  local platform="${os}-${arch}"

  # Request signed download URL from ping_api with arch param
  local ping_url="${NSELF_PING_URL:-https://ping.nself.org}"
  local url_response
  url_response=$(curl -sf --connect-timeout 10 --max-time 15 \
    -X POST \
    -H "Content-Type: application/json" \
    -H "X-License-Key: $key" \
    "${ping_url}/plugins/${plugin_name}/download-url?arch=${platform}" 2>/dev/null)

  if [[ -z "$url_response" ]]; then
    log_error "Failed to get download URL for $plugin_name ($platform)"
    return 1
  fi

  local signed_url sha256
  signed_url=$(printf '%s' "$url_response" | grep -o '"url":"[^"]*"' | cut -d'"' -f4)
  sha256=$(printf '%s' "$url_response" | grep -o '"sha256":"[^"]*"' | cut -d'"' -f4)

  if [[ -z "$signed_url" ]]; then
    log_error "No download URL in response for $plugin_name"
    return 1
  fi

  # Download binary
  local binary_path="${temp_dir}/${plugin_name}"
  if ! curl -sf --connect-timeout 10 --max-time 120 \
    -o "$binary_path" \
    "$signed_url" 2>/dev/null; then
    log_error "Failed to download $plugin_name binary"
    return 1
  fi

  # Verify SHA-256 if provided
  if [[ -n "$sha256" ]]; then
    local actual_sha=""
    if command -v sha256sum >/dev/null 2>&1; then
      actual_sha=$(sha256sum "$binary_path" | awk '{print $1}')
    elif command -v shasum >/dev/null 2>&1; then
      actual_sha=$(shasum -a 256 "$binary_path" | awk '{print $1}')
    fi
    if [[ -n "$actual_sha" && "$actual_sha" != "$sha256" ]]; then
      log_error "SHA-256 mismatch for $plugin_name binary (expected: $sha256, got: $actual_sha)"
      rm -f "$binary_path"
      return 1
    fi
  fi

  # Install binary
  local install_dir="$PLUGIN_DIR/$plugin_name/bin"
  mkdir -p "$install_dir"
  mv "$binary_path" "${install_dir}/${plugin_name}"
  chmod +x "${install_dir}/${plugin_name}"

  log_success "Rust binary installed: ${install_dir}/${plugin_name} ($platform)"
  return 0
}

# Generate a systemd user unit file for a Rust plugin and enable it when
# systemd --user is available (Linux only).  On macOS or when systemd is not
# present the unit file is written but not enabled.
_generate_systemd_unit() {
  local plugin_name="$1"
  local binary_path="$PLUGIN_DIR/$plugin_name/bin/$plugin_name"
  local env_file="$PLUGIN_DIR/$plugin_name/.env"
  local unit_dir="$HOME/.config/systemd/user"
  local unit_file="$unit_dir/nself-${plugin_name}.service"

  mkdir -p "$unit_dir"

  # Write the unit file — use printf to avoid heredoc variable-expansion issues
  # in Bash 3.2 and to stay consistent with the no-echo-e rule.
  printf '[Unit]\n' >"$unit_file"
  printf 'Description=nself plugin: %s\n' "$plugin_name" >>"$unit_file"
  printf 'After=network.target\n' >>"$unit_file"
  printf 'PartOf=nself.target\n' >>"$unit_file"
  printf '\n' >>"$unit_file"
  printf '[Service]\n' >>"$unit_file"
  printf 'Type=simple\n' >>"$unit_file"
  printf 'ExecStart=%s\n' "$binary_path" >>"$unit_file"
  printf 'Restart=on-failure\n' >>"$unit_file"
  printf 'RestartSec=5\n' >>"$unit_file"
  printf 'EnvironmentFile=-%s\n' "$env_file" >>"$unit_file"
  printf 'StandardOutput=journal\n' >>"$unit_file"
  printf 'StandardError=journal\n' >>"$unit_file"
  printf 'SyslogIdentifier=nself-%s\n' "$plugin_name" >>"$unit_file"
  printf '\n' >>"$unit_file"
  printf '[Install]\n' >>"$unit_file"
  printf 'WantedBy=default.target\n' >>"$unit_file"

  if command -v systemctl >/dev/null 2>&1 && systemctl --user status >/dev/null 2>&1; then
    systemctl --user daemon-reload 2>/dev/null || true
    systemctl --user enable "nself-${plugin_name}.service" 2>/dev/null || true
    log_success "systemd unit installed: $unit_file"
  else
    log_info "systemd unit created (not enabled — systemd --user not available): $unit_file"
  fi
}

# Start a Rust plugin process.  Prefers systemd --user when available; falls
# back to direct process management with a PID file.
_start_rust_plugin() {
  local plugin_name="$1"
  local health_endpoint="${2:-/health}"
  local binary_path="$PLUGIN_DIR/$plugin_name/bin/$plugin_name"
  local pid_file="$PLUGIN_DIR/$plugin_name/pid"
  local env_file="$PLUGIN_DIR/$plugin_name/.env"
  local log_file="$PLUGIN_DIR/$plugin_name/plugin.log"

  if [[ ! -x "$binary_path" ]]; then
    log_error "Binary not found: $binary_path"
    return 1
  fi

  # Try systemd --user first on Linux
  if command -v systemctl >/dev/null 2>&1 && \
     [[ -f "$HOME/.config/systemd/user/nself-${plugin_name}.service" ]]; then
    systemctl --user start "nself-${plugin_name}.service" 2>/dev/null
    log_success "Started $plugin_name via systemd"
    return 0
  fi

  # Fallback: direct process management
  if [[ -f "$pid_file" ]]; then
    local existing_pid
    existing_pid=$(cat "$pid_file" 2>/dev/null)
    if kill -0 "$existing_pid" 2>/dev/null; then
      log_info "$plugin_name is already running (PID $existing_pid)"
      return 0
    fi
  fi

  # Build env prefix from env file when present (filter comments and blank lines)
  local env_prefix=""
  if [[ -f "$env_file" ]]; then
    env_prefix="env $(grep -v '^#' "$env_file" | grep '=' | tr '\n' ' ')"
  fi

  # Start in background
  eval "$env_prefix" "$binary_path" >>"$log_file" 2>&1 &
  local pid=$!
  printf '%s' "$pid" >"$pid_file"

  # Determine port for health check
  local port
  port=$(grep -o 'PORT=[0-9]*' "$env_file" 2>/dev/null | cut -d= -f2 || true)
  port="${port:-8080}"

  # Poll health endpoint — 10 retries, 0.5 s apart (5 s total)
  local retries=10
  local healthy=false
  while [ "$retries" -gt 0 ]; do
    sleep 0.5
    if curl -sf --connect-timeout 1 "http://127.0.0.1:${port}${health_endpoint}" >/dev/null 2>&1; then
      healthy=true
      break
    fi
    retries=$((retries - 1))
  done

  if [ "$healthy" = true ]; then
    log_success "Plugin $plugin_name started (PID $pid)"
  else
    log_warn "Plugin $plugin_name started (PID $pid) — health check timed out, may still be initializing"
  fi
}

# Stop a Rust plugin process.  Prefers systemd --user; falls back to
# SIGTERM via the PID file with a SIGKILL escalation after 1 s.
_stop_rust_plugin() {
  local plugin_name="$1"
  local pid_file="$PLUGIN_DIR/$plugin_name/pid"

  # Try systemd first
  if command -v systemctl >/dev/null 2>&1 && \
     [[ -f "$HOME/.config/systemd/user/nself-${plugin_name}.service" ]]; then
    systemctl --user stop "nself-${plugin_name}.service" 2>/dev/null
    log_success "Stopped $plugin_name via systemd"
    return 0
  fi

  if [[ -f "$pid_file" ]]; then
    local pid
    pid=$(cat "$pid_file")
    if kill -TERM "$pid" 2>/dev/null; then
      sleep 1
      kill -0 "$pid" 2>/dev/null && kill -KILL "$pid" 2>/dev/null || true
      rm -f "$pid_file"
      log_success "Stopped $plugin_name (PID $pid)"
    else
      rm -f "$pid_file"
      log_info "$plugin_name was not running"
    fi
  else
    log_info "No PID file found for $plugin_name"
  fi
}

# ============================================================================
# PLUGIN SOURCE SYNC
# ============================================================================

# sync_plugin_source <plugin_name>
#
# Copies TypeScript plugin source from the global plugin store
# (~/.nself/plugins/<name>/ts/src/) to the project services directory
# ({project}/services/<name>/src/).
#
# WHY THIS EXISTS:
#   After `nself plugin install` or `nself plugin update`, the new source
#   lands in PLUGIN_DIR but the project's services/<name>/src/ is NOT
#   updated. Consequently `nself build` regenerates docker-compose.yml
#   using the OLD source — the container runs stale code until the user
#   manually copies files. This function eliminates that manual step.
#
# Bash 3.2 compatible — no rsync required (falls back to cp).
sync_plugin_source() {
  local plugin_name="$1"
  local project_dir="${NSELF_PROJECT_DIR:-$(pwd)}"

  # Source: ~/.nself/plugins/<name>/ts/src/
  local plugin_ts_src="${PLUGIN_DIR}/${plugin_name}/ts/src"

  # Only applies to TypeScript plugins (Rust plugins use pre-compiled binaries)
  if [[ ! -d "$plugin_ts_src" ]]; then
    return 0
  fi

  # Only run inside a project directory (must have .env)
  if [[ ! -f "${project_dir}/.env" ]]; then
    return 0
  fi

  local service_src="${project_dir}/services/${plugin_name}/src"
  mkdir -p "$service_src"

  # Sync ts/src/ → services/<name>/src/
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --delete "${plugin_ts_src}/" "${service_src}/"
  else
    # POSIX cp fallback: overwrites changed files (does not delete removed files,
    # but is safe for the install/update path since the source is authoritative)
    cp -r "${plugin_ts_src}/." "${service_src}/"
  fi

  log_info "Plugin source synced to ${service_src}"

  # Also update package.json if the plugin ships one at ts/package.json
  local plugin_pkg="${PLUGIN_DIR}/${plugin_name}/ts/package.json"
  local service_pkg="${project_dir}/services/${plugin_name}/package.json"
  if [[ -f "$plugin_pkg" ]] && [[ -f "$service_pkg" ]]; then
    cp "$plugin_pkg" "$service_pkg"
    log_info "Updated package.json in ${project_dir}/services/${plugin_name}/"
  fi
}

# cmd_sync [<plugin_name>]
#
# Manually sync plugin TypeScript source to the project services directory.
# Useful when auto-sync did not run (e.g. plugin was installed from outside
# the project directory) or to verify sources are up to date.
#
# Usage:
#   nself plugin sync <name>   Sync a specific plugin
#   nself plugin sync          Sync all installed TypeScript plugins
cmd_sync() {
  local plugin_name="${1:-}"

  export NSELF_PROJECT_DIR="$(pwd)"

  if [[ -z "$plugin_name" ]]; then
    # Sync all installed TypeScript plugins
    local synced=0
    for plugin_dir_entry in "${PLUGIN_DIR}"/*/; do
      if [[ -d "${plugin_dir_entry}ts/src" ]]; then
        local name
        name=$(basename "$plugin_dir_entry")
        if sync_plugin_source "$name"; then
          synced=$((synced + 1))
        fi
      fi
    done
    if [[ $synced -eq 0 ]]; then
      log_info "No TypeScript plugins found to sync"
    else
      log_success "Synced ${synced} plugin(s)"
      printf "\nRun 'nself build' then 'nself restart' to apply.\n"
    fi
    return 0
  fi

  if ! is_plugin_installed "$plugin_name"; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  local plugin_ts_src="${PLUGIN_DIR}/${plugin_name}/ts/src"
  if [[ ! -d "$plugin_ts_src" ]]; then
    log_error "Plugin '$plugin_name' does not have TypeScript source at ${plugin_ts_src}"
    log_info "Rust plugins use pre-compiled binaries and do not need source sync."
    return 1
  fi

  sync_plugin_source "$plugin_name"
  log_success "Synced '$plugin_name'"
  printf "\nRun: nself build && nself restart %s\n" "$plugin_name"
}

install_local_plugin() {
  local plugin_path="$1"

  if [[ ! -f "$plugin_path/plugin.json" ]]; then
    log_error "Invalid plugin: missing plugin.json"
    return 1
  fi

  local plugin_name
  plugin_name=$(grep '"name"' "$plugin_path/plugin.json" | head -1 | sed 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')

  log_info "Installing local plugin: $plugin_name"

  mkdir -p "$PLUGIN_DIR"
  rm -rf "$PLUGIN_DIR/$plugin_name"
  cp -r "$plugin_path" "$PLUGIN_DIR/$plugin_name"

  run_plugin_installer "$plugin_name"

  log_success "Local plugin '$plugin_name' installed"
}

run_plugin_installer() {
  local plugin_name="$1"
  local plugin_dir="$PLUGIN_DIR/$plugin_name"

  if [[ -f "$plugin_dir/install.sh" ]]; then
    log_info "Running plugin installer..."

    # Make shared utilities available
    export PLUGIN_DIR="$plugin_dir"
    export NSELF_PROJECT_DIR="$(pwd)"

    # Update shared path in install script
    local shared_path="$PLUGIN_DIR/_shared"
    if [[ -d "$shared_path" ]]; then
      export SHARED_DIR="$shared_path"
    fi

    bash "$plugin_dir/install.sh"
  fi
}

# ============================================================================
# UPDATE CHECKING
# ============================================================================

# Check for plugin updates
cmd_updates() {
  local quiet=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --quiet | -q)
        quiet=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  if [[ "$quiet" == "true" ]]; then
    # Quiet mode: just output update info
    if declare -f registry_check_updates >/dev/null 2>&1; then
      registry_check_updates 2>/dev/null
    fi
  else
    printf "\n=== Plugin Updates ===\n\n"

    # Check if any plugins are installed
    local installed_count=0
    for plugin_dir in "$PLUGIN_DIR"/*/; do
      [[ -f "$plugin_dir/plugin.json" ]] && installed_count=$((installed_count + 1))
    done

    if [[ $installed_count -eq 0 ]]; then
      log_info "No plugins installed"
      printf "\nInstall plugins with: nself plugin install <name>\n"
      return 0
    fi

    log_info "Checking for updates ($installed_count plugins installed)..."
    echo ""

    if declare -f registry_check_updates_formatted >/dev/null 2>&1; then
      registry_check_updates_formatted
    else
      # Fallback if registry.sh not loaded
      log_warning "Registry module not loaded. Skipping update check."
    fi
  fi
}

# Refresh plugin registry cache
cmd_refresh() {
  log_info "Refreshing plugin registry..."

  if declare -f registry_fetch >/dev/null 2>&1; then
    if registry_fetch "true" >/dev/null 2>&1; then
      log_success "Registry cache refreshed"

      # Show registry info
      if declare -f registry_get_metadata >/dev/null 2>&1; then
        echo ""
        registry_get_metadata
      fi
    else
      log_error "Failed to refresh registry"
      return 1
    fi
  else
    # Fallback
    local registry
    if registry=$(curl -sf "$PLUGIN_REGISTRY_URL/registry.json" 2>/dev/null || curl -sf "$PLUGIN_REGISTRY_FALLBACK" 2>/dev/null); then
      mkdir -p "$PLUGIN_CACHE_DIR"
      printf '%s' "$registry" >"$PLUGIN_CACHE_DIR/registry.json"
      log_success "Registry cache refreshed"
    else
      log_error "Failed to fetch registry"
      return 1
    fi
  fi
}

# ============================================================================
# OUTDATED / VERSION MANIFEST (T-0208)
# ============================================================================

PLUGIN_MANIFEST_URL="${NSELF_PLUGIN_MANIFEST_URL:-https://plugins.nself.org/manifest.json}"

# cmd_outdated — show installed plugins with current vs latest version
# Fetches manifest.json from plugins.nself.org and compares installed versions.
cmd_outdated() {
  local quiet=false
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --quiet | -q) quiet=true; shift ;;
      *) shift ;;
    esac
  done

  # Fetch manifest
  local manifest
  manifest=$(curl -sf "$PLUGIN_MANIFEST_URL" 2>/dev/null)
  if [[ -z "$manifest" ]]; then
    log_error "Failed to fetch plugin manifest from $PLUGIN_MANIFEST_URL"
    return 1
  fi

  if ! command -v jq >/dev/null 2>&1; then
    log_error "jq is required to check plugin versions"
    return 1
  fi

  local found_outdated=false

  for plugin_dir in "$PLUGIN_DIR"/*/; do
    [[ -f "$plugin_dir/plugin.json" ]] || continue
    local slug installed_ver latest_ver
    slug=$(jq -r '.name // .slug // ""' "$plugin_dir/plugin.json" 2>/dev/null)
    installed_ver=$(jq -r '.version // "unknown"' "$plugin_dir/plugin.json" 2>/dev/null)
    [[ -z "$slug" ]] && continue

    latest_ver=$(printf '%s' "$manifest" | jq -r --arg s "$slug" '.[$s].latest_version // ""' 2>/dev/null)
    [[ -z "$latest_ver" ]] && continue

    if [[ "$installed_ver" != "$latest_ver" ]]; then
      if [[ "$quiet" == "true" ]]; then
        printf '%s %s %s\n' "$slug" "$installed_ver" "$latest_ver"
      else
        printf '  %-30s installed: %-10s latest: %s\n' "$slug" "$installed_ver" "$latest_ver"
      fi
      found_outdated=true
    fi
  done

  if [[ "$found_outdated" == "false" ]]; then
    if [[ "$quiet" != "true" ]]; then
      log_success "All installed plugins are up to date"
    fi
  fi
}

# ============================================================================
# PLUGIN ROLLBACK (T-0309)
# ============================================================================

# ---------------------------------------------------------------------------
# _plugin_update_version_files <plugin_name> <new_version> [old_version]
#
# Writes two tracking files into the plugin directory:
#   .version          — the currently active image version
#   .previous-version — the version that was active before this change
#
# Also updates plugin.json in-place (version + previous_version fields)
# when jq is available.
#
# Called from _pull_plugin_docker_image (on install/update) and from
# cmd_plugin_rollback so rollback always has a recorded previous version.
#
# Bash 3.2 compatible — no declare -A, no ${var,,}, no echo -e
# ---------------------------------------------------------------------------
_plugin_update_version_files() {
  local plugin_name="$1"
  local new_version="$2"
  local old_version="${3:-}"
  local plugin_dir="$PLUGIN_DIR/$plugin_name"

  mkdir -p "$plugin_dir"

  # If old_version not supplied, read it from the existing .version file
  if [[ -z "$old_version" && -f "$plugin_dir/.version" ]]; then
    old_version=$(cat "$plugin_dir/.version" 2>/dev/null || true)
  fi

  # Write .version
  printf '%s\n' "$new_version" > "$plugin_dir/.version"

  # Write .previous-version only when there is a meaningful prior version
  if [[ -n "$old_version" && "$old_version" != "$new_version" ]]; then
    printf '%s\n' "$old_version" > "$plugin_dir/.previous-version"
  fi

  # Keep plugin.json in sync when jq is available
  local manifest="$plugin_dir/plugin.json"
  if [[ -f "$manifest" ]] && command -v jq >/dev/null 2>&1; then
    local tmp_file
    tmp_file=$(mktemp)
    if [[ -n "$old_version" && "$old_version" != "$new_version" ]]; then
      jq --arg v "$new_version" --arg pv "$old_version" \
        '.version = $v | .previous_version = $pv' \
        "$manifest" > "$tmp_file" && mv "$tmp_file" "$manifest"
    else
      jq --arg v "$new_version" \
        '.version = $v' \
        "$manifest" > "$tmp_file" && mv "$tmp_file" "$manifest"
    fi
  fi
}

# cmd_plugin_rollback — roll back a plugin to its previous or specified version
#
# Usage:
#   nself plugin rollback <name>                    Roll back to previous version
#   nself plugin rollback <name> <version>          Roll back to specific version
#   nself plugin rollback <name> --list             List available versions from GHCR
#   nself plugin rollback <name> [version] --force  Skip confirmation prompt
#
# Bash 3.2 compatible — no declare -A, no ${var,,}, no echo -e
cmd_plugin_rollback() {
  local plugin_name="${1:-}"
  local target_version=""
  local force=false

  # Parse remaining args: version string, --list, --force, --help
  shift || true
  local _rb_arg
  for _rb_arg in "$@"; do
    case "$_rb_arg" in
      --force|-f)   force=true ;;
      --list)       target_version="--list" ;;
      --help|-h)
        printf "Usage: nself plugin rollback <name> [version] [options]\n\n"
        printf "Roll back a plugin to its previous or a specific version.\n\n"
        printf "Arguments:\n"
        printf "  name        Plugin name (must be installed)\n"
        printf "  version     Target version (optional, uses previous version if omitted)\n\n"
        printf "Options:\n"
        printf "  --list      List available versions pulled from GHCR\n"
        printf "  --force     Skip confirmation prompt\n"
        printf "  --help, -h  Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin rollback ai\n"
        printf "  nself plugin rollback ai 0.0.9\n"
        printf "  nself plugin rollback ai --list\n"
        printf "  nself plugin rollback ai 0.0.9 --force\n"
        return 0
        ;;
      -*)
        # Unknown flag — skip silently for forward compatibility
        ;;
      *)
        # First non-flag positional after name is the version
        if [[ -z "$target_version" ]]; then
          target_version="$_rb_arg"
        fi
        ;;
    esac
  done
  unset _rb_arg

  if [[ -z "$plugin_name" ]]; then
    log_error "Plugin name required"
    printf "\nUsage:\n"
    printf "  nself plugin rollback <name>              Roll back to previous version\n"
    printf "  nself plugin rollback <name> <version>    Roll back to specific version\n"
    printf "  nself plugin rollback <name> --list       List available versions\n"
    return 1
  fi

  # Validate plugin is installed
  if ! is_plugin_installed "$plugin_name"; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  local plugin_dir="$PLUGIN_DIR/$plugin_name"
  local manifest="$plugin_dir/plugin.json"
  local current_version=""
  local previous_version=""

  # Read current version — prefer .version file, fall back to plugin.json
  if [[ -f "$plugin_dir/.version" ]]; then
    current_version=$(cat "$plugin_dir/.version" 2>/dev/null || true)
  fi
  if [[ -z "$current_version" && -f "$manifest" ]]; then
    if command -v jq >/dev/null 2>&1; then
      current_version=$(jq -r '.version // ""' "$manifest" 2>/dev/null)
    else
      current_version=$(grep '"version"' "$manifest" | head -1 | \
        sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    fi
  fi

  # Read previous version — prefer .previous-version file, fall back to plugin.json
  if [[ -f "$plugin_dir/.previous-version" ]]; then
    previous_version=$(cat "$plugin_dir/.previous-version" 2>/dev/null || true)
  fi
  if [[ -z "$previous_version" && -f "$manifest" ]]; then
    if command -v jq >/dev/null 2>&1; then
      previous_version=$(jq -r '.previous_version // ""' "$manifest" 2>/dev/null)
    else
      previous_version=$(grep '"previous_version"' "$manifest" | head -1 | \
        sed 's/.*"previous_version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    fi
  fi

  # --list: show locally cached tags plus remote GHCR listing
  if [[ "$target_version" == "--list" ]]; then
    printf "Available versions for %s:\n\n" "$plugin_name"

    # Local tags already pulled to the docker daemon
    if command -v docker >/dev/null 2>&1; then
      local local_tags
      local_tags=$(docker image ls \
        --filter "reference=ghcr.io/nself-org/nself-${plugin_name}*" \
        --format "{{.Tag}}" 2>/dev/null | grep -v '<none>' | sort -rV || true)
      if [[ -n "$local_tags" ]]; then
        printf "Locally cached:\n"
        printf '%s\n' "$local_tags" | while IFS= read -r tag; do
          [[ -n "$tag" ]] && printf "  %s\n" "$tag"
        done
        printf "\n"
      fi
    fi

    # Remote listing via anonymous GHCR token exchange
    log_info "Fetching remote tags from GHCR..."
    local token
    token=$(curl -sf \
      "https://ghcr.io/token?scope=repository:nself-org/nself-${plugin_name}:pull" 2>/dev/null | \
      grep -o '"token":"[^"]*"' | sed 's/"token":"//;s/"$//' || true)

    if [[ -n "$token" ]]; then
      local tags_json
      tags_json=$(curl -sf \
        -H "Authorization: Bearer $token" \
        "https://ghcr.io/v2/nself-org/nself-${plugin_name}/tags/list" 2>/dev/null || true)
      if [[ -n "$tags_json" ]]; then
        printf "Remote tags:\n"
        printf '%s' "$tags_json" | grep -o '"[0-9][^"]*"' | tr -d '"' | sort -rV | \
          while IFS= read -r tag; do
            [[ -n "$tag" ]] && printf "  %s\n" "$tag"
          done
        printf "\n"
      fi
    else
      log_warning "Could not fetch remote tag list from GHCR. Check network connectivity."
    fi

    printf "Current:  %s\n" "${current_version:-unknown}"
    [[ -n "$previous_version" ]] && printf "Previous: %s\n" "$previous_version"
    return 0
  fi

  # Determine rollback target
  local rollback_to="$target_version"
  if [[ -z "$rollback_to" ]]; then
    if [[ -z "$previous_version" ]]; then
      log_error "No previous version recorded for '$plugin_name'."
      printf "Specify a version: nself plugin rollback %s <version>\n" "$plugin_name"
      printf "List versions:     nself plugin rollback %s --list\n" "$plugin_name"
      return 1
    fi
    rollback_to="$previous_version"
  fi

  if [[ "$rollback_to" == "$current_version" ]]; then
    log_warning "Plugin '$plugin_name' is already at version $rollback_to"
    return 0
  fi

  printf "Rolling back '%s':\n" "$plugin_name"
  printf "  Current:  %s\n" "${current_version:-unknown}"
  printf "  Target:   %s\n" "$rollback_to"

  if [[ "$force" != "true" ]]; then
    printf "\nContinue? [y/N] "
    local confirm=""
    read -r confirm
    case "$confirm" in
      [yY]|[yY][eE][sS]) ;;
      *) log_info "Rollback cancelled."; return 0 ;;
    esac
  fi

  local image="${NSELF_GHCR_BASE}/nself-${plugin_name}:${rollback_to}"

  # Authenticate to GHCR if a token is available (private images)
  local ghcr_token="${GHCR_TOKEN:-${GITHUB_TOKEN:-}}"
  if [[ -n "$ghcr_token" ]]; then
    printf '%s' "$ghcr_token" | docker login ghcr.io -u nself-bot --password-stdin >/dev/null 2>&1 || true
  fi

  log_info "Pulling image: $image"
  if ! docker pull "$image"; then
    log_error "Failed to pull image $image. Version may not exist on GHCR."
    printf "Check available versions: nself plugin rollback %s --list\n" "$plugin_name"
    return 1
  fi

  # Update .ghcr-image so docker-compose picks up the pinned image reference
  printf '%s\n' "$image" > "$plugin_dir/.ghcr-image"

  # Stop current container: prefer docker compose (inside project dir),
  # fall back to runtime library's stop_plugin helper.
  log_info "Stopping nself-${plugin_name}..."
  if [[ -f ".env" ]] && command -v docker >/dev/null 2>&1; then
    docker compose stop "nself-${plugin_name}" 2>/dev/null || true
  elif declare -f stop_plugin >/dev/null 2>&1; then
    stop_plugin "$plugin_name" 2>/dev/null || true
  fi

  # Write .version / .previous-version / update plugin.json
  _plugin_update_version_files "$plugin_name" "$rollback_to" "$current_version"

  # Regenerate docker-compose.yml with the new image ref, then start the service
  log_info "Starting nself-${plugin_name} at $rollback_to..."
  if [[ -f ".env" ]] && command -v docker >/dev/null 2>&1; then
    if command -v nself >/dev/null 2>&1; then
      nself build >/dev/null 2>&1 || \
        log_warning "nself build failed — run 'nself build' manually if compose is stale"
    fi
    docker compose up -d "nself-${plugin_name}" 2>/dev/null || \
      log_warning "Could not start nself-${plugin_name} — run: docker compose up -d nself-${plugin_name}"
  elif declare -f start_plugin >/dev/null 2>&1; then
    start_plugin "$plugin_name" 2>/dev/null || true
  fi

  # Health check — poll for up to 30 s (10 retries x 3 s)
  local retries=10
  local healthy=false
  log_info "Waiting for health check..."
  while [[ $retries -gt 0 ]]; do
    if declare -f health_check_plugin >/dev/null 2>&1 && health_check_plugin "$plugin_name" 2>/dev/null; then
      healthy=true
      break
    fi
    retries=$((retries - 1))
    sleep 3
  done

  if [[ "$healthy" == "true" ]]; then
    log_success "Plugin '$plugin_name' rolled back to $rollback_to and is healthy"
  else
    log_warning "Plugin rolled back to $rollback_to but health check did not pass within 30s."
    printf "Check status: nself plugin health\n"
  fi
}

# ============================================================================
# PLUGIN CONFIG (T-0253)
# ============================================================================

# _plugin_config_read_var <var_name>
# Read a variable's current value from .env, .env.local, .env.prod (last wins).
# Outputs the raw value (empty string if not set).
_plugin_config_read_var() {
  local var_name="$1"
  local val=""
  local f
  for f in ".env" ".env.local" ".env.prod"; do
    if [[ -f "$f" ]]; then
      local line
      line=$(grep -E "^${var_name}=" "$f" 2>/dev/null | tail -1)
      if [[ -n "$line" ]]; then
        # Use cut -d= -f2- to preserve = characters in values (e.g. base64)
        val=$(printf '%s' "$line" | cut -d= -f2-)
      fi
    fi
  done
  printf '%s' "$val"
}

# _plugin_config_upsert_env_local <key> <value>
# Write or replace a KEY=VALUE line in .env.local (project directory).
# Uses a temp-file swap — Bash 3.2 compatible, no sed -i.
_plugin_config_upsert_env_local() {
  local key="$1"
  local value="$2"
  local env_local=".env.local"
  if [[ -f "$env_local" ]] && grep -qE "^${key}=" "$env_local" 2>/dev/null; then
    local tmp_file
    tmp_file=$(mktemp)
    grep -v "^${key}=" "$env_local" > "$tmp_file" 2>/dev/null || true
    printf '%s=%s\n' "$key" "$value" >> "$tmp_file"
    mv "$tmp_file" "$env_local"
  else
    printf '%s=%s\n' "$key" "$value" >> "$env_local"
  fi
}

# _plugin_config_parse_env_vars <manifest>
# Parse the env_vars array from plugin.json (Bash 3.2 compatible, no jq required).
# Outputs one line per var: NAME|DESCRIPTION|REQUIRED|SECRET|EXAMPLE
# where REQUIRED and SECRET are "true"/"false" strings.
_plugin_config_parse_env_vars() {
  local manifest="$1"

  if command -v jq >/dev/null 2>&1; then
    # jq path: emit pipe-delimited records
    jq -r '
      .env_vars[]? |
      [
        (.name // ""),
        (.description // ""),
        (if .required then "true" else "false" end),
        (if .secret then "true" else "false" end),
        (.example // "")
      ] | join("|")
    ' "$manifest" 2>/dev/null
    return
  fi

  # No jq fallback: hand-parse the env_vars array (handles compact + multi-line JSON).
  # This is intentionally simple — we only care about 5 known string/bool keys per object.
  local in_env_vars=false in_obj=false
  local name="" description="" required="false" secret="false" example=""

  while IFS= read -r line; do
    # Start of env_vars array
    case "$line" in
      *'"env_vars"'*'['*)
        in_env_vars=true
        continue
        ;;
    esac

    [[ "$in_env_vars" != "true" ]] && continue

    # End of env_vars array
    case "$line" in
      *']'*)
        # Flush any open object before exiting
        if [[ "$in_obj" == "true" && -n "$name" ]]; then
          printf '%s|%s|%s|%s|%s\n' "$name" "$description" "$required" "$secret" "$example"
        fi
        in_env_vars=false
        in_obj=false
        name=""; description=""; required="false"; secret="false"; example=""
        continue
        ;;
    esac

    # Object boundaries
    case "$line" in
      *'{'*)
        in_obj=true
        name=""; description=""; required="false"; secret="false"; example=""
        ;;
      *'}'*)
        if [[ "$in_obj" == "true" && -n "$name" ]]; then
          printf '%s|%s|%s|%s|%s\n' "$name" "$description" "$required" "$secret" "$example"
        fi
        in_obj=false
        name=""; description=""; required="false"; secret="false"; example=""
        ;;
    esac

    [[ "$in_obj" != "true" ]] && continue

    # Extract field values
    case "$line" in
      *'"name"'*)
        name=$(printf '%s' "$line" | sed 's/.*"name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
        ;;
      *'"description"'*)
        description=$(printf '%s' "$line" | sed 's/.*"description"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
        ;;
      *'"example"'*)
        example=$(printf '%s' "$line" | sed 's/.*"example"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
        ;;
      *'"required"'*'true'*)
        required="true"
        ;;
      *'"required"'*'false'*)
        required="false"
        ;;
      *'"secret"'*'true'*)
        secret="true"
        ;;
      *'"secret"'*'false'*)
        secret="false"
        ;;
    esac
  done < "$manifest"
}

# cmd_plugin_config — interactive env var setup for a plugin
#
# Usage:
#   nself plugin config <name>          Interactive prompts for required env vars
#   nself plugin config <name> --show   Show current values (secrets masked as ****)
#   nself plugin config <name> --reset  Clear plugin-specific vars from .env.local
#
# Reads env_vars from ~/.nself/plugins/<name>/plugin.json.
# Writes interactive results to .env.local in the current project directory.
# Bash 3.2 compatible — no declare -A, no ${var,,}, no echo -e, no mapfile.
cmd_plugin_config() {
  local plugin_name="${1:-}"
  shift || true

  # Help
  case "$plugin_name" in
    --help|-h|"")
      printf "Usage: nself plugin config <name> [options]\n\n"
      printf "Interactively configure required environment variables for a plugin.\n\n"
      printf "Arguments:\n"
      printf "  name         Plugin name (must be installed)\n\n"
      printf "Options:\n"
      printf "  --show       Show current values (secrets masked as ****)\n"
      printf "  --reset      Remove plugin vars from .env.local\n"
      printf "  --help, -h   Show this help text\n\n"
      printf "Examples:\n"
      printf "  nself plugin config ai\n"
      printf "  nself plugin config ai --show\n"
      printf "  nself plugin config ai --reset\n"
      return 0
      ;;
  esac

  # Parse mode flag
  local mode="interactive"
  for _cfg_arg in "$@"; do
    case "$_cfg_arg" in
      --show)   mode="show" ;;
      --reset)  mode="reset" ;;
      --help|-h)
        printf "Usage: nself plugin config <name> [--show|--reset]\n"
        return 0
        ;;
    esac
  done
  unset _cfg_arg

  if ! is_plugin_installed "$plugin_name" 2>/dev/null; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
  if [[ ! -f "$manifest" ]]; then
    log_error "Plugin manifest not found: $manifest"
    return 1
  fi

  # Derive plugin name prefix for env vars: PLUGIN_<UPPER_NAME>_*
  local upper_name
  upper_name=$(printf '%s' "$plugin_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')

  # Parse env_vars from manifest
  local env_vars_raw
  env_vars_raw=$(_plugin_config_parse_env_vars "$manifest")

  if [[ -z "$env_vars_raw" ]]; then
    log_info "Plugin '$plugin_name' does not declare any env_vars in its manifest."
    printf "You may still set variables manually in .env or .env.local.\n"
    return 0
  fi

  # ── --show mode ────────────────────────────────────────────────────────────
  if [[ "$mode" == "show" ]]; then
    printf "\n=== Config: %s ===\n\n" "$plugin_name"
    printf "%-40s  %-8s  %s\n" "VARIABLE" "REQUIRED" "VALUE"
    printf "%-40s  %-8s  %s\n" "--------" "--------" "-----"
    while IFS='|' read -r var_name var_desc var_required var_secret var_example; do
      [[ -z "$var_name" ]] && continue
      local current_val
      current_val=$(_plugin_config_read_var "$var_name")
      local display_val
      if [[ "$var_secret" == "true" && -n "$current_val" ]]; then
        display_val="****"
      elif [[ -z "$current_val" ]]; then
        display_val="(not set)"
      else
        display_val="$current_val"
      fi
      printf "%-40s  %-8s  %s\n" "$var_name" "$var_required" "$display_val"
    done <<EOF
$env_vars_raw
EOF
    printf "\n"
    return 0
  fi

  # ── --reset mode ───────────────────────────────────────────────────────────
  if [[ "$mode" == "reset" ]]; then
    local env_local=".env.local"
    if [[ ! -f "$env_local" ]]; then
      log_info "No .env.local found — nothing to reset."
      return 0
    fi
    local tmp_file
    tmp_file=$(mktemp)
    # Remove all lines matching PLUGIN_<UPPER_NAME>_* vars declared in manifest
    local removed=0
    while IFS= read -r fline; do
      local keep=true
      while IFS='|' read -r var_name var_desc var_required var_secret var_example; do
        [[ -z "$var_name" ]] && continue
        case "$fline" in
          "${var_name}="*)
            keep=false
            removed=$((removed + 1))
            break
            ;;
        esac
      done <<EOF2
$env_vars_raw
EOF2
      [[ "$keep" == "true" ]] && printf '%s\n' "$fline" >> "$tmp_file"
    done < "$env_local"
    mv "$tmp_file" "$env_local"
    if [[ $removed -gt 0 ]]; then
      log_success "Removed $removed variable(s) for '$plugin_name' from .env.local."
      printf "Run 'nself build' to apply.\n"
    else
      log_info "No variables for '$plugin_name' found in .env.local."
    fi
    return 0
  fi

  # ── Interactive mode ───────────────────────────────────────────────────────
  # Must be inside a project directory (has .env) so we know where to write
  if [[ ! -f ".env" ]]; then
    log_error "Not in a project directory (no .env found)."
    printf "Run this command from your nself project root.\n"
    return 1
  fi

  printf "\nConfiguring env vars for plugin '%s'\n" "$plugin_name"
  printf "Saves to .env.local — press Enter to keep the current value.\n\n"

  local wrote=0
  while IFS='|' read -r var_name var_desc var_required var_secret var_example; do
    [[ -z "$var_name" ]] && continue

    # Read current value for display in prompt
    local current_val
    current_val=$(_plugin_config_read_var "$var_name")

    # Build prompt line
    local prompt_current=""
    if [[ -n "$current_val" ]]; then
      if [[ "$var_secret" == "true" ]]; then
        prompt_current="current: ****"
      else
        prompt_current="current: $current_val"
      fi
    elif [[ -n "$var_example" ]]; then
      prompt_current="example: $var_example"
    fi

    local prompt_label="$var_name"
    [[ -n "$var_desc" ]] && prompt_label="$var_name ($var_desc)"
    [[ "$var_required" == "true" ]] && prompt_label="${prompt_label} [required]"

    local new_val=""
    if [[ "$var_secret" == "true" ]]; then
      # Masked input for secret vars
      if [[ -n "$prompt_current" ]]; then
        printf '%s [%s]: ' "$prompt_label" "$prompt_current"
      else
        printf '%s: ' "$prompt_label"
      fi
      stty -echo 2>/dev/null || true
      read -r new_val
      stty echo 2>/dev/null || true
      printf '\n'
    else
      if [[ -n "$prompt_current" ]]; then
        printf '%s [%s]: ' "$prompt_label" "$prompt_current"
      else
        printf '%s: ' "$prompt_label"
      fi
      read -r new_val
    fi

    # Empty input: keep existing value, skip write
    if [[ -z "$new_val" ]]; then
      if [[ "$var_required" == "true" && -z "$current_val" ]]; then
        log_warning "$var_name is required but was left empty."
      fi
      continue
    fi

    # Validate URL-type vars (ending in _URL or _ENDPOINT)
    case "$var_name" in
      *_URL|*_ENDPOINT)
        case "$new_val" in
          http://*|https://*)
            ;;
          *)
            log_warning "$var_name should be a URL (http:// or https://). Saving anyway."
            ;;
        esac
        ;;
    esac

    _plugin_config_upsert_env_local "$var_name" "$new_val"
    wrote=$((wrote + 1))
  done <<EOF3
$env_vars_raw
EOF3

  if [[ $wrote -gt 0 ]]; then
    printf "\nSaved %d variable(s) to .env.local. Run 'nself build' to apply.\n" "$wrote"
  else
    log_info "No changes made."
  fi
}

# ============================================================================
# LICENSE MANAGEMENT
# ============================================================================

# Manage the Pro Plugins license key
cmd_plugin_license() {
  local subcmd="${1:-show}"
  shift || true

  case "$subcmd" in
    show | status)
      license_show_status
      ;;

    set)
      local key="${1:-}"
      if [[ -z "$key" ]]; then
        log_error "Usage: nself plugin license set <key>"
        printf "Keys start with 'nself_pro_' — get one at: %s\n" "${NSELF_PRICING_URL:-https://nself.org/pricing}"
        return 1
      fi
      if ! license_validate_format "$key"; then
        log_error "Invalid license key format."
        printf "Key must start with 'nself_pro_' and be at least 32 characters.\n"
        return 1
      fi
      license_save_key "$key"
      log_success "License key saved to ~/.nself/license/key"
      printf "Run 'nself plugin license validate' to verify with the server.\n"
      ;;

    clear | remove)
      license_clear_key
      log_success "License key removed."
      ;;

    validate)
      local license_key
      license_key=$(license_get_key) || true
      if [[ -z "$license_key" ]]; then
        log_error "No license key configured."
        printf "Set one with: nself plugin license set nself_pro_...\n"
        printf "Get a license at: %s\n" "${NSELF_PRICING_URL:-https://nself.org/pricing}"
        return 1
      fi
      if ! license_validate_format "$license_key"; then
        log_error "Invalid license key format."
        printf "Key must start with 'nself_pro_' and be at least 32 characters.\n"
        return 1
      fi
      log_info "Validating license against server..."
      if license_validate_remote "$license_key"; then
        log_success "License is valid."
      else
        log_error "License validation failed. Check or renew at: ${NSELF_PRICING_URL:-https://nself.org/pricing}"
        return 1
      fi
      ;;

    plugins | list)
      printf "\nPro Plugins (require license):\n\n"
      for plugin_name in $NSELF_PRO_PLUGINS; do
        printf "  %s\n" "$plugin_name"
      done
      printf "\nTotal: $(printf '%s' "$NSELF_PRO_PLUGINS" | wc -w | tr -d ' ') pro plugins\n"
      printf "Details: %s\n\n" "${NSELF_PRICING_URL:-https://nself.org/pricing}"
      ;;

    help | --help | -h)
      printf "Usage: nself plugin license <subcommand>\n\n"
      printf "Subcommands:\n"
      printf "  set <key>  Save your Pro Plugins license key\n"
      printf "  clear      Remove saved license key\n"
      printf "  show       Show current license key and status (default)\n"
      printf "  validate   Force-validate license key against the API\n"
      printf "  plugins    List all Pro Plugins covered by a license\n"
      printf "\nQuick start:\n"
      printf "  nself plugin license set nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n"
      printf "  nself plugin install analytics\n"
      printf "\nOr add to your .env:\n"
      printf "  NSELF_PLUGIN_LICENSE_KEY=nself_pro_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx\n"
      printf "\nGet a license at: %s\n\n" "${NSELF_PRICING_URL:-https://nself.org/pricing}"
      ;;

    *)
      log_error "Unknown subcommand: $subcmd"
      printf "Run 'nself plugin license help' for usage.\n"
      return 1
      ;;
  esac
}

# ============================================================================
# T-0250: cmd_plugin_list — installed plugins with version + health
# ============================================================================

cmd_plugin_list() {
  local show_json=false
  local show_all=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --help|-h)
        printf "Usage: nself plugin plugin-list [options]\n\n"
        printf "List installed plugins with version, port, and health status.\n\n"
        printf "Options:\n"
        printf "  --all, -a      Include uninstalled plugins from the registry\n"
        printf "  --json         Output parseable JSON\n"
        printf "  --help, -h     Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin plugin-list\n"
        printf "  nself plugin plugin-list --all\n"
        printf "  nself plugin plugin-list --json\n"
        return 0
        ;;
      --json)
        show_json=true
        shift
        ;;
      --all|-a)
        show_all=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  # Collect running container names once (Bash 3.2 compatible)
  local running_containers=""
  if command -v docker >/dev/null 2>&1; then
    running_containers=$(docker ps --format "{{.Names}}" 2>/dev/null || true)
  fi

  if [[ "$show_json" == "true" ]]; then
    # JSON output
    local first=true
    printf '[\n'
    for plugin_dir in "$PLUGIN_DIR"/*/; do
      [[ -d "$plugin_dir" ]] || continue
      local pname
      pname=$(basename "$plugin_dir")
      [[ "$pname" == "_shared" ]] && continue
      [[ -f "$plugin_dir/plugin.json" ]] || continue

      local version port
      version=$(grep '"version"' "$plugin_dir/plugin.json" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
      port=$(grep '"port"' "$plugin_dir/plugin.json" | head -1 | sed 's/[^0-9]//g')
      version="${version:-unknown}"
      port="${port:-}"

      local container_name="nself-${pname}"
      local healthy="false"
      if printf '%s\n' "$running_containers" | grep -qx "$container_name" 2>/dev/null; then
        healthy="true"
      fi

      [[ "$first" == "true" ]] || printf ',\n'
      first=false
      printf '  {"name":"%s","version":"%s","port":"%s","healthy":%s}' \
        "$pname" "$version" "$port" "$healthy"
    done
    printf '\n]\n'
    return 0
  fi

  # Human-readable table
  local GREEN="\033[0;32m"
  local RED="\033[0;31m"
  local RESET="\033[0m"

  printf "\n=== Installed Plugins ===\n\n"
  printf "%-22s %-10s %-6s %-8s\n" "NAME" "VERSION" "PORT" "HEALTH"
  printf "%-22s %-10s %-6s %-8s\n" "----" "-------" "----" "------"

  local found=0
  for plugin_dir in "$PLUGIN_DIR"/*/; do
    [[ -d "$plugin_dir" ]] || continue
    local pname
    pname=$(basename "$plugin_dir")
    [[ "$pname" == "_shared" ]] && continue
    [[ -f "$plugin_dir/plugin.json" ]] || continue

    local version port
    version=$(grep '"version"' "$plugin_dir/plugin.json" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    port=$(grep '"port"' "$plugin_dir/plugin.json" | head -1 | sed 's/[^0-9]//g')
    version="${version:-unknown}"
    port="${port:--}"

    local container_name="nself-${pname}"
    local health_sym health_label
    if printf '%s\n' "$running_containers" | grep -qx "$container_name" 2>/dev/null; then
      health_sym="${GREEN}✓${RESET}"
      health_label="healthy"
    else
      health_sym="${RED}✗${RESET}"
      health_label="stopped"
    fi

    printf "%-22s %-10s %-6s " "$pname" "$version" "$port"
    printf "${health_sym} %s\n" "$health_label"
    found=$((found + 1))
  done

  if [[ $found -eq 0 ]]; then
    printf "  (no plugins installed)\n"
    printf "\nInstall with: nself plugin install <name>\n\n"
    return 0
  fi

  # Show uninstalled registry plugins if --all
  if [[ "$show_all" == "true" ]]; then
    local registry
    registry=$(fetch_registry 2>/dev/null || true)
    if [[ -n "$registry" ]]; then
      printf "\n--- Available (not installed) ---\n"
      local free_plugins
      free_plugins=$(_registry_plugin_names "$registry")
      for rp in $free_plugins; do
        if ! is_plugin_installed "$rp"; then
          printf "%-22s %-10s %-6s %s\n" "$rp" "-" "-" "(not installed)"
        fi
      done
    fi
  fi

  printf "\n  %d plugin(s) installed\n\n" "$found"
}

# ============================================================================
# T-0251: cmd_plugin_info — description, env vars, tier, deps
# ============================================================================

cmd_plugin_info() {
  local plugin_name="${1:-}"

  case "$plugin_name" in
    --help|-h|"")
      printf "Usage: nself plugin info <name>\n\n"
      printf "Show detailed information about a plugin.\n\n"
      printf "Arguments:\n"
      printf "  name    Plugin name (installed or in registry)\n\n"
      printf "Examples:\n"
      printf "  nself plugin info ai\n"
      printf "  nself plugin info notify\n"
      return 0
      ;;
  esac

  local plugin_dir="$PLUGIN_DIR/$plugin_name"
  local manifest="$plugin_dir/plugin.json"

  if [[ ! -f "$manifest" ]]; then
    log_error "Plugin '$plugin_name' is not installed. Install it first: nself plugin install $plugin_name"
    return 1
  fi

  # Extract fields — jq preferred, sed fallback (Bash 3.2 compatible)
  local description version tier docker_image
  if command -v jq >/dev/null 2>&1; then
    description=$(jq -r '.description // "(no description)"' "$manifest" 2>/dev/null)
    version=$(jq -r '.version // "unknown"' "$manifest" 2>/dev/null)
    docker_image=$(jq -r '.image // ""' "$manifest" 2>/dev/null)
    tier="Free"
    if declare -f license_is_paid_plugin >/dev/null 2>&1 && license_is_paid_plugin "$plugin_name"; then
      tier="Pro"
    fi
  else
    description=$(grep '"description"' "$manifest" | head -1 | sed 's/.*"description"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    version=$(grep '"version"' "$manifest" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    docker_image=$(grep '"image"' "$manifest" | head -1 | sed 's/.*"image"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
    tier="Free"
    if declare -f license_is_paid_plugin >/dev/null 2>&1 && license_is_paid_plugin "$plugin_name"; then
      tier="Pro"
    fi
  fi

  description="${description:-(no description)}"
  version="${version:-unknown}"

  printf "\n=== Plugin: %s ===\n\n" "$plugin_name"
  printf "  Description:  %s\n" "$description"
  printf "  Version:      %s\n" "$version"
  printf "  Tier:         %s\n" "$tier"
  [[ -n "$docker_image" ]] && printf "  Docker image: %s\n" "$docker_image"

  # Dependencies
  local deps
  deps=$(_plugin_deps "$plugin_name")
  if [[ -n "$deps" ]]; then
    printf "\n  Dependencies:\n"
    for dep in $deps; do
      [[ -z "$dep" ]] && continue
      if is_plugin_installed "$dep"; then
        printf "    %-20s (installed)\n" "$dep"
      else
        printf "    %-20s (not installed)\n" "$dep"
      fi
    done
  else
    printf "\n  Dependencies:  none\n"
  fi

  # Required env vars
  if command -v jq >/dev/null 2>&1; then
    local req_vars
    req_vars=$(jq -r '.envVars.required[]? // empty' "$manifest" 2>/dev/null)
    if [[ -n "$req_vars" ]]; then
      printf "\n  Required environment variables:\n"
      printf '%s\n' "$req_vars" | while IFS= read -r var; do
        [[ -z "$var" ]] && continue
        printf "    %s=\n" "$var"
      done
    fi

    local opt_vars
    opt_vars=$(jq -r '.envVars.optional[]? // empty' "$manifest" 2>/dev/null)
    if [[ -n "$opt_vars" ]]; then
      printf "\n  Optional environment variables:\n"
      printf '%s\n' "$opt_vars" | while IFS= read -r var; do
        [[ -z "$var" ]] && continue
        printf "    # %s=\n" "$var"
      done
    fi
  else
    # Fallback: grep for known patterns
    local req_section
    req_section=$(grep -A20 '"required"' "$manifest" | grep '"[A-Z_][A-Z0-9_]*"' | sed 's/.*"\([A-Z_][A-Z0-9_]*\)".*/\1/' | grep -v '{' || true)
    if [[ -n "$req_section" ]]; then
      printf "\n  Required environment variables:\n"
      printf '%s\n' "$req_section" | while IFS= read -r var; do
        [[ -z "$var" ]] && continue
        printf "    %s=\n" "$var"
      done
    fi
  fi

  printf "\n  Path: %s\n\n" "$plugin_dir"
}

# ============================================================================
# T-0254: _check_plugin_prerequisites — dep check before install
# ============================================================================

# _check_plugin_prerequisites <plugin_name>
# Called before installing a plugin. Checks declared dependencies.
# Hard deps: prompts user to install or aborts. Soft deps: warning only.
# Returns 0 if all hard deps satisfied or user confirmed, 1 to abort.
_check_plugin_prerequisites() {
  local plugin_name="$1"

  # Known hard dependency map (Bash 3.2 — no declare -A)
  # Format: space-separated list for each plugin
  local hard_deps=""
  case "$plugin_name" in
    claw)    hard_deps="ai mux" ;;
    mux)     hard_deps="ai" ;;
    voice)   hard_deps="notify" ;;
    cron)    hard_deps="redis" ;;
    browser) hard_deps="ai" ;;
    *)       hard_deps="" ;;
  esac

  # Also read from installed/downloaded plugin.json when available
  local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
  if [[ -f "$manifest" ]]; then
    local manifest_deps
    manifest_deps=$(_plugin_deps "$plugin_name")
    if [[ -n "$manifest_deps" ]]; then
      hard_deps="$manifest_deps"
    fi
  fi

  if [[ -z "$hard_deps" ]]; then
    return 0
  fi

  local missing_deps=""
  for dep in $hard_deps; do
    [[ -z "$dep" ]] && continue
    if ! is_plugin_installed "$dep"; then
      missing_deps="$missing_deps $dep"
    fi
  done
  missing_deps="${missing_deps# }"

  if [[ -z "$missing_deps" ]]; then
    return 0
  fi

  for dep in $missing_deps; do
    [[ -z "$dep" ]] && continue
    printf "\n"
    log_warning "nself-$plugin_name requires nself-$dep, which is not installed."
    printf "Install it first? [y/N] "
    local answer=""
    read -r answer
    case "$answer" in
      [yY]|[yY][eE][sS])
        log_info "Installing prerequisite: $dep"
        if ! cmd_install "$dep"; then
          log_error "Failed to install prerequisite '$dep'. Aborting."
          return 1
        fi
        ;;
      *)
        log_error "Cannot install '$plugin_name' without '$dep'. Install it first:"
        printf "  nself plugin install %s\n\n" "$dep"
        return 1
        ;;
    esac
  done

  return 0
}

# ============================================================================
# T-0255: _check_port_conflicts — port conflict detection
# ============================================================================

# _check_port_conflicts <plugin_name> [port]
# Called before nself plugin install or start.
# Reads port from plugin.json when port arg is not provided.
# Returns 0 if port is free, 1 if conflict detected.
_check_port_conflicts() {
  local plugin_name="$1"
  local port="${2:-}"

  # Resolve port from plugin.json if not provided
  if [[ -z "$port" ]]; then
    local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
    if [[ -f "$manifest" ]]; then
      port=$(grep '"port"' "$manifest" | head -1 | sed 's/[^0-9]//g')
    fi
  fi

  if [[ -z "$port" ]]; then
    # No port configured — nothing to check
    return 0
  fi

  # Get list of listening ports (Bash 3.2 compatible — ss preferred, netstat fallback)
  local listening_ports=""
  if command -v ss >/dev/null 2>&1; then
    listening_ports=$(ss -tlnp 2>/dev/null | grep "LISTEN" | grep -o ':[0-9]*' | tr -d ':' || true)
  elif command -v netstat >/dev/null 2>&1; then
    listening_ports=$(netstat -tlnp 2>/dev/null | grep "LISTEN" | grep -o ':[0-9]*' | tr -d ':' || true)
  else
    # Cannot check — warn but do not block
    log_warning "Cannot check port conflicts: neither ss nor netstat found."
    return 0
  fi

  if printf '%s\n' "$listening_ports" | grep -qx "$port" 2>/dev/null; then
    local upper_name
    upper_name=$(printf '%s' "$plugin_name" | tr '[:lower:]' '[:upper:]' | tr '-' '_')
    log_error "Port $port is already in use."
    printf "Set %s_PORT=%s (different value) in .env or stop the conflicting process.\n" \
      "$upper_name" "$port"
    return 1
  fi

  return 0
}

# ============================================================================
# T-0257: cmd_plugin_status — unified health dashboard
# ============================================================================

cmd_plugin_status() {
  local watch_mode=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --help|-h)
        printf "Usage: nself plugin plugin-status [options]\n\n"
        printf "Show a unified health dashboard for all installed plugins.\n\n"
        printf "Options:\n"
        printf "  --watch        Refresh every 5 seconds (clear + redraw)\n"
        printf "  --help, -h     Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin plugin-status\n"
        printf "  nself plugin plugin-status --watch\n"
        return 0
        ;;
      --watch|-w)
        watch_mode=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  _render_plugin_status_dashboard() {
    local GREEN="\033[0;32m"
    local RED="\033[0;31m"
    local YELLOW="\033[0;33m"
    local RESET="\033[0m"

    local healthy_count=0
    local total_count=0

    # Collect running containers once
    local running_containers=""
    if command -v docker >/dev/null 2>&1; then
      running_containers=$(docker ps --format "{{.Names}}" 2>/dev/null || true)
    fi

    printf "\n=== Plugin Health Dashboard ===\n\n"
    printf "%-22s %-10s %-6s %-10s %-20s %-20s\n" \
      "NAME" "VERSION" "PORT" "HEALTH" "CPU" "MEMORY"
    printf "%-22s %-10s %-6s %-10s %-20s %-20s\n" \
      "----" "-------" "----" "------" "---" "------"

    for plugin_dir in "$PLUGIN_DIR"/*/; do
      [[ -d "$plugin_dir" ]] || continue
      local pname
      pname=$(basename "$plugin_dir")
      [[ "$pname" == "_shared" ]] && continue
      [[ -f "$plugin_dir/plugin.json" ]] || continue

      total_count=$((total_count + 1))

      local version port
      version=$(grep '"version"' "$plugin_dir/plugin.json" | head -1 | sed 's/.*"version"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/')
      port=$(grep '"port"' "$plugin_dir/plugin.json" | head -1 | sed 's/[^0-9]//g')
      version="${version:-unknown}"
      port="${port:--}"

      local container_name="nself-${pname}"
      local health_label cpu_pct mem_usage last_restart
      cpu_pct="-"
      mem_usage="-"
      last_restart="-"

      if printf '%s\n' "$running_containers" | grep -qx "$container_name" 2>/dev/null; then
        health_label="${GREEN}healthy${RESET}"
        healthy_count=$((healthy_count + 1))
        # Get resource stats (non-streaming — single snapshot)
        if command -v docker >/dev/null 2>&1; then
          local stats_line
          stats_line=$(docker stats --no-stream --format "{{.CPUPerc}} {{.MemUsage}}" \
            "$container_name" 2>/dev/null | head -1 || true)
          if [[ -n "$stats_line" ]]; then
            cpu_pct=$(printf '%s' "$stats_line" | awk '{print $1}')
            mem_usage=$(printf '%s' "$stats_line" | awk '{print $2}')
          fi
          # Last restart time from docker inspect
          last_restart=$(docker inspect --format "{{.State.StartedAt}}" \
            "$container_name" 2>/dev/null | cut -c1-16 | tr 'T' ' ' || true)
          last_restart="${last_restart:--}"
        fi
      else
        health_label="${RED}stopped${RESET}"
      fi

      printf "%-22s %-10s %-6s " "$pname" "$version" "$port"
      printf "${health_label}"
      printf "%-$((10 - ${#health_label} + 31))s" ""
      printf "%-20s %-20s\n" "$cpu_pct" "$mem_usage"
    done

    if [[ $total_count -eq 0 ]]; then
      printf "  (no plugins installed)\n"
      printf "\nInstall with: nself plugin install <name>\n\n"
      return 0
    fi

    printf "\n  %d/%d plugins healthy\n\n" "$healthy_count" "$total_count"
  }

  if [[ "$watch_mode" == "true" ]]; then
    while true; do
      clear
      _render_plugin_status_dashboard
      printf "  Refreshing every 5s — Ctrl+C to exit\n\n"
      sleep 5
    done
  else
    _render_plugin_status_dashboard
  fi
}

# ============================================================================
# T-0272: cmd_channel — canary release channel management
# ============================================================================

# cmd_channel — manage release channels for installed Rust/Docker plugins
#
# Usage:
#   nself plugin channel <name> <stable|canary|beta>
#       Switch plugin to the given channel. Writes channel to
#       ~/.nself/plugins/<name>/.channel, re-pulls the GHCR image for the new
#       channel, rebuilds docker-compose.yml, and restarts the plugin container.
#
#   nself plugin channel --list
#       List the current release channel for every installed plugin.
#
# Bash 3.2 compatible — no declare -A, no ${var,,}, no echo -e, no mapfile.
cmd_channel() {
  # Handle --list / -l before positional arg parsing
  case "${1:-}" in
    --list|-l)
      printf "\n=== Plugin Release Channels ===\n\n"
      printf "%-22s %s\n" "PLUGIN" "CHANNEL"
      printf "%-22s %s\n" "------" "-------"
      local found=0
      for plugin_dir in "$PLUGIN_DIR"/*/; do
        [[ -d "$plugin_dir" ]] || continue
        local pname
        pname=$(basename "$plugin_dir")
        [[ "$pname" == "_shared" ]] && continue
        [[ -f "$plugin_dir/plugin.json" ]] || continue
        local ch="stable"
        if [[ -f "$plugin_dir/.channel" ]]; then
          ch=$(cat "$plugin_dir/.channel" 2>/dev/null)
          ch="${ch:-stable}"
        fi
        printf "%-22s %s\n" "$pname" "$ch"
        found=$((found + 1))
      done
      if [[ $found -eq 0 ]]; then
        log_info "No plugins installed"
        printf "\nInstall with: nself plugin install <name>\n"
      fi
      printf "\n"
      return 0
      ;;
    --help|-h|"")
      printf "Usage: nself plugin channel <name> <stable|canary|beta>\n"
      printf "       nself plugin channel --list\n\n"
      printf "Switch a plugin's release channel or list current channels.\n\n"
      printf "Subcommands:\n"
      printf "  <name> <channel>   Switch plugin to channel (stable, canary, beta)\n"
      printf "  --list, -l         Show current channel for every installed plugin\n\n"
      printf "Examples:\n"
      printf "  nself plugin channel ai canary\n"
      printf "  nself plugin channel ai stable\n"
      printf "  nself plugin channel --list\n\n"
      return 0
      ;;
  esac

  local plugin_name="${1:-}"
  local new_channel="${2:-}"

  if [[ -z "$plugin_name" ]]; then
    log_error "Plugin name required"
    printf "Usage: nself plugin channel <name> <stable|canary|beta>\n"
    return 1
  fi

  if [[ -z "$new_channel" ]]; then
    log_error "Channel required"
    printf "Valid channels: stable, canary, beta\n"
    printf "Usage: nself plugin channel %s <stable|canary|beta>\n" "$plugin_name"
    return 1
  fi

  # Validate channel
  case "$new_channel" in
    stable|canary|beta) ;;
    *)
      log_error "Invalid channel: $new_channel. Valid channels: stable, canary, beta"
      return 1
      ;;
  esac

  # Plugin must be installed
  if ! is_plugin_installed "$plugin_name"; then
    log_error "Plugin '$plugin_name' is not installed"
    return 1
  fi

  # Only Rust/Docker plugins have GHCR images with channels
  if ! plugin_is_rust "$plugin_name" 2>/dev/null && ! _plugin_is_rust_known "$plugin_name"; then
    log_error "Plugin '$plugin_name' does not use Docker/GHCR images and does not support channels"
    return 1
  fi

  # Read current channel
  local current_channel="stable"
  local channel_file="$PLUGIN_DIR/$plugin_name/.channel"
  if [[ -f "$channel_file" ]]; then
    current_channel=$(cat "$channel_file" 2>/dev/null)
    current_channel="${current_channel:-stable}"
  fi

  if [[ "$current_channel" == "$new_channel" ]]; then
    log_info "Plugin '$plugin_name' is already on channel: $new_channel"
    return 0
  fi

  printf "Switching '%s' from '%s' to '%s'...\n" "$plugin_name" "$current_channel" "$new_channel"

  # Write new channel before pulling so .channel file is always consistent
  printf '%s\n' "$new_channel" > "$channel_file"

  # Pull the new channel image
  log_info "Pulling image for channel: $new_channel"
  if ! _pull_plugin_docker_image "$plugin_name" "" "$new_channel"; then
    log_warning "Could not pull $new_channel image — reverting channel to $current_channel"
    printf '%s\n' "$current_channel" > "$channel_file"
    return 1
  fi

  # Rebuild compose and restart the plugin container
  _plugin_activate_docker_service "$plugin_name"

  log_success "Plugin '$plugin_name' switched to channel: $new_channel"
  printf "\nCheck status: nself plugin plugin-status\n"
}

# ============================================================================
# HELP
# ============================================================================

show_help() {
  printf "
nself plugin - Plugin Management

Usage: nself plugin <command> [options]

Commands:
  list [options]          List available plugins
    --installed, -i         Show only installed plugins
    --detailed, -d          Show detailed status (with --installed)
    --category, -c <cat>    Filter by category (billing, ecommerce, devops)

  plugin-list [options]   List installed plugins with version, port, and health
    --all, -a               Include uninstalled plugins from the registry
    --json                  Output parseable JSON

  install <name>          Install a plugin from registry
  install <path>          Install a local plugin
    --channel <channel>     Release channel: stable (default), canary, beta

  remove <name>           Remove a plugin
    --keep-data             Keep database tables

  update [name]           Update a specific plugin
    --all, -a               Update all installed plugins

  rollback <name> [ver]   Roll back to previous or specific version
    --list                  List available versions from GHCR
    --force                 Skip confirmation prompt

  status [name]           Show plugin status and health

  plugin-status           Unified health dashboard (CPU, memory, uptime)
    --watch                 Refresh every 5 seconds

  updates                 Check for available plugin updates
    --quiet, -q             Output only update info (for scripts)

  refresh                 Force refresh the plugin registry cache

  license [subcommand]    Manage Pro Plugins license
    set <key>               Save your license key persistently
    clear                   Remove saved license key
    show                    Show current license key and status (default)
    validate                Force-validate license key against API
    plugins                 List all 49 Pro Plugins covered by license

  config <name>           Interactive env var setup (reads plugin.json env_vars)
    --show                  Show current values (secrets masked as ****)
    --reset                 Remove plugin vars from .env.local

  channel <name> <ch>     Switch plugin to a release channel (stable, canary, beta)
  channel --list          Show current release channel for every installed plugin

  check-deps <name>       Check system dependencies for a plugin
  install-deps <name>     Install missing system dependencies
    --check-only            Dry run (show what would be installed)
  check-conflicts         Scan all installed plugins for dependency conflicts
    --fix                   Auto-install required versions where possible

  deps <name>             Show declared dependencies for an installed plugin
  deps --all              List all installed plugins and their dependencies

  metrics [name]          Fetch Prometheus metrics from running plugin services
                          Shows all Rust plugins when name is omitted

Remote Deploy (pro plugins — bypasses registry, uses rsync+SSH):
  deploy <name>           Sync source to remote server and rebuild container
    --server <host>         SSH hostname or IP of the target server (required)
    --user <user>           SSH user (default: root)
    --dir <dir>             Remote nself project dir (default: /opt/nself)
    --dry-run               Show what would happen without executing
    --no-restart            Rebuild image but skip container restart

  deploy-all              Deploy all plugins in plugins-pro/paid/ to a remote server
    --server <host>         SSH hostname or IP of the target server (required)

Runtime Management:
  start <name>            Start a plugin as external process
  start --all             Start all installed plugins (respects dependencies)
  stop <name>             Stop a running plugin
    --force                 Force-stop immediately (SIGKILL)
  stop --all              Stop all running plugins
    --force                 Force-stop all immediately
  restart <name>          Restart a plugin
  logs <name>             View plugin logs (Docker or file, color-coded)
    -f, --follow            Follow log output in real time
    --no-follow             Exit after printing (default)
    --lines N, -n N         Number of lines to show (default: 100)
  ps | running            List running plugins
  health                  Health check all running plugins

Plugin Actions:
  <plugin> <action>       Run plugin action (e.g., stripe sync)
  <plugin> --help         Show plugin's available actions

Built-in Plugin Actions:
  <plugin> init           Initialize database schema for the plugin
  <plugin> integrate      Show CS_N service configuration for .env

Examples:
  # Installation & management
  nself plugin list
  nself plugin list --installed
  nself plugin list --installed --detailed   # Show states, PIDs, ports
  nself plugin install stripe
  nself plugin update --all
  nself plugin status

  # Runtime (external processes)
  nself plugin start vpn              # Start single plugin
  nself plugin start --all            # Start all (respects dependencies)
  nself plugin stop vpn               # Stop plugin gracefully
  nself plugin stop vpn --force       # Force-stop immediately
  nself plugin stop --all             # Stop all gracefully
  nself plugin stop --all --force     # Force-stop all immediately
  nself plugin restart vpn            # Restart plugin
  nself plugin logs vpn               # View logs
  nself plugin logs vpn -f            # Follow logs
  nself plugin ps                     # List running
  nself plugin health                 # Health check all

  # Plugin actions
  nself plugin stripe sync
  nself plugin stripe customers list
  nself plugin devices init           # Initialize schema
  nself plugin devices integrate      # Show CS_N config

  # Dependencies
  nself plugin check-deps stripe
  nself plugin install-deps stripe
  nself plugin check-conflicts         # Scan for version conflicts
  nself plugin check-conflicts --fix   # Auto-resolve update-needed conflicts
  nself plugin deps ai                 # Show requires[] for installed ai plugin
  nself plugin deps --all              # Show deps for all installed plugins

  # Metrics
  nself plugin metrics                 # Show metrics for all running Rust plugins
  nself plugin metrics ai              # Show metrics for nself-ai only

  # Release channels (canary/beta/stable — Rust/Docker plugins only)
  nself plugin install ai --channel canary    # Install from canary channel
  nself plugin channel ai canary              # Switch installed plugin to canary
  nself plugin channel ai stable             # Switch back to stable
  nself plugin channel --list                # Show channels for all plugins

  # License
  nself plugin license               # Show license status
  nself plugin license validate      # Force-validate against API
  nself plugin license plugins       # List all Pro Plugins

  # Remote deploy (pro plugins — bypasses registry, pushes source directly)
  nself plugin deploy mux --server 49.13.140.12 --dir /opt/nclaw
  nself plugin deploy ai --server 49.13.140.12 --dry-run
  nself plugin deploy-all --server 49.13.140.12 --dir /opt/nclaw

Plugin Features:
  • Lifecycle states (starting/running/stopping/stopped/failed)
  • Dependency management (automatic startup ordering)
  • Required environment variable validation
  • Progress indicators for builds
  • Plugin URLs in 'nself urls' command
  • PID/log consolidation in ~/.nself/runtime/

Available Plugins:
  stripe    - Payment processing & subscriptions (billing)
  github    - Repository & CI integration (devops)
  shopify   - E-commerce store sync (ecommerce)
  vpn       - VPN management (NordVPN, PIA, Mullvad)
  See full list: nself plugin list

Registry:
  Primary:  https://plugins.nself.org
  Fallback: https://github.com/nself-org/plugins

Environment:
  NSELF_PLUGIN_DIR          Plugin installation directory (~/.nself/plugins)
  NSELF_PLUGIN_REGISTRY     Custom registry URL (default: https://plugins.nself.org)
  NSELF_REGISTRY_CACHE_TTL  Registry cache TTL in seconds (default: 300)
  NSELF_PLUGIN_RUNTIME      Plugin runtime directory (~/.nself/runtime)
  NSELF_PLUGIN_LICENSE_KEY  Pro Plugins license key (nself_pro_...)

"
}

# ============================================================================
# T-0256: _run_ai_setup_wizard — first-run AI provider configuration
# ============================================================================
#
# Runs after `nself plugin install ai` if PLUGIN_AI_DEFAULT_PROVIDER is not set.
# Prompts for provider, API key (masked), tests connection, writes to .env.local.
# Skips silently if already configured.
#
# Bash 3.2 compatible — no echo -e, no ${var,,}, no declare -A, no mapfile
#
_run_ai_setup_wizard() {
  local env_local="${NSELF_ENV_LOCAL:-.env.local}"

  # Skip if already configured
  if grep -q "^PLUGIN_AI_DEFAULT_PROVIDER=" "$env_local" 2>/dev/null; then
    return 0
  fi

  printf "\n"
  printf "┌─────────────────────────────────────────────┐\n"
  printf "│  nself-ai — First Run Setup                  │\n"
  printf "└─────────────────────────────────────────────┘\n\n"
  printf "Which AI provider would you like to use?\n\n"
  printf "  1) OpenAI     (GPT-4o, GPT-4, GPT-3.5)\n"
  printf "  2) Anthropic  (Claude 3 Opus, Sonnet, Haiku)\n"
  printf "  3) Gemini     (Gemini 1.5 Pro, Flash)\n"
  printf "  4) Ollama     (local models — Llama, Mistral, etc.)\n\n"
  printf "Enter choice [1-4]: "
  local choice=""
  read -r choice

  case "$choice" in
    1) _setup_ai_openai ;;
    2) _setup_ai_anthropic ;;
    3) _setup_ai_gemini ;;
    4) _setup_ai_ollama ;;
    *) printf "Skipping setup. Run 'nself plugin config ai' to configure later.\n"; return 0 ;;
  esac
}

# Write a key=value pair to .env.local (append if file exists)
_ai_write_env() {
  local key="$1"
  local val="$2"
  local env_local="${NSELF_ENV_LOCAL:-.env.local}"
  # Remove existing entry for this key first
  if [[ -f "$env_local" ]]; then
    local tmp_file
    tmp_file=$(mktemp)
    grep -v "^${key}=" "$env_local" > "$tmp_file" 2>/dev/null || true
    printf '%s=%s\n' "$key" "$val" >> "$tmp_file"
    # Use cp + rm instead of mv for Bash 3.2 compatibility across filesystems
    cp "$tmp_file" "$env_local"
    rm -f "$tmp_file"
  else
    printf '%s=%s\n' "$key" "$val" > "$env_local"
  fi
}

_setup_ai_openai() {
  local env_local="${NSELF_ENV_LOCAL:-.env.local}"
  printf "\nOpenAI API key (sk-...): "
  local key=""
  read -rs key
  printf "\n"

  if [[ ${#key} -lt 20 ]]; then
    printf "Error: key too short (minimum 20 characters)\n"
    return 1
  fi

  printf "Testing connection... "
  if curl -s https://api.openai.com/v1/models \
       -H "Authorization: Bearer $key" 2>/dev/null | grep -q '"id"'; then
    printf "OK\n"
  else
    printf "FAILED\n"
    printf "Warning: Connection test failed. Key saved anyway — verify at nself.org/docs/plugins/ai\n"
  fi

  _ai_write_env "PLUGIN_AI_DEFAULT_PROVIDER" "openai"
  _ai_write_env "PLUGIN_OPENAI_API_KEY" "$key"
  _ai_write_env "PLUGIN_AI_DEFAULT_MODEL" "gpt-4o"
  printf "Setup complete. Test with: curl https://api.\${NSELF_DOMAIN}/ai/v1/chat/completions\n"
}

_setup_ai_anthropic() {
  local env_local="${NSELF_ENV_LOCAL:-.env.local}"
  printf "\nAnthropic API key: "
  local key=""
  read -rs key
  printf "\n"

  if [[ ${#key} -lt 20 ]]; then
    printf "Error: key too short (minimum 20 characters)\n"
    return 1
  fi

  printf "Testing connection... "
  if curl -s https://api.anthropic.com/v1/models \
       -H "x-api-key: $key" \
       -H "anthropic-version: 2023-06-01" 2>/dev/null | grep -q '"id"'; then
    printf "OK\n"
  else
    printf "FAILED\n"
    printf "Warning: Connection test failed. Key saved anyway — verify at nself.org/docs/plugins/ai\n"
  fi

  _ai_write_env "PLUGIN_AI_DEFAULT_PROVIDER" "anthropic"
  _ai_write_env "PLUGIN_ANTHROPIC_API_KEY" "$key"
  _ai_write_env "PLUGIN_AI_DEFAULT_MODEL" "claude-3-5-sonnet-20241022"
  printf "Setup complete. Test with: curl https://api.\${NSELF_DOMAIN}/ai/v1/chat/completions\n"
}

_setup_ai_gemini() {
  local env_local="${NSELF_ENV_LOCAL:-.env.local}"
  printf "\nGemini API key: "
  local key=""
  read -rs key
  printf "\n"

  if [[ ${#key} -lt 20 ]]; then
    printf "Error: key too short (minimum 20 characters)\n"
    return 1
  fi

  printf "Testing connection... "
  if curl -s "https://generativelanguage.googleapis.com/v1beta/models?key=${key}" \
       2>/dev/null | grep -q '"name"'; then
    printf "OK\n"
  else
    printf "FAILED\n"
    printf "Warning: Connection test failed. Key saved anyway — verify at nself.org/docs/plugins/ai\n"
  fi

  _ai_write_env "PLUGIN_AI_DEFAULT_PROVIDER" "gemini"
  _ai_write_env "PLUGIN_GEMINI_API_KEY" "$key"
  _ai_write_env "PLUGIN_AI_DEFAULT_MODEL" "gemini-1.5-pro"
  printf "Setup complete. Test with: curl https://api.\${NSELF_DOMAIN}/ai/v1/chat/completions\n"
}

_setup_ai_ollama() {
  local env_local="${NSELF_ENV_LOCAL:-.env.local}"
  printf "\nOllama URL [http://localhost:11434]: "
  local url=""
  read -r url
  url="${url:-http://localhost:11434}"

  printf "Model name [llama3.2]: "
  local model=""
  read -r model
  model="${model:-llama3.2}"

  printf "Testing connection... "
  if curl -s "${url}/api/tags" 2>/dev/null | grep -q '"models"'; then
    printf "OK\n"
  else
    printf "FAILED\n"
    printf "Warning: Connection test failed. Settings saved anyway — verify at nself.org/docs/plugins/ai\n"
  fi

  _ai_write_env "PLUGIN_AI_DEFAULT_PROVIDER" "ollama"
  _ai_write_env "PLUGIN_OLLAMA_URL" "$url"
  _ai_write_env "PLUGIN_AI_DEFAULT_MODEL" "$model"
  printf "Setup complete. Test with: curl https://api.\${NSELF_DOMAIN}/ai/v1/chat/completions\n"
}

# ============================================================================
# T-1366: cmd_plugin_migrate_schema — move np_{name}_* tables from public to
# the isolated np_{name} schema.
# Bash 3.2 compatible.
# ============================================================================
#
# Usage:
#   nself plugin migrate-schema <name>
#   nself plugin migrate-schema <name> --dry-run
#   nself plugin migrate-schema <name> --rollback
#
# What it does:
#   For each table in public matching the pattern np_{name}_%:
#     1. ALTER TABLE public.np_{name}_{suffix} SET SCHEMA np_{name}
#     2. ALTER TABLE np_{name}.np_{name}_{suffix} RENAME TO {suffix}
#      (drops the redundant np_{name}_ prefix inside the schema)
#   Postgres automatically updates FKs and indexes on schema move.
#
# --rollback reverses: moves tables back to public with the np_{name}_ prefix
#   restored.
#
# nself doctor calls check_plugin_schema_isolation() from schema-isolation.sh
# to warn when public tables are found.

cmd_plugin_migrate_schema() {
  local plugin_name=""
  local dry_run=false
  local rollback=false

  for _arg in "$@"; do
    case "$_arg" in
      --help|-h)
        printf "Usage: nself plugin migrate-schema <name> [--dry-run] [--rollback]\n\n"
        printf "Move plugin tables from public schema into the isolated np_{name} schema.\n\n"
        printf "Arguments:\n"
        printf "  name         Plugin name (e.g. claw, stripe, notify)\n\n"
        printf "Options:\n"
        printf "  --dry-run    Print what would be moved without executing\n"
        printf "  --rollback   Move tables back to public with prefix restored\n"
        printf "  --help, -h   Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin migrate-schema claw\n"
        printf "  nself plugin migrate-schema claw --dry-run\n"
        printf "  nself plugin migrate-schema claw --rollback\n\n"
        printf "What it does:\n"
        printf "  Moves each public.np_{name}_{suffix} table into the np_{name} schema,\n"
        printf "  renaming it to {suffix} (drops the now-redundant prefix).\n"
        printf "  FKs and indexes are updated automatically by Postgres.\n"
        return 0
        ;;
      --dry-run)
        dry_run=true
        ;;
      --rollback)
        rollback=true
        ;;
      -*)
        ;;
      *)
        if [[ -z "$plugin_name" ]]; then
          plugin_name="$_arg"
        fi
        ;;
    esac
  done

  if [[ -z "$plugin_name" ]]; then
    log_error "Plugin name required"
    printf "\nUsage: nself plugin migrate-schema <name>\n"
    return 1
  fi

  # Build schema and role names
  local schema_name
  schema_name="np_$(printf '%s' "$plugin_name" | tr '[:upper:]' '[:lower:]' | tr '-' '_')"
  local table_prefix="${schema_name}_"

  # Load environment for DB connection details
  if declare -f load_env_with_priority >/dev/null 2>&1; then
    load_env_with_priority true 2>/dev/null || true
  elif [[ -f ".env" ]]; then
    set -a
    { set +u; source ".env"; } 2>/dev/null || true
    set +a
  fi

  local project_name="${PROJECT_NAME:-nself}"
  local db_container="${project_name}_postgres"
  local db_user="${POSTGRES_USER:-postgres}"
  local db_name="${POSTGRES_DB:-${project_name}}"

  # Check container
  if ! docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${db_container}$"; then
    log_error "Database container not running: $db_container"
    log_info "Start services with: nself start"
    return 1
  fi

  if [[ "$rollback" == "true" ]]; then
    # ── ROLLBACK: move tables from np_{name} back to public with prefix ─────
    printf "\n"
    log_info "Rolling back schema isolation for plugin: $plugin_name"
    printf "  Moving tables from np_%s back to public with prefix np_%s_\n" \
      "$(printf '%s' "$plugin_name" | tr '[:upper:]' '[:lower:]' | tr '-' '_')" \
      "$(printf '%s' "$plugin_name" | tr '[:upper:]' '[:lower:]' | tr '-' '_')"
    printf "\n"

    # Get tables currently in the np_{name} schema
    local isolated_tables
    isolated_tables=$(docker exec "$db_container" psql -U "$db_user" -d "$db_name" \
      -t -c "SELECT table_name FROM information_schema.tables
             WHERE table_schema = '${schema_name}'
             ORDER BY table_name;" 2>/dev/null | tr -d ' ' | grep -v '^$' || true)

    if [[ -z "$isolated_tables" ]]; then
      log_info "No tables found in schema $schema_name — nothing to roll back"
      return 0
    fi

    local moved=0
    while IFS= read -r tbl; do
      [[ -z "$tbl" ]] && continue
      local public_name="${table_prefix}${tbl}"
      if [[ "$dry_run" == "true" ]]; then
        printf "  [dry-run] RENAME np_%s.%s -> public.%s\n" \
          "$(printf '%s' "$plugin_name" | tr '[:upper:]' '[:lower:]' | tr '-' '_')" \
          "$tbl" "$public_name"
      else
        printf "  Rolling back: %s.%s -> public.%s\n" "$schema_name" "$tbl" "$public_name"
        docker exec -i "$db_container" psql -U "$db_user" -d "$db_name" \
          -c "ALTER TABLE ${schema_name}.${tbl} RENAME TO ${public_name};
              ALTER TABLE ${schema_name}.${public_name} SET SCHEMA public;" \
          >/dev/null 2>&1 || true
        moved=$((moved + 1))
      fi
    done <<TBLS
$isolated_tables
TBLS

    if [[ "$dry_run" == "true" ]]; then
      printf "\n  [dry-run] No changes made\n"
    else
      log_success "Rollback complete: $moved table(s) returned to public schema"
    fi
    return 0
  fi

  # ── FORWARD: move tables from public to np_{name} schema ─────────────────
  printf "\n"
  log_info "Migrating plugin tables to schema: $schema_name"
  printf "  Pattern: public.%s{suffix} -> %s.{suffix}\n" "$table_prefix" "$schema_name"
  printf "\n"

  # Find matching tables in public schema
  local public_tables
  public_tables=$(docker exec "$db_container" psql -U "$db_user" -d "$db_name" \
    -t -c "SELECT table_name FROM information_schema.tables
           WHERE table_schema = 'public'
           AND table_name LIKE '${table_prefix}%'
           ORDER BY table_name;" 2>/dev/null | tr -d ' ' | grep -v '^$' || true)

  if [[ -z "$public_tables" ]]; then
    log_info "No tables matching 'public.${table_prefix}*' — nothing to migrate"
    printf "  Plugin %s may already be isolated, or has no tables yet.\n" "$plugin_name"
    return 0
  fi

  # Ensure target schema exists (idempotent)
  if [[ "$dry_run" == "false" ]]; then
    if declare -f create_plugin_schema >/dev/null 2>&1; then
      create_plugin_schema "$plugin_name" 2>/dev/null || true
    else
      docker exec -i "$db_container" psql -U "$db_user" -d "$db_name" \
        -c "CREATE SCHEMA IF NOT EXISTS ${schema_name};" >/dev/null 2>&1 || true
    fi
  fi

  local moved=0
  while IFS= read -r full_tbl; do
    [[ -z "$full_tbl" ]] && continue

    # Derive the suffix by stripping the np_{name}_ prefix
    local suffix="${full_tbl#${table_prefix}}"

    if [[ "$dry_run" == "true" ]]; then
      printf "  [dry-run] public.%s  ->  %s.%s\n" "$full_tbl" "$schema_name" "$suffix"
    else
      printf "  Migrating: public.%s -> %s.%s\n" "$full_tbl" "$schema_name" "$suffix"

      # Step 1: move table to target schema (keeps original name temporarily)
      if ! docker exec -i "$db_container" psql -U "$db_user" -d "$db_name" \
          -v ON_ERROR_STOP=1 \
          -c "ALTER TABLE public.${full_tbl} SET SCHEMA ${schema_name};" \
          >/dev/null 2>&1; then
        log_warning "Failed to move table: $full_tbl (skipping)"
        continue
      fi

      # Step 2: rename to drop the now-redundant prefix
      if [[ "$suffix" != "$full_tbl" ]]; then
        docker exec -i "$db_container" psql -U "$db_user" -d "$db_name" \
          -c "ALTER TABLE ${schema_name}.${full_tbl} RENAME TO ${suffix};" \
          >/dev/null 2>&1 || \
          log_warning "Rename failed for ${schema_name}.${full_tbl} -> ${suffix} (table moved but not renamed)"
      fi

      moved=$((moved + 1))
    fi
  done <<PTBLS
$public_tables
PTBLS

  if [[ "$dry_run" == "true" ]]; then
    printf "\n  [dry-run] No changes made.\n"
    printf "  Run without --dry-run to execute.\n"
  else
    if [[ $moved -gt 0 ]]; then
      log_success "Schema migration complete: $moved table(s) moved to $schema_name"
      printf "\n"
      log_info "Next steps:"
      printf "  1. Update any app code that references public.%s* tables\n" "$table_prefix"
      printf "  2. Update Hasura metadata to reflect new schema location\n"
      printf "  3. Run: nself build && nself restart\n"
    else
      log_warning "No tables were moved (check for errors above)"
    fi
  fi

  return 0
}

# ============================================================================
# MAIN
# ============================================================================

main() {
  local command="${1:-}"
  shift || true

  case "$command" in
    list | ls)
      cmd_list "$@"
      ;;
    install | add)
      cmd_install "$@"
      ;;
    remove | rm | uninstall)
      cmd_remove "$@"
      ;;
    update | upgrade)
      cmd_update "$@"
      ;;
    status)
      cmd_status "$@"
      ;;
    updates | check-updates)
      cmd_updates "$@"
      ;;
    refresh | sync-registry)
      cmd_refresh "$@"
      ;;
    check-deps)
      local plugin_name="$1"
      if [[ -z "$plugin_name" ]]; then
        log_error "Plugin name required"
        printf "\nUsage: nself plugin check-deps <name>\n"
        return 1
      fi
      if ! is_plugin_installed "$plugin_name"; then
        log_error "Plugin '$plugin_name' is not installed"
        return 1
      fi
      check_plugin_dependencies "$plugin_name"
      ;;
    check-conflicts)
      local fix_flag="${1:-}"
      if declare -f plugin_resolve_conflicts >/dev/null 2>&1; then
        plugin_resolve_conflicts "$fix_flag"
      else
        log_error "plugin_resolve_conflicts not available"
        return 1
      fi
      ;;
    install-deps)
      local plugin_name="$1"
      shift || true
      local check_only=false
      while [[ $# -gt 0 ]]; do
        case "$1" in
          --check-only)
            check_only=true
            shift
            ;;
          *)
            shift
            ;;
        esac
      done
      if [[ -z "$plugin_name" ]]; then
        log_error "Plugin name required"
        printf "\nUsage: nself plugin install-deps <name> [--check-only]\n"
        return 1
      fi
      if ! is_plugin_installed "$plugin_name"; then
        log_error "Plugin '$plugin_name' is not installed"
        return 1
      fi
      install_plugin_dependencies "$plugin_name" "$check_only"
      ;;
    start)
      # Export project directory for runtime functions to read backend .env files
      export NSELF_PROJECT_DIR="$(pwd)"

      # Check for --all first
      if [[ "$1" == "--all" ]] || [[ "$1" == "-a" ]]; then
        start_all_plugins
      else
        local plugin_name="$1"
        if [[ -z "$plugin_name" ]]; then
          log_error "Plugin name required"
          printf "\nUsage: nself plugin start <name>  OR  nself plugin start --all\n"
          return 1
        fi
        if ! is_plugin_installed "$plugin_name"; then
          log_error "Plugin '$plugin_name' is not installed"
          return 1
        fi
        start_plugin "$plugin_name"
      fi
      ;;
    stop)
      # Export project directory for runtime functions
      export NSELF_PROJECT_DIR="$(pwd)"

      # Check for --all first
      if [[ "$1" == "--all" ]] || [[ "$1" == "-a" ]]; then
        stop_all_plugins
      else
        local plugin_name="$1"
        if [[ -z "$plugin_name" ]]; then
          log_error "Plugin name required"
          printf "\nUsage: nself plugin stop <name>  OR  nself plugin stop --all\n"
          return 1
        fi
        stop_plugin "$plugin_name"
      fi
      ;;
    restart)
      # Export project directory for runtime functions
      export NSELF_PROJECT_DIR="$(pwd)"

      local plugin_name="$1"
      if [[ -z "$plugin_name" ]]; then
        log_error "Plugin name required"
        printf "\nUsage: nself plugin restart <name>\n"
        return 1
      fi
      if ! is_plugin_installed "$plugin_name"; then
        log_error "Plugin '$plugin_name' is not installed"
        return 1
      fi
      restart_plugin "$plugin_name"
      ;;
    logs)
      local plugin_name="$1"
      shift || true
      local follow=false
      local lines=100
      while [[ $# -gt 0 ]]; do
        case "$1" in
          -f | --follow)
            follow=true
            shift
            ;;
          --no-follow)
            follow=false
            shift
            ;;
          --lines|-n)
            shift
            lines="${1:-100}"
            shift
            ;;
          --help|-h)
            printf "Usage: nself plugin logs <name> [options]\n\n"
            printf "View plugin container logs with color-coded output.\n\n"
            printf "Options:\n"
            printf "  -f, --follow      Follow log output in real time\n"
            printf "  --no-follow       Exit after printing (default)\n"
            printf "  --lines N, -n N   Number of lines to show (default: 100)\n"
            printf "  --help, -h        Show this help text\n\n"
            printf "Examples:\n"
            printf "  nself plugin logs ai\n"
            printf "  nself plugin logs ai -f\n"
            printf "  nself plugin logs ai --lines 500\n"
            return 0
            ;;
          *)
            shift
            ;;
        esac
      done
      if [[ -z "$plugin_name" ]]; then
        log_error "Plugin name required"
        printf "\nUsage: nself plugin logs <name> [-f|--follow] [--lines N]\n"
        return 1
      fi
      show_plugin_logs "$plugin_name" "$follow" "$lines"
      ;;
    ps | running)
      list_running_plugins
      ;;
    health)
      health_check_all
      ;;
    license)
      cmd_plugin_license "$@"
      ;;
    outdated)
      cmd_outdated "$@"
      ;;
    rollback)
      cmd_plugin_rollback "$@"
      ;;
    channel)
      cmd_channel "$@"
      ;;
    sync | source-sync)
      cmd_sync "$@"
      ;;
    info)
      cmd_plugin_info "$@"
      ;;
    plugin-list)
      cmd_plugin_list "$@"
      ;;
    plugin-status)
      cmd_plugin_status "$@"
      ;;
    create)
      case "${1:-}" in
        --help|-h|"")
          printf "Usage: nself plugin create <name>\n\n"
          printf "Scaffold a new plugin from a template.\n\n"
          printf "Arguments:\n"
          printf "  name    Name for the new plugin\n\n"
          printf "Options:\n"
          printf "  --help, -h  Show this help text\n\n"
          printf "Examples:\n"
          printf "  nself plugin create my-plugin\n"
          return 0
          ;;
      esac
      log_error "Plugin scaffolding is not yet available in this version"
      return 1
      ;;
    config)
      cmd_plugin_config "$@"
      ;;
    watch)
      cmd_plugin_watch "$@"
      ;;
    deps)
      cmd_plugin_deps "$@"
      ;;
    metrics)
      cmd_plugin_metrics "$@"
      ;;
    deploy)
      cmd_plugin_deploy "$@"
      ;;
    deploy-all)
      cmd_plugin_deploy_all "$@"
      ;;
    migrate-schema)
      cmd_plugin_migrate_schema "$@"
      ;;
    -h | --help | help | "")
      show_help
      ;;
    *)
      # Check if it's a plugin name (for running actions)
      if is_plugin_installed "$command"; then
        cmd_run_action "$command" "$@"
      else
        log_error "Unknown command: $command"
        show_help
        return 1
      fi
      ;;
  esac
}

# ============================================================================
# T-0223: cmd_plugin_watch — auto-restart daemon with exponential backoff
# ============================================================================
#
# Polls np_plugin_registry every N seconds (default 30).
# If healthy=false for >90s: restarts the container with backoff.
# Exponential backoff: 30s -> 60s -> 120s -> 300s per plugin.
# After 5 restarts in 24h: enters "dead" state, calls notify if available.
# --daemon: runs in background with nohup.
# --stop: kills background daemon.
#
# State files: /tmp/nself-plugin-watch-<slug>.state
# Format (line per line): restarts=N lastRestart=EPOCH lastHealthy=EPOCH deadSince=EPOCH
#
# Bash 3.2 compatible — no echo -e, no ${var,,}, no declare -A, no mapfile
#
cmd_plugin_watch() {
  local daemon_mode=false
  local poll_interval=30
  local stop_mode=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --help|-h)
        printf "Usage: nself plugin watch [--daemon] [--interval N] [--stop]\n\n"
        printf "Monitor plugin health and auto-restart crashed plugins.\n\n"
        printf "Options:\n"
        printf "  --daemon          Run in background (nohup)\n"
        printf "  --interval N      Poll interval in seconds (default: 30)\n"
        printf "  --stop            Kill background daemon\n"
        printf "  --help, -h        Show this help text\n\n"
        printf "Behavior:\n"
        printf "  Polls np_plugin_registry every N seconds via Hasura Postgres.\n"
        printf "  If healthy=false for >90s: attempts docker restart with backoff.\n"
        printf "  Backoff: 30s -> 60s -> 120s -> 300s.\n"
        printf "  After 5 restarts in 24h: marks plugin dead, stops auto-restart.\n\n"
        printf "Examples:\n"
        printf "  nself plugin watch\n"
        printf "  nself plugin watch --daemon\n"
        printf "  nself plugin watch --interval 60\n"
        printf "  nself plugin watch --stop\n"
        return 0
        ;;
      --daemon)
        daemon_mode=true
        shift
        ;;
      --interval)
        poll_interval="${2:-30}"
        shift 2
        ;;
      --stop)
        stop_mode=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  # --stop: kill background daemon
  if [[ "$stop_mode" == "true" ]]; then
    local pid_file="${NSELF_LOG_DIR:-$HOME/.nself/logs}/plugin-watch.pid"
    if [[ -f "$pid_file" ]]; then
      local pid
      pid=$(cat "$pid_file" 2>/dev/null)
      if [[ -n "$pid" ]] && kill -0 "$pid" 2>/dev/null; then
        kill "$pid" && rm -f "$pid_file"
        printf "plugin watch daemon stopped (pid %s)\n" "$pid"
      else
        rm -f "$pid_file"
        printf "No running plugin watch daemon found\n"
      fi
    else
      printf "No plugin watch daemon running\n"
    fi
    return 0
  fi

  # --daemon: relaunch self in background
  if [[ "$daemon_mode" == "true" ]]; then
    local log_dir="${NSELF_LOG_DIR:-$HOME/.nself/logs}"
    mkdir -p "$log_dir"
    local log_file="$log_dir/plugin-watch.log"
    local pid_file="$log_dir/plugin-watch.pid"

    # Kill any existing daemon first
    if [[ -f "$pid_file" ]]; then
      local old_pid
      old_pid=$(cat "$pid_file" 2>/dev/null)
      if [[ -n "$old_pid" ]] && kill -0 "$old_pid" 2>/dev/null; then
        kill "$old_pid" 2>/dev/null || true
      fi
      rm -f "$pid_file"
    fi

    nohup "$0" plugin watch --interval "$poll_interval" >> "$log_file" 2>&1 &
    local bg_pid=$!
    printf '%s\n' "$bg_pid" > "$pid_file"
    printf "plugin watch daemon started (pid %s)\n" "$bg_pid"
    printf "Logs: %s\n" "$log_file"
    return 0
  fi

  # Load env to get Postgres credentials
  local pg_user pg_db pg_container
  pg_user="${POSTGRES_USER:-postgres}"
  pg_db="${POSTGRES_DB:-postgres}"
  pg_container="${PROJECT_NAME:-nself}_postgres"

  printf "nself plugin watch — polling every %ss\n" "$poll_interval"
  printf "Container: %s | Ctrl+C to stop\n\n" "$pg_container"

  # Main polling loop
  while true; do
    # Query np_plugin_registry for health status
    local registry_rows=""
    registry_rows=$(docker exec "$pg_container" psql -U "$pg_user" -d "$pg_db" \
      -t -c "SELECT plugin_slug, healthy FROM np_plugin_registry;" 2>/dev/null) || true

    local now
    now=$(date +%s 2>/dev/null || printf '0')

    # Print status table header
    printf "\n[%s] Plugin Health Poll\n" "$(date '+%H:%M:%S' 2>/dev/null || printf 'unknown')"
    printf "%-20s %-10s %-10s\n" "Plugin" "Status" "Restarts"
    printf "%-20s %-10s %-10s\n" "------" "------" "--------"

    # Process each row from pg output
    # pg -t output format: " plugin_slug | t " or " plugin_slug | f "
    if [[ -n "$registry_rows" ]]; then
      while IFS='|' read -r slug healthy_raw; do
        # Trim whitespace (Bash 3.2 compatible)
        slug=$(printf '%s' "$slug" | tr -d ' ')
        healthy_raw=$(printf '%s' "$healthy_raw" | tr -d ' ')

        [[ -z "$slug" ]] && continue

        local state_file="/tmp/nself-plugin-watch-${slug}.state"

        # Read state (Bash 3.2: parse key=value lines manually)
        local restarts=0 last_restart=0 last_healthy=0 dead_since=0
        if [[ -f "$state_file" ]]; then
          while IFS='=' read -r key val; do
            case "$key" in
              restarts)    restarts="$val" ;;
              lastRestart) last_restart="$val" ;;
              lastHealthy) last_healthy="$val" ;;
              deadSince)   dead_since="$val" ;;
            esac
          done < "$state_file"
        fi

        local status_label="healthy"
        if [[ "$healthy_raw" == "t" ]]; then
          # Plugin is healthy — update lastHealthy, reset backoff flag only
          last_healthy="$now"
          dead_since=0
          # Write updated state
          printf 'restarts=%s\nlastRestart=%s\nlastHealthy=%s\ndeadSince=%s\n' \
            "$restarts" "$last_restart" "$last_healthy" "$dead_since" > "$state_file"
        else
          status_label="unhealthy"

          # Check if already in dead state (5 restarts in 24h)
          if [[ "$dead_since" -gt 0 ]]; then
            status_label="dead"
          else
            # How long has it been unhealthy?
            local unhealthy_secs=0
            if [[ "$last_healthy" -gt 0 ]]; then
              unhealthy_secs=$((now - last_healthy))
            else
              # Never recorded healthy — treat as unhealthy since epoch 0
              unhealthy_secs=999
            fi

            if [[ "$unhealthy_secs" -gt 90 ]]; then
              # Check restart count in last 24h
              local restarts_24h=0
              if [[ "$last_restart" -gt 0 ]]; then
                local age_last=$((now - last_restart))
                if [[ "$age_last" -lt 86400 ]]; then
                  restarts_24h="$restarts"
                fi
              fi

              if [[ "$restarts_24h" -ge 5 ]]; then
                # Mark dead — stop auto-restart
                dead_since="$now"
                status_label="dead"
                printf 'restarts=%s\nlastRestart=%s\nlastHealthy=%s\ndeadSince=%s\n' \
                  "$restarts" "$last_restart" "$last_healthy" "$dead_since" > "$state_file"
                printf "WARN: nself-%s has crashed 5 times in 24h — stopped auto-restart\n" "$slug"
                # Notify via nself-notify if available
                if docker inspect nself-notify >/dev/null 2>&1 && \
                   docker exec nself-notify curl -s http://127.0.0.1:3712/health 2>/dev/null | grep -q ok; then
                  docker exec nself-notify curl -s -X POST http://127.0.0.1:3712/notify \
                    -H "Content-Type: application/json" \
                    -d "{\"message\":\"nself-${slug} has crashed 5 times in 24h — auto-restart stopped\"}" \
                    >/dev/null 2>&1 || true
                fi
              else
                # Compute backoff delay based on restart count
                local backoff=30
                case "$restarts" in
                  0) backoff=30 ;;
                  1) backoff=60 ;;
                  2) backoff=120 ;;
                  *) backoff=300 ;;
                esac

                # Only restart if enough time has passed since last restart
                local time_since_last=$((now - last_restart))
                if [[ "$last_restart" -eq 0 ]] || [[ "$time_since_last" -ge "$backoff" ]]; then
                  status_label="restarting"
                  docker restart "nself-${slug}" >/dev/null 2>&1 || true
                  restarts=$((restarts + 1))
                  last_restart="$now"
                  printf 'restarts=%s\nlastRestart=%s\nlastHealthy=%s\ndeadSince=%s\n' \
                    "$restarts" "$last_restart" "$last_healthy" "$dead_since" > "$state_file"
                  printf "Restarting nself-%s (attempt %s, backoff was %ss)\n" \
                    "$slug" "$restarts" "$backoff"
                fi
              fi
            fi
          fi
        fi

        printf "%-20s %-10s %-10s\n" "$slug" "$status_label" "$restarts"
      done <<EOF
$registry_rows
EOF
    else
      printf "  (no plugins in registry or Postgres not reachable)\n"
    fi

    sleep "$poll_interval"
  done
}

# ============================================================================
# T-0849: _plugin_install_with_deps — install with automatic dependency resolution
# ============================================================================
#
# Wraps cmd_install with:
#   - Circular dependency detection (DEPS_BEING_INSTALLED global, space-separated)
#   - Automatic pre-install of declared `requires` entries from plugin.json
#   - --no-deps flag to skip resolution
#
# Usage: _plugin_install_with_deps <plugin_name> [no_deps_flag]
#   no_deps_flag: "true" to skip dependency resolution (default: "false")
#
# Note: DEPS_BEING_INSTALLED must be exported by the caller if needed across
# subshells, but within a single shell session it persists as a global.

DEPS_BEING_INSTALLED="${DEPS_BEING_INSTALLED:-}"

_plugin_read_requires() {
  local plugin_name="$1"
  local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
  if [ ! -f "$manifest" ]; then
    return
  fi
  # Parse the "requires" field (distinct from "dependencies") — Bash 3.2 compatible
  local in_requires=false result="" dep inner line
  while IFS= read -r line; do
    case "$line" in
      *'"requires"'*)
        in_requires=true
        case "$line" in
          *"]"*)
            # Inline array: "requires": ["a", "b"]
            inner="${line#*\[}"
            inner="${inner%%\]*}"
            while [ "${inner#*\"}" != "$inner" ]; do
              dep="${inner#*\"}"
              dep="${dep%%\"*}"
              [ -n "$dep" ] && result="$result $dep"
              inner="${inner#*\"}"
              inner="${inner#*\"}"
            done
            in_requires=false
            ;;
        esac
        continue
        ;;
      *"]"*)
        [ "$in_requires" = "true" ] && { in_requires=false; break; }
        ;;
      *'"'*)
        if [ "$in_requires" = "true" ]; then
          dep="${line#*\"}"
          dep="${dep%%\"*}"
          [ -n "$dep" ] && result="$result $dep"
        fi
        ;;
    esac
  done < "$manifest"
  printf '%s' "$result"
}

_plugin_install_with_deps() {
  local plugin_name="$1"
  local no_deps="${2:-false}"

  # Circular dependency detection
  case " $DEPS_BEING_INSTALLED " in
    *" $plugin_name "*)
      printf "Error: circular dependency detected for plugin '%s'\n" "$plugin_name" >&2
      return 1
      ;;
  esac

  DEPS_BEING_INSTALLED="${DEPS_BEING_INSTALLED} ${plugin_name}"

  # Resolve requires[] dependencies before installing the main plugin
  if [ "$no_deps" != "true" ]; then
    # Download/check manifest for requires — first try already-installed manifest,
    # then fall back to the registry fetch that cmd_install does internally.
    # We pre-check installed manifest; if not installed yet, deps are resolved
    # after cmd_install places the manifest. For pre-install resolution we rely
    # on the static fallback inside _plugin_deps (which covers known plugins).
    local requires dep
    requires=$(_plugin_read_requires "$plugin_name" 2>/dev/null || printf '')
    if [ -z "$requires" ]; then
      # Plugin not yet installed — use _plugin_deps static fallback for known plugins
      requires=$(_plugin_deps "$plugin_name" 2>/dev/null || printf '')
    fi
    for dep in $requires; do
      [ -z "$dep" ] && continue
      if ! is_plugin_installed "$dep"; then
        printf "Installing required dependency: %s\n" "$dep"
        _plugin_install_with_deps "$dep" "false" || return 1
      fi
    done
  fi

  # Run the actual install
  cmd_install "$plugin_name"
}

# ============================================================================
# T-0850: cmd_plugin_deps — show dependency tree for installed plugins
# ============================================================================
#
# Usage:
#   nself plugin deps <plugin-name>   Show deps for a specific installed plugin
#   nself plugin deps --all           List all installed plugins and their deps

cmd_plugin_deps() {
  local plugin_name="${1:-}"

  case "$plugin_name" in
    --help|-h)
      printf "Usage: nself plugin deps <plugin-name>\n"
      printf "       nself plugin deps --all\n\n"
      printf "Show declared dependencies for installed plugins.\n\n"
      printf "Arguments:\n"
      printf "  plugin-name   Show deps for a specific installed plugin\n"
      printf "  --all         List all installed plugins and their deps\n\n"
      printf "Examples:\n"
      printf "  nself plugin deps ai\n"
      printf "  nself plugin deps --all\n"
      return 0
      ;;
    --all|-a)
      local pname manifest requires requires_str found_any=false
      local plugin_dir
      for plugin_dir in "$PLUGIN_DIR"/*/; do
        [ -d "$plugin_dir" ] || continue
        pname=$(basename "$plugin_dir")
        manifest="$plugin_dir/plugin.json"
        if [ -f "$manifest" ]; then
          found_any=true
          requires=$(_plugin_read_requires "$pname" 2>/dev/null || printf '')
          if [ -z "$requires" ]; then
            requires_str="none"
          else
            # Convert space-separated to comma-separated
            requires_str=$(printf '%s' "$requires" | tr ' ' ',' | sed 's/^,//')
          fi
          printf "%s -> [%s]\n" "$pname" "$requires_str"
        fi
      done
      if [ "$found_any" = "false" ]; then
        printf "No plugins installed in %s\n" "$PLUGIN_DIR"
      fi
      ;;
    "")
      printf "Usage: nself plugin deps <plugin-name>\n"
      printf "       nself plugin deps --all\n"
      return 1
      ;;
    *)
      local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
      if [ ! -f "$manifest" ]; then
        printf "Plugin '%s' is not installed.\n" "$plugin_name" >&2
        return 1
      fi
      local requires
      requires=$(_plugin_read_requires "$plugin_name" 2>/dev/null || printf '')
      printf "Dependencies for %s:\n" "$plugin_name"
      if [ -z "$requires" ]; then
        printf "  requires: (none)\n"
      else
        local dep
        for dep in $requires; do
          [ -z "$dep" ] && continue
          if is_plugin_installed "$dep"; then
            printf "  requires: %s (installed)\n" "$dep"
          else
            printf "  requires: %s (NOT installed)\n" "$dep"
          fi
        done
      fi
      ;;
  esac
}

# ============================================================================
# T-0857: cmd_plugin_metrics — fetch and display Prometheus metrics per plugin
# ============================================================================
#
# Usage:
#   nself plugin metrics              Show metrics for all known Rust plugins
#   nself plugin metrics <name>       Show metrics for a specific plugin
#
# Plugin internal URLs are resolved from env vars:
#   PLUGIN_AI_INTERNAL_URL      (default: http://localhost:3711)
#   PLUGIN_CLAW_INTERNAL_URL    (default: http://localhost:3710)
#   PLUGIN_MUX_INTERNAL_URL     (default: http://localhost:3712)
#   PLUGIN_VOICE_INTERNAL_URL   (default: http://localhost:3714)
#   PLUGIN_BROWSER_INTERNAL_URL (default: http://localhost:3716)

_plugin_default_url() {
  local name="$1"
  case "$name" in
    ai)      printf 'http://localhost:3711' ;;
    claw)    printf 'http://localhost:3710' ;;
    mux)     printf 'http://localhost:3712' ;;
    voice)   printf 'http://localhost:3714' ;;
    browser) printf 'http://localhost:3716' ;;
    *)       printf 'http://localhost:3711' ;;
  esac
}

_plugin_resolve_internal_url() {
  local name="$1"
  local upper_name env_var_name url default_url
  upper_name=$(printf '%s' "$name" | tr '[:lower:]' '[:upper:]')
  env_var_name="PLUGIN_${upper_name}_INTERNAL_URL"
  default_url=$(_plugin_default_url "$name")
  # Bash 3.2 compatible indirect variable expansion via eval
  eval "url=\"\${${env_var_name}:-}\""
  if [ -z "$url" ]; then
    url="$default_url"
  fi
  printf '%s' "$url"
}

_show_plugin_metrics() {
  local name="$1"
  local base_url="$2"
  printf "\n=== %s metrics ===\n" "$name"
  if ! command -v curl >/dev/null 2>&1; then
    printf "  [curl not available]\n"
    return
  fi
  local output
  output=$(curl -sf --max-time 5 "${base_url}/metrics" 2>/dev/null) || {
    printf "  [unavailable — is nself-plugin-%s running?]\n" "$name"
    return
  }
  if [ -z "$output" ]; then
    printf "  [no metrics output]\n"
    return
  fi
  # Print non-comment, non-empty lines, limit to 20 lines
  local count=0 line
  while IFS= read -r line; do
    case "$line" in
      '#'*|'') continue ;;
    esac
    printf "%s\n" "$line"
    count=$((count + 1))
    [ "$count" -ge 20 ] && break
  done <<EOF
$output
EOF
}

cmd_plugin_metrics() {
  local plugin_name="${1:-}"

  case "$plugin_name" in
    --help|-h)
      printf "Usage: nself plugin metrics [plugin-name]\n\n"
      printf "Fetch and display Prometheus metrics from running plugin services.\n\n"
      printf "Arguments:\n"
      printf "  plugin-name   Show metrics for a specific plugin (optional)\n\n"
      printf "Supported plugins: ai, claw, mux, voice, browser\n\n"
      printf "Plugin URLs are resolved from env vars:\n"
      printf "  PLUGIN_AI_INTERNAL_URL      (default: http://localhost:3711)\n"
      printf "  PLUGIN_CLAW_INTERNAL_URL    (default: http://localhost:3710)\n"
      printf "  PLUGIN_MUX_INTERNAL_URL     (default: http://localhost:3712)\n"
      printf "  PLUGIN_VOICE_INTERNAL_URL   (default: http://localhost:3714)\n"
      printf "  PLUGIN_BROWSER_INTERNAL_URL (default: http://localhost:3716)\n\n"
      printf "Examples:\n"
      printf "  nself plugin metrics\n"
      printf "  nself plugin metrics ai\n"
      printf "  nself plugin metrics claw\n"
      return 0
      ;;
    "")
      # Show metrics for all known Rust plugins
      local name url
      for name in ai claw mux voice browser; do
        url=$(_plugin_resolve_internal_url "$name")
        _show_plugin_metrics "$name" "$url"
      done
      printf "\n"
      ;;
    *)
      local url
      url=$(_plugin_resolve_internal_url "$plugin_name")
      _show_plugin_metrics "$plugin_name" "$url"
      printf "\n"
      ;;
  esac
}

# ============================================================================
# T-1320: cmd_plugin_deploy — push updated plugin source to remote server
# ============================================================================
#
# Syncs plugin source from the local plugins-pro checkout to a remote nself
# instance via rsync+SSH, then rebuilds and restarts the plugin container.
#
# Usage:
#   nself plugin deploy <plugin-name> [--server <host>] [--user <user>] [--dir <dir>]
#   nself plugin deploy-all --server <host> [--user <user>] [--dir <dir>]
#
# Options:
#   --server <host>    SSH hostname or IP of the target server (required)
#   --user <user>      SSH user (default: root)
#   --dir <dir>        Remote nself project directory (default: /opt/nself)
#   --dry-run          Show what would happen without doing it
#   --no-restart       Rebuild image but skip container restart
#
# Bash 3.2 compatible — no declare -A, no ${var,,}, no echo -e, no mapfile.

cmd_plugin_deploy() {
  local plugin_name="${1:-}"
  local server_host=""
  local server_user="root"
  local remote_dir="/opt/nself"
  local dry_run=false
  local restart=true

  shift 2>/dev/null || true
  while [ $# -gt 0 ]; do
    case "$1" in
      --server)    server_host="$2"; shift 2 ;;
      --user)      server_user="$2"; shift 2 ;;
      --dir)       remote_dir="$2"; shift 2 ;;
      --dry-run)   dry_run=true; shift ;;
      --no-restart) restart=false; shift ;;
      --help|-h)
        printf "Usage: nself plugin deploy <plugin-name> [options]\n\n"
        printf "Sync plugin source to a remote nself instance and rebuild.\n\n"
        printf "Arguments:\n"
        printf "  plugin-name    Name of the pro plugin to deploy\n\n"
        printf "Options:\n"
        printf "  --server <host>   SSH hostname or IP (required)\n"
        printf "  --user <user>     SSH user (default: root)\n"
        printf "  --dir <dir>       Remote nself project dir (default: /opt/nself)\n"
        printf "  --dry-run         Show what would happen without executing\n"
        printf "  --no-restart      Rebuild image but skip container restart\n"
        printf "  --help, -h        Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin deploy mux --server 49.13.140.12 --dir /opt/nclaw\n"
        printf "  nself plugin deploy ai --server 49.13.140.12 --dry-run\n"
        return 0
        ;;
      *) shift ;;
    esac
  done

  if [ -z "$plugin_name" ]; then
    log_error "Plugin name required"
    printf "\nUsage: nself plugin deploy <plugin-name> [--server <host>] [--user <user>] [--dir <dir>]\n"
    printf "Example: nself plugin deploy mux --server 49.13.140.12 --dir /opt/nclaw\n"
    return 1
  fi

  # Locate plugin source in plugins-pro
  local plugins_pro_dir=""
  if [ -d "$HOME/Sites/nself/plugins-pro/paid/${plugin_name}" ]; then
    plugins_pro_dir="$HOME/Sites/nself/plugins-pro/paid/${plugin_name}"
  fi

  if [ -z "$plugins_pro_dir" ]; then
    log_error "Plugin source not found for '${plugin_name}'"
    log_error "Expected: \$HOME/Sites/nself/plugins-pro/paid/${plugin_name}/"
    return 1
  fi

  log_info "Plugin source: $plugins_pro_dir"

  if [ -z "$server_host" ]; then
    log_error "No --server specified."
    printf "\nUsage: nself plugin deploy <plugin-name> --server <ip-or-hostname>\n"
    printf "Example: nself plugin deploy mux --server 49.13.140.12 --dir /opt/nclaw\n"
    return 1
  fi

  local remote_plugin_dir="${remote_dir}/plugins-pro/paid/${plugin_name}"

  if [ "$dry_run" = "true" ]; then
    log_info "[DRY RUN] Would rsync: ${plugins_pro_dir}/src/ -> ${server_user}@${server_host}:${remote_plugin_dir}/src/"
    if [ -f "${plugins_pro_dir}/Cargo.toml" ]; then
      log_info "[DRY RUN] Would rsync: ${plugins_pro_dir}/Cargo.toml -> ${server_user}@${server_host}:${remote_plugin_dir}/Cargo.toml"
    fi
    log_info "[DRY RUN] Would rebuild: docker compose build plugin-${plugin_name}"
    if [ "$restart" = "true" ]; then
      log_info "[DRY RUN] Would restart: docker compose up -d --no-deps plugin-${plugin_name}"
    fi
    return 0
  fi

  # Check required tools
  if ! command -v rsync >/dev/null 2>&1; then
    log_error "rsync is required for plugin deploy"
    return 1
  fi
  if ! command -v ssh >/dev/null 2>&1; then
    log_error "ssh is required for plugin deploy"
    return 1
  fi

  # 1. Ensure remote plugin src directory exists
  # shellcheck disable=SC2029
  ssh "${server_user}@${server_host}" "mkdir -p '${remote_plugin_dir}/src'" 2>/dev/null || true

  # 2. Sync source files
  log_info "Syncing ${plugin_name} source to ${server_host}..."
  if ! rsync -avz --delete \
    "${plugins_pro_dir}/src/" \
    "${server_user}@${server_host}:${remote_plugin_dir}/src/"; then
    log_error "rsync of src/ failed"
    return 1
  fi

  # Sync Cargo.toml if it exists at plugin root
  if [ -f "${plugins_pro_dir}/Cargo.toml" ]; then
    if ! rsync -avz "${plugins_pro_dir}/Cargo.toml" \
      "${server_user}@${server_host}:${remote_plugin_dir}/Cargo.toml"; then
      log_warning "Cargo.toml rsync failed (continuing)"
    fi
  fi

  log_success "Source synced"

  # 3. Rebuild Docker image on remote
  log_info "Rebuilding plugin-${plugin_name} on ${server_host}..."
  # shellcheck disable=SC2029
  if ! ssh "${server_user}@${server_host}" \
    "cd '${remote_dir}' && docker compose build plugin-${plugin_name} 2>&1"; then
    log_error "Remote build failed"
    log_info "Check build logs above. Common causes:"
    log_info "  - Rust compile error in source"
    log_info "  - Missing dependency in Cargo.toml"
    return 1
  fi

  log_success "Build complete"

  # 4. Restart container
  if [ "$restart" = "true" ]; then
    log_info "Restarting plugin-${plugin_name}..."
    # shellcheck disable=SC2029
    if ssh "${server_user}@${server_host}" \
      "cd '${remote_dir}' && docker compose up -d --no-deps plugin-${plugin_name} 2>&1"; then
      log_success "plugin-${plugin_name} restarted"
    else
      log_error "Restart failed — build succeeded but container not running"
      log_info "SSH in and check: docker compose logs plugin-${plugin_name}"
      return 1
    fi
  fi

  log_success "plugin-${plugin_name} deployed to ${server_host}"
}

# Deploy all plugins in plugins-pro/paid/ to a remote server
cmd_plugin_deploy_all() {
  local server_host=""
  local server_user="root"
  local remote_dir="/opt/nself"
  local dry_run=false
  local restart=true

  while [ $# -gt 0 ]; do
    case "$1" in
      --server)    server_host="$2"; shift 2 ;;
      --user)      server_user="$2"; shift 2 ;;
      --dir)       remote_dir="$2"; shift 2 ;;
      --dry-run)   dry_run=true; shift ;;
      --no-restart) restart=false; shift ;;
      --help|-h)
        printf "Usage: nself plugin deploy-all --server <host> [options]\n\n"
        printf "Deploy all plugins from plugins-pro to a remote nself instance.\n\n"
        printf "Options:\n"
        printf "  --server <host>   SSH hostname or IP (required)\n"
        printf "  --user <user>     SSH user (default: root)\n"
        printf "  --dir <dir>       Remote nself project dir (default: /opt/nself)\n"
        printf "  --dry-run         Show what would happen without executing\n"
        printf "  --no-restart      Rebuild images but skip container restarts\n"
        printf "  --help, -h        Show this help text\n\n"
        printf "Examples:\n"
        printf "  nself plugin deploy-all --server 49.13.140.12 --dir /opt/nclaw\n"
        return 0
        ;;
      *) shift ;;
    esac
  done

  if [ -z "$server_host" ]; then
    log_error "Usage: nself plugin deploy-all --server <host>"
    return 1
  fi

  local plugins_pro_dir="$HOME/Sites/nself/plugins-pro/paid"
  if [ ! -d "$plugins_pro_dir" ]; then
    log_error "plugins-pro not found at $plugins_pro_dir"
    return 1
  fi

  local deployed=0
  local failed=0

  for plugin_dir in "$plugins_pro_dir"/*/; do
    [ -d "$plugin_dir" ] || continue
    local pname
    pname="$(basename "$plugin_dir")"
    log_info "Deploying ${pname}..."
    if cmd_plugin_deploy "$pname" --server "$server_host" --user "$server_user" --dir "$remote_dir" \
        $([ "$dry_run" = "true" ] && printf '%s' '--dry-run') \
        $([ "$restart" = "false" ] && printf '%s' '--no-restart'); then
      deployed=$((deployed + 1))
    else
      failed=$((failed + 1))
      log_warning "Failed: $pname (continuing)"
    fi
  done

  log_success "Deployed: ${deployed} | Failed: ${failed}"
}

main "$@"
