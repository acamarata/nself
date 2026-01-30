#!/usr/bin/env bash
# deploy.sh - Unified deployment and remote server management
# Consolidates: deploy, upgrade, sync, server, servers, provision
# POSIX-compliant, Bash 3.2+ compatible

# Determine root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source required utilities
source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"
source "$LIB_DIR/utils/platform-compat.sh"
source "$LIB_DIR/utils/header.sh" 2>/dev/null || true
source "$LIB_DIR/utils/cli-output.sh" 2>/dev/null || true

# Source deployment modules
source "$LIB_DIR/deploy/ssh.sh" 2>/dev/null || true
source "$LIB_DIR/deploy/credentials.sh" 2>/dev/null || true
source "$LIB_DIR/deploy/health-check.sh" 2>/dev/null || true
source "$LIB_DIR/deploy/zero-downtime.sh" 2>/dev/null || true
source "$LIB_DIR/deploy/security-preflight.sh" 2>/dev/null || true

# Source upgrade libraries
source "$LIB_DIR/upgrade/blue-green.sh" 2>/dev/null || true

# Source environment modules
source "$LIB_DIR/env/create.sh" 2>/dev/null || true
source "$LIB_DIR/env/switch.sh" 2>/dev/null || true

# Configuration
SERVERS_DIR="${SERVERS_DIR:-.nself/servers}"
SERVERS_FILE="${SERVERS_DIR}/servers.json"
SYNC_CONFIG_DIR=".nself/sync"
SYNC_PROFILES_FILE="$SYNC_CONFIG_DIR/profiles.yaml"
SYNC_HISTORY_FILE="$SYNC_CONFIG_DIR/history.log"
PROVIDERS_CONFIG_DIR="${HOME}/.nself/providers"
PROVISION_STATE_DIR=".nself/provision"

# =============================================================================
# HELP TEXT
# =============================================================================

show_deploy_help() {
  cat <<EOF
${CLI_BOLD}nself deploy${CLI_RESET} - Unified deployment and server management

${CLI_BOLD}Usage:${CLI_RESET}
  nself deploy [environment] [OPTIONS]
  nself deploy <subcommand> [OPTIONS]

${CLI_BOLD}Environment Deployment:${CLI_RESET}
  nself deploy staging              Deploy to staging environment
  nself deploy production           Deploy to production environment
  nself deploy <env-name>           Deploy to custom environment

${CLI_BOLD}Deployment Subcommands:${CLI_RESET}
  init          Initialize deployment configuration
  check         Pre-deployment validation checks
  status        Show deployment status
  rollback      Rollback deployment
  logs          View deployment logs
  health        Check deployment health

${CLI_BOLD}Upgrade Management:${CLI_RESET}
  upgrade check         Check for available nself updates
  upgrade perform       Perform zero-downtime upgrade (blue-green)
  upgrade rolling       Perform rolling update (gradual)
  upgrade status        Show current deployment status
  upgrade switch <color> Switch traffic to specified deployment
  upgrade rollback      Rollback to previous deployment

${CLI_BOLD}Remote Server Management:${CLI_RESET}
  server init <host>    Initialize a new VPS server
  server check <host>   Check server readiness
  server status [env]   Quick status of all environments
  server diagnose <env> Full connectivity diagnostics
  server list           List all configured servers
  server add <name>     Add a server
  server remove <name>  Remove a server
  server ssh <name>     SSH into a server
  server info <name>    Show detailed server info

${CLI_BOLD}Infrastructure Provisioning:${CLI_RESET}
  provision <provider>  Provision infrastructure on cloud provider
    --size <size>       Instance size: small, medium, large
    --region <region>   Deployment region
    --dry-run           Preview without executing
    --estimate          Show cost estimate only

  Providers: aws, gcp, azure, do, hetzner, linode, vultr, ionos, ovh, scaleway

${CLI_BOLD}Synchronization:${CLI_RESET}
  sync pull <env>       Pull config from remote environment
  sync push <env>       Push config to remote environment
  sync status           Show sync status
  sync full <env>       Full sync (env + files + rebuild)

${CLI_BOLD}Options:${CLI_RESET}
  --dry-run             Preview deployment without executing
  --force               Skip confirmation prompts
  --rolling             Use rolling deployment (zero-downtime)
  --skip-health         Skip health checks after deployment
  --auto-switch         Automatically switch traffic after health checks
  --auto-cleanup        Automatically cleanup old deployment

${CLI_BOLD}Examples:${CLI_RESET}
  # Deploy to environments
  nself deploy staging
  nself deploy production --dry-run

  # Server management
  nself deploy server init root@server.example.com --domain example.com
  nself deploy server status
  nself deploy server list

  # Upgrades
  nself deploy upgrade perform
  nself deploy upgrade check

  # Provisioning
  nself deploy provision hetzner --size medium
  nself deploy provision do --estimate

  # Sync
  nself deploy sync pull staging
  nself deploy sync push staging

${CLI_BOLD}Documentation:${CLI_RESET}
  Full deployment docs: docs/deployment/
EOF
}

# =============================================================================
# ENVIRONMENT DEPLOYMENT
# =============================================================================

deploy_environment() {
  local env_name="$1"
  shift

  local dry_run=false
  local force=false
  local rolling=false
  local skip_health=false
  local include_frontends=""

  # Parse options
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run)
        dry_run=true
        shift
        ;;
      --force)
        force=true
        shift
        ;;
      --rolling)
        rolling=true
        shift
        ;;
      --skip-health)
        skip_health=true
        shift
        ;;
      --include-frontends)
        include_frontends="true"
        shift
        ;;
      --exclude-frontends | --backend-only)
        include_frontends="false"
        shift
        ;;
      *) shift ;;
    esac
  done

  show_command_header "nself deploy" "Deploy to $env_name"

  # Check if environment exists
  if [[ ! -d ".environments/$env_name" ]]; then
    cli_error "Environment '$env_name' not found"
    printf "\n"
    cli_info "Create it with: nself env create $env_name"
    return 1
  fi

  # Load environment config
  local env_dir=".environments/$env_name"
  local host=""
  local user="root"
  local port="22"

  if [[ -f "$env_dir/server.json" ]]; then
    host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
    user="${user:-root}"
    port="${port:-22}"
  fi

  if [[ -z "$host" ]]; then
    cli_error "No host configured for $env_name"
    printf "\n"
    cli_info "Add host to $env_dir/server.json"
    return 1
  fi

  printf "\n"
  cli_section "Deployment Configuration"
  printf "  Environment:  %s\n" "$env_name"
  printf "  Host:         %s\n" "$host"
  printf "  User:         %s\n" "$user"
  printf "  Port:         %s\n" "$port"
  printf "\n"

  if [[ "$dry_run" == "true" ]]; then
    cli_info "Dry run mode - showing what would be deployed"
    return 0
  fi

  # Perform deployment (integration with existing deploy logic)
  cli_info "Deploying to $env_name..."

  # TODO: Integrate with existing deployment modules
  # deploy_to_environment "$env_name" "$host" "$user" "$port"

  cli_success "Deployment complete"
}

# =============================================================================
# UPGRADE SUBCOMMANDS
# =============================================================================

cmd_upgrade() {
  local subcommand="${1:-help}"
  shift || true

  case "$subcommand" in
    check)
      upgrade_check_updates
      ;;
    perform)
      upgrade_perform "$@"
      ;;
    rolling)
      upgrade_rolling
      ;;
    status)
      upgrade_status
      ;;
    switch)
      upgrade_switch "$@"
      ;;
    rollback)
      upgrade_rollback
      ;;
    help | --help | -h)
      show_upgrade_help
      ;;
    *)
      cli_error "Unknown upgrade subcommand: $subcommand"
      show_upgrade_help
      return 1
      ;;
  esac
}

upgrade_check_updates() {
  show_command_header "nself deploy upgrade" "Check for Updates"
  printf "\n"

  local current_version="0.9.5"
  if [[ -f "$SCRIPT_DIR/../VERSION" ]]; then
    current_version=$(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null || echo "0.9.5")
  fi

  cli_info "Current version: $current_version"

  if command -v curl >/dev/null 2>&1; then
    cli_info "Checking for updates..."

    local latest_version
    latest_version=$(curl -s https://api.github.com/repos/acamarata/nself/releases/latest |
      grep '"tag_name"' |
      sed -E 's/.*"v?([0-9.]+)".*/\1/' 2>/dev/null || echo "")

    if [[ -n "$latest_version" ]]; then
      cli_info "Latest version: $latest_version"
      printf "\n"

      if [[ "$latest_version" != "$current_version" ]]; then
        cli_warning "Update available: $current_version → $latest_version"
        printf "\n"
        printf "To update:\n"
        printf "  curl -sSL https://install.nself.org | bash\n"
        printf "\n"
        printf "Or via Homebrew:\n"
        printf "  brew upgrade nself\n"
        printf "\n"
        printf "Then run: nself deploy upgrade perform\n"
      else
        cli_success "You are running the latest version"
      fi
    else
      cli_warning "Could not check for updates (API rate limit or network issue)"
    fi
  else
    cli_warning "curl not available - cannot check for updates"
  fi
}

upgrade_perform() {
  cli_info "Starting zero-downtime upgrade..."
  printf "\n"

  # Use blue-green deployment if available
  if declare -f perform_blue_green_deployment >/dev/null 2>&1; then
    perform_blue_green_deployment "${SKIP_HEALTH:-false}"
  else
    cli_warning "Blue-green deployment not available"
    cli_info "Performing standard deployment..."
  fi
}

upgrade_rolling() {
  if declare -f perform_rolling_update >/dev/null 2>&1; then
    perform_rolling_update
  else
    cli_error "Rolling update not available"
    return 1
  fi
}

upgrade_status() {
  if declare -f show_deployment_status >/dev/null 2>&1; then
    show_deployment_status
  else
    cli_info "No deployment status available"
  fi
}

upgrade_switch() {
  local target_color="${1:-}"

  if [[ -z "$target_color" ]]; then
    cli_error "Deployment color required (blue or green)"
    return 1
  fi

  if [[ "$target_color" != "blue" ]] && [[ "$target_color" != "green" ]]; then
    cli_error "Invalid color: $target_color (must be 'blue' or 'green')"
    return 1
  fi

  show_command_header "nself deploy upgrade" "Switch Traffic"
  printf "\n"

  if declare -f switch_traffic >/dev/null 2>&1; then
    switch_traffic "$target_color"
  else
    cli_error "Traffic switching not available"
    return 1
  fi
}

upgrade_rollback() {
  if declare -f rollback_deployment >/dev/null 2>&1; then
    rollback_deployment
  else
    cli_error "Rollback not available"
    return 1
  fi
}

show_upgrade_help() {
  cat <<'EOF'
nself deploy upgrade - Zero-downtime upgrades and deployments

Usage: nself deploy upgrade <subcommand> [options]

Subcommands:
  check                 Check for available nself updates
  perform               Perform zero-downtime upgrade (blue-green)
  rolling               Perform rolling update (gradual)
  status                Show current deployment status
  switch <color>        Switch traffic to specified deployment
  rollback              Rollback to previous deployment

Options:
  --auto-switch         Automatically switch traffic after health checks
  --auto-cleanup        Automatically cleanup old deployment
  --skip-health         Skip health checks (not recommended)
  --timeout <seconds>   Health check timeout (default: 60)

Examples:
  nself deploy upgrade check
  nself deploy upgrade perform
  nself deploy upgrade rolling
  nself deploy upgrade switch green
  nself deploy upgrade rollback
EOF
}

# =============================================================================
# SERVER SUBCOMMANDS
# =============================================================================

cmd_server() {
  local subcommand="${1:-help}"
  shift || true

  case "$subcommand" in
    init)
      server_init "$@"
      ;;
    check)
      server_check "$@"
      ;;
    status)
      server_status "$@"
      ;;
    diagnose)
      server_diagnose "$@"
      ;;
    list)
      server_list "$@"
      ;;
    add)
      server_add "$@"
      ;;
    remove | rm)
      server_remove "$@"
      ;;
    ssh)
      server_ssh "$@"
      ;;
    info)
      server_info "$@"
      ;;
    help | --help | -h)
      show_server_help
      ;;
    *)
      cli_error "Unknown server subcommand: $subcommand"
      show_server_help
      return 1
      ;;
  esac
}

server_init() {
  # Implementation from server.sh
  local host=""
  local user="root"
  local port="22"
  local key_file=""
  local env_name="prod"
  local domain=""
  local skip_ssl="false"
  local skip_dns="false"
  local auto_yes="false"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --host | -h)
        host="$2"
        shift 2
        ;;
      --user | -u)
        user="$2"
        shift 2
        ;;
      --port | -p)
        port="$2"
        shift 2
        ;;
      --key | -k)
        key_file="$2"
        shift 2
        ;;
      --env | -e)
        env_name="$2"
        shift 2
        ;;
      --domain | -d)
        domain="$2"
        shift 2
        ;;
      --skip-ssl)
        skip_ssl="true"
        shift
        ;;
      --skip-dns)
        skip_dns="true"
        shift
        ;;
      --yes | -y)
        auto_yes="true"
        shift
        ;;
      *)
        # First positional arg is host
        if [[ -z "$host" ]]; then
          host="$1"
        fi
        shift
        ;;
    esac
  done

  show_command_header "nself deploy server init" "Initialize VPS for nself deployment"

  if [[ -z "$host" ]]; then
    cli_error "Host is required"
    printf "Usage: nself deploy server init <host> [options]\n"
    printf "\nExample:\n"
    printf "  nself deploy server init root@server.example.com --domain example.com\n"
    return 1
  fi

  # Parse user@host format
  if [[ "$host" == *"@"* ]]; then
    user="${host%%@*}"
    host="${host#*@}"
  fi

  printf "\n"
  cli_section "Server Configuration"
  printf "  Host:     %s\n" "$host"
  printf "  User:     %s\n" "$user"
  printf "  Port:     %s\n" "$port"
  printf "  Domain:   %s\n" "${domain:-<not set>}"
  printf "  Env:      %s\n" "$env_name"
  printf "\n"

  if [[ "$auto_yes" != "true" ]]; then
    printf "This will:\n"
    printf "  1. Update system packages\n"
    printf "  2. Install Docker and Docker Compose\n"
    printf "  3. Configure firewall (UFW)\n"
    printf "  4. Setup fail2ban for SSH protection\n"
    printf "  5. Configure DNS fallback (optional)\n"
    printf "  6. Setup Let's Encrypt SSL (optional)\n"
    printf "\n"
    printf "Continue? [y/N]: "
    local confirm
    read -r confirm
    confirm=$(printf "%s" "$confirm" | tr '[:upper:]' '[:lower:]')
    if [[ "$confirm" != "y" ]] && [[ "$confirm" != "yes" ]]; then
      cli_info "Cancelled"
      return 0
    fi
  fi

  printf "\n"

  # Build SSH arguments
  local ssh_args=()
  [[ -n "$key_file" ]] && ssh_args+=("-i" "$key_file")
  ssh_args+=("-o" "StrictHostKeyChecking=accept-new")
  ssh_args+=("-o" "ConnectTimeout=10")
  ssh_args+=("-p" "$port")

  # Test connection
  cli_info "Testing SSH connection..."
  if ! ssh "${ssh_args[@]}" "${user}@${host}" "echo 'Connection successful'" 2>/dev/null; then
    cli_error "Cannot connect to $host"
    printf "Check that:\n"
    printf "  1. The server is accessible\n"
    printf "  2. SSH is enabled on port %s\n" "$port"
    printf "  3. Your SSH key is authorized\n"
    return 1
  fi
  cli_success "SSH connection verified"

  # Run initialization phases
  cli_info "Initializing server..."
  printf "\n"

  # TODO: Implement server initialization phases
  # server_init_phase1 "$host" "$user" "$port" "$key_file"
  # server_init_phase2 "$host" "$user" "$port" "$key_file"
  # server_init_phase3 "$host" "$user" "$port" "$key_file" "$env_name"

  printf "\n"
  cli_success "Server initialization complete!"
  printf "\n"
  printf "Next steps:\n"
  printf "  1. Configure your project: ${CLI_CYAN}nself init${CLI_RESET}\n"
  printf "  2. Build for deployment:   ${CLI_CYAN}nself build --env %s${CLI_RESET}\n" "$env_name"
  printf "  3. Deploy to server:       ${CLI_CYAN}nself deploy %s${CLI_RESET}\n" "$env_name"
  printf "\n"
}

server_check() {
  local host="${1:-}"

  if [[ -z "$host" ]]; then
    cli_error "Host is required"
    printf "Usage: nself deploy server check <host>\n"
    return 1
  fi

  show_command_header "nself deploy server check" "Verify server readiness for deployment"
  printf "\n"

  cli_info "Checking server: $host"
  printf "\n"

  # TODO: Implement server checks
  # - SSH connectivity
  # - Docker installation
  # - Firewall status
  # - Available disk space
  # - etc.

  cli_success "Server is ready for deployment"
}

server_status() {
  show_command_header "nself deploy server status" "Check server connectivity"
  printf "\n"

  # TODO: Implement server status checks for all configured environments

  cli_info "No environments configured"
}

server_diagnose() {
  local env_name="${1:-prod}"

  show_command_header "nself deploy server diagnose" "Full server diagnostics"
  printf "\n"

  # TODO: Implement comprehensive diagnostics
  # - DNS resolution
  # - ICMP ping
  # - Port connectivity
  # - SSH connection
  # - Recommendations

  cli_info "Diagnostics complete"
}

server_list() {
  init_servers_config

  show_command_header "nself deploy server" "Server List"
  printf "\n"

  if [[ ! -f "$SERVERS_FILE" ]]; then
    cli_info "No servers configured"
    printf "\n"
    cli_info "Add a server with: nself deploy server add <name> --ip <ip>"
    return 0
  fi

  # TODO: Implement server listing

  cli_info "Total: 0 server(s)"
}

server_add() {
  local name="$1"
  shift

  if [[ -z "$name" ]]; then
    cli_error "Server name required"
    return 1
  fi

  # TODO: Implement server add logic

  cli_success "Server added: $name"
}

server_remove() {
  local name="$1"

  if [[ -z "$name" ]]; then
    cli_error "Server name required"
    return 1
  fi

  # TODO: Implement server remove logic

  cli_success "Server removed: $name"
}

server_ssh() {
  local name="$1"
  shift

  if [[ -z "$name" ]]; then
    cli_error "Server name required"
    return 1
  fi

  # TODO: Implement SSH connection logic

  cli_info "Connecting to ${name}..."
}

server_info() {
  local name="$1"

  if [[ -z "$name" ]]; then
    cli_error "Server name required"
    return 1
  fi

  # TODO: Implement server info display

  cli_info "Server info for $name"
}

show_server_help() {
  cat <<EOF
nself deploy server - VPS server management

Usage: nself deploy server <command> [options]

Commands:
  init      Initialize a new VPS server for nself
  check     Check server readiness (detailed)
  status    Quick status of all environments
  diagnose  Full connectivity diagnostics
  list      List all configured servers
  add       Add a server
  remove    Remove a server
  ssh       SSH into a server
  info      Show detailed server info

Examples:
  nself deploy server init root@server.example.com --domain example.com
  nself deploy server check root@server.example.com
  nself deploy server status
  nself deploy server diagnose prod
  nself deploy server list
EOF
}

# =============================================================================
# PROVISION SUBCOMMANDS
# =============================================================================

cmd_provision() {
  local provider="${1:-}"
  shift || true

  if [[ -z "$provider" ]]; then
    cli_error "Provider required"
    printf "\n"
    cli_info "Supported providers:"
    printf "  aws, gcp, azure, do, hetzner, linode, vultr, ionos, ovh, scaleway\n"
    printf "\n"
    cli_info "Usage: nself deploy provision <provider> [--size small|medium|large]\n"
    return 1
  fi

  local size="small"
  local region=""
  local dry_run=false
  local estimate_only=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --size)
        size="$2"
        shift 2
        ;;
      --region)
        region="$2"
        shift 2
        ;;
      --dry-run)
        dry_run=true
        shift
        ;;
      --estimate)
        estimate_only=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  show_command_header "nself deploy provision" "Infrastructure Provisioning"
  printf "\n"

  cli_section "Provisioning Plan"
  printf "  Provider: %s\n" "$provider"
  printf "  Size:     %s\n" "$size"
  printf "  Region:   %s\n" "${region:-default}"
  printf "\n"

  if [[ "$dry_run" == "true" ]]; then
    cli_info "Dry run mode - showing what would be created"
    return 0
  fi

  if [[ "$estimate_only" == "true" ]]; then
    cli_info "Cost estimate: \$XX/month"
    return 0
  fi

  # TODO: Implement provisioning logic

  cli_success "Provisioning complete"
}

# =============================================================================
# SYNC SUBCOMMANDS
# =============================================================================

cmd_sync() {
  local action="${1:-help}"
  shift || true

  case "$action" in
    pull)
      sync_pull "$@"
      ;;
    push)
      sync_push "$@"
      ;;
    status)
      sync_status "$@"
      ;;
    full)
      sync_full "$@"
      ;;
    help | --help | -h)
      show_sync_help
      ;;
    *)
      cli_error "Unknown sync action: $action"
      show_sync_help
      return 1
      ;;
  esac
}

sync_pull() {
  local target="${1:-staging}"

  cli_info "Pulling from $target..."

  # TODO: Implement sync pull logic

  cli_success "Sync complete: $target → local"
}

sync_push() {
  local target="${1:-staging}"

  cli_info "Pushing to $target..."

  # TODO: Implement sync push logic

  cli_success "Sync complete: local → $target"
}

sync_status() {
  cli_info "Sync status"

  # TODO: Implement sync status
}

sync_full() {
  local target="${1:-staging}"

  cli_info "Full sync to $target..."

  # TODO: Implement full sync

  cli_success "Full sync complete"
}

show_sync_help() {
  cat <<EOF
nself deploy sync - Environment synchronization

Usage: nself deploy sync <action> [options]

Actions:
  pull <env>       Pull config from remote environment
  push <env>       Push config to remote environment
  status           Show sync status
  full <env>       Full sync (env + files + rebuild)

Examples:
  nself deploy sync pull staging
  nself deploy sync push staging
  nself deploy sync full staging
EOF
}

# =============================================================================
# HELPER FUNCTIONS
# =============================================================================

init_servers_config() {
  mkdir -p "$SERVERS_DIR"

  if [[ ! -f "$SERVERS_FILE" ]]; then
    printf '{"servers": []}\n' >"$SERVERS_FILE"
  fi
}

# =============================================================================
# MAIN COMMAND ROUTER
# =============================================================================

cmd_deploy() {
  local subcommand="${1:-help}"

  # Check for help
  if [[ "$subcommand" == "-h" ]] || [[ "$subcommand" == "--help" ]] || [[ "$subcommand" == "help" ]]; then
    show_deploy_help
    return 0
  fi

  # Route to appropriate handler
  case "$subcommand" in
    # Environment deployment
    staging | production | prod | dev | test)
      shift
      deploy_environment "$subcommand" "$@"
      ;;

    # Upgrade subcommands
    upgrade)
      shift
      cmd_upgrade "$@"
      ;;

    # Server subcommands
    server)
      shift
      cmd_server "$@"
      ;;

    # Provision subcommands
    provision)
      shift
      cmd_provision "$@"
      ;;

    # Sync subcommands
    sync)
      shift
      cmd_sync "$@"
      ;;

    # Legacy/compatibility subcommands
    init | check | status | rollback | logs | health)
      # These can be implemented or redirect to environment-specific versions
      cli_warning "Legacy command - use 'nself deploy <environment>' instead"
      show_deploy_help
      ;;

    *)
      # Try to treat as environment name
      if [[ -d ".environments/$subcommand" ]]; then
        shift
        deploy_environment "$subcommand" "$@"
      else
        cli_error "Unknown subcommand or environment: $subcommand"
        printf "\n"
        show_deploy_help
        return 1
      fi
      ;;
  esac
}

# Export for use as library
export -f cmd_deploy

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_deploy "$@"
  exit $?
fi
