#!/usr/bin/env bash

# upgrade.sh - Zero-downtime upgrades and deployment management
# v0.4.8 - Sprint 20: Migration & Upgrade Tools

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Source upgrade libraries
source "$CLI_SCRIPT_DIR/../lib/upgrade/blue-green.sh" 2>/dev/null || true

# Color fallbacks
: "${COLOR_GREEN:=\033[0;32m}"
: "${COLOR_YELLOW:=\033[0;33m}"
: "${COLOR_RED:=\033[0;31m}"
: "${COLOR_CYAN:=\033[0;36m}"
: "${COLOR_BLUE:=\033[0;34m}"
: "${COLOR_RESET:=\033[0m}"
: "${COLOR_DIM:=\033[2m}"

# Show help
show_upgrade_help() {
  cat << 'EOF'
nself upgrade - Zero-downtime upgrades and deployments

Usage: nself upgrade <subcommand> [options]

Subcommands:
  check                 Check for available nself updates
  perform               Perform zero-downtime upgrade (blue-green)
  rolling               Perform rolling update (gradual)
  status                Show current deployment status
  switch <color>        Switch traffic to specified deployment
  rollback              Rollback to previous deployment
  cleanup <color>       Cleanup specified deployment

Deployment Strategies:
  blue-green            Zero-downtime using parallel deployments
  rolling               Gradual service-by-service updates

Options:
  --auto-switch         Automatically switch traffic after health checks
  --auto-cleanup        Automatically cleanup old deployment
  --skip-health         Skip health checks (not recommended)
  --timeout <seconds>   Health check timeout (default: 60)
  -h, --help            Show this help message

Examples:
  # Check for updates
  nself upgrade check

  # Perform zero-downtime upgrade
  nself upgrade perform

  # Perform rolling update
  nself upgrade rolling

  # Show deployment status
  nself upgrade status

  # Manual traffic switch
  nself upgrade switch green

  # Rollback to previous deployment
  nself upgrade rollback

  # Automated upgrade (no prompts)
  nself upgrade perform --auto-switch --auto-cleanup

Blue-Green Deployment:
  This strategy runs two identical production environments (blue and green).
  Only one serves live traffic at a time. To upgrade:
    1. Deploy new version to inactive environment
    2. Run health checks on new deployment
    3. Switch traffic to new environment (zero downtime)
    4. Keep old environment running for quick rollback

Rolling Update:
  This strategy updates services gradually, one at a time.
  Less resource-intensive but may have brief service interruptions.
    1. Update non-critical services first
    2. Update critical services one by one
    3. Wait for health checks between updates
EOF
}

# Check for nself updates
cmd_check_updates() {
  show_command_header "nself upgrade" "Check for Updates"
  echo ""

  # Get current version
  local current_version="0.4.8"  # Will be dynamically loaded
  if [[ -f "$CLI_SCRIPT_DIR/../VERSION" ]]; then
    current_version=$(cat "$CLI_SCRIPT_DIR/../VERSION" 2>/dev/null || echo "0.4.8")
  fi

  log_info "Current version: $current_version"

  # Check GitHub for latest release
  if command -v curl >/dev/null 2>&1; then
    log_info "Checking for updates..."

    local latest_version=$(curl -s https://api.github.com/repos/acamarata/nself/releases/latest \
      | grep '"tag_name"' \
      | sed -E 's/.*"v?([0-9.]+)".*/\1/' 2>/dev/null || echo "")

    if [[ -n "$latest_version" ]]; then
      log_info "Latest version: $latest_version"
      echo ""

      if [[ "$latest_version" != "$current_version" ]]; then
        log_warning "Update available: $current_version → $latest_version"
        echo ""
        echo "To update:"
        echo "  curl -sSL https://install.nself.org | bash"
        echo ""
        echo "Or via Homebrew:"
        echo "  brew upgrade nself"
        echo ""
        echo "Then run: nself upgrade perform"
      else
        log_success "You are running the latest version"
      fi
    else
      log_warning "Could not check for updates (API rate limit or network issue)"
    fi
  else
    log_warning "curl not available - cannot check for updates"
  fi
}

# Perform zero-downtime upgrade
cmd_perform_upgrade() {
  local auto_switch="${AUTO_SWITCH:-false}"
  local auto_cleanup="${AUTO_CLEANUP:-false}"
  local skip_health="${SKIP_HEALTH:-false}"

  # Check if there are updates to apply
  log_info "Starting zero-downtime upgrade..."
  echo ""

  # Perform blue-green deployment
  perform_blue_green_deployment "$skip_health"
}

# Perform rolling update
cmd_rolling_upgrade() {
  perform_rolling_update
}

# Show deployment status
cmd_status() {
  show_deployment_status
}

# Switch traffic
cmd_switch() {
  local target_color="${1:-}"

  if [[ -z "$target_color" ]]; then
    log_error "Deployment color required (blue or green)"
    return 1
  fi

  if [[ "$target_color" != "blue" ]] && [[ "$target_color" != "green" ]]; then
    log_error "Invalid color: $target_color (must be 'blue' or 'green')"
    return 1
  fi

  show_command_header "nself upgrade" "Switch Traffic"
  echo ""

  switch_traffic "$target_color"
}

# Rollback deployment
cmd_rollback_upgrade() {
  rollback_deployment
}

# Cleanup deployment
cmd_cleanup() {
  local color="${1:-}"

  if [[ -z "$color" ]]; then
    log_error "Deployment color required (blue or green)"
    return 1
  fi

  show_command_header "nself upgrade" "Cleanup Deployment"
  echo ""

  cleanup_deployment "$color"
}

# Main command handler
cmd_upgrade() {
  local subcommand="${1:-}"

  # Check for help first
  if [[ "$subcommand" == "-h" ]] || [[ "$subcommand" == "--help" ]] || [[ -z "$subcommand" ]]; then
    show_upgrade_help
    return 0
  fi

  # Parse global options
  local args=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --auto-switch)
        AUTO_SWITCH=true
        shift
        ;;
      --auto-cleanup)
        AUTO_CLEANUP=true
        shift
        ;;
      --skip-health)
        SKIP_HEALTH=true
        shift
        ;;
      --timeout)
        HEALTH_CHECK_TIMEOUT="$2"
        shift 2
        ;;
      -h|--help)
        show_upgrade_help
        return 0
        ;;
      *)
        args+=("$1")
        shift
        ;;
    esac
  done

  # Restore positional arguments
  set -- "${args[@]}"
  subcommand="${1:-}"

  case "$subcommand" in
    check)
      cmd_check_updates
      ;;
    perform)
      cmd_perform_upgrade
      ;;
    rolling)
      cmd_rolling_upgrade
      ;;
    status)
      cmd_status
      ;;
    switch)
      shift
      cmd_switch "$@"
      ;;
    rollback)
      cmd_rollback_upgrade
      ;;
    cleanup)
      shift
      cmd_cleanup "$@"
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      show_upgrade_help
      return 1
      ;;
  esac
}

# Export for use as library
export -f cmd_upgrade

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "upgrade" || exit $?
  cmd_upgrade "$@"
  exit_code=$?
  post_command "upgrade" $exit_code
  exit $exit_code
fi
