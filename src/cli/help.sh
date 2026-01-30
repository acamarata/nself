#!/usr/bin/env bash
set -euo pipefail

# help.sh - Show help information

# Get script directory with absolute path
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SCRIPT_DIR="$CLI_SCRIPT_DIR"

# Source shared utilities
[[ -z "${DISPLAY_SOURCED:-}" ]] && source "$CLI_SCRIPT_DIR/../lib/utils/display.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"
[[ -z "${CONSTANTS_SOURCED:-}" ]] && source "$CLI_SCRIPT_DIR/../lib/config/constants.sh"

# Command function
cmd_help() {
  local command="${1:-}"

  if [[ -n "$command" ]]; then
    # Show help for specific command
    show_command_help "$command"
  else
    # Show general help
    show_general_help
  fi
}

# Show general help
show_general_help() {
  # Get version from VERSION file
  local version="unknown"
  if [[ -f "$SCRIPT_DIR/../VERSION" ]]; then
    version=$(cat "$SCRIPT_DIR/../VERSION" 2>/dev/null || echo "unknown")
  fi

  show_command_header "nself v${version}" "Self-Hosted Infrastructure Manager"
  echo
  echo "Usage: nself <command> [options]"

  show_section "Core Commands"
  printf "  ${COLOR_BLUE}init${COLOR_RESET}          Initialize a new project\n"
  printf "  ${COLOR_BLUE}build${COLOR_RESET}         Build project structure and Docker images\n"
  printf "  ${COLOR_BLUE}start${COLOR_RESET}         Start all services\n"
  printf "  ${COLOR_BLUE}stop${COLOR_RESET}          Stop all services\n"
  printf "  ${COLOR_BLUE}restart${COLOR_RESET}       Restart all services\n"
  echo
  printf "  ${COLOR_BLUE}reset${COLOR_RESET}         Reset project to clean state\n"
  printf "  ${COLOR_BLUE}clean${COLOR_RESET}         Clean up Docker resources\n"
  printf "  ${COLOR_BLUE}restore${COLOR_RESET}       Restore configuration from backup\n"

  show_section "Status Commands"
  printf "  ${COLOR_BLUE}status${COLOR_RESET}        Show service status\n"
  printf "  ${COLOR_BLUE}logs${COLOR_RESET}          View service logs\n"
  printf "  ${COLOR_BLUE}exec${COLOR_RESET}          Execute commands in containers\n"
  printf "  ${COLOR_BLUE}urls${COLOR_RESET}          Show service URLs\n"
  echo
  printf "  ${COLOR_BLUE}doctor${COLOR_RESET}        Run system diagnostics\n"
  printf "  ${COLOR_BLUE}version${COLOR_RESET}       Show version information\n"
  printf "  ${COLOR_BLUE}update${COLOR_RESET}        Update nself to latest version\n"
  printf "  ${COLOR_BLUE}help${COLOR_RESET}          Show this help message\n"

  show_section "Database Commands"
  printf "  ${COLOR_BLUE}db${COLOR_RESET}            Database tools (migrate, seed, mock, backup, restore, schema, types)\n"

  show_section "Service Management (v0.4.7)"
  printf "  ${COLOR_BLUE}service${COLOR_RESET}       Unified optional service management\n"
  printf "                   ├─ list, enable, disable, status, restart, logs\n"
  printf "                   ├─ email: test, inbox, config\n"
  printf "                   ├─ search: index, query, stats\n"
  printf "                   ├─ functions: deploy, invoke, logs, list\n"
  printf "                   ├─ mlflow: ui, experiments, runs, artifacts\n"
  printf "                   ├─ storage: buckets, upload, download, presign\n"
  printf "                   └─ cache: stats, flush, keys\n"
  echo
  printf "  ${COLOR_BLUE}ssl${COLOR_RESET}           Manage SSL certificates\n"
  printf "  ${COLOR_BLUE}trust${COLOR_RESET}         Trust local SSL certificates\n"

  show_section "Cloud Infrastructure (v0.4.7)"
  printf "  ${COLOR_BLUE}cloud${COLOR_RESET}         Unified cloud infrastructure management\n"
  printf "                   ├─ provider: list, init, validate, info\n"
  printf "                   ├─ server: create, destroy, list, status, ssh, add, remove\n"
  printf "                   ├─ cost: estimate, compare\n"
  printf "                   └─ deploy: quick, full\n"
  echo
  printf "  Supported: DigitalOcean, Linode, Vultr, Hetzner, OVH, Scaleway, UpCloud,\n"
  printf "             AWS, GCP, Azure, Oracle, IBM, Contabo, Hostinger, Kamatera,\n"
  printf "             SSDNodes, Exoscale, Alibaba, Tencent, Yandex, RackNerd, BuyVM,\n"
  printf "             Time4VPS, Raspberry Pi, Custom SSH\n"

  show_section "Kubernetes & Helm (v0.4.7)"
  printf "  ${COLOR_BLUE}k8s${COLOR_RESET}           Kubernetes management\n"
  printf "                   ├─ init, convert, apply, deploy, status, logs\n"
  printf "                   ├─ scale, rollback, delete\n"
  printf "                   ├─ cluster: list, connect, info\n"
  printf "                   └─ namespace: list, create, delete, switch\n"
  echo
  printf "  ${COLOR_BLUE}helm${COLOR_RESET}          Helm chart management\n"
  printf "                   ├─ init, generate, install, upgrade, rollback, uninstall\n"
  printf "                   ├─ list, status, values, template, package\n"
  printf "                   └─ repo: add, remove, update, list\n"

  show_section "Deployment & Environments"
  printf "  ${COLOR_BLUE}env${COLOR_RESET}           Environment management (local/staging/prod)\n"
  printf "  ${COLOR_BLUE}deploy${COLOR_RESET}        Deploy with advanced strategies\n"
  printf "                   ├─ staging, production, rollback\n"
  printf "                   ├─ preview: ephemeral preview environments\n"
  printf "                   ├─ canary: percentage-based rollout\n"
  printf "                   └─ blue-green: instant traffic switching\n"
  echo
  printf "  ${COLOR_BLUE}prod${COLOR_RESET}          Production configuration and hardening\n"
  printf "  ${COLOR_BLUE}staging${COLOR_RESET}       Staging environment management\n"
  printf "  ${COLOR_BLUE}sync${COLOR_RESET}          Sync data between environments\n"
  printf "                   ├─ db, files, config, full\n"
  printf "                   ├─ auto: continuous file watching\n"
  printf "                   └─ watch: manual watch mode\n"

  show_section "Performance & Scaling (v0.4.6)"
  printf "  ${COLOR_BLUE}perf${COLOR_RESET}          Performance profiling and analysis\n"
  printf "  ${COLOR_BLUE}bench${COLOR_RESET}         Benchmarking and load testing\n"
  printf "  ${COLOR_BLUE}scale${COLOR_RESET}         Service scaling and autoscaling\n"
  printf "  ${COLOR_BLUE}migrate${COLOR_RESET}       Cross-environment migration\n"

  show_section "Operations & Monitoring (v0.4.6)"
  printf "  ${COLOR_BLUE}health${COLOR_RESET}        Health check management and monitoring\n"
  printf "  ${COLOR_BLUE}frontend${COLOR_RESET}      Frontend application management\n"
  printf "  ${COLOR_BLUE}history${COLOR_RESET}       Deployment and operation audit trail\n"
  printf "  ${COLOR_BLUE}config${COLOR_RESET}        Configuration management\n"

  show_section "Plugins (v0.4.8)"
  printf "  ${COLOR_BLUE}plugin${COLOR_RESET}        Plugin management and execution\n"
  printf "                   ├─ list: show available plugins\n"
  printf "                   ├─ install, remove, update, status\n"
  printf "                   └─ <plugin> <action>: run plugin commands\n"
  echo
  printf "  Available: stripe (billing), shopify (e-commerce), github (devops)\n"
  printf "  Planned: linear, intercom, resend, notion, airtable, plaid\n"

  show_section "Multi-Tenancy (v0.5.0)"
  printf "  ${COLOR_BLUE}tenant${COLOR_RESET}        Multi-tenant management\n"
  printf "                   ├─ Tenant lifecycle: init, create, list, show, suspend, activate, delete\n"
  printf "                   ├─ billing: Usage tracking, invoices, quotas, plans\n"
  printf "                   ├─ branding: Logo, colors, themes, custom CSS\n"
  printf "                   ├─ domains: Custom domains, SSL, verification\n"
  printf "                   ├─ email: Template management and testing\n"
  printf "                   └─ themes: Create, edit, preview, activate themes\n"

  show_section "Utility Commands"
  printf "  ${COLOR_BLUE}ci${COLOR_RESET}            CI/CD workflow generation (GitHub/GitLab)\n"
  printf "  ${COLOR_BLUE}completion${COLOR_RESET}    Shell completion (bash/zsh/fish)\n"
  echo
  echo "For command-specific help: nself help <command>"
  echo "                      or: nself <command> --help"
}

# Show command-specific help
show_command_help() {
  local command="$1"
  local command_file=""

  # Find command file (check multiple locations)
  if [[ -f "$SCRIPT_DIR/${command}.sh" ]]; then
    command_file="$SCRIPT_DIR/${command}.sh"
  elif [[ -f "$SCRIPT_DIR/../tools/dev/${command}.sh" ]]; then
    command_file="$SCRIPT_DIR/../tools/dev/${command}.sh"
  fi

  # Check if command exists
  if [[ -z "$command_file" ]] || [[ ! -f "$command_file" ]]; then
    log_error "Unknown command: $command"
    echo
    echo "Run 'nself help' to see available commands"
    return 1
  fi

  # Try to run the command with --help
  if bash "$command_file" --help 2>/dev/null; then
    return 0
  else
    # Fallback: show basic info
    echo "Help for: nself $command"
    echo
    echo "Run: nself $command --help"
    echo "Or check the documentation"
  fi
}

# Export for use as library
export -f cmd_help

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "help" || exit $?
  cmd_help "$@"
  exit_code=$?
  post_command "help" $exit_code
  exit $exit_code
fi
