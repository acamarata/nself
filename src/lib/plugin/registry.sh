#!/usr/bin/env bash
# registry.sh - Plugin registry client for nself
# Handles fetching, caching, and querying the plugin registry

# ============================================================================
# Configuration
# ============================================================================

# Primary registry endpoint (Cloudflare Worker)
PLUGIN_REGISTRY_URL="${NSELF_PLUGIN_REGISTRY:-https://plugins.nself.org}"

# Fallback to GitHub raw (when primary is unavailable)
PLUGIN_REGISTRY_FALLBACK="https://raw.githubusercontent.com/acamarata/nself-plugins/main/registry.json"

# Cache configuration
PLUGIN_CACHE_DIR="${NSELF_PLUGIN_CACHE:-$HOME/.nself/cache/plugins}"
PLUGIN_REGISTRY_CACHE="${PLUGIN_CACHE_DIR}/registry.json"
PLUGIN_REGISTRY_CACHE_TTL="${NSELF_REGISTRY_CACHE_TTL:-300}"  # 5 minutes

# Plugin installation directory
PLUGIN_DIR="${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}"

# ============================================================================
# Cache Management
# ============================================================================

# Ensure cache directory exists
registry_init_cache() {
    mkdir -p "$PLUGIN_CACHE_DIR"
}

# Check if cache is fresh
registry_cache_is_fresh() {
    local cache_file="$1"
    local ttl="${2:-$PLUGIN_REGISTRY_CACHE_TTL}"

    if [[ ! -f "$cache_file" ]]; then
        return 1
    fi

    local cache_age
    local current_time
    current_time=$(date +%s)

    # Cross-platform stat for modification time
    if [[ "$OSTYPE" == "darwin"* ]]; then
        cache_age=$(stat -f %m "$cache_file" 2>/dev/null)
    else
        cache_age=$(stat -c %Y "$cache_file" 2>/dev/null)
    fi

    if [[ -z "$cache_age" ]]; then
        return 1
    fi

    if (( current_time - cache_age < ttl )); then
        return 0
    fi

    return 1
}

# ============================================================================
# Registry Fetching
# ============================================================================

# Fetch registry with caching and fallback
registry_fetch() {
    local force_refresh="${1:-false}"

    registry_init_cache

    # Check cache first (unless force refresh)
    if [[ "$force_refresh" != "true" ]] && registry_cache_is_fresh "$PLUGIN_REGISTRY_CACHE"; then
        cat "$PLUGIN_REGISTRY_CACHE"
        return 0
    fi

    local registry=""

    # Try primary registry (plugins.nself.org)
    if registry=$(curl -sf --connect-timeout 5 --max-time 10 "${PLUGIN_REGISTRY_URL}/registry.json" 2>/dev/null); then
        if [[ -n "$registry" ]] && printf '%s' "$registry" | grep -q '"plugins"'; then
            printf '%s' "$registry" > "$PLUGIN_REGISTRY_CACHE"
            printf '%s' "$registry"
            return 0
        fi
    fi

    # Try fallback (GitHub raw)
    if registry=$(curl -sf --connect-timeout 5 --max-time 10 "$PLUGIN_REGISTRY_FALLBACK" 2>/dev/null); then
        if [[ -n "$registry" ]] && printf '%s' "$registry" | grep -q '"plugins"'; then
            printf '%s' "$registry" > "$PLUGIN_REGISTRY_CACHE"
            printf '%s' "$registry"
            return 0
        fi
    fi

    # Use stale cache if available
    if [[ -f "$PLUGIN_REGISTRY_CACHE" ]]; then
        cat "$PLUGIN_REGISTRY_CACHE"
        return 0
    fi

    return 1
}

# Fetch specific plugin info from registry
registry_get_plugin() {
    local plugin_name="$1"
    local version="${2:-latest}"

    # Try primary registry API first
    local plugin_info
    if plugin_info=$(curl -sf --connect-timeout 5 --max-time 10 "${PLUGIN_REGISTRY_URL}/plugins/${plugin_name}/${version}" 2>/dev/null); then
        if [[ -n "$plugin_info" ]] && printf '%s' "$plugin_info" | grep -q '"name"'; then
            printf '%s' "$plugin_info"
            return 0
        fi
    fi

    # Fall back to registry.json lookup
    local registry
    registry=$(registry_fetch) || return 1

    if command -v jq >/dev/null 2>&1; then
        printf '%s' "$registry" | jq -r ".plugins[] | select(.name==\"$plugin_name\")" 2>/dev/null
    else
        # Basic grep-based extraction (limited)
        printf '%s' "$registry" | grep -o "\"name\"[[:space:]]*:[[:space:]]*\"${plugin_name}\"[^}]*}" | head -1
    fi
}

# ============================================================================
# Plugin Listing
# ============================================================================

# List all available plugins from registry
registry_list_available() {
    local category="${1:-}"
    local registry

    registry=$(registry_fetch) || {
        printf "Failed to fetch plugin registry\n" >&2
        return 1
    }

    if command -v jq >/dev/null 2>&1; then
        if [[ -n "$category" ]]; then
            printf '%s' "$registry" | jq -r ".plugins[] | select(.category==\"$category\") | .name"
        else
            printf '%s' "$registry" | jq -r '.plugins[].name'
        fi
    else
        # Basic grep-based extraction
        printf '%s' "$registry" | grep -o '"name"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/'
    fi
}

# List installed plugins
registry_list_installed() {
    local plugins=()

    if [[ ! -d "$PLUGIN_DIR" ]]; then
        return 0
    fi

    for plugin_dir in "$PLUGIN_DIR"/*/; do
        if [[ -f "$plugin_dir/plugin.json" ]]; then
            local name
            name=$(basename "$plugin_dir")
            [[ "$name" != "_shared" ]] && plugins+=("$name")
        fi
    done

    printf '%s\n' "${plugins[@]}"
}

# Get installed plugin version
registry_get_installed_version() {
    local plugin_name="$1"
    local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"

    if [[ ! -f "$manifest" ]]; then
        return 1
    fi

    if command -v jq >/dev/null 2>&1; then
        jq -r '.version // "unknown"' "$manifest" 2>/dev/null
    else
        grep '"version"' "$manifest" 2>/dev/null | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/'
    fi
}

# ============================================================================
# Version Comparison
# ============================================================================

# Compare semantic versions
# Returns 0 if v1 >= v2, 1 if v1 < v2
registry_version_compare() {
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
# Update Checking
# ============================================================================

# Check for plugin updates
# Returns list of "plugin_name:current_version:latest_version"
registry_check_updates() {
    local updates=()
    local registry

    registry=$(registry_fetch) || return 1

    for plugin_dir in "$PLUGIN_DIR"/*/; do
        [[ -f "$plugin_dir/plugin.json" ]] || continue

        local name version latest
        name=$(basename "$plugin_dir")
        [[ "$name" == "_shared" ]] && continue

        version=$(registry_get_installed_version "$name")

        if command -v jq >/dev/null 2>&1; then
            latest=$(printf '%s' "$registry" | jq -r ".plugins[] | select(.name==\"$name\") | .version" 2>/dev/null)
        else
            latest=$(printf '%s' "$registry" | grep -A5 "\"name\"[[:space:]]*:[[:space:]]*\"${name}\"" | grep '"version"' | head -1 | sed 's/.*: *"\([^"]*\)".*/\1/')
        fi

        if [[ -n "$latest" ]] && [[ "$latest" != "null" ]]; then
            if ! registry_version_compare "$version" "$latest"; then
                updates+=("${name}:${version}:${latest}")
            fi
        fi
    done

    if [[ ${#updates[@]} -gt 0 ]]; then
        printf '%s\n' "${updates[@]}"
        return 0
    fi

    return 1
}

# Check updates with formatted output
registry_check_updates_formatted() {
    local updates
    updates=$(registry_check_updates 2>/dev/null)

    if [[ -z "$updates" ]]; then
        printf "\033[32m✓ All plugins up to date\033[0m\n"
        return 0
    fi

    printf "\033[33m⚠ Updates available:\033[0m\n"
    while IFS=: read -r name current latest; do
        printf "  %s: %s → %s\n" "$name" "$current" "$latest"
    done <<< "$updates"

    echo ""
    printf "Run '\033[1mnself plugin update\033[0m' to update all plugins\n"
    printf "Run '\033[1mnself plugin update <name>\033[0m' to update a specific plugin\n"

    return 1
}

# ============================================================================
# Plugin Download
# ============================================================================

# Get download URL for a plugin
registry_get_download_url() {
    local plugin_name="$1"
    local version="${2:-latest}"

    local plugin_info
    plugin_info=$(registry_get_plugin "$plugin_name" "$version") || return 1

    local download_url=""
    if command -v jq >/dev/null 2>&1; then
        download_url=$(printf '%s' "$plugin_info" | jq -r '.downloadUrl // empty' 2>/dev/null)

        # If no direct download URL, construct from repository + path
        if [[ -z "$download_url" ]]; then
            local repo path
            repo=$(printf '%s' "$plugin_info" | jq -r '.repository // empty' 2>/dev/null)
            path=$(printf '%s' "$plugin_info" | jq -r '.path // empty' 2>/dev/null)

            if [[ -n "$repo" ]]; then
                # Convert GitHub repo URL to archive download URL
                if [[ "$repo" == *"github.com"* ]]; then
                    # Extract owner/repo from URL
                    local owner_repo
                    owner_repo=$(printf '%s' "$repo" | sed 's|.*github.com/||' | sed 's|\.git$||')
                    download_url="https://api.github.com/repos/${owner_repo}/tarball/main"
                fi
            fi
        fi
    fi

    if [[ -n "$download_url" ]]; then
        printf '%s' "$download_url"
        return 0
    fi

    return 1
}

# Verify plugin checksum
registry_verify_checksum() {
    local file="$1"
    local expected_checksum="$2"

    if [[ -z "$expected_checksum" ]]; then
        return 0  # No checksum to verify
    fi

    local actual_checksum

    # Extract hash algorithm and value
    local algorithm hash_value
    if [[ "$expected_checksum" == *":"* ]]; then
        algorithm="${expected_checksum%%:*}"
        hash_value="${expected_checksum#*:}"
    else
        algorithm="sha256"
        hash_value="$expected_checksum"
    fi

    case "$algorithm" in
        sha256)
            if command -v sha256sum >/dev/null 2>&1; then
                actual_checksum=$(sha256sum "$file" | cut -d' ' -f1)
            elif command -v shasum >/dev/null 2>&1; then
                actual_checksum=$(shasum -a 256 "$file" | cut -d' ' -f1)
            else
                printf "Warning: No SHA256 tool available for checksum verification\n" >&2
                return 0
            fi
            ;;
        sha1)
            if command -v sha1sum >/dev/null 2>&1; then
                actual_checksum=$(sha1sum "$file" | cut -d' ' -f1)
            elif command -v shasum >/dev/null 2>&1; then
                actual_checksum=$(shasum -a 1 "$file" | cut -d' ' -f1)
            else
                printf "Warning: No SHA1 tool available for checksum verification\n" >&2
                return 0
            fi
            ;;
        md5)
            if command -v md5sum >/dev/null 2>&1; then
                actual_checksum=$(md5sum "$file" | cut -d' ' -f1)
            elif command -v md5 >/dev/null 2>&1; then
                actual_checksum=$(md5 -q "$file")
            else
                printf "Warning: No MD5 tool available for checksum verification\n" >&2
                return 0
            fi
            ;;
        *)
            printf "Warning: Unknown checksum algorithm: %s\n" "$algorithm" >&2
            return 0
            ;;
    esac

    if [[ "$actual_checksum" == "$hash_value" ]]; then
        return 0
    else
        printf "Checksum mismatch!\n" >&2
        printf "  Expected: %s\n" "$hash_value" >&2
        printf "  Actual:   %s\n" "$actual_checksum" >&2
        return 1
    fi
}

# ============================================================================
# Registry Metadata
# ============================================================================

# Get registry version/timestamp
registry_get_metadata() {
    local registry
    registry=$(registry_fetch) || return 1

    if command -v jq >/dev/null 2>&1; then
        local version updated plugin_count
        version=$(printf '%s' "$registry" | jq -r '.version // "unknown"' 2>/dev/null)
        updated=$(printf '%s' "$registry" | jq -r '.updated // "unknown"' 2>/dev/null)
        plugin_count=$(printf '%s' "$registry" | jq -r '.plugins | length' 2>/dev/null)

        printf "Registry Version: %s\n" "$version"
        printf "Last Updated: %s\n" "$updated"
        printf "Available Plugins: %s\n" "$plugin_count"
    else
        printf "Registry loaded (jq not available for detailed metadata)\n"
    fi
}

# Get available categories
registry_get_categories() {
    local registry
    registry=$(registry_fetch) || return 1

    if command -v jq >/dev/null 2>&1; then
        printf '%s' "$registry" | jq -r '.categories // [] | .[]' 2>/dev/null
    else
        # Fallback: extract from plugins
        printf '%s' "$registry" | grep -o '"category"[[:space:]]*:[[:space:]]*"[^"]*"' | sed 's/.*"\([^"]*\)"$/\1/' | sort -u
    fi
}

# ============================================================================
# Export Functions
# ============================================================================

export -f registry_init_cache
export -f registry_cache_is_fresh
export -f registry_fetch
export -f registry_get_plugin
export -f registry_list_available
export -f registry_list_installed
export -f registry_get_installed_version
export -f registry_version_compare
export -f registry_check_updates
export -f registry_check_updates_formatted
export -f registry_get_download_url
export -f registry_verify_checksum
export -f registry_get_metadata
export -f registry_get_categories
