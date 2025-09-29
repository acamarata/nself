#!/usr/bin/env bash

# backup.sh - Comprehensive backup and restore for nself

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/docker.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Backup configuration - Support both old and new naming for backward compatibility
BACKUP_ENABLED="${BACKUP_ENABLED:-${DB_BACKUP_ENABLED:-false}}"
BACKUP_SCHEDULE="${BACKUP_SCHEDULE:-${DB_BACKUP_SCHEDULE:-}}"
BACKUP_DIR="${BACKUP_DIR:-./backups}"
BACKUP_RETENTION_DAYS="${BACKUP_RETENTION_DAYS:-${DB_BACKUP_RETENTION_DAYS:-30}}"
BACKUP_STORAGE="${BACKUP_STORAGE:-${DB_BACKUP_STORAGE:-local}}"
BACKUP_TYPES="${BACKUP_TYPES:-${DB_BACKUP_TYPES:-database}}"
BACKUP_COMPRESSION="${BACKUP_COMPRESSION:-${DB_BACKUP_COMPRESSION:-true}}"
BACKUP_ENCRYPTION="${BACKUP_ENCRYPTION:-${DB_BACKUP_ENCRYPTION:-false}}"
BACKUP_RETENTION_MIN="${BACKUP_RETENTION_MIN:-3}"  # Minimum backups to keep
BACKUP_RETENTION_WEEKLY="${BACKUP_RETENTION_WEEKLY:-4}"  # Weekly backups to keep
BACKUP_RETENTION_MONTHLY="${BACKUP_RETENTION_MONTHLY:-12}"  # Monthly backups to keep

# Cloud storage configuration
BACKUP_CLOUD_PROVIDER="${BACKUP_CLOUD_PROVIDER:-}"  # s3, dropbox, gdrive, onedrive, rclone
S3_BUCKET="${S3_BUCKET:-}"
S3_ENDPOINT="${S3_ENDPOINT:-}"  # For MinIO/S3-compatible
DROPBOX_TOKEN="${DROPBOX_TOKEN:-}"
DROPBOX_FOLDER="${DROPBOX_FOLDER:-/nself-backups}"
GDRIVE_FOLDER_ID="${GDRIVE_FOLDER_ID:-}"
ONEDRIVE_FOLDER="${ONEDRIVE_FOLDER:-nself-backups}"
RCLONE_REMOTE="${RCLONE_REMOTE:-}"  # Name of rclone remote
RCLONE_PATH="${RCLONE_PATH:-nself-backups}"  # Path on remote

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
  
  # Upload to cloud if configured
  upload_to_cloud "$backup_path" "$backup_name"
  
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
  for env_file in .env .env.dev .env.production; do
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
    for file in .env .env.dev .env.production; do
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

# Prune old backups with advanced retention
cmd_backup_prune() {
  local policy="${1:-age}"  # age, gfs (grandfather-father-son), smart
  local days="${2:-$BACKUP_RETENTION_DAYS}"
  
  show_command_header "nself backup prune" "Remove old backups"
  
  case "$policy" in
    age)
      prune_by_age "$days"
      ;;
    gfs)
      prune_gfs_policy
      ;;
    smart)
      prune_smart_policy
      ;;
    cloud)
      prune_cloud_backups "$days"
      ;;
    *)
      log_error "Unknown prune policy: $policy"
      log_info "Valid policies: age, gfs, smart, cloud"
      return 1
      ;;
  esac
}

# Simple age-based pruning
prune_by_age() {
  local days="$1"
  
  log_info "Removing backups older than $days days..."
  echo ""
  
  local count=0
  local freed_space=0
  local kept_count=0
  
  # Count total backups
  local total_backups=$(find "$BACKUP_DIR" -name "*.tar.gz" -type f 2>/dev/null | wc -l)
  
  # Find old backups
  if [[ -d "$BACKUP_DIR" ]]; then
    while IFS= read -r backup; do
      if [[ -f "$backup" ]]; then
        # Check if we should keep minimum backups
        if [[ $((total_backups - count)) -le ${BACKUP_RETENTION_MIN:-3} ]]; then
          log_info "  Keeping (minimum retention): $(basename "$backup")"
          kept_count=$((kept_count + 1))
        else
          local size=$(du -k "$backup" | cut -f1)
          local name=$(basename "$backup")
          
          log_info "  Removing: $name"
          rm -f "$backup"
          
          count=$((count + 1))
          freed_space=$((freed_space + size))
        fi
      fi
    done < <(find "$BACKUP_DIR" -name "*.tar.gz" -type f -mtime +${days} | sort)
  fi
  
  # Summary
  echo ""
  if [[ $count -gt 0 ]]; then
    local freed_mb=$((freed_space / 1024))
    log_success "Removed $count backup(s), freed ${freed_mb}MB"
  fi
  if [[ $kept_count -gt 0 ]]; then
    log_info "Kept $kept_count backup(s) (minimum retention policy)"
  fi
  if [[ $count -eq 0 ]] && [[ $kept_count -eq 0 ]]; then
    log_info "No backups older than $days days found"
  fi
  echo ""
}

# Grandfather-Father-Son retention policy
prune_gfs_policy() {
  log_info "Applying GFS retention policy..."
  echo ""
  log_info "  Keeping: Last 7 daily, 4 weekly, 12 monthly backups"
  echo ""
  
  local daily="${BACKUP_RETENTION_DAILY:-7}"
  local weekly="${BACKUP_RETENTION_WEEKLY:-4}"
  local monthly="${BACKUP_RETENTION_MONTHLY:-12}"
  
  # This is a complex policy - for now, keep recent + sample older
  # In production, this would analyze backup dates and keep strategic samples
  
  local count=0
  local kept=0
  
  # Sort backups by date (newest first)
  local backups=($(ls -t "$BACKUP_DIR"/*.tar.gz 2>/dev/null))
  
  for i in "${!backups[@]}"; do
    local backup="${backups[$i]}"
    local name=$(basename "$backup")
    
    if [[ $i -lt $daily ]]; then
      # Keep daily backups
      log_info "  [DAILY] Keeping: $name"
      kept=$((kept + 1))
    elif [[ $((i % 7)) -eq 0 ]] && [[ $i -lt $((daily + weekly * 7)) ]]; then
      # Keep weekly backups
      log_info "  [WEEKLY] Keeping: $name"
      kept=$((kept + 1))
    elif [[ $((i % 30)) -eq 0 ]] && [[ $i -lt $((daily + weekly * 7 + monthly * 30)) ]]; then
      # Keep monthly backups
      log_info "  [MONTHLY] Keeping: $name"
      kept=$((kept + 1))
    else
      # Remove old backup
      log_info "  Removing: $name"
      rm -f "$backup"
      count=$((count + 1))
    fi
  done
  
  echo ""
  log_success "GFS policy applied: Kept $kept, removed $count backup(s)"
  echo ""
}

# Smart retention policy (keeps important backups)
prune_smart_policy() {
  log_info "Applying smart retention policy..."
  echo ""
  
  # Keep backups based on importance:
  # - All backups from last 24 hours
  # - Daily backups for last week
  # - Weekly backups for last month
  # - Monthly backups for last year
  # - Yearly backups forever
  
  local now=$(date +%s)
  local count=0
  local kept=0
  
  if [[ -d "$BACKUP_DIR" ]]; then
    for backup in "$BACKUP_DIR"/*.tar.gz; do
      if [[ -f "$backup" ]]; then
        local name=$(basename "$backup")
        local backup_time=$(stat -f %m "$backup" 2>/dev/null || stat -c %Y "$backup")
        local age_days=$(( (now - backup_time) / 86400 ))
        
        local keep=false
        local reason=""
        
        if [[ $age_days -le 1 ]]; then
          keep=true
          reason="last 24 hours"
        elif [[ $age_days -le 7 ]]; then
          keep=true
          reason="last week"
        elif [[ $age_days -le 30 ]] && [[ $((age_days % 7)) -eq 0 ]]; then
          keep=true
          reason="weekly (last month)"
        elif [[ $age_days -le 365 ]] && [[ $((age_days % 30)) -eq 0 ]]; then
          keep=true
          reason="monthly (last year)"
        elif [[ $((age_days % 365)) -eq 0 ]]; then
          keep=true
          reason="yearly"
        fi
        
        if [[ "$keep" == true ]]; then
          log_info "  Keeping ($reason): $name"
          kept=$((kept + 1))
        else
          log_info "  Removing: $name"
          rm -f "$backup"
          count=$((count + 1))
        fi
      fi
    done
  fi
  
  echo ""
  log_success "Smart policy applied: Kept $kept, removed $count backup(s)"
  echo ""
}

# Prune cloud backups
prune_cloud_backups() {
  local days="${1:-30}"
  
  log_info "Pruning cloud backups older than $days days..."
  echo ""
  
  local provider="${BACKUP_CLOUD_PROVIDER:-}"
  
  case "$provider" in
    s3)
      if command -v aws >/dev/null 2>&1; then
        log_info "Pruning S3 backups..."
        # List and delete old S3 objects
        local cutoff_date=$(date -d "$days days ago" +%Y-%m-%d 2>/dev/null || date -v -${days}d +%Y-%m-%d)
        aws s3 ls "s3://$S3_BUCKET/nself-backups/" | while read -r line; do
          local file_date=$(echo "$line" | awk '{print $1}')
          local file_name=$(echo "$line" | awk '{print $4}')
          if [[ "$file_date" < "$cutoff_date" ]]; then
            log_info "  Removing from S3: $file_name"
            aws s3 rm "s3://$S3_BUCKET/nself-backups/$file_name"
          fi
        done
      fi
      ;;
    rclone)
      if command -v rclone >/dev/null 2>&1; then
        log_info "Pruning rclone backups..."
        rclone delete "${RCLONE_REMOTE}:${RCLONE_PATH}" --min-age "${days}d"
      fi
      ;;
    *)
      log_warning "Cloud pruning not implemented for: $provider"
      ;;
  esac
  
  echo ""
  log_success "Cloud pruning complete"
  echo ""
}

# Universal cloud upload function
upload_to_cloud() {
  local file_path="$1"
  local file_name="$2"
  
  # Determine which cloud provider to use
  local provider="${BACKUP_CLOUD_PROVIDER:-}"
  
  # Auto-detect if not specified
  if [[ -z "$provider" ]]; then
    if [[ -n "$S3_BUCKET" ]]; then
      provider="s3"
    elif [[ -n "$DROPBOX_TOKEN" ]]; then
      provider="dropbox"
    elif [[ -n "$GDRIVE_FOLDER_ID" ]]; then
      provider="gdrive"
    elif [[ -n "$RCLONE_REMOTE" ]]; then
      provider="rclone"
    fi
  fi
  
  # Upload based on provider
  case "$provider" in
    s3)
      upload_to_s3 "$file_path" "$file_name"
      ;;
    dropbox)
      upload_to_dropbox "$file_path" "$file_name"
      ;;
    gdrive)
      upload_to_gdrive "$file_path" "$file_name"
      ;;
    onedrive)
      upload_to_onedrive "$file_path" "$file_name"
      ;;
    rclone)
      upload_to_rclone "$file_path" "$file_name"
      ;;
    "")
      # No cloud provider configured, skip
      ;;
    *)
      log_warning "Unknown cloud provider: $provider"
      ;;
  esac
}

# Upload to Dropbox
upload_to_dropbox() {
  local file_path="$1"
  local file_name="$2"
  
  if [[ -z "$DROPBOX_TOKEN" ]]; then
    log_warning "Dropbox token not configured"
    return 1
  fi
  
  log_info "Uploading to Dropbox..."
  
  local dropbox_path="${DROPBOX_FOLDER}/${file_name}"
  
  # Use Dropbox API
  if command -v curl >/dev/null 2>&1; then
    local response=$(curl -s -X POST https://content.dropboxapi.com/2/files/upload \
      --header "Authorization: Bearer ${DROPBOX_TOKEN}" \
      --header "Dropbox-API-Arg: {\"path\": \"${dropbox_path}\", \"mode\": \"overwrite\"}" \
      --header "Content-Type: application/octet-stream" \
      --data-binary @"${file_path}" 2>&1)
    
    if echo "$response" | grep -q "error"; then
      log_error "Dropbox upload failed: $response"
      return 1
    else
      log_success "  Uploaded to Dropbox: $dropbox_path"
    fi
  else
    log_warning "curl not found, cannot upload to Dropbox"
    return 1
  fi
}

# Upload to Google Drive
upload_to_gdrive() {
  local file_path="$1"
  local file_name="$2"
  
  log_info "Uploading to Google Drive..."
  
  # Check for gdrive CLI tool
  if command -v gdrive >/dev/null 2>&1; then
    if [[ -n "$GDRIVE_FOLDER_ID" ]]; then
      gdrive upload --parent "$GDRIVE_FOLDER_ID" "$file_path" || {
        log_warning "Failed to upload to Google Drive"
        return 1
      }
    else
      gdrive upload "$file_path" || {
        log_warning "Failed to upload to Google Drive"
        return 1
      }
    fi
    log_success "  Uploaded to Google Drive: $file_name"
  else
    log_warning "gdrive CLI not installed. Install from: https://github.com/prasmussen/gdrive"
    log_info "  Or use rclone instead: nself backup cloud setup"
    return 1
  fi
}

# Upload to OneDrive
upload_to_onedrive() {
  local file_path="$1"
  local file_name="$2"
  
  log_info "Uploading to OneDrive..."
  
  # OneDrive requires rclone or similar tool
  if command -v rclone >/dev/null 2>&1; then
    rclone copy "$file_path" "onedrive:${ONEDRIVE_FOLDER}/" || {
      log_warning "Failed to upload to OneDrive"
      return 1
    }
    log_success "  Uploaded to OneDrive: ${ONEDRIVE_FOLDER}/$file_name"
  else
    log_warning "rclone not installed. OneDrive requires rclone."
    log_info "  Run: nself backup cloud setup"
    return 1
  fi
}

# Upload using rclone (supports 40+ providers)
upload_to_rclone() {
  local file_path="$1"
  local file_name="$2"
  
  if [[ -z "$RCLONE_REMOTE" ]]; then
    log_warning "rclone remote not configured"
    return 1
  fi
  
  log_info "Uploading via rclone to $RCLONE_REMOTE..."
  
  if command -v rclone >/dev/null 2>&1; then
    rclone copy "$file_path" "${RCLONE_REMOTE}:${RCLONE_PATH}/" || {
      log_warning "Failed to upload via rclone"
      return 1
    }
    log_success "  Uploaded via rclone: ${RCLONE_REMOTE}:${RCLONE_PATH}/$file_name"
  else
    log_warning "rclone not installed"
    log_info "  Run: nself backup cloud setup"
    return 1
  fi
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

# Cloud setup wizard
cmd_backup_cloud() {
  local action="${1:-setup}"
  
  case "$action" in
    setup)
      backup_cloud_setup
      ;;
    status)
      backup_cloud_status
      ;;
    test)
      backup_cloud_test
      ;;
    *)
      log_error "Unknown cloud action: $action"
      echo "Usage: nself backup cloud [setup|status|test]"
      return 1
      ;;
  esac
}

# Setup cloud backup provider
backup_cloud_setup() {
  show_command_header "nself backup cloud setup" "Configure cloud backup provider"
  
  echo "Select cloud provider:"
  echo "  1) Amazon S3"
  echo "  2) Dropbox"
  echo "  3) Google Drive"
  echo "  4) OneDrive"
  echo "  5) rclone (supports 40+ providers)"
  echo "  6) None (disable cloud backups)"
  echo ""
  echo -n "Choice [1-6]: "
  read choice
  
  case "$choice" in
    1)
      setup_s3
      ;;
    2)
      setup_dropbox
      ;;
    3)
      setup_gdrive
      ;;
    4)
      setup_onedrive
      ;;
    5)
      setup_rclone
      ;;
    6)
      echo "BACKUP_CLOUD_PROVIDER=" >> .env
      log_info "Cloud backups disabled"
      ;;
    *)
      log_error "Invalid choice"
      return 1
      ;;
  esac
}

# Setup S3
setup_s3() {
  log_info "Setting up Amazon S3..."
  echo ""
  
  echo -n "S3 bucket name: "
  read s3_bucket
  
  echo -n "S3 endpoint (leave empty for AWS): "
  read s3_endpoint
  
  echo -n "AWS Access Key ID: "
  read aws_key
  
  echo -n "AWS Secret Access Key: "
  read -s aws_secret
  echo ""
  
  # Save configuration
  {
    echo "BACKUP_CLOUD_PROVIDER=s3"
    echo "S3_BUCKET=$s3_bucket"
    [[ -n "$s3_endpoint" ]] && echo "S3_ENDPOINT=$s3_endpoint"
    echo "AWS_ACCESS_KEY_ID=$aws_key"
    echo "AWS_SECRET_ACCESS_KEY=$aws_secret"
  } >> .env
  
  log_success "S3 configuration saved"
  echo ""
  log_info "Testing S3 connection..."
  backup_cloud_test
}

# Setup Dropbox
setup_dropbox() {
  log_info "Setting up Dropbox..."
  echo ""
  echo "To use Dropbox backup, you need an access token:"
  echo "  1. Go to https://www.dropbox.com/developers/apps"
  echo "  2. Create a new app (scoped access)"
  echo "  3. Generate an access token"
  echo ""
  echo -n "Dropbox access token: "
  read -s dropbox_token
  echo ""
  
  echo -n "Dropbox folder path [/nself-backups]: "
  read dropbox_folder
  dropbox_folder="${dropbox_folder:-/nself-backups}"
  
  # Save configuration
  {
    echo "BACKUP_CLOUD_PROVIDER=dropbox"
    echo "DROPBOX_TOKEN=$dropbox_token"
    echo "DROPBOX_FOLDER=$dropbox_folder"
  } >> .env
  
  log_success "Dropbox configuration saved"
  echo ""
  log_info "Testing Dropbox connection..."
  backup_cloud_test
}

# Setup Google Drive
setup_gdrive() {
  log_info "Setting up Google Drive..."
  echo ""
  
  # Check if gdrive is installed
  if ! command -v gdrive >/dev/null 2>&1; then
    log_warning "gdrive CLI not installed"
    echo ""
    echo "Install gdrive CLI:"
    echo "  1. Download from: https://github.com/prasmussen/gdrive"
    echo "  2. Follow authentication instructions"
    echo ""
    echo "Or use rclone instead (option 5)"
    return 1
  fi
  
  echo "Authenticating with Google Drive..."
  gdrive list
  
  echo -n "Google Drive folder ID (optional): "
  read gdrive_folder
  
  # Save configuration
  {
    echo "BACKUP_CLOUD_PROVIDER=gdrive"
    [[ -n "$gdrive_folder" ]] && echo "GDRIVE_FOLDER_ID=$gdrive_folder"
  } >> .env
  
  log_success "Google Drive configuration saved"
}

# Setup OneDrive
setup_onedrive() {
  log_info "Setting up OneDrive..."
  echo ""
  echo "OneDrive requires rclone. Setting up rclone for OneDrive..."
  echo ""
  
  # Install rclone if needed
  if ! command -v rclone >/dev/null 2>&1; then
    log_info "Installing rclone..."
    curl https://rclone.org/install.sh | sudo bash
  fi
  
  log_info "Configuring rclone for OneDrive..."
  rclone config create onedrive onedrive
  
  echo -n "OneDrive folder path [nself-backups]: "
  read onedrive_folder
  onedrive_folder="${onedrive_folder:-nself-backups}"
  
  # Save configuration
  {
    echo "BACKUP_CLOUD_PROVIDER=onedrive"
    echo "ONEDRIVE_FOLDER=$onedrive_folder"
  } >> .env
  
  log_success "OneDrive configuration saved"
}

# Setup rclone
setup_rclone() {
  log_info "Setting up rclone..."
  echo ""
  
  # Install rclone if needed
  if ! command -v rclone >/dev/null 2>&1; then
    log_info "Installing rclone..."
    curl https://rclone.org/install.sh | sudo bash
  fi
  
  log_info "Running rclone configuration wizard..."
  echo "rclone supports 40+ cloud providers including:"
  echo "  Box, Dropbox, Google Drive, OneDrive, MEGA, pCloud, etc."
  echo ""
  rclone config
  
  echo -n "rclone remote name: "
  read rclone_remote
  
  echo -n "Remote path [nself-backups]: "
  read rclone_path
  rclone_path="${rclone_path:-nself-backups}"
  
  # Save configuration
  {
    echo "BACKUP_CLOUD_PROVIDER=rclone"
    echo "RCLONE_REMOTE=$rclone_remote"
    echo "RCLONE_PATH=$rclone_path"
  } >> .env
  
  log_success "rclone configuration saved"
}

# Show cloud backup status
backup_cloud_status() {
  show_command_header "nself backup cloud status" "Cloud backup configuration"
  
  local provider="${BACKUP_CLOUD_PROVIDER:-none}"
  
  echo "Provider: $provider"
  echo ""
  
  case "$provider" in
    s3)
      echo "S3 Bucket: ${S3_BUCKET:-not configured}"
      echo "S3 Endpoint: ${S3_ENDPOINT:-AWS}"
      echo "AWS Key: ${AWS_ACCESS_KEY_ID:+configured}"
      ;;
    dropbox)
      echo "Dropbox Token: ${DROPBOX_TOKEN:+configured}"
      echo "Dropbox Folder: ${DROPBOX_FOLDER:-/nself-backups}"
      ;;
    gdrive)
      echo "Google Drive: ${GDRIVE_FOLDER_ID:-root folder}"
      ;;
    onedrive)
      echo "OneDrive Folder: ${ONEDRIVE_FOLDER:-nself-backups}"
      ;;
    rclone)
      echo "rclone Remote: ${RCLONE_REMOTE:-not configured}"
      echo "rclone Path: ${RCLONE_PATH:-nself-backups}"
      ;;
    none)
      echo "No cloud provider configured"
      echo "Run 'nself backup cloud setup' to configure"
      ;;
  esac
  echo ""
}

# Test cloud backup connection
backup_cloud_test() {
  show_command_header "nself backup cloud test" "Testing cloud connection"
  
  local test_file=$(mktemp)
  echo "nself backup test" > "$test_file"
  local test_name="test_$(date +%s).txt"
  
  upload_to_cloud "$test_file" "$test_name"
  local result=$?
  
  rm -f "$test_file"
  
  if [[ $result -eq 0 ]]; then
    log_success "Cloud backup test successful!"
  else
    log_error "Cloud backup test failed"
  fi
  echo ""
}

# Schedule automatic backups
cmd_backup_schedule() {
  local frequency="${1:-daily}"
  
  show_command_header "nself backup schedule" "Setup automatic backups"
  
  log_info "Setting up $frequency backups..."
  echo ""
  
  # Create backup script
  local backup_script="/usr/local/bin/nself-backup"
  cat > "$backup_script" << 'EOF'
#!/bin/bash
cd $(dirname $(nself which))
nself backup create full
nself backup prune smart
EOF
  chmod +x "$backup_script"
  
  # Setup cron job
  local cron_entry=""
  case "$frequency" in
    hourly)
      cron_entry="0 * * * * $backup_script"
      ;;
    daily)
      cron_entry="0 3 * * * $backup_script"
      ;;
    weekly)
      cron_entry="0 3 * * 0 $backup_script"
      ;;
    monthly)
      cron_entry="0 3 1 * * $backup_script"
      ;;
    *)
      log_error "Invalid frequency: $frequency"
      echo "Valid options: hourly, daily, weekly, monthly"
      return 1
      ;;
  esac
  
  # Add to crontab
  (crontab -l 2>/dev/null | grep -v "nself-backup"; echo "$cron_entry") | crontab -
  
  log_success "Scheduled $frequency backups"
  echo ""
  log_info "View schedule: crontab -l"
  log_info "Remove schedule: crontab -e (delete nself-backup line)"
  echo ""
}

# Show help
show_backup_help() {
  echo "Usage: nself backup <command> [options]"
  echo ""
  echo "Commands:"
  echo "  create [type] [name]     Create a backup (types: full, database, config)"
  echo "  list                     List available backups"
  echo "  restore <name> [type]    Restore from backup"
  echo "  prune [policy] [days]    Remove old backups"
  echo "                           Policies: age, gfs, smart, cloud"
  echo "  cloud [action]           Manage cloud backups"
  echo "                           Actions: setup, status, test"
  echo "  schedule [frequency]     Schedule automatic backups"
  echo "                           Frequencies: hourly, daily, weekly, monthly"
  echo ""
  echo "Environment Variables:"
  echo "  BACKUP_DIR                Directory for backups (default: ./backups)"
  echo "  BACKUP_RETENTION_DAYS     Days to keep backups (default: 30)"
  echo "  BACKUP_RETENTION_MIN      Minimum backups to keep (default: 3)"
  echo "  BACKUP_CLOUD_PROVIDER     Cloud provider: s3, dropbox, gdrive, onedrive, rclone"
  echo ""
  echo "Cloud Provider Variables:"
  echo "  S3: S3_BUCKET, S3_ENDPOINT, AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY"
  echo "  Dropbox: DROPBOX_TOKEN, DROPBOX_FOLDER"
  echo "  Google Drive: GDRIVE_FOLDER_ID"
  echo "  OneDrive: ONEDRIVE_FOLDER"
  echo "  rclone: RCLONE_REMOTE, RCLONE_PATH"
  echo ""
  echo "Examples:"
  echo "  nself backup create                      # Create full backup"
  echo "  nself backup create database my-backup   # Database backup with custom name"
  echo "  nself backup list                        # Show all backups"
  echo "  nself backup restore backup.tar.gz       # Restore full backup"
  echo "  nself backup prune age 7                 # Remove backups older than 7 days"
  echo "  nself backup prune gfs                   # Apply GFS retention policy"
  echo "  nself backup prune smart                 # Apply smart retention policy"
  echo "  nself backup cloud setup                 # Configure cloud provider"
  echo "  nself backup schedule daily              # Schedule daily backups"
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
    cloud)
      cmd_backup_cloud "$@"
      ;;
    schedule)
      cmd_backup_schedule "$@"
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