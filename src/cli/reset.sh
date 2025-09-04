#!/usr/bin/env bash
set -euo pipefail

# reset.sh - Reset project to clean state

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/header.sh"

# Command function
cmd_reset() {
  local force_reset=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --force | -f)
      force_reset=true
      shift
      ;;
    --help | -h)
      show_reset_help
      return 0
      ;;
    *)
      log_error "Unknown option: $1"
      show_reset_help
      return 1
      ;;
    esac
  done

  # Show standardized header
  show_command_header "nself reset" "Reset project to clean state"

  echo -e "${COLOR_YELLOW}⚠${COLOR_RESET}  This will:"
  echo "  • Stop and remove all containers"
  echo "  • Delete all Docker volumes"
  echo "  • Remove all generated files"
  echo "  • Backup env files with .old suffix"
  echo

  if [[ "$force_reset" != "true" ]]; then
    read -p "Are you sure you want to reset everything? [y/N]: " -n 1 -r
    echo

    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      log_info "Reset cancelled"
      return 1
    fi
  fi

  echo

  # Get project name first
  local project="${PROJECT_NAME:-myproject}"
  if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
    set -a
    load_env_with_priority 2>/dev/null || true
    set +a
    project="${PROJECT_NAME:-myproject}"
  fi

  # Stopping services
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Stopping services..."

  # First try docker-compose down if file exists
  if [[ -f "docker-compose.yml" ]]; then
    docker compose down -v >/dev/null 2>&1 || true
  fi

  # Then forcefully stop and remove ALL project containers
  local container_count=$(docker ps -aq --filter "name=${project}" | wc -l | tr -d ' ')
  if [[ $container_count -gt 0 ]]; then
    # Stop all project containers
    docker ps -q --filter "name=${project}" | xargs -r docker stop >/dev/null 2>&1 || true
    # Remove all project containers
    docker ps -aq --filter "name=${project}" | xargs -r docker rm -f >/dev/null 2>&1 || true
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Stopped and removed $container_count containers         \n"
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No containers to stop                           \n"
  fi

  # Remove all volumes for this project
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Removing Docker volumes..."
  local volume_count=$(docker volume ls -q | grep "^${project}_" | wc -l | tr -d ' ')
  if [[ $volume_count -gt 0 ]]; then
    docker volume ls -q | grep "^${project}_" | xargs -r docker volume rm -f >/dev/null 2>&1 || true
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed $volume_count Docker volumes                     \n"
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No volumes to remove                            \n"
  fi

  # Also remove the Docker network
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Removing Docker network..."
  if docker network ls | grep -q "${project}_network"; then
    docker network rm "${project}_network" >/dev/null 2>&1 || true
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed Docker network                          \n"
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No network to remove                            \n"
  fi

  # Backup environment files and schema to _backup folder
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Backing up configuration files..."
  
  # Create backup folder with timestamp subfolder
  local timestamp=$(date +%Y%m%d_%H%M%S)
  local backup_dir="_backup/${timestamp}"
  
  # Create backup directory structure
  mkdir -p "$backup_dir"
  
  # Clean up any loose files in _backup root (move to archive folder)
  if [[ -d "_backup" ]]; then
    local loose_files=false
    for item in _backup/*; do
      if [[ -f "$item" ]]; then
        loose_files=true
        break
      fi
    done
    
    if [[ "$loose_files" == true ]]; then
      local archive_dir="_backup/archive_${timestamp}"
      mkdir -p "$archive_dir"
      for item in _backup/*; do
        if [[ -f "$item" ]]; then
          mv "$item" "$archive_dir/" 2>/dev/null || true
        fi
      done
    fi
  fi
  
  # Move files to backup
  local backed_up=0
  [[ -f ".env" ]] && mv .env "$backup_dir/.env" && ((backed_up++))
  [[ -f ".env.dev" ]] && mv .env.dev "$backup_dir/.env.dev" && ((backed_up++))
  [[ -f ".env.dev" ]] && mv .env.dev "$backup_dir/.env.dev" && ((backed_up++))
  [[ -f ".env.staging" ]] && mv .env.staging "$backup_dir/.env.staging" && ((backed_up++))
  [[ -f ".env.prod" ]] && mv .env.prod "$backup_dir/.env.prod" && ((backed_up++))
  [[ -f ".env.secrets" ]] && mv .env.secrets "$backup_dir/.env.secrets" && ((backed_up++))
  [[ -f "schema.dbml" ]] && mv schema.dbml "$backup_dir/schema.dbml" && ((backed_up++))
  [[ -f ".gitignore" ]] && cp .gitignore "$backup_dir/.gitignore" && ((backed_up++))
  
  # Also move any old .env.*.old files to backup
  for oldfile in .env*.old schema.dbml.old; do
    if [[ -f "$oldfile" ]]; then
      mv "$oldfile" "$backup_dir/$oldfile" 2>/dev/null && ((backed_up++))
    fi
  done

  if [[ $backed_up -gt 0 ]]; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Backed up $backed_up files to $backup_dir/              \n"
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} No configuration files to backup                \n"
  fi

  # Remove ALL generated files and directories
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Removing generated files..."
  local items_to_remove=(
    # Docker files
    "docker-compose.yml"
    "docker-compose.yml.backup"
    "docker-compose.override.yml"
    ".dockerignore"

    # Template and example files
    ".env.example"
    ".env.prod-template"
    ".env.prod-secrets"
    ".gitignore"
    
    # Any leftover backup files (removed after backing up)

    # Service directories
    "nginx"
    "nginx.backup"
    "postgres"
    "hasura"
    "functions"
    "services"
    "nestjs-run"
    "backend"
    "storage"
    "auth"
    "minio"
    "dashboard"

    # Certificates and binaries
    "certs"
    "bin"

    # Data directories
    "data"
    "volumes"
    ".volumes"
    "postgres-data"
    "minio-data"
    "redis-data"

    # Database files
    "db"
    "init.sql"
    "schema.dbml"
    "seeds"
    "migrations"
    "metadata"
    "postgres/init"

    # Log files
    "logs"

    # Temporary files and scripts
    ".needs-rebuild"
    ".last-build-hash"
    ".port-overrides"
    "fix-healthchecks.sh"
    "fix-*.sh"

    # Other generated files
    "node_modules"
    "package-lock.json"
    "yarn.lock"
    "go.mod"
    "go.sum"
    "requirements.txt"
    "Pipfile"
    "Pipfile.lock"
  )

  local removed_count=0
  for item in "${items_to_remove[@]}"; do
    if [[ -e "$item" ]]; then
      rm -rf "$item"
      ((removed_count++))
    fi
  done

  # Remove files matching patterns
  for pattern in *.log *.pid *.lock .DS_Store; do
    for file in $pattern; do
      if [[ -f "$file" ]]; then
        rm -f "$file"
        ((removed_count++))
      fi
    done
  done
  
  # Remove any .backup and .old files
  for backup_file in .env.*.backup *.old .*.old; do
    if [[ -f "$backup_file" ]]; then
      rm -f "$backup_file"
      ((removed_count++))
    fi
  done
  
  # Clean up old backup folders (consolidating into _backup)
  for old_backup in _backup_*; do
    if [[ -d "$old_backup" ]]; then
      # Move contents to new timestamp folder in _backup
      if [[ "$(ls -A $old_backup)" ]]; then
        local old_timestamp="${old_backup#_backup_}_migrated"
        mkdir -p "_backup/${old_timestamp}"
        mv "$old_backup"/* "_backup/${old_timestamp}/" 2>/dev/null || true
      fi
      rm -rf "$old_backup"
      ((removed_count++))
    fi
  done

  # Remove any unity-* or project-prefixed directories (leftover from previous runs)
  for dir in ${project}-* unity-*; do
    if [[ -d "$dir" ]]; then
      rm -rf "$dir"
      ((removed_count++))
    fi
  done

  printf "\r${COLOR_GREEN}✓${COLOR_RESET} Removed $removed_count files and directories           \n"
  
  # Update .gitignore to exclude backup folders
  if [[ -f ".gitignore" ]]; then
    # Check if _backup* is already in .gitignore
    if ! grep -q "^_backup" .gitignore; then
      echo -e "\n# Backup folders from nself reset\n_backup*" >> .gitignore
      printf "${COLOR_GREEN}✓${COLOR_RESET} Added _backup* to .gitignore\n"
    fi
  else
    # Don't create .gitignore - leave directory completely clean except _backup
    # User can run 'nself init' to get a fresh start with proper .gitignore
    :
  fi

  # Clean up Docker system (optional)
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Cleaning Docker system..."
  if docker system prune -f >/dev/null 2>&1; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Docker system cleaned                           \n"
  else
    printf "\r${COLOR_YELLOW}⚠${COLOR_RESET} Docker cleanup skipped                          \n"
  fi

  echo
  log_success "Project reset complete!"
  echo

  echo -e "${COLOR_CYAN}➞ Next Steps${COLOR_RESET}"
  echo

  echo -e "${COLOR_BLUE}Start Fresh:${COLOR_RESET}"
  echo -e "  ${COLOR_BLUE}nself init${COLOR_RESET}     ${COLOR_DIM}# Create new configuration${COLOR_RESET}"
  echo -e "  ${COLOR_BLUE}nself build${COLOR_RESET}    ${COLOR_DIM}# Generate infrastructure${COLOR_RESET}"
  echo -e "  ${COLOR_BLUE}nself start${COLOR_RESET}       ${COLOR_DIM}# Start services${COLOR_RESET}"
  echo

  echo -e "${COLOR_BLUE}Restore Previous:${COLOR_RESET}"
  echo -e "  ${COLOR_BLUE}mv .env.old .env${COLOR_RESET}"
  echo -e "  ${COLOR_BLUE}nself build${COLOR_RESET}"
  echo -e "  ${COLOR_BLUE}nself start${COLOR_RESET}"
  echo

  log_info "Configuration backed up with .old suffix"

  return 0
}

# Show help
show_reset_help() {
  echo "Usage: nself reset [options]"
  echo
  echo "Reset the project to a clean state"
  echo
  echo "Options:"
  echo "  -f, --force    Skip confirmation prompt"
  echo "  -h, --help     Show this help message"
  echo
  echo "Actions:"
  echo "  • Stop and remove all containers"
  echo "  • Delete all Docker volumes"
  echo "  • Remove all generated files"
  echo "  • Backup env files with .old suffix"
  echo
  echo "Examples:"
  echo "  nself reset              # Interactive reset"
  echo "  nself reset --force      # Skip confirmation"
  echo
  echo "WARNING: This will remove all data and cannot be undone!"
}

# Export for use as library
export -f cmd_reset

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "reset" || exit $?
  cmd_reset "$@"
  exit_code=$?
  post_command "reset" $exit_code
  exit $exit_code
fi
