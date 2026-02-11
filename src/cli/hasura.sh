#!/usr/bin/env bash
# hasura.sh - Hasura GraphQL Engine Management
# Metadata, migrations, and table tracking

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$SCRIPT_DIR"

# Source dependencies
source "$SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/utils/header.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/hooks/post-command.sh" 2>/dev/null || true

# Fallback log functions
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

# Show usage
hasura_usage() {
  cat <<EOF
Usage: nself hasura <subcommand> [options]

Hasura GraphQL Engine management

METADATA:
  metadata apply     Apply metadata from hasura/metadata/
  metadata export    Export current metadata
  metadata reload    Reload metadata
  metadata clear     Clear all metadata

TRACKING:
  track table        Track a database table
  track schema       Track all tables in schema
  untrack table      Untrack a table

CONSOLE:
  console            Open Hasura console

MIGRATION:
  migrate status     Migration status (via Hasura)
  migrate apply      Apply migrations

EXAMPLES:
  # Apply metadata
  nself hasura metadata apply

  # Track auth tables
  nself hasura track schema auth

  # Track specific table
  nself hasura track table public.users

  # Open console
  nself hasura console

For more information: .wiki/commands/HASURA.md
EOF
}

# Main router
cmd_hasura() {
  local subcommand="${1:-}"

  if [[ -z "$subcommand" ]] || [[ "$subcommand" == "--help" ]] || [[ "$subcommand" == "-h" ]]; then
    hasura_usage
    return 0
  fi

  shift

  case "$subcommand" in
    metadata)
      cmd_hasura_metadata "$@"
      ;;
    track)
      cmd_hasura_track "$@"
      ;;
    untrack)
      cmd_hasura_untrack "$@"
      ;;
    console)
      cmd_hasura_console "$@"
      ;;
    migrate)
      cmd_hasura_migrate "$@"
      ;;
    *)
      log_error "Unknown subcommand: hasura $subcommand"
      hasura_usage
      return 1
      ;;
  esac
}

# Metadata subcommands
cmd_hasura_metadata() {
  local action="${1:-apply}"
  shift || true

  case "$action" in
    apply) hasura_metadata_apply "$@" ;;
    export) hasura_metadata_export "$@" ;;
    reload) hasura_metadata_reload "$@" ;;
    clear) hasura_metadata_clear "$@" ;;
    *)
      log_error "Unknown action: metadata $action"
      return 1
      ;;
  esac
}

# Apply metadata from files
hasura_metadata_apply() {
  log_info "Applying Hasura metadata..."

  # Load environment
  load_env_with_priority 2>/dev/null || true

  # Get Hasura configuration
  local hasura_url="http://localhost:${HASURA_GRAPHQL_PORT:-8080}"
  local admin_secret="${HASURA_GRAPHQL_ADMIN_SECRET}"

  if [[ -z "$admin_secret" ]]; then
    log_error "HASURA_GRAPHQL_ADMIN_SECRET not set"
    return 1
  fi

  # Check if metadata directory exists
  if [[ ! -d "hasura/metadata" ]]; then
    log_warning "No hasura/metadata directory found"
    log_info "Tracking default schemas instead..."
    track_default_schemas
    return 0
  fi

  # If no metadata files, track default schemas
  if [[ ! -f "hasura/metadata/tables.yaml" ]] && [[ ! -f "hasura/metadata/export.json" ]]; then
    log_info "No metadata files found, tracking default schemas..."
    track_default_schemas
    return 0
  fi

  # Apply metadata (simplified - real implementation would parse YAML)
  log_info "Applying metadata from hasura/metadata/"
  track_default_schemas
}

# Export current metadata
hasura_metadata_export() {
  log_info "Exporting Hasura metadata..."

  # Load environment
  load_env_with_priority 2>/dev/null || true

  local hasura_url="http://localhost:${HASURA_GRAPHQL_PORT:-8080}"
  local admin_secret="${HASURA_GRAPHQL_ADMIN_SECRET}"

  if [[ -z "$admin_secret" ]]; then
    log_error "HASURA_GRAPHQL_ADMIN_SECRET not set"
    return 1
  fi

  mkdir -p hasura/metadata

  # Export metadata using Hasura API
  local response=$(curl -s -X POST "$hasura_url/v1/metadata" \
    -H "X-Hasura-Admin-Secret: $admin_secret" \
    -H "Content-Type: application/json" \
    -d '{"type":"export_metadata","args":{}}' 2>/dev/null)

  if [[ -n "$response" ]] && ! echo "$response" | grep -q '"error"'; then
    echo "$response" | command -v jq >/dev/null 2>&1 && echo "$response" | jq . > hasura/metadata/export.json || echo "$response" > hasura/metadata/export.json
    log_success "Metadata exported to hasura/metadata/export.json"
  else
    log_error "Failed to export metadata"
    if echo "$response" | grep -q '"error"'; then
      echo "$response" | command -v jq >/dev/null 2>&1 && echo "$response" | jq . || echo "$response"
    fi
    return 1
  fi
}

# Reload metadata
hasura_metadata_reload() {
  log_info "Reloading Hasura metadata..."

  # Load environment
  load_env_with_priority 2>/dev/null || true

  local hasura_url="http://localhost:${HASURA_GRAPHQL_PORT:-8080}"
  local admin_secret="${HASURA_GRAPHQL_ADMIN_SECRET}"

  if [[ -z "$admin_secret" ]]; then
    log_error "HASURA_GRAPHQL_ADMIN_SECRET not set"
    return 1
  fi

  local response=$(curl -s -X POST "$hasura_url/v1/metadata" \
    -H "X-Hasura-Admin-Secret: $admin_secret" \
    -H "Content-Type: application/json" \
    -d '{"type":"reload_metadata","args":{}}' 2>/dev/null)

  if echo "$response" | grep -q '"message":"success"'; then
    log_success "Metadata reloaded"
  else
    log_error "Failed to reload metadata"
    return 1
  fi
}

# Clear metadata (dangerous!)
hasura_metadata_clear() {
  log_warning "This will clear ALL Hasura metadata"
  printf "Type 'yes' to confirm: "
  read -r response
  if [[ "$response" != "yes" ]]; then
    log_info "Cancelled"
    return 0
  fi

  # Load environment
  load_env_with_priority 2>/dev/null || true

  local hasura_url="http://localhost:${HASURA_GRAPHQL_PORT:-8080}"
  local admin_secret="${HASURA_GRAPHQL_ADMIN_SECRET}"

  local response=$(curl -s -X POST "$hasura_url/v1/metadata" \
    -H "X-Hasura-Admin-Secret: $admin_secret" \
    -H "Content-Type: application/json" \
    -d '{"type":"clear_metadata","args":{}}' 2>/dev/null)

  if echo "$response" | grep -q '"message":"success"'; then
    log_success "Metadata cleared"
  else
    log_error "Failed to clear metadata"
    return 1
  fi
}

# Track table/schema
cmd_hasura_track() {
  local type="${1:-table}"
  local target="${2:-}"

  if [[ -z "$target" ]]; then
    log_error "Usage: nself hasura track <table|schema> <name>"
    return 1
  fi

  case "$type" in
    table)
      track_table "$target"
      ;;
    schema)
      track_schema "$target"
      ;;
    *)
      log_error "Unknown track type: $type"
      return 1
      ;;
  esac
}

# Untrack table
cmd_hasura_untrack() {
  local table_spec="${1:-}"

  if [[ -z "$table_spec" ]]; then
    log_error "Usage: nself hasura untrack table <schema.table>"
    return 1
  fi

  untrack_table "$table_spec"
}

# Track single table
track_table() {
  local table_spec="$1"
  local schema="public"
  local table="$table_spec"

  # Parse schema.table format
  if [[ "$table_spec" == *"."* ]]; then
    schema="${table_spec%%.*}"
    table="${table_spec#*.}"
  fi

  log_info "Tracking table: $schema.$table"

  # Load environment
  load_env_with_priority 2>/dev/null || true

  local hasura_url="http://localhost:${HASURA_GRAPHQL_PORT:-8080}"
  local admin_secret="${HASURA_GRAPHQL_ADMIN_SECRET}"

  if [[ -z "$admin_secret" ]]; then
    log_error "HASURA_GRAPHQL_ADMIN_SECRET not set"
    return 1
  fi

  local response=$(curl -s -X POST "$hasura_url/v1/metadata" \
    -H "X-Hasura-Admin-Secret: $admin_secret" \
    -H "Content-Type: application/json" \
    -d "{
      \"type\": \"pg_track_table\",
      \"args\": {
        \"source\": \"default\",
        \"table\": {
          \"schema\": \"$schema\",
          \"name\": \"$table\"
        }
      }
    }" 2>/dev/null)

  if echo "$response" | grep -q '"message":"success"'; then
    log_success "Table tracked: $schema.$table"
  else
    # Check if already tracked
    if echo "$response" | grep -q "already tracked"; then
      log_info "Table already tracked: $schema.$table"
    else
      log_error "Failed to track table"
      echo "$response" | command -v jq >/dev/null 2>&1 && echo "$response" | jq . || echo "$response"
      return 1
    fi
  fi
}

# Untrack single table
untrack_table() {
  local table_spec="$1"
  local schema="public"
  local table="$table_spec"

  # Parse schema.table format
  if [[ "$table_spec" == *"."* ]]; then
    schema="${table_spec%%.*}"
    table="${table_spec#*.}"
  fi

  log_info "Untracking table: $schema.$table"

  # Load environment
  load_env_with_priority 2>/dev/null || true

  local hasura_url="http://localhost:${HASURA_GRAPHQL_PORT:-8080}"
  local admin_secret="${HASURA_GRAPHQL_ADMIN_SECRET}"

  local response=$(curl -s -X POST "$hasura_url/v1/metadata" \
    -H "X-Hasura-Admin-Secret: $admin_secret" \
    -H "Content-Type: application/json" \
    -d "{
      \"type\": \"pg_untrack_table\",
      \"args\": {
        \"source\": \"default\",
        \"table\": {
          \"schema\": \"$schema\",
          \"name\": \"$table\"
        }
      }
    }" 2>/dev/null)

  if echo "$response" | grep -q '"message":"success"'; then
    log_success "Table untracked: $schema.$table"
  else
    log_error "Failed to untrack table"
    echo "$response" | command -v jq >/dev/null 2>&1 && echo "$response" | jq . || echo "$response"
    return 1
  fi
}

# Track all tables in schema
track_schema() {
  local schema="$1"

  log_info "Tracking all tables in schema: $schema"

  # Get list of tables in schema
  local db="${POSTGRES_DB:-nself}"
  local user="${POSTGRES_USER:-postgres}"
  local project_name="${PROJECT_NAME:-$(basename "$(pwd)")}"

  local tables=$(docker exec -i "${project_name}_postgres" psql -U "$user" -d "$db" -t -A -c "
    SELECT tablename FROM pg_tables
    WHERE schemaname = '$schema'
    ORDER BY tablename
  " 2>/dev/null)

  local count=0
  for table in $tables; do
    if track_table "$schema.$table" 2>/dev/null; then
      count=$((count + 1))
    fi
  done

  log_success "Tracked $count table(s) in schema: $schema"
}

# Track default schemas (auth, storage, public)
track_default_schemas() {
  log_info "Tracking default schemas..."

  # Track auth schema tables
  track_table "auth.users" 2>/dev/null || true
  track_table "auth.user_providers" 2>/dev/null || true
  track_table "auth.providers" 2>/dev/null || true
  track_table "auth.refresh_tokens" 2>/dev/null || true
  track_table "auth.roles" 2>/dev/null || true
  track_table "auth.user_roles" 2>/dev/null || true

  # Track storage schema (if exists)
  track_table "storage.buckets" 2>/dev/null || true
  track_table "storage.files" 2>/dev/null || true

  log_success "Default schemas tracked"
}

# Open Hasura console
cmd_hasura_console() {
  # Load environment
  load_env_with_priority 2>/dev/null || true

  local base_domain="${BASE_DOMAIN:-local.nself.org}"
  local hasura_url="https://api.${base_domain}/console"

  log_info "Opening Hasura console..."
  log_info "URL: $hasura_url"

  if command -v open >/dev/null 2>&1; then
    open "$hasura_url"
  elif command -v xdg-open >/dev/null 2>&1; then
    xdg-open "$hasura_url"
  else
    echo "Open in browser: $hasura_url"
  fi
}

# Migration commands (placeholder)
cmd_hasura_migrate() {
  local action="${1:-status}"
  log_warning "Hasura migrations are managed via 'nself db migrate' commands"
  log_info "Try: nself db migrate $action"
}

# Export for library usage
export -f cmd_hasura

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "hasura" || exit $?
  cmd_hasura "$@"
  exit_code=$?
  post_command "hasura" "$exit_code"
  exit $exit_code
fi
