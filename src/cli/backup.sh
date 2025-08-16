#!/usr/bin/env bash

# backup.sh - Comprehensive backup and restore for nself

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Backup configuration
BACKUP_DIR="${BACKUP_DIR:-./backups}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-30}"
S3_BUCKET="${S3_BUCKET:-}"
S3_ENDPOINT="${S3_ENDPOINT:-}"  # For MinIO/S3-compatible

# Ensure backup directory exists
mkdir -p "$BACKUP_DIR"

# Generate backup filename
generate_backup_name() {
  local type="${1:-full}"
  local timestamp=$(date +%Y%m%d_%H%M%S)
  echo "nself_backup_${type}_${timestamp}.tar.gz"
}

# Create backup
cmd_backup_create() {
  local backup_type="${1:-full}"
  local custom_name="${2:-}"
  
  show_command_header "nself backup create" "Create backup of nself data"
  
  # Determine backup name
  local backup_name="${custom_name:-$(generate_backup_name "$backup_type")}"
  local backup_path="$BACKUP_DIR/$backup_name"
  local temp_dir=$(mktemp -d)
  
  log_info "Creating backup: $backup_name"
  echo ""
  
  # Backup components based on type
  case "$backup_type" in
    full)
      log_info "Backing up all components..."
      backup_database "$temp_dir"
      backup_config "$temp_dir"
      backup_volumes "$temp_dir"
      backup_certificates "$temp_dir"
      ;;
    database)
      log_info "Backing up database only..."
      backup_database "$temp_dir"
      ;;
    config)
      log_info "Backing up configuration only..."
      backup_config "$temp_dir"
      ;;
    *)
      log_error "Unknown backup type: $backup_type"
      log_info "Valid types: full, database, config"
      rm -rf "$temp_dir"
      return 1
      ;;
  esac
  
  # Create tarball
  log_info "Creating archive..."
  tar -czf "$backup_path" -C "$temp_dir" . 2>/dev/null
  
  # Calculate size
  local size=$(du -h "$backup_path" | cut -f1)
  
  # Upload to S3 if configured
  if [[ -n "$S3_BUCKET" ]]; then
    upload_to_s3 "$backup_path" "$backup_name"
  fi
  
  # Cleanup
  rm -rf "$temp_dir"
  
  # Success message
  echo ""
  log_success "Backup created successfully!"
  echo "  Location: $backup_path"
  echo "  Size: $size"
  echo "  Type: $backup_type"
  
  if [[ -n "$S3_BUCKET" ]]; then
    echo "  S3: s3://$S3_BUCKET/nself-backups/$backup_name"
  fi
  
  echo ""
  log_info "To restore this backup, run:"
  echo "  nself backup restore $backup_name"
  echo ""
}

# Backup database
backup_database() {
  local dest_dir="$1"
  local db_backup_dir="$dest_dir/database"
  mkdir -p "$db_backup_dir"
  
  # Check if postgres is running
  if docker ps --format "{{.Names}}" | grep -q postgres; then
    log_info "  • Dumping PostgreSQL database..."
    
    # Get database credentials from .env
    local db_name="${POSTGRES_DB:-postgres}"
    local db_user="${POSTGRES_USER:-postgres}"
    
    # Dump all databases
    docker exec postgres pg_dumpall -U "$db_user" > "$db_backup_dir/postgres_dump.sql" 2>/dev/null || {
      log_warning "  Failed to dump database (container may not be running)"
    }
    
    # Also backup Hasura metadata if available
    if [[ -d "./hasura/metadata" ]]; then
      log_info "  • Copying Hasura metadata..."
      cp -r ./hasura/metadata "$db_backup_dir/hasura_metadata"
    fi
  else
    log_warning "  PostgreSQL not running, skipping database backup"
  fi
}

# Backup configuration
backup_config() {
  local dest_dir="$1"
  local config_backup_dir="$dest_dir/config"
  mkdir -p "$config_backup_dir"
  
  log_info "  • Backing up configuration files..."
  
  # Backup environment files
  for env_file in .env .env.local .env.production; do
    if [[ -f "$env_file" ]]; then
      cp "$env_file" "$config_backup_dir/"
    fi
  done
  
  # Backup docker-compose files
  for compose_file in docker-compose.yml docker-compose.override.yml docker-compose.prod.yml; do
    if [[ -f "$compose_file" ]]; then
      cp "$compose_file" "$config_backup_dir/"
    fi
  done
  
  # Backup nginx config if exists
  if [[ -d "./nginx" ]]; then
    cp -r ./nginx "$config_backup_dir/"
  fi
}

# Backup Docker volumes
backup_volumes() {
  local dest_dir="$1"
  local volumes_backup_dir="$dest_dir/volumes"
  mkdir -p "$volumes_backup_dir"
  
  log_info "  • Backing up Docker volumes..."
  
  # Get list of nself volumes
  local volumes=$(docker volume ls --format "{{.Name}}" | grep -E "^${PROJECT_NAME:-nself}" || true)
  
  if [[ -n "$volumes" ]]; then
    for volume in $volumes; do
      local volume_name=$(echo "$volume" | sed "s/${PROJECT_NAME:-nself}_//")
      log_info "    - Volume: $volume_name"
      
      # Create temporary container to export volume
      docker run --rm -v "$volume:/data" -v "$volumes_backup_dir:/backup" \
        alpine tar -czf "/backup/${volume_name}.tar.gz" -C /data . 2>/dev/null || {
        log_warning "    Failed to backup volume: $volume"
      }
    done
  else
    log_info "    No volumes found"
  fi
}

# Backup certificates
backup_certificates() {
  local dest_dir="$1"
  
  if [[ -d "./certs" ]]; then
    log_info "  • Backing up SSL certificates..."
    cp -r ./certs "$dest_dir/"
  fi
}

# List backups
cmd_backup_list() {
  show_command_header "nself backup list" "Available backups"
  
  # Local backups
  echo -e "${COLOR_BOLD}Local Backups:${COLOR_RESET}"
  echo ""
  
  if [[ -d "$BACKUP_DIR" ]] && [[ -n "$(ls -A "$BACKUP_DIR" 2>/dev/null)" ]]; then
    printf "%-40s %-10s %-20s\n" "Name" "Size" "Created"
    printf "%-40s %-10s %-20s\n" "────" "────" "───────"
    
    for backup in "$BACKUP_DIR"/*.tar.gz; do
      if [[ -f "$backup" ]]; then
        local name=$(basename "$backup")
        local size=$(du -h "$backup" | cut -f1)
        local created=$(stat -f "%Sm" -t "%Y-%m-%d %H:%M" "$backup" 2>/dev/null || stat -c "%y" "$backup" 2>/dev/null | cut -d' ' -f1,2)
        printf "%-40s %-10s %-20s\n" "$name" "$size" "$created"
      fi
    done
  else
    echo "  No local backups found"
  fi
  
  echo ""
  
  # S3 backups if configured
  if [[ -n "$S3_BUCKET" ]]; then
    echo -e "${COLOR_BOLD}S3 Backups:${COLOR_RESET}"
    echo ""
    
    if command -v aws >/dev/null 2>&1; then
      aws s3 ls "s3://$S3_BUCKET/nself-backups/" --human-readable 2>/dev/null || {
        echo "  Unable to list S3 backups"
      }
    else
      echo "  AWS CLI not installed (needed for S3 backups)"
    fi
    echo ""
  fi
}

# Restore backup
cmd_backup_restore() {
  local backup_name="$1"
  local restore_type="${2:-full}"
  
  if [[ -z "$backup_name" ]]; then
    log_error "Backup name required"
    echo "Usage: nself backup restore <backup_name> [full|database|config]"
    return 1
  fi
  
  show_command_header "nself backup restore" "Restore from backup"
  
  # Find backup file
  local backup_path=""
  if [[ -f "$BACKUP_DIR/$backup_name" ]]; then
    backup_path="$BACKUP_DIR/$backup_name"
  elif [[ -f "$backup_name" ]]; then
    backup_path="$backup_name"
  else
    log_error "Backup not found: $backup_name"
    return 1
  fi
  
  log_warning "⚠️  This will overwrite existing data!"
  echo -n "Are you sure you want to restore from $backup_name? (y/N): "
  read -r confirm
  if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
    log_info "Restore cancelled"
    return 0
  fi
  
  # Create temporary directory for extraction
  local temp_dir=$(mktemp -d)
  
  log_info "Extracting backup..."
  tar -xzf "$backup_path" -C "$temp_dir"
  
  # Restore components based on type
  case "$restore_type" in
    full)
      log_info "Restoring all components..."
      restore_database "$temp_dir"
      restore_config "$temp_dir"
      restore_volumes "$temp_dir"
      restore_certificates "$temp_dir"
      ;;
    database)
      log_info "Restoring database only..."
      restore_database "$temp_dir"
      ;;
    config)
      log_info "Restoring configuration only..."
      restore_config "$temp_dir"
      ;;
    *)
      log_error "Unknown restore type: $restore_type"
      rm -rf "$temp_dir"
      return 1
      ;;
  esac
  
  # Cleanup
  rm -rf "$temp_dir"
  
  echo ""
  log_success "Restore completed successfully!"
  echo ""
  log_info "Next steps:"
  echo "  1. Run 'nself restart' to apply restored configuration"
  echo "  2. Run 'nself status' to verify services"
  echo "  3. Run 'nself doctor' to check system health"
  echo ""
}

# Restore database
restore_database() {
  local source_dir="$1"
  
  if [[ -f "$source_dir/database/postgres_dump.sql" ]]; then
    log_info "  • Restoring PostgreSQL database..."
    
    # Ensure postgres is running
    if ! docker ps --format "{{.Names}}" | grep -q postgres; then
      log_info "    Starting PostgreSQL..."
      docker compose up -d postgres
      sleep 5
    fi
    
    # Restore database
    local db_user="${POSTGRES_USER:-postgres}"
    docker exec -i postgres psql -U "$db_user" < "$source_dir/database/postgres_dump.sql" 2>/dev/null || {
      log_error "    Failed to restore database"
      return 1
    }
    
    # Restore Hasura metadata if available
    if [[ -d "$source_dir/database/hasura_metadata" ]]; then
      log_info "  • Restoring Hasura metadata..."
      rm -rf ./hasura/metadata
      cp -r "$source_dir/database/hasura_metadata" ./hasura/metadata
    fi
  else
    log_warning "  No database backup found in archive"
  fi
}

# Restore configuration
restore_config() {
  local source_dir="$1"
  
  if [[ -d "$source_dir/config" ]]; then
    log_info "  • Restoring configuration files..."
    
    # Backup existing config first
    for file in .env .env.local .env.production; do
      if [[ -f "$file" ]]; then
        cp "$file" "${file}.backup-$(date +%Y%m%d_%H%M%S)"
      fi
    done
    
    # Restore config files
    cp -f "$source_dir/config"/.env* . 2>/dev/null || true
    cp -f "$source_dir/config"/docker-compose*.yml . 2>/dev/null || true
    
    if [[ -d "$source_dir/config/nginx" ]]; then
      rm -rf ./nginx
      cp -r "$source_dir/config/nginx" ./
    fi
  else
    log_warning "  No configuration backup found in archive"
  fi
}

# Restore volumes
restore_volumes() {
  local source_dir="$1"
  
  if [[ -d "$source_dir/volumes" ]]; then
    log_info "  • Restoring Docker volumes..."
    
    for volume_archive in "$source_dir/volumes"/*.tar.gz; do
      if [[ -f "$volume_archive" ]]; then
        local volume_name=$(basename "$volume_archive" .tar.gz)
        local full_volume_name="${PROJECT_NAME:-nself}_${volume_name}"
        
        log_info "    - Restoring volume: $volume_name"
        
        # Create volume if it doesn't exist
        docker volume create "$full_volume_name" >/dev/null 2>&1 || true
        
        # Restore volume data
        docker run --rm -v "$full_volume_name:/data" -v "$source_dir/volumes:/backup" \
          alpine tar -xzf "/backup/${volume_name}.tar.gz" -C /data 2>/dev/null || {
          log_warning "    Failed to restore volume: $volume_name"
        }
      fi
    done
  else
    log_info "  No volume backups found in archive"
  fi
}

# Restore certificates
restore_certificates() {
  local source_dir="$1"
  
  if [[ -d "$source_dir/certs" ]]; then
    log_info "  • Restoring SSL certificates..."
    rm -rf ./certs
    cp -r "$source_dir/certs" ./
  fi
}

# Prune old backups
cmd_backup_prune() {
  local days="${1:-$BACKUP_RETENTION_DAYS}"
  
  show_command_header "nself backup prune" "Remove old backups"
  
  log_info "Removing backups older than $days days..."
  echo ""
  
  local count=0
  local freed_space=0
  
  # Find and remove old backups
  if [[ -d "$BACKUP_DIR" ]]; then
    while IFS= read -r backup; do
      if [[ -f "$backup" ]]; then
        local size=$(du -k "$backup" | cut -f1)
        local name=$(basename "$backup")
        
        log_info "  Removing: $name"
        rm -f "$backup"
        
        count=$((count + 1))
        freed_space=$((freed_space + size))
      fi
    done < <(find "$BACKUP_DIR" -name "*.tar.gz" -type f -mtime +${days})
  fi
  
  # Summary
  echo ""
  if [[ $count -gt 0 ]]; then
    local freed_mb=$((freed_space / 1024))
    log_success "Removed $count old backup(s), freed ${freed_mb}MB"
  else
    log_info "No backups older than $days days found"
  fi
  echo ""
}

# Upload to S3
upload_to_s3() {
  local file_path="$1"
  local file_name="$2"
  
  log_info "Uploading to S3..."
  
  if command -v aws >/dev/null 2>&1; then
    local s3_path="s3://$S3_BUCKET/nself-backups/$file_name"
    
    if [[ -n "$S3_ENDPOINT" ]]; then
      aws s3 cp "$file_path" "$s3_path" --endpoint-url "$S3_ENDPOINT" || {
        log_warning "  Failed to upload to S3"
        return 1
      }
    else
      aws s3 cp "$file_path" "$s3_path" || {
        log_warning "  Failed to upload to S3"
        return 1
      }
    fi
    
    log_success "  Uploaded to S3: $s3_path"
  else
    log_warning "  AWS CLI not installed, skipping S3 upload"
  fi
}

# Show help
show_backup_help() {
  echo "Usage: nself backup <command> [options]"
  echo ""
  echo "Commands:"
  echo "  create [type] [name]   Create a backup (types: full, database, config)"
  echo "  list                   List available backups"
  echo "  restore <name> [type]  Restore from backup"
  echo "  prune [days]           Remove backups older than N days (default: 30)"
  echo ""
  echo "Environment Variables:"
  echo "  BACKUP_DIR             Directory for backups (default: ./backups)"
  echo "  BACKUP_RETENTION_DAYS  Days to keep backups (default: 30)"
  echo "  S3_BUCKET              S3 bucket for remote backups (optional)"
  echo "  S3_ENDPOINT            S3 endpoint for MinIO/compatible (optional)"
  echo ""
  echo "Examples:"
  echo "  nself backup create                    # Create full backup"
  echo "  nself backup create database            # Database only"
  echo "  nself backup list                       # Show all backups"
  echo "  nself backup restore backup_20240101.tar.gz  # Restore backup"
  echo "  nself backup prune 7                    # Remove backups older than 7 days"
}

# Main command router
cmd_backup() {
  local subcommand="${1:-help}"
  shift || true
  
  case "$subcommand" in
    create)
      cmd_backup_create "$@"
      ;;
    list|ls)
      cmd_backup_list "$@"
      ;;
    restore)
      cmd_backup_restore "$@"
      ;;
    prune|clean)
      cmd_backup_prune "$@"
      ;;
    help|-h|--help)
      show_backup_help
      ;;
    *)
      log_error "Unknown backup command: $subcommand"
      show_backup_help
      return 1
      ;;
  esac
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "backup" || exit $?
  cmd_backup "$@"
  exit_code=$?
  post_command "backup" $exit_code
  exit $exit_code
fi