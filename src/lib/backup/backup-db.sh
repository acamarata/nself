#!/usr/bin/env bash
# backup-db.sh - Database backup and restore with optional S3/MinIO upload
# Part of nself SysOps - Bash 3.2 compatible

set -euo pipefail

# Get the directory where this script is located
_BACKUP_DB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_BACKUP_DB_LIB_ROOT="$(dirname "$_BACKUP_DB_DIR")"

# Source dependencies
source "$_BACKUP_DB_LIB_ROOT/utils/display.sh" 2>/dev/null || true
source "$_BACKUP_DB_LIB_ROOT/utils/platform-compat.sh" 2>/dev/null || true

# Configuration defaults
BACKUP_DIR="${BACKUP_DIR:-./backups/db}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-7}"
BACKUP_S3_BUCKET="${BACKUP_S3_BUCKET:-}"
BACKUP_S3_ENDPOINT="${BACKUP_S3_ENDPOINT:-}"
BACKUP_S3_ACCESS_KEY="${BACKUP_S3_ACCESS_KEY:-}"
BACKUP_S3_SECRET_KEY="${BACKUP_S3_SECRET_KEY:-}"

# Create a compressed database backup via pg_dump in the postgres container
backup_db() {
  local db_name="${1:-${POSTGRES_DB:-nself_db}}"
  local db_user="${2:-${POSTGRES_USER:-postgres}}"
  local custom_name="${3:-}"

  # Find the postgres container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  if [[ -z "$container" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "PostgreSQL container not found or not running" >&2
    return 1
  fi

  # Prepare backup directory and filename
  mkdir -p "$BACKUP_DIR"
  local timestamp
  timestamp=$(date +%Y%m%d_%H%M%S)
  local backup_name="${custom_name:-${db_name}_${timestamp}.sql.gz}"
  local backup_path="${BACKUP_DIR}/${backup_name}"

  printf "\033[34m%s\033[0m %s\n" "INFO" "Backing up database '$db_name' from container '$container'..."

  # Run pg_dump inside the container, pipe through gzip on host
  if ! docker exec "$container" pg_dump -U "$db_user" "$db_name" 2>/dev/null | gzip > "$backup_path"; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "pg_dump failed for database '$db_name'" >&2
    rm -f "$backup_path"
    return 1
  fi

  # Verify the backup is non-empty
  if [[ ! -s "$backup_path" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "Backup file is empty" >&2
    rm -f "$backup_path"
    return 1
  fi

  local size
  size=$(du -h "$backup_path" | cut -f1)
  printf "\033[32m%s\033[0m %s (%s)\n" "SUCCESS" "Backup saved: $backup_path" "$size"

  # Upload to S3/MinIO if configured
  if [[ -n "$BACKUP_S3_BUCKET" ]]; then
    backup_db_upload_s3 "$backup_path" "$backup_name"
  fi

  # Prune old local backups based on retention
  backup_db_prune_local "$db_name"

  printf "%s\n" "$backup_path"
}

# Upload a backup file to S3/MinIO bucket
backup_db_upload_s3() {
  local file_path="$1"
  local object_name="${2:-$(basename "$file_path")}"
  local bucket="${BACKUP_S3_BUCKET}"

  if [[ -z "$bucket" ]]; then
    printf "\033[33m%s\033[0m %s\n" "WARNING" "BACKUP_S3_BUCKET not set, skipping upload"
    return 0
  fi

  # Prefer mc (MinIO Client) if available
  if command -v mc >/dev/null 2>&1; then
    # Configure alias if endpoint is provided
    if [[ -n "${BACKUP_S3_ENDPOINT:-}" ]]; then
      mc alias set nself_backup "$BACKUP_S3_ENDPOINT" \
        "${BACKUP_S3_ACCESS_KEY:-}" "${BACKUP_S3_SECRET_KEY:-}" \
        --api S3v4 >/dev/null 2>&1 || true
      local target="nself_backup/${bucket}/${object_name}"
    else
      local target="${bucket}/${object_name}"
    fi

    printf "\033[34m%s\033[0m %s\n" "INFO" "Uploading to S3: $target"
    if mc cp "$file_path" "$target" >/dev/null 2>&1; then
      printf "\033[32m%s\033[0m %s\n" "SUCCESS" "Uploaded to $target"
    else
      printf "\033[31m%s\033[0m %s\n" "ERROR" "Failed to upload via mc" >&2
      return 1
    fi

    # Prune old remote backups
    backup_db_prune_s3 "$bucket"

  elif command -v curl >/dev/null 2>&1 && [[ -n "${BACKUP_S3_ENDPOINT:-}" ]]; then
    # Fallback: use curl with S3 PUT (unsigned, for MinIO with public write)
    local url="${BACKUP_S3_ENDPOINT}/${bucket}/${object_name}"
    printf "\033[34m%s\033[0m %s\n" "INFO" "Uploading via curl to: $url"
    if curl -s -X PUT -T "$file_path" "$url" >/dev/null 2>&1; then
      printf "\033[32m%s\033[0m %s\n" "SUCCESS" "Uploaded to $url"
    else
      printf "\033[33m%s\033[0m %s\n" "WARNING" "curl upload failed (may need mc for authenticated uploads)"
    fi
  else
    printf "\033[33m%s\033[0m %s\n" "WARNING" "Neither mc nor curl with endpoint available, skipping S3 upload"
  fi
}

# Prune local backups older than BACKUP_RETENTION_DAYS
backup_db_prune_local() {
  local db_name="${1:-}"
  local retention_days="${BACKUP_RETENTION_DAYS:-7}"

  if [[ ! -d "$BACKUP_DIR" ]]; then
    return 0
  fi

  local pattern="*.sql.gz"
  if [[ -n "$db_name" ]]; then
    pattern="${db_name}_*.sql.gz"
  fi

  local count=0
  # Use find with mtime for age-based pruning
  while IFS= read -r old_backup; do
    if [[ -f "$old_backup" ]]; then
      rm -f "$old_backup"
      count=$((count + 1))
    fi
  done < <(find "$BACKUP_DIR" -name "$pattern" -type f -mtime "+${retention_days}" 2>/dev/null)

  if [[ "$count" -gt 0 ]]; then
    printf "\033[34m%s\033[0m Pruned %d backup(s) older than %d days\n" "INFO" "$count" "$retention_days"
  fi
}

# Prune remote S3 backups older than retention period (requires mc)
backup_db_prune_s3() {
  local bucket="$1"
  local retention_days="${BACKUP_RETENTION_DAYS:-7}"

  if ! command -v mc >/dev/null 2>&1; then
    return 0
  fi

  local target="nself_backup/${bucket}"
  # mc rm with --older-than flag
  mc rm --recursive --force --older-than "${retention_days}d" "$target/" >/dev/null 2>&1 || true
}

# Restore a database from a gzip dump file
backup_db_restore() {
  local backup_file="$1"
  local db_name="${2:-${POSTGRES_DB:-nself_db}}"
  local db_user="${3:-${POSTGRES_USER:-postgres}}"

  if [[ -z "$backup_file" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "Backup file path required" >&2
    printf "Usage: backup_db_restore <file.sql.gz> [database] [user]\n" >&2
    return 1
  fi

  # Support both local files and S3 paths
  local local_file="$backup_file"
  if [[ "$backup_file" = s3://* ]] || [[ "$backup_file" = */* && ! -f "$backup_file" ]]; then
    # Try to download from S3 if it looks like a remote path
    if [[ -n "${BACKUP_S3_BUCKET:-}" ]] && command -v mc >/dev/null 2>&1; then
      local_file="/tmp/nself_restore_$$.sql.gz"
      printf "\033[34m%s\033[0m %s\n" "INFO" "Downloading backup from S3..."
      mc cp "nself_backup/${backup_file}" "$local_file" >/dev/null 2>&1 || {
        printf "\033[31m%s\033[0m %s\n" "ERROR" "Failed to download backup from S3" >&2
        return 1
      }
    fi
  fi

  if [[ ! -f "$local_file" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "Backup file not found: $local_file" >&2
    return 1
  fi

  # Find the postgres container
  local container
  container=$(docker ps --filter 'name=postgres' --format '{{.Names}}' | head -1)
  if [[ -z "$container" ]]; then
    printf "\033[31m%s\033[0m %s\n" "ERROR" "PostgreSQL container not found or not running" >&2
    return 1
  fi

  printf "\033[34m%s\033[0m Restoring database '%s' from '%s'...\n" "INFO" "$db_name" "$(basename "$local_file")"

  # Decompress and pipe into psql inside the container
  # Drop and recreate the database first
  docker exec "$container" psql -U "$db_user" -d postgres -c \
    "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = '$db_name' AND pid <> pg_backend_pid();" \
    >/dev/null 2>&1 || true

  docker exec "$container" psql -U "$db_user" -d postgres -c \
    "DROP DATABASE IF EXISTS \"$db_name\";" >/dev/null 2>&1

  docker exec "$container" psql -U "$db_user" -d postgres -c \
    "CREATE DATABASE \"$db_name\";" >/dev/null 2>&1

  # Restore from the gzip dump
  if gunzip -c "$local_file" | docker exec -i "$container" psql -U "$db_user" -d "$db_name" >/dev/null 2>&1; then
    printf "\033[32m%s\033[0m Database '%s' restored successfully\n" "SUCCESS" "$db_name"
  else
    printf "\033[31m%s\033[0m Restore completed with warnings (check for errors above)\n" "WARNING" >&2
  fi

  # Clean up temporary download
  if [[ "$local_file" != "$backup_file" ]]; then
    rm -f "$local_file"
  fi
}

# Export functions
export -f backup_db 2>/dev/null || true
export -f backup_db_restore 2>/dev/null || true
export -f backup_db_upload_s3 2>/dev/null || true
export -f backup_db_prune_local 2>/dev/null || true
