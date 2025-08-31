#!/usr/bin/env bash
set -euo pipefail


# db.sh - Database management tools for nself
# Handles migrations, seeding, schema sync, and backups

set +e # Don't exit on error for db commands

# Get script directory (macOS compatible)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source environment utilities for safe loading
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/header.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"
# Source auto-config only if it exists
if [[ -f "$SCRIPT_DIR/../lib/database/auto-config.sh" ]]; then
  source "$SCRIPT_DIR/../lib/database/auto-config.sh"
fi
# Color output functions

# Ensure required directories exist
ensure_directories() {
  mkdir -p hasura/migrations/default
  mkdir -p hasura/metadata
  mkdir -p hasura/seeds
  mkdir -p seeds/common
  mkdir -p seeds/development
  mkdir -p seeds/staging
  mkdir -p seeds/production
  mkdir -p bin/dbsyncs
}

# ====================
# Schema Management
# ====================

# Check if DBML CLI is installed
check_dbml_cli() {
  if ! command -v dbml2sql &>/dev/null; then
    log_warning "DBML CLI not installed. Installing..."
    npm install -g @dbml/cli
  fi
}

# Convert DBML to SQL
dbml_to_sql() {
  local dbml_file="$1"
  local sql_file="$2"

  check_dbml_cli

  if dbml2sql "$dbml_file" --postgres -o "$sql_file" 2>/dev/null; then
    return 0
  else
    log_error "Failed to convert DBML to SQL"
    return 1
  fi
}

# Calculate file hash
calculate_hash() {
  local file="$1"
  if [ ! -f "$file" ]; then
    echo "file_not_found"
    return 1
  fi

  if command -v shasum &>/dev/null; then
    shasum -a 256 "$file" | cut -d' ' -f1
  elif command -v sha256sum &>/dev/null; then
    sha256sum "$file" | cut -d' ' -f1
  elif command -v md5 &>/dev/null; then
    md5 -q "$file"
  else
    md5sum "$file" | cut -d' ' -f1
  fi
}

# Backup current state before changes
backup_current_state() {
  local backup_dir="bin/dbsyncs/$(date +%Y-%m-%d_%H-%M-%S)"

  log_info "Backing up current database state to $backup_dir"
  mkdir -p "$backup_dir"

  # Backup schema
  if [ -f "schema.dbml" ]; then
    cp schema.dbml "$backup_dir/"
  fi

  # Backup migrations
  if [ -d "hasura/migrations" ]; then
    cp -r hasura/migrations "$backup_dir/"
  fi

  # Backup seeds
  if [ -d "seeds" ]; then
    cp -r seeds "$backup_dir/"
  fi

  # Save metadata
  cat >"$backup_dir/metadata.json" <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "environment": "${ENVIRONMENT:-development}",
  "schema_hash": "$(calculate_hash schema.dbml 2>/dev/null || echo 'none')",
  "description": "$1"
}
EOF

  log_success "Backup saved to $backup_dir"
  return 0
}

# ====================
# Main DB Run Command
# ====================

# Main run command - analyze schema and generate migrations
cmd_run() {
  local schema_file="${LOCAL_SCHEMA_FILE:-schema.dbml}"
  local force_regenerate=false
  
  # Check for --force flag
  if [[ "$1" == "--force" ]] || [[ "$1" == "-f" ]]; then
    force_regenerate=true
    log_info "Force regeneration enabled"
  fi

  # Check if schema file exists
  if [ ! -f "$schema_file" ]; then
    log_error "Schema file not found: $schema_file"
    log_info "Create a schema.dbml file or run 'nself db sample' to generate one"
    return 1
  fi

  log_info "Analyzing schema: $schema_file"

  # Backup current state
  backup_current_state "Before schema sync"

  # Check if schema has changed
  local hash_file=".nself/schema.hash"
  mkdir -p .nself

  local new_hash=$(calculate_hash "$schema_file")
  local has_changes=false

  if [ "$force_regenerate" = true ]; then
    has_changes=true
    log_info "Forcing migration regeneration"
  elif [ -f "$hash_file" ]; then
    local old_hash=$(cat "$hash_file")
    if [ "$new_hash" != "$old_hash" ]; then
      has_changes=true
    fi
  else
    has_changes=true
  fi

  if [ "$has_changes" = true ]; then
    log_success "Schema changes detected!"

    # Generate migration
    local timestamp=$(date +%Y%m%d%H%M%S)
    local migration_name="schema_update"
    local migration_dir="hasura/migrations/default/${timestamp}_${migration_name}"

    log_info "Generating migration: ${timestamp}_${migration_name}"

    # Create migration directory
    mkdir -p "$migration_dir"

    # Generate SQL from DBML
    local temp_sql="/tmp/dbml_${timestamp}.sql"
    if ! dbml_to_sql "$schema_file" "$temp_sql"; then
      return 1
    fi

    # Create up migration
    cat >"$migration_dir/up.sql" <<EOF
-- Auto-generated migration from schema.dbml
-- Generated: $(date)

EOF

    # Add the generated SQL with IF NOT EXISTS clauses
    sed 's/CREATE TABLE/CREATE TABLE IF NOT EXISTS/g' "$temp_sql" |
      sed 's/CREATE TYPE/CREATE TYPE IF NOT EXISTS/g' |
      sed 's/CREATE INDEX/CREATE INDEX IF NOT EXISTS/g' >>"$migration_dir/up.sql"

    # Create down migration
    cat >"$migration_dir/down.sql" <<'EOF'
-- Rollback migration
-- WARNING: Review and modify as needed before using in production

EOF

    # Extract table names for down migration
    grep -E "^CREATE TABLE" "$temp_sql" |
      sed 's/CREATE TABLE IF NOT EXISTS/DROP TABLE IF EXISTS/' |
      sed 's/CREATE TABLE/DROP TABLE IF EXISTS/' |
      sed 's/ (.*/ CASCADE;/' >>"$migration_dir/down.sql"

    rm -f "$temp_sql"

    # Update hash
    echo "$new_hash" >"$hash_file"

    log_success "Migration created: $migration_dir"
    log_info "Review the migration files before applying:"
    log_info "  - $migration_dir/up.sql"
    log_info "  - $migration_dir/down.sql"
    echo ""
    log_info "To apply migrations:"
    log_info "  - Lead developers: nself db migrate:up"
    log_info "  - All developers: nself db update"
  else
    log_info "Schema is up to date - no changes detected"
    log_info "Use 'nself db run --force' to regenerate migrations"
  fi
}

# ====================
# Sync from dbdiagram.io
# ====================

cmd_sync() {
  local dbdiagram_url="${DBDIAGRAM_URL}"

  if [ -z "$dbdiagram_url" ]; then
    log_error "No DBDIAGRAM_URL configured in .env.local"
    log_info "Add to .env.local:"
    log_info "  DBDIAGRAM_URL=https://dbdiagram.io/d/your-project-id"
    return 1
  fi

  log_info "Syncing from dbdiagram.io: $dbdiagram_url"
  echo ""
  log_warning "dbdiagram.io does not provide a public API."
  log_info "Please follow these steps:"
  echo ""
  log_info "1. Open your diagram:"
  log_info "   $dbdiagram_url"
  echo ""
  log_info "2. Click 'Export' → 'Export to DBML'"
  echo ""
  log_info "3. Copy the DBML content"
  echo ""
  log_info "4. Paste below (press Ctrl+D when done):"
  echo "----------------------------------------"

  # Backup existing schema
  if [ -f "schema.dbml" ]; then
    cp schema.dbml schema.dbml.backup
    log_info "Backed up existing schema to schema.dbml.backup"
  fi

  # Read new schema
  cat >schema.dbml.tmp

  if [ -s "schema.dbml.tmp" ]; then
    mv schema.dbml.tmp schema.dbml
    echo "----------------------------------------"
    log_success "Schema updated from dbdiagram.io"
    echo ""
    log_info "Running 'nself db run' to generate migrations..."
    cmd_run
  else
    rm -f schema.dbml.tmp
    log_error "No content received"
  fi
}

# ====================
# Migration Commands
# ====================

cmd_migrate_create() {
  local name="$1"

  if [ -z "$name" ]; then
    log_error "Please provide a migration name"
    log_info "Usage: nself db migrate:create <name>"
    return 1
  fi

  # Create timestamped migration
  local timestamp=$(date +%Y%m%d%H%M%S)
  local migration_dir="hasura/migrations/default/${timestamp}_${name}"

  mkdir -p "$migration_dir"

  # Create empty migration files
  cat >"$migration_dir/up.sql" <<EOF
-- Migration: $name
-- Created: $(date)

-- Add your forward migration SQL here

EOF

  cat >"$migration_dir/down.sql" <<EOF
-- Rollback for: $name
-- Created: $(date)

-- Add your rollback SQL here

EOF

  log_success "Created migration: $migration_dir"
}

cmd_migrate_up() {
  log_info "Running migrations (up)..."

  # Check if Hasura CLI is available
  if command -v hasura &>/dev/null; then
    cd hasura 2>/dev/null && hasura migrate apply --database-name default || true
    cd - >/dev/null
  else
    # Fallback to direct SQL execution
    log_info "Hasura CLI not found, using direct SQL execution..."

    # Check if postgres container is running
    if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
      log_error "PostgreSQL container is not running. Run 'nself start' first."
      return 1
    fi

    # Apply migrations in order
    for migration in hasura/migrations/default/*/up.sql; do
      if [ -f "$migration" ]; then
        local migration_name=$(basename $(dirname "$migration"))
        log_info "Applying migration: $migration_name"
        docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -d "${POSTGRES_DB:-nhost}" <"$migration" || true
      fi
    done
  fi

  log_success "Migrations completed"
}

cmd_migrate_down() {
  local steps="${1:-1}"

  log_info "Rolling back $steps migration(s)..."

  # Get list of migrations in reverse order
  local migrations=($(ls -r hasura/migrations/default/*/down.sql 2>/dev/null | head -n "$steps"))

  if [ ${#migrations[@]} -eq 0 ]; then
    log_warning "No migrations to rollback"
    return 0
  fi

  # Backup before rollback
  backup_current_state "Before rollback"

  for migration in "${migrations[@]}"; do
    if [ -f "$migration" ]; then
      local migration_name=$(basename $(dirname "$migration"))
      log_info "Rolling back: $migration_name"
      docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -d "${POSTGRES_DB:-nhost}" <"$migration" || true
    fi
  done

  log_success "Rollback completed"
}

# ====================
# Revert Command
# ====================

cmd_revert() {
  log_info "Reverting to previous database state..."

  # Find latest backup
  local latest_backup=$(ls -d bin/dbsyncs/*/ 2>/dev/null | tail -1)

  if [ -z "$latest_backup" ] || [ ! -d "$latest_backup" ]; then
    log_error "No backups found"
    return 1
  fi

  log_info "Reverting to: $latest_backup"

  # Backup current state first
  backup_current_state "Before revert"

  # Restore schema
  if [ -f "$latest_backup/schema.dbml" ]; then
    cp "$latest_backup/schema.dbml" schema.dbml
    log_success "Restored schema.dbml"
  fi

  # Restore migrations
  if [ -d "$latest_backup/migrations" ]; then
    rm -rf hasura/migrations
    cp -r "$latest_backup/migrations" hasura/
    log_success "Restored migrations"
  fi

  # Restore seeds
  if [ -d "$latest_backup/seeds" ]; then
    rm -rf seeds
    cp -r "$latest_backup/seeds" .
    log_success "Restored seeds"
  fi

  log_success "Revert completed"
  log_info "Run 'nself db migrate:up' to apply the restored state"
}

# ====================
# Seeding Commands
# ====================

cmd_seed() {
  # Use ENV if available, otherwise fall back to ENVIRONMENT
  local env_mode="${ENV:-${ENVIRONMENT:-development}}"
  local use_env_seeds="${DB_ENV_SEEDS:-true}" # Default to true for better practices

  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi

  # Map ENV to standard environment names
  local env_name="development"
  if [[ "$env_mode" == "prod" ]] || [[ "$env_mode" == "production" ]]; then
    env_name="production"
  elif [[ "$env_mode" == "staging" ]]; then
    env_name="staging"
  elif [[ "$env_mode" == "dev" ]] || [[ "$env_mode" == "development" ]]; then
    env_name="development"
  fi

  # Use environment-based seeding strategy if enabled
  if [[ "$use_env_seeds" == "true" ]]; then
    # Standards-compliant approach using Hasura/PostgreSQL conventions
    log_info "Seeding database for environment: $env_name"

    # Apply common seeds first (shared across all environments)
    if [ -d "seeds/common" ]; then
      log_info "Applying common seeds..."
      for seed in seeds/common/*.sql; do
        if [ -f "$seed" ]; then
          log_info "  • $(basename $seed)"
          if ! docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -d "${POSTGRES_DB:-nhost}" <"$seed"; then
            log_warning "  Failed to apply: $(basename $seed)"
          fi
        fi
      done
    fi

    # Apply environment-specific seeds
    if [ -d "seeds/$env_name" ]; then
      log_info "Applying $env_name seeds..."
      for seed in seeds/$env_name/*.sql; do
        if [ -f "$seed" ]; then
          log_info "  • $(basename $seed)"
          if ! docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -d "${POSTGRES_DB:-nhost}" <"$seed"; then
            log_warning "  Failed to apply: $(basename $seed)"
          fi
        fi
      done
    else
      log_info "No $env_name seeds found in seeds/$env_name/"
      log_info "Directory structure (Hasura/PostgreSQL standard):"
      log_info "  seeds/common/       - Shared data for all environments"
      log_info "  seeds/development/  - Mock/test data for local development"
      log_info "  seeds/staging/      - Staging environment data"
      log_info "  seeds/production/   - Minimal production data"
    fi

    log_success "Database seeded for $env_name environment"
  else
    # No environment branching - just use default directory
    log_info "Seeding database (no environment branching)"

    if [ -d "seeds/default" ]; then
      for seed in seeds/default/*.sql; do
        if [ -f "$seed" ]; then
          log_info "Applying seed: $(basename $seed)"
          docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -d "${POSTGRES_DB:-nhost}" <"$seed" || true
        fi
      done
    else
      log_info "No seeds found in seeds/default/"
      log_info "Create seed files in seeds/default/ directory"
    fi

    log_success "Database seeded"
  fi
}

# ====================
# Update Command (Safe migration for junior devs)
# ====================

cmd_update() {
  log_info "Checking for database updates..."

  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi

  # Check for pending migrations
  check_pending_migrations
  local pending_count=$?

  if [ "$pending_count" -eq 0 ]; then
    log_success "Database is up to date!"
    return 0
  fi

  log_warning "Found $pending_count pending migration(s)"
  echo ""

  # Show pending migrations
  log_info "Pending migrations:"
  for migration in hasura/migrations/default/*/up.sql; do
    if [ -f "$migration" ]; then
      local migration_name=$(basename $(dirname "$migration"))
      # Validate migration name (alphanumeric, underscore, dash only)
      if [[ ! "$migration_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
        log_warning "Skipping invalid migration name: $migration_name"
        continue
      fi
      # For now, just list all migrations (tracking not implemented)
      echo "  - $migration_name"
    fi
  done

  echo ""
  log_info "This will apply all pending migrations to bring your database up to date."
  read -p "Continue? (y/N): " -r
  echo

  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "Update cancelled"
    return 0
  fi

  # Backup before applying
  backup_current_state "Before update"

  # Apply migrations
  cmd_migrate_up

  # Seed based on environment (dev seeds for non-production, prod seeds for production)
  log_info "Applying seeds for ${ENV:-${ENVIRONMENT:-development}} environment..."
  cmd_seed

  log_success "Database updated successfully!"
}

# Check for pending migrations (returns count)
check_pending_migrations() {
  local migration_count=0

  # Count migration directories
  if [ -d "hasura/migrations/default" ]; then
    migration_count=$(ls -d hasura/migrations/default/*/ 2>/dev/null | wc -l)
  fi

  # If can't connect to database, assume all migrations are pending
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres" 2>/dev/null; then
    return $migration_count
  fi

  # Check if tables exist (basic check for applied migrations)
  local table_count=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -d "${POSTGRES_DB:-nhost}" -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'" 2>/dev/null || echo "0")
  table_count=$(echo "$table_count" | tr -d ' ')

  # If we have migrations but very few tables, migrations are likely pending
  if [ "$migration_count" -gt 0 ] && [ "$table_count" -lt 5 ]; then
    return $migration_count
  fi

  return 0
}

# ====================
# Other Commands
# ====================

cmd_reset() {
  log_warning "This will delete all data and reset the database!"
  read -p "Are you sure? Type 'yes' to confirm: " -r
  echo

  if [[ ! $REPLY == "yes" ]]; then
    log_info "Reset cancelled"
    return 0
  fi

  # Backup before reset
  backup_current_state "Before reset"

  log_info "Resetting database..."

  # Validate database name (alphanumeric, underscore, dash only)
  local db_name="${POSTGRES_DB:-nhost}"
  if [[ ! "$db_name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
    log_error "Invalid database name: $db_name"
    log_info "Database names must contain only letters, numbers, underscores, and dashes"
    return 1
  fi
  
  # Drop and recreate database (using validated name)
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -c "DROP DATABASE IF EXISTS \"$db_name\";"
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql -U postgres -c "CREATE DATABASE \"$db_name\";"

  # Re-run migrations
  cmd_migrate_up

  # Re-seed
  cmd_seed

  log_success "Database reset completed"
}

cmd_status() {
  log_info "Database Status"
  echo ""

  # Check schema
  if [ -f "schema.dbml" ]; then
    log_success "Schema file: schema.dbml"
    local hash=$(calculate_hash schema.dbml)
    log_info "  Hash: ${hash:0:12}..."
  else
    log_warning "No schema.dbml file found"
  fi

  # Check migrations
  local migration_count=$(ls -d hasura/migrations/default/*/ 2>/dev/null | wc -l)
  log_info "Migrations: $migration_count"

  if [ "$migration_count" -gt 0 ]; then
    log_info "Latest migrations:"
    ls -d hasura/migrations/default/*/ 2>/dev/null | tail -3 | while read dir; do
      echo "  - $(basename $dir)"
    done
  fi

  # Check backups
  local backup_count=$(ls -d bin/dbsyncs/*/ 2>/dev/null | wc -l)
  log_info "Backups: $backup_count"

  if [ "$backup_count" -gt 0 ]; then
    log_info "Latest backup: $(ls -d bin/dbsyncs/*/ 2>/dev/null | tail -1)"
  fi

  # Check if database is running
  if docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_success "PostgreSQL: Running"
  else
    log_warning "PostgreSQL: Not running"
  fi
}

cmd_sample() {
  log_info "Creating sample schema.dbml..."

  cat >schema.dbml <<'EOF'
// Sample Database Schema
// Edit this file to define your database structure
// Then run 'nself db run' to generate migrations

Project MyApp {
  database_type: 'PostgreSQL'
  Note: 'Application database schema'
}

// Users table
Table users {
  id uuid [pk, default: `gen_random_uuid()`]
  email varchar(255) [unique, not null]
  username varchar(100) [unique]
  full_name varchar(255)
  avatar_url text
  role varchar(50) [default: 'user']
  email_verified boolean [default: false]
  created_at timestamptz [default: `now()`]
  updated_at timestamptz [default: `now()`]
  
  Indexes {
    email
    username
    created_at
  }
}

// User profiles
Table profiles {
  id uuid [pk, default: `gen_random_uuid()`]
  user_id uuid [ref: - users.id, not null]
  bio text
  website varchar(255)
  location varchar(255)
  preferences jsonb [default: '{}']
  created_at timestamptz [default: `now()`]
  updated_at timestamptz [default: `now()`]
  
  Indexes {
    user_id
  }
}

// Organizations for multi-tenancy
Table organizations {
  id uuid [pk, default: `gen_random_uuid()`]
  name varchar(255) [not null]
  slug varchar(100) [unique, not null]
  owner_id uuid [ref: > users.id, not null]
  settings jsonb [default: '{}']
  created_at timestamptz [default: `now()`]
  updated_at timestamptz [default: `now()`]
  
  Indexes {
    slug
    owner_id
  }
}

// Organization members
Table organization_members {
  organization_id uuid [ref: > organizations.id]
  user_id uuid [ref: > users.id]
  role varchar(50) [default: 'member']
  joined_at timestamptz [default: `now()`]
  
  Indexes {
    (organization_id, user_id) [pk]
  }
}

// Add more tables as needed...
EOF

  log_success "Created sample schema.dbml"
  log_info "Edit this file to match your application needs"
  log_info "Then run 'nself db run' to generate migrations"
}

# ====================
# New Production Commands
# ====================

# Interactive PostgreSQL console
cmd_console() {
  log_info "Opening PostgreSQL console..."
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  # Connect to database
  docker exec -it "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}"
}

# Export database or table
cmd_export() {
  local target="${1:-}"
  local format="${2:-sql}"  # sql, csv, json
  local output="${3:-}"
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  # Generate output filename
  if [[ -z "$output" ]]; then
    output="${target:-database}_$(date +%Y%m%d_%H%M%S).${format}"
  fi
  
  log_info "Exporting to $output (format: $format)..."
  
  case "$format" in
    sql)
      if [[ -z "$target" ]]; then
        # Export entire database
        docker exec "${PROJECT_NAME:-myproject}_postgres" pg_dump \
          -U "${POSTGRES_USER:-postgres}" \
          -d "${POSTGRES_DB:-nhost}" \
          --no-owner --no-acl > "$output"
      else
        # Export specific table
        docker exec "${PROJECT_NAME:-myproject}_postgres" pg_dump \
          -U "${POSTGRES_USER:-postgres}" \
          -d "${POSTGRES_DB:-nhost}" \
          -t "$target" --no-owner --no-acl > "$output"
      fi
      ;;
    csv)
      if [[ -z "$target" ]]; then
        log_error "Table name required for CSV export"
        return 1
      fi
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "COPY $target TO STDOUT WITH CSV HEADER" > "$output"
      ;;
    json)
      if [[ -z "$target" ]]; then
        log_error "Table name required for JSON export"
        return 1
      fi
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -t -c "SELECT json_agg(t) FROM $target t" > "$output"
      ;;
    *)
      log_error "Unknown format: $format (use sql, csv, or json)"
      return 1
      ;;
  esac
  
  log_success "Export completed: $output"
}

# Import data from file
cmd_import() {
  local file="$1"
  local table="${2:-}"
  
  if [[ ! -f "$file" ]]; then
    log_error "File not found: $file"
    return 1
  fi
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  # Detect format from extension
  local ext="${file##*.}"
  
  log_info "Importing from $file..."
  
  case "$ext" in
    sql)
      docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" < "$file"
      ;;
    csv)
      if [[ -z "$table" ]]; then
        log_error "Table name required for CSV import"
        log_info "Usage: nself db import file.csv table_name"
        return 1
      fi
      docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "COPY $table FROM STDIN WITH CSV HEADER" < "$file"
      ;;
    json)
      if [[ -z "$table" ]]; then
        log_error "Table name required for JSON import"
        log_info "Usage: nself db import file.json table_name"
        return 1
      fi
      # Create temp table and import JSON
      docker exec -i "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" <<EOF
CREATE TEMP TABLE json_import (data jsonb);
\COPY json_import FROM STDIN;
$(cat "$file")
\.
INSERT INTO $table 
SELECT * FROM json_populate_recordset(NULL::$table, 
  (SELECT json_agg(data) FROM json_import)::json);
EOF
      ;;
    *)
      log_error "Unknown format: $ext (use .sql, .csv, or .json)"
      return 1
      ;;
  esac
  
  log_success "Import completed"
}

# Analyze database for query optimization
cmd_analyze() {
  log_info "Analyzing database tables..."
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  # Run ANALYZE on all tables
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "ANALYZE VERBOSE;" 2>&1 | while read line; do
    echo "  $line"
  done
  
  log_success "Database analysis completed"
  
  # Show table statistics
  log_info "Table statistics:"
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "SELECT schemaname, tablename, n_live_tup as rows, 
        pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
        FROM pg_stat_user_tables 
        ORDER BY n_live_tup DESC;" 2>/dev/null || true
}

# Vacuum database to reclaim space
cmd_vacuum() {
  local mode="${1:-standard}"  # standard, full, analyze
  
  log_info "Running VACUUM ($mode)..."
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  case "$mode" in
    standard)
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "VACUUM VERBOSE;" 2>&1 | while read line; do
        echo "  $line"
      done
      ;;
    full)
      log_warning "Full VACUUM will lock tables and may take a while..."
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "VACUUM FULL VERBOSE;" 2>&1 | while read line; do
        echo "  $line"
      done
      ;;
    analyze)
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "VACUUM ANALYZE VERBOSE;" 2>&1 | while read line; do
        echo "  $line"
      done
      ;;
    *)
      log_error "Unknown mode: $mode (use standard, full, or analyze)"
      return 1
      ;;
  esac
  
  log_success "VACUUM completed"
}

# Reindex database for performance
cmd_reindex() {
  local target="${1:-database}"  # database, table, index
  local name="${2:-}"
  
  log_info "Reindexing $target..."
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  case "$target" in
    database)
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "REINDEX DATABASE \"${POSTGRES_DB:-nhost}\";"
      ;;
    table)
      if [[ -z "$name" ]]; then
        log_error "Table name required"
        return 1
      fi
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "REINDEX TABLE $name;"
      ;;
    index)
      if [[ -z "$name" ]]; then
        log_error "Index name required"
        return 1
      fi
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -d "${POSTGRES_DB:-nhost}" \
        -c "REINDEX INDEX $name;"
      ;;
    *)
      log_error "Unknown target: $target (use database, table, or index)"
      return 1
      ;;
  esac
  
  log_success "Reindex completed"
}

# Optimize database (VACUUM + ANALYZE + REINDEX)
cmd_optimize() {
  log_info "Optimizing database performance..."
  echo ""
  
  # Run VACUUM ANALYZE
  log_info "Step 1/3: Running VACUUM ANALYZE..."
  cmd_vacuum analyze
  echo ""
  
  # Run REINDEX
  log_info "Step 2/3: Reindexing database..."
  cmd_reindex database
  echo ""
  
  # Update table statistics
  log_info "Step 3/3: Updating statistics..."
  cmd_analyze
  echo ""
  
  log_success "Database optimization completed!"
  
  # Show cache hit ratio
  log_info "Cache hit ratio:"
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "SELECT 
      sum(heap_blks_hit) / nullif(sum(heap_blks_hit) + sum(heap_blks_read), 0) * 100 as cache_hit_ratio
      FROM pg_statio_user_tables;" 2>/dev/null || true
}

# Show current locks
cmd_locks() {
  log_info "Current database locks:"
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "SELECT 
      pid,
      usename,
      application_name,
      client_addr,
      query_start,
      state,
      wait_event_type,
      wait_event,
      substring(query, 1, 50) as query
    FROM pg_stat_activity
    WHERE pid != pg_backend_pid()
    AND query != '<IDLE>'
    ORDER BY query_start;" 2>/dev/null || {
    log_info "No active locks found"
  }
}

# Show active connections
cmd_connections() {
  log_info "Active database connections:"
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  # Show connection summary
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "SELECT 
      count(*) as total,
      count(*) FILTER (WHERE state = 'active') as active,
      count(*) FILTER (WHERE state = 'idle') as idle,
      count(*) FILTER (WHERE state = 'idle in transaction') as idle_in_transaction
    FROM pg_stat_activity
    WHERE pid != pg_backend_pid();" 2>/dev/null || true
  
  echo ""
  log_info "Connections by application:"
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "SELECT 
      application_name,
      count(*) as connections,
      max(query_start) as last_activity
    FROM pg_stat_activity
    WHERE pid != pg_backend_pid()
    GROUP BY application_name
    ORDER BY connections DESC;" 2>/dev/null || true
}

# Kill a database connection
cmd_kill() {
  local pid="$1"
  
  if [[ -z "$pid" ]]; then
    log_error "PID required"
    log_info "Usage: nself db kill <pid>"
    log_info "Get PIDs with: nself db connections"
    return 1
  fi
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  log_info "Terminating connection $pid..."
  
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "SELECT pg_terminate_backend($pid);" 2>/dev/null && {
    log_success "Connection terminated"
  } || {
    log_error "Failed to terminate connection"
    return 1
  }
}

# Clone database for testing
cmd_clone() {
  local target="${1:-${POSTGRES_DB:-nhost}_clone}"
  local template="${POSTGRES_DB:-nhost}"
  
  log_info "Cloning database '$template' to '$target'..."
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  # Check if target already exists
  local exists=$(docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -t -c "SELECT 1 FROM pg_database WHERE datname='$target'" 2>/dev/null | xargs)
  
  if [[ "$exists" == "1" ]]; then
    log_warning "Database '$target' already exists"
    read -p "Drop and recreate? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
      docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
        -U "${POSTGRES_USER:-postgres}" \
        -c "DROP DATABASE \"$target\";" 2>/dev/null || true
    else
      return 0
    fi
  fi
  
  # Terminate connections to template database
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity 
        WHERE datname='$template' AND pid != pg_backend_pid();" 2>/dev/null || true
  
  # Create clone
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -c "CREATE DATABASE \"$target\" WITH TEMPLATE \"$template\";" && {
    log_success "Database cloned successfully!"
    log_info "Connect with: docker exec -it ${PROJECT_NAME:-myproject}_postgres psql -U ${POSTGRES_USER:-postgres} -d $target"
  } || {
    log_error "Failed to clone database"
    return 1
  }
}

# Show database and table sizes
cmd_size() {
  log_info "Database sizes:"
  
  # Check if postgres is running
  if ! docker ps | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
    log_error "PostgreSQL container is not running. Run 'nself start' first."
    return 1
  fi
  
  # Show database size
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -c "SELECT 
      datname as database,
      pg_size_pretty(pg_database_size(datname)) as size
    FROM pg_database
    WHERE datname NOT IN ('template0', 'template1', 'postgres')
    ORDER BY pg_database_size(datname) DESC;" 2>/dev/null || true
  
  echo ""
  log_info "Top 10 tables by size:"
  docker exec "${PROJECT_NAME:-myproject}_postgres" psql \
    -U "${POSTGRES_USER:-postgres}" \
    -d "${POSTGRES_DB:-nhost}" \
    -c "SELECT 
      schemaname || '.' || tablename as table,
      pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as total_size,
      pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
      pg_size_pretty(pg_indexes_size(schemaname||'.'||tablename)) as indexes_size
    FROM pg_tables
    WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
    ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
    LIMIT 10;" 2>/dev/null || true
}

# ====================
# Help Command
# ====================

cmd_help() {
  log_info "nself db - Database management tools"
  echo ""
  log_success "Schema Management:"
  echo "  run                  Analyze schema.dbml and generate migrations"
  echo "  run --force          Force regenerate migrations"
  echo "  sync                 Pull schema from dbdiagram.io (requires DBDIAGRAM_URL)"
  echo "  sample               Create sample schema.dbml"
  echo ""
  log_success "Migration Commands:"
  echo "  migrate:create <name> Create new migration"
  echo "  migrate:up           Apply pending migrations"
  echo "  migrate:down [n]     Rollback n migrations (default: 1)"
  echo ""
  log_success "Database Operations:"
  echo "  console              Interactive PostgreSQL console"
  echo "  export [table] [fmt] Export data (fmt: sql, csv, json)"
  echo "  import <file> [tbl]  Import data from file"
  echo "  clone [name]         Clone database for testing"
  echo "  update               Safely apply pending migrations and seeds"
  echo "  seed                 Seed database (dev or prod based on ENV)"
  echo "  reset                Reset database (drop and recreate)"
  echo "  status               Show database status"
  echo "  revert               Revert to previous backup"
  echo "  size                 Show database and table sizes"
  echo ""
  log_success "Performance & Maintenance:"
  echo "  optimize             Run VACUUM, ANALYZE, and REINDEX"
  echo "  analyze              Update table statistics"
  echo "  vacuum [mode]        Reclaim space (mode: standard, full, analyze)"
  echo "  reindex [target]     Rebuild indexes (target: database, table, index)"
  echo ""
  log_success "Monitoring & Debugging:"
  echo "  connections          Show active connections"
  echo "  locks                Show current locks"
  echo "  kill <pid>           Terminate a connection"
  echo ""
  log_success "Workflow:"
  echo "  1. Edit schema.dbml (or sync from dbdiagram.io)"
  echo "  2. Run 'nself db run' to generate migrations"
  echo "  3. Run 'nself db update' to apply migrations + seeds"
  echo ""
  log_success "Production Features:"
  echo "  • Auto-configuration based on available resources"
  echo "  • Smart defaults for security (SSL, encryption in prod)"
  echo "  • Connection pooling (auto-enabled when needed)"
  echo "  • Point-in-time recovery (auto-enabled in prod)"
  echo "  • Monitoring integration (with nself monitor)"
  echo ""
  log_success "Configuration (.env.local):"
  echo "  ENV                  Environment: 'dev' or 'prod' (default: dev)"
  echo "  DB_*                 Database settings (see .env.example)"
  echo "  All settings use smart defaults - override only if needed"
  echo ""
  log_success "Backups:"
  echo "  Use 'nself backup' for comprehensive backup management"
  echo "  All schema changes are backed up to bin/dbsyncs/"
}

# ====================
# Main Command Handler
# ====================

main() {
  if [[ -f ".env.local" ]]; then
    load_env_with_priority
  fi
  ensure_directories

  local command="${1:-help}"
  shift || true

  # Show command header (except for help command)
  if [[ "$command" != "help" ]] && [[ "$command" != "--help" ]] && [[ "$command" != "-h" ]] && [[ -n "$command" ]]; then
    show_command_header "nself db" "Database operations and management"
  fi

  case "$command" in
  # Schema Management
  run)
    cmd_run "$@"
    ;;
  sync)
    cmd_sync "$@"
    ;;
  sample)
    cmd_sample "$@"
    ;;
    
  # Migration Commands
  migrate:create)
    cmd_migrate_create "$@"
    ;;
  migrate:up)
    cmd_migrate_up "$@"
    ;;
  migrate:down)
    cmd_migrate_down "$@"
    ;;
    
  # Database Operations
  console)
    cmd_console "$@"
    ;;
  export)
    cmd_export "$@"
    ;;
  import)
    cmd_import "$@"
    ;;
  clone)
    cmd_clone "$@"
    ;;
  update)
    cmd_update "$@"
    ;;
  seed)
    cmd_seed "$@"
    ;;
  reset)
    cmd_reset "$@"
    ;;
  status)
    cmd_status "$@"
    ;;
  revert)
    cmd_revert "$@"
    ;;
  size)
    cmd_size "$@"
    ;;
    
  # Performance & Maintenance
  optimize)
    cmd_optimize "$@"
    ;;
  analyze)
    cmd_analyze "$@"
    ;;
  vacuum)
    cmd_vacuum "$@"
    ;;
  reindex)
    cmd_reindex "$@"
    ;;
    
  # Monitoring & Debugging
  connections)
    cmd_connections "$@"
    ;;
  locks)
    cmd_locks "$@"
    ;;
  kill)
    cmd_kill "$@"
    ;;
    
  # Help
  help | --help | -h | "")
    cmd_help
    ;;
  *)
    log_error "Unknown command: $command"
    echo ""
    cmd_help
    exit 1
    ;;
  esac
}

main "$@"
