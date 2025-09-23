#!/usr/bin/env bash
set -euo pipefail

# restore.sh - Restore backed up configuration files

# Source display utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/utils/header.sh" 2>/dev/null || true

# Main restore function
cmd_restore() {
  local backup_dir="_backup"
  local specific_backup=""
  local list_only=false
  local verbose=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --list|-l)
        list_only=true
        shift
        ;;
      --verbose|-v)
        verbose=true
        shift
        ;;
      --help|-h)
        show_restore_help
        return 0
        ;;
      *)
        # Assume it's a backup timestamp
        specific_backup="$1"
        shift
        ;;
    esac
  done

  # Show header
  show_command_header "nself restore" "Restore configuration from backup"

  # Check if backup directory exists
  if [[ ! -d "$backup_dir" ]]; then
    log_error "No backup directory found"
    echo "Run 'nself reset' to create a backup first"
    return 1
  fi

  # Get list of available backups
  local backups=($(ls -1d "$backup_dir"/*/ 2>/dev/null | sort -r))

  if [[ ${#backups[@]} -eq 0 ]]; then
    log_error "No backups found"
    echo "Run 'nself reset' to create a backup first"
    return 1
  fi

  # List mode
  if [[ "$list_only" == true ]]; then
    echo "Available backups:"
    echo
    for backup in "${backups[@]}"; do
      local timestamp=$(basename "$backup")
      local date_formatted="${timestamp:0:4}-${timestamp:4:2}-${timestamp:6:2} ${timestamp:9:2}:${timestamp:11:2}:${timestamp:13:2}"
      local file_count=$(ls -1 "$backup" 2>/dev/null | wc -l | tr -d ' ')
      echo -e "  ${COLOR_BLUE}$timestamp${COLOR_RESET}  ($date_formatted)  - $file_count files"
    done
    echo
    echo "To restore a specific backup:"
    echo -e "  ${COLOR_BLUE}nself restore $timestamp${COLOR_RESET}"
    return 0
  fi

  # Determine which backup to restore
  local backup_to_restore=""
  if [[ -n "$specific_backup" ]]; then
    # Check if specific backup exists
    backup_to_restore="$backup_dir/$specific_backup"
    if [[ ! -d "$backup_to_restore" ]]; then
      log_error "Backup not found: $specific_backup"
      echo "Run 'nself restore --list' to see available backups"
      return 1
    fi
  else
    # Use most recent backup
    backup_to_restore="${backups[0]}"
    specific_backup=$(basename "$backup_to_restore")
  fi

  # Show what we're restoring
  echo "Restoring from: ${COLOR_BLUE}$specific_backup${COLOR_RESET}"
  echo

  # List files to restore
  local files_to_restore=($(ls -1 "$backup_to_restore" 2>/dev/null))

  if [[ ${#files_to_restore[@]} -eq 0 ]]; then
    log_error "No files found in backup"
    return 1
  fi

  echo "Files to restore:"
  for file in "${files_to_restore[@]}"; do
    echo "  • $file"
  done
  echo

  # Ask for confirmation
  echo -e "${COLOR_YELLOW}⚠${COLOR_RESET}  This will overwrite existing files"
  echo -n "Continue? [y/N]: "
  read -r response
  if [[ ! "$response" =~ ^[Yy]$ ]]; then
    echo "Restore cancelled"
    return 0
  fi
  echo

  # Perform restoration
  local restored_count=0
  local failed_count=0

  for file in "${files_to_restore[@]}"; do
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Restoring $file..."

    # Backup current file if it exists
    if [[ -f "$file" ]]; then
      cp "$file" "$file.before-restore" 2>/dev/null || true
    fi

    # Restore the file
    if cp "$backup_to_restore/$file" "$file" 2>/dev/null; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Restored $file                              \n"
      ((restored_count++))
    else
      printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to restore $file                     \n"
      ((failed_count++))
    fi
  done

  echo
  if [[ $failed_count -eq 0 ]]; then
    log_success "Restored $restored_count files successfully"
  else
    log_warning "Restored $restored_count files, $failed_count failed"
  fi

  # Show next steps
  echo
  echo "Next steps:"
  echo
  echo -e "  ${COLOR_BLUE}nself build${COLOR_RESET}  - Generate infrastructure"
  echo -e "  ${COLOR_BLUE}nself start${COLOR_RESET}  - Start services"
  echo
  echo "For more help, use: nself help or nself help restore"

  return 0
}

# Show help
show_restore_help() {
  echo "Usage: nself restore [options] [timestamp]"
  echo
  echo "Restore configuration from a previous backup"
  echo
  echo "Options:"
  echo "  -l, --list     List available backups"
  echo "  -v, --verbose  Show detailed output"
  echo "  -h, --help     Show this help message"
  echo
  echo "Arguments:"
  echo "  timestamp      Specific backup to restore (YYYYMMDD_HHMMSS format)"
  echo "                 If not specified, restores the most recent backup"
  echo
  echo "Examples:"
  echo "  nself restore                    # Restore most recent backup"
  echo "  nself restore --list             # List available backups"
  echo "  nself restore 20250923_120000    # Restore specific backup"
  echo
  echo "Notes:"
  echo "  • Backups are created when running 'nself reset'"
  echo "  • Current files are backed up with .before-restore suffix"
  echo "  • Only .env files are restored by default"
}

# Export command
export -f cmd_restore

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_restore "$@"
fi