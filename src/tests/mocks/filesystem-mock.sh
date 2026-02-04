#!/usr/bin/env bash
# filesystem-mock.sh - In-memory filesystem for fast, isolated testing
#
# Provides isolated temporary filesystem operations with automatic cleanup
# All file operations happen in tmpfs when available for maximum speed

set -euo pipefail

# ============================================================================
# Mock Filesystem Configuration
# ============================================================================

# Base directory for mock filesystem
MOCK_FS_BASE="${MOCK_FS_BASE:-/tmp/mock-fs-$$}"

# Whether to use tmpfs (in-memory) if available
MOCK_FS_USE_TMPFS="${MOCK_FS_USE_TMPFS:-true}"

# Maximum size for tmpfs (in MB)
MOCK_FS_TMPFS_SIZE="${MOCK_FS_TMPFS_SIZE:-100}"

# ============================================================================
# Mock Filesystem Initialization
# ============================================================================

# Initialize mock filesystem
init_filesystem_mock() {
  local use_tmpfs="${1:-$MOCK_FS_USE_TMPFS}"

  # Create base directory
  mkdir -p "$MOCK_FS_BASE"

  # Try to mount tmpfs if requested and available (Linux only)
  if [[ "$use_tmpfs" == true ]] && [[ "$(uname)" == "Linux" ]] && [[ $EUID -eq 0 ]]; then
    # Running as root, can mount tmpfs
    if ! mountpoint -q "$MOCK_FS_BASE" 2>/dev/null; then
      mount -t tmpfs -o size="${MOCK_FS_TMPFS_SIZE}M" tmpfs "$MOCK_FS_BASE" 2>/dev/null || true
    fi
  fi

  # Create standard directory structure
  mkdir -p "$MOCK_FS_BASE"/{home,etc,var,tmp,opt}

  export MOCK_FS_BASE
}

# Cleanup mock filesystem
cleanup_filesystem_mock() {
  if [[ -d "$MOCK_FS_BASE" ]]; then
    # Unmount if it's a tmpfs mount
    if mountpoint -q "$MOCK_FS_BASE" 2>/dev/null; then
      umount "$MOCK_FS_BASE" 2>/dev/null || true
    fi

    # Remove directory
    rm -rf "$MOCK_FS_BASE"
  fi
}

# ============================================================================
# Path Translation
# ============================================================================

# Translate path to mock filesystem
# Usage: mock_path /etc/config.yml -> /tmp/mock-fs-$$/etc/config.yml
mock_path() {
  local original_path="$1"

  # Remove leading slash and prepend mock base
  local clean_path="${original_path#/}"
  printf "%s/%s\n" "$MOCK_FS_BASE" "$clean_path"
}

# Create directory structure for path
# Usage: ensure_mock_dir /etc/app/config
ensure_mock_dir() {
  local path="$1"
  local mock_dir_path
  mock_dir_path=$(mock_path "$path")

  mkdir -p "$(dirname "$mock_dir_path")"
}

# ============================================================================
# Mock File Operations
# ============================================================================

# Create mock file with content
# Usage: create_mock_file /etc/app.conf "content here"
create_mock_file() {
  local file_path="$1"
  local content="${2:-}"
  local mock_file_path

  mock_file_path=$(mock_path "$file_path")
  ensure_mock_dir "$file_path"

  printf "%s\n" "$content" > "$mock_file_path"
}

# Create mock file from template
# Usage: create_mock_file_from_template /etc/app.conf template.txt
create_mock_file_from_template() {
  local file_path="$1"
  local template_file="$2"
  local mock_file_path

  mock_file_path=$(mock_path "$file_path")
  ensure_mock_dir "$file_path"

  cp "$template_file" "$mock_file_path"
}

# Read mock file
# Usage: read_mock_file /etc/app.conf
read_mock_file() {
  local file_path="$1"
  local mock_file_path
  mock_file_path=$(mock_path "$file_path")

  if [[ -f "$mock_file_path" ]]; then
    cat "$mock_file_path"
  else
    printf "Error: File not found: %s\n" "$file_path" >&2
    return 1
  fi
}

# Check if mock file exists
# Usage: mock_file_exists /etc/app.conf
mock_file_exists() {
  local file_path="$1"
  local mock_file_path
  mock_file_path=$(mock_path "$file_path")

  [[ -f "$mock_file_path" ]]
}

# Check if mock directory exists
# Usage: mock_dir_exists /etc/app
mock_dir_exists() {
  local dir_path="$1"
  local mock_dir_path
  mock_dir_path=$(mock_path "$dir_path")

  [[ -d "$mock_dir_path" ]]
}

# Remove mock file
# Usage: remove_mock_file /etc/app.conf
remove_mock_file() {
  local file_path="$1"
  local mock_file_path
  mock_file_path=$(mock_path "$file_path")

  rm -f "$mock_file_path"
}

# Remove mock directory
# Usage: remove_mock_dir /etc/app
remove_mock_dir() {
  local dir_path="$1"
  local mock_dir_path
  mock_dir_path=$(mock_path "$dir_path")

  rm -rf "$mock_dir_path"
}

# ============================================================================
# Mock Configuration Files
# ============================================================================

# Create standard mock environment
create_standard_mock_env() {
  # Create common configuration directories
  mkdir -p "$(mock_path /etc/nself)"
  mkdir -p "$(mock_path /var/log/nself)"
  mkdir -p "$(mock_path /opt/nself)"
  mkdir -p "$(mock_path /tmp)"

  # Create mock .env file
  create_mock_file /opt/nself/.env "PROJECT_NAME=test-project
ENV=dev
BASE_DOMAIN=localhost
POSTGRES_DB=test_db
POSTGRES_USER=postgres
POSTGRES_PASSWORD=test-password"

  # Create mock docker-compose.yml
  create_mock_file /opt/nself/docker-compose.yml "version: '3.8'
services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: test_db"
}

# ============================================================================
# File Snapshot & Diff
# ============================================================================

# Create snapshot of mock filesystem state
# Usage: snapshot_mock_fs snapshot_name
snapshot_mock_fs() {
  local snapshot_name="$1"
  local snapshot_dir="$MOCK_FS_BASE/.snapshots/$snapshot_name"

  mkdir -p "$snapshot_dir"
  # Use tar to preserve permissions and structure
  tar -czf "$snapshot_dir/snapshot.tar.gz" -C "$MOCK_FS_BASE" \
    --exclude=".snapshots" . 2>/dev/null || true
}

# Restore snapshot
# Usage: restore_mock_fs snapshot_name
restore_mock_fs() {
  local snapshot_name="$1"
  local snapshot_file="$MOCK_FS_BASE/.snapshots/$snapshot_name/snapshot.tar.gz"

  if [[ -f "$snapshot_file" ]]; then
    # Clear current state (except snapshots)
    find "$MOCK_FS_BASE" -mindepth 1 -maxdepth 1 ! -name ".snapshots" -exec rm -rf {} +

    # Restore snapshot
    tar -xzf "$snapshot_file" -C "$MOCK_FS_BASE" 2>/dev/null
  else
    printf "Error: Snapshot not found: %s\n" "$snapshot_name" >&2
    return 1
  fi
}

# List files changed since snapshot
# Usage: diff_mock_fs snapshot_name
diff_mock_fs() {
  local snapshot_name="$1"
  local temp_restore_dir="/tmp/mock-fs-restore-$$"

  mkdir -p "$temp_restore_dir"
  tar -xzf "$MOCK_FS_BASE/.snapshots/$snapshot_name/snapshot.tar.gz" \
    -C "$temp_restore_dir" 2>/dev/null || return 1

  # Use diff to show changes
  diff -rq "$temp_restore_dir" "$MOCK_FS_BASE" 2>/dev/null | \
    grep -v ".snapshots" || true

  rm -rf "$temp_restore_dir"
}

# ============================================================================
# Test Helpers
# ============================================================================

# Assert mock file contains pattern
# Usage: assert_mock_file_contains /etc/app.conf "setting=value"
assert_mock_file_contains() {
  local file_path="$1"
  local pattern="$2"
  local mock_file_path
  mock_file_path=$(mock_path "$file_path")

  if [[ ! -f "$mock_file_path" ]]; then
    printf "Assertion failed: File does not exist: %s\n" "$file_path" >&2
    return 1
  fi

  if grep -q "$pattern" "$mock_file_path" 2>/dev/null; then
    return 0
  else
    printf "Assertion failed: Pattern not found in %s\n" "$file_path" >&2
    printf "  Pattern: %s\n" "$pattern" >&2
    printf "  File contents:\n" >&2
    head -20 "$mock_file_path" | sed 's/^/    /' >&2
    return 1
  fi
}

# Assert mock file has specific permissions
# Usage: assert_mock_file_permissions /etc/secret.conf 600
assert_mock_file_permissions() {
  local file_path="$1"
  local expected_perms="$2"
  local mock_file_path
  mock_file_path=$(mock_path "$file_path")

  if [[ ! -f "$mock_file_path" ]]; then
    printf "Assertion failed: File does not exist: %s\n" "$file_path" >&2
    return 1
  fi

  local actual_perms
  if stat --version 2>/dev/null | grep -q GNU; then
    actual_perms=$(stat -c "%a" "$mock_file_path")
  else
    actual_perms=$(stat -f "%OLp" "$mock_file_path")
  fi

  if [[ "$actual_perms" == "$expected_perms" ]]; then
    return 0
  else
    printf "Assertion failed: Wrong permissions on %s\n" "$file_path" >&2
    printf "  Expected: %s\n" "$expected_perms" >&2
    printf "  Actual: %s\n" "$actual_perms" >&2
    return 1
  fi
}

# Count files in mock directory
# Usage: count_mock_files /etc/app/configs
count_mock_files() {
  local dir_path="$1"
  local mock_dir_path
  mock_dir_path=$(mock_path "$dir_path")

  if [[ -d "$mock_dir_path" ]]; then
    find "$mock_dir_path" -type f | wc -l
  else
    printf "0\n"
  fi
}

# ============================================================================
# Performance Monitoring
# ============================================================================

# Get mock filesystem statistics
get_mock_fs_stats() {
  printf "Mock Filesystem Statistics:\n"
  printf "  Base: %s\n" "$MOCK_FS_BASE"
  printf "  Files: %d\n" "$(find "$MOCK_FS_BASE" -type f 2>/dev/null | wc -l)"
  printf "  Directories: %d\n" "$(find "$MOCK_FS_BASE" -type d 2>/dev/null | wc -l)"

  # Show size
  if command -v du >/dev/null 2>&1; then
    printf "  Size: %s\n" "$(du -sh "$MOCK_FS_BASE" 2>/dev/null | cut -f1)"
  fi

  # Check if tmpfs
  if mountpoint -q "$MOCK_FS_BASE" 2>/dev/null; then
    printf "  Type: tmpfs (in-memory)\n"
  else
    printf "  Type: disk\n"
  fi
}

# ============================================================================
# Export Functions
# ============================================================================

export -f init_filesystem_mock
export -f cleanup_filesystem_mock
export -f mock_path
export -f ensure_mock_dir
export -f create_mock_file
export -f create_mock_file_from_template
export -f read_mock_file
export -f mock_file_exists
export -f mock_dir_exists
export -f remove_mock_file
export -f remove_mock_dir
export -f create_standard_mock_env
export -f snapshot_mock_fs
export -f restore_mock_fs
export -f diff_mock_fs
export -f assert_mock_file_contains
export -f assert_mock_file_permissions
export -f count_mock_files
export -f get_mock_fs_stats
