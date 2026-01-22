#!/usr/bin/env bash
# env.sh - Environment management CLI command
# POSIX-compliant, no Bash 4+ features

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source dependencies
source "$LIB_DIR/utils/display.sh" 2>/dev/null || true
source "$LIB_DIR/utils/header.sh" 2>/dev/null || true
source "$LIB_DIR/utils/platform-compat.sh" 2>/dev/null || true

# Source environment modules
source "$LIB_DIR/env/create.sh" 2>/dev/null || true
source "$LIB_DIR/env/switch.sh" 2>/dev/null || true
source "$LIB_DIR/env/diff.sh" 2>/dev/null || true
source "$LIB_DIR/env/validate.sh" 2>/dev/null || true

# Show help
show_env_help() {
  cat <<EOF
nself env - Environment management

Usage: nself env <command> [options]

Commands:
  create <name> [template]   Create a new environment
  list                       List all environments
  switch <name>              Switch to an environment
  status                     Show current environment status
  info [name]                Show environment details
  diff <env1> <env2>         Compare two environments
  validate [name]            Validate environment configuration
  delete <name>              Delete an environment
  export <name>              Export environment configuration
  import <file>              Import environment configuration

Templates:
  local       Local development (default)
  staging     Staging environment
  prod        Production environment

Examples:
  nself env create staging            # Create staging environment
  nself env create prod production    # Create production environment
  nself env switch staging            # Switch to staging
  nself env diff local staging        # Compare local vs staging
  nself env validate prod             # Validate production config

Environment Structure:
  .environments/
    ├── local/
    │   ├── .env              # Environment configuration
    │   └── server.json       # Server connection info
    ├── staging/
    │   ├── .env
    │   ├── .env.secrets      # Sensitive values (chmod 600)
    │   └── server.json
    └── prod/
        ├── .env
        ├── .env.secrets
        └── server.json

Notes:
  • Environments inherit from .env.dev as base
  • .env.secrets files should have 600 permissions
  • Use 'nself env validate' before deploying
EOF
}

# Create subcommand
cmd_env_create() {
  local name=""
  local template="local"
  local force=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -f|--force)
        force=true
        shift
        ;;
      -h|--help)
        printf "Usage: nself env create <name> [template]\n\n"
        printf "Templates: local, staging, prod\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        if [[ -z "$name" ]]; then
          name="$1"
        else
          template="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$name" ]]; then
    log_error "Environment name is required"
    printf "Usage: nself env create <name> [template]\n"
    return 1
  fi

  env::create "$name" "$template" "$force"
}

# List subcommand
cmd_env_list() {
  env::list
}

# Switch subcommand
cmd_env_switch() {
  local name="$1"
  local preview=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --preview)
        preview=true
        shift
        ;;
      -h|--help)
        printf "Usage: nself env switch <name> [--preview]\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        name="$1"
        shift
        ;;
    esac
  done

  if [[ -z "$name" ]]; then
    log_error "Environment name is required"
    printf "Usage: nself env switch <name>\n"
    return 1
  fi

  if [[ "$preview" == "true" ]]; then
    env::preview_switch "$name"
  else
    env::switch "$name"
  fi
}

# Status subcommand
cmd_env_status() {
  local current
  current=$(env::get_current 2>/dev/null || echo "local")

  printf "Current environment: ${COLOR_GREEN}%s${COLOR_RESET}\n" "$current"

  local env_dir=".environments/$current"
  if [[ -d "$env_dir" ]]; then
    env::show_status "$current"
  else
    printf "\n${COLOR_YELLOW}Environment directory not found${COLOR_RESET}\n"
    printf "Create it with: nself env create %s\n" "$current"
  fi
}

# Info subcommand
cmd_env_info() {
  local name="${1:-}"

  if [[ -z "$name" ]]; then
    name=$(env::get_current 2>/dev/null || echo "local")
  fi

  env::info "$name"
}

# Diff subcommand
cmd_env_diff() {
  local env_a=""
  local env_b=""
  local show_values=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --values)
        show_values=true
        shift
        ;;
      -h|--help)
        printf "Usage: nself env diff <env1> <env2> [--values]\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        if [[ -z "$env_a" ]]; then
          env_a="$1"
        elif [[ -z "$env_b" ]]; then
          env_b="$1"
        fi
        shift
        ;;
    esac
  done

  if [[ -z "$env_a" ]] || [[ -z "$env_b" ]]; then
    log_error "Two environment names are required"
    printf "Usage: nself env diff <env1> <env2>\n"
    return 1
  fi

  env::diff "$env_a" "$env_b" "$show_values"
}

# Validate subcommand
cmd_env_validate() {
  local name=""
  local all=false
  local strict=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --all)
        all=true
        shift
        ;;
      --strict)
        strict=true
        shift
        ;;
      -h|--help)
        printf "Usage: nself env validate [name] [--all] [--strict]\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        name="$1"
        shift
        ;;
    esac
  done

  if [[ "$all" == "true" ]]; then
    env::validate_all
  else
    if [[ -z "$name" ]]; then
      name=$(env::get_current 2>/dev/null || echo "local")
    fi
    env::validate "$name" "$strict"
  fi
}

# Delete subcommand
cmd_env_delete() {
  local name="$1"
  local force=false

  while [[ $# -gt 0 ]]; do
    case "$1" in
      -f|--force)
        force=true
        shift
        ;;
      -h|--help)
        printf "Usage: nself env delete <name> [-f|--force]\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        name="$1"
        shift
        ;;
    esac
  done

  if [[ -z "$name" ]]; then
    log_error "Environment name is required"
    printf "Usage: nself env delete <name>\n"
    return 1
  fi

  env::delete "$name" "$force"
}

# Export subcommand
cmd_env_export() {
  local name="${1:-}"
  local output="${2:-}"

  if [[ -z "$name" ]]; then
    name=$(env::get_current 2>/dev/null || echo "local")
  fi

  local env_dir=".environments/$name"

  if [[ ! -d "$env_dir" ]]; then
    log_error "Environment '$name' does not exist"
    return 1
  fi

  # Create tarball excluding secrets
  local tarball="${output:-${name}-env-export.tar.gz}"

  tar -czf "$tarball" \
    --exclude='.env.secrets' \
    -C ".environments" \
    "$name" 2>/dev/null

  log_success "Exported environment to: $tarball"
  log_warning "Note: .env.secrets was NOT included for security"
}

# Import subcommand
cmd_env_import() {
  local file="$1"
  local name="${2:-}"

  if [[ -z "$file" ]]; then
    log_error "Import file is required"
    printf "Usage: nself env import <file> [name]\n"
    return 1
  fi

  if [[ ! -f "$file" ]]; then
    log_error "File not found: $file"
    return 1
  fi

  # Extract environment name from tarball if not provided
  if [[ -z "$name" ]]; then
    name=$(tar -tzf "$file" 2>/dev/null | head -1 | cut -d'/' -f1)
  fi

  if [[ -z "$name" ]]; then
    log_error "Could not determine environment name"
    return 1
  fi

  # Extract to environments directory
  mkdir -p ".environments"
  tar -xzf "$file" -C ".environments"

  log_success "Imported environment: $name"
  log_info "Remember to configure .env.secrets if needed"
}

# Main command handler
cmd_env() {
  local subcommand="${1:-}"
  shift || true

  # Show header for most commands
  case "$subcommand" in
    ""|--help|-h)
      show_command_header "nself env" "Environment management"
      show_env_help
      return 0
      ;;
  esac

  # Route to subcommand
  case "$subcommand" in
    create)
      show_command_header "nself env create" "Create new environment"
      cmd_env_create "$@"
      ;;
    list|ls)
      show_command_header "nself env list" "Available environments"
      cmd_env_list "$@"
      ;;
    switch|use)
      cmd_env_switch "$@"
      ;;
    status)
      show_command_header "nself env status" "Current environment"
      cmd_env_status "$@"
      ;;
    info|show)
      show_command_header "nself env info" "Environment details"
      cmd_env_info "$@"
      ;;
    diff|compare)
      show_command_header "nself env diff" "Compare environments"
      cmd_env_diff "$@"
      ;;
    validate|check)
      show_command_header "nself env validate" "Validate configuration"
      cmd_env_validate "$@"
      ;;
    delete|rm|remove)
      cmd_env_delete "$@"
      ;;
    export)
      cmd_env_export "$@"
      ;;
    import)
      cmd_env_import "$@"
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      printf "\nRun 'nself env --help' for usage information\n"
      return 1
      ;;
  esac
}

# Export for use as library
export -f cmd_env

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_env "$@"
fi
