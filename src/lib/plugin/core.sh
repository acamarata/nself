#!/usr/bin/env bash
# core.sh - Core plugin utilities for nself
# Provides common functions for plugin management

# ============================================================================
# Plugin Registry
# ============================================================================

PLUGIN_REGISTRY_URL="${NSELF_PLUGIN_REGISTRY:-https://raw.githubusercontent.com/acamarata/nself-plugins/main/registry.json}"
PLUGIN_DIR="${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}"
PLUGIN_CACHE_DIR="${NSELF_PLUGIN_CACHE:-$HOME/.nself/cache/plugins}"

# ============================================================================
# Plugin Information
# ============================================================================

# Get plugin info from manifest
plugin_get_info() {
    local plugin_name="$1"
    local field="$2"
    local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"

    if [[ ! -f "$manifest" ]]; then
        return 1
    fi

    grep "\"$field\"" "$manifest" | head -1 | sed 's/.*"'"$field"'"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/'
}

# Get list of installed plugins
plugin_list_installed() {
    local plugins=()

    for plugin_dir in "$PLUGIN_DIR"/*/; do
        if [[ -f "$plugin_dir/plugin.json" ]]; then
            plugins+=("$(basename "$plugin_dir")")
        fi
    done

    printf '%s\n' "${plugins[@]}"
}

# Check if plugin is compatible with current nself version
plugin_check_compatibility() {
    local plugin_name="$1"

    local min_version
    min_version=$(plugin_get_info "$plugin_name" "minNselfVersion")

    if [[ -z "$min_version" ]]; then
        return 0
    fi

    local current_version
    current_version=$(cat "$(dirname "${BASH_SOURCE[0]}")/../../VERSION" 2>/dev/null || echo "0.0.0")

    # Compare versions
    plugin_version_compare "$current_version" "$min_version"
}

# Compare semantic versions
# Returns 0 if v1 >= v2, 1 if v1 < v2
plugin_version_compare() {
    local v1="$1"
    local v2="$2"

    # Remove 'v' prefix if present
    v1="${v1#v}"
    v2="${v2#v}"

    # Split into parts using IFS
    local v1_major v1_minor v1_patch
    local v2_major v2_minor v2_patch

    IFS='.' read -r v1_major v1_minor v1_patch <<< "$v1"
    IFS='.' read -r v2_major v2_minor v2_patch <<< "$v2"

    # Default to 0 if empty
    v1_major="${v1_major:-0}"
    v1_minor="${v1_minor:-0}"
    v1_patch="${v1_patch:-0}"
    v2_major="${v2_major:-0}"
    v2_minor="${v2_minor:-0}"
    v2_patch="${v2_patch:-0}"

    if (( v1_major > v2_major )); then
        return 0
    elif (( v1_major < v2_major )); then
        return 1
    fi

    if (( v1_minor > v2_minor )); then
        return 0
    elif (( v1_minor < v2_minor )); then
        return 1
    fi

    if (( v1_patch >= v2_patch )); then
        return 0
    else
        return 1
    fi
}

# ============================================================================
# Plugin Database Operations
# ============================================================================

# Get database connection for plugin operations
plugin_db_connection() {
    local db_host="${POSTGRES_HOST:-localhost}"
    local db_port="${POSTGRES_PORT:-5432}"
    local db_name="${POSTGRES_DB:-nself}"
    local db_user="${POSTGRES_USER:-postgres}"
    local db_pass="${POSTGRES_PASSWORD:-}"

    printf "postgresql://%s:%s@%s:%s/%s" "$db_user" "$db_pass" "$db_host" "$db_port" "$db_name"
}

# Execute SQL for plugin
plugin_db_exec() {
    local sql="$1"
    local container="${PROJECT_NAME:-nself}_postgres"

    if docker ps --format "{{.Names}}" 2>/dev/null | grep -q "^${container}$"; then
        docker exec -i "$container" psql -U "${POSTGRES_USER:-postgres}" -d "${POSTGRES_DB:-nself}" -c "$sql" 2>/dev/null
    elif command -v psql >/dev/null 2>&1; then
        psql "$(plugin_db_connection)" -c "$sql" 2>/dev/null
    else
        return 1
    fi
}

# ============================================================================
# Plugin Hooks
# ============================================================================

# Run pre-install hooks
plugin_pre_install() {
    local plugin_name="$1"

    # Check compatibility
    if ! plugin_check_compatibility "$plugin_name"; then
        log_error "Plugin '$plugin_name' requires a newer version of nself"
        return 1
    fi

    return 0
}

# Run post-install hooks
plugin_post_install() {
    local plugin_name="$1"

    # Verify installation
    if [[ ! -f "$PLUGIN_DIR/$plugin_name/plugin.json" ]]; then
        log_error "Plugin installation failed"
        return 1
    fi

    return 0
}

# ============================================================================
# Plugin Environment
# ============================================================================

# Load plugin environment
plugin_load_env() {
    local plugin_name="$1"

    # Source project .env if exists
    if [[ -f ".env" ]]; then
        set -a
        source ".env" 2>/dev/null || true
        set +a
    fi
}

# Check required environment variables
plugin_check_env() {
    local plugin_name="$1"
    local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"

    if [[ ! -f "$manifest" ]]; then
        return 1
    fi

    # Extract required env vars
    local required_vars
    required_vars=$(grep -A10 '"required"' "$manifest" | grep -o '"[A-Z_]*"' | tr -d '"' || true)

    local missing=0
    for var in $required_vars; do
        if [[ -z "${!var:-}" ]]; then
            log_warning "Missing required variable: $var"
            ((missing++))
        fi
    done

    return $missing
}
