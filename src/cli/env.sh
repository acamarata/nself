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

# Fallback logging (if display.sh failed to load)
if ! declare -f log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_warning >/dev/null 2>&1; then
  log_warning() { printf "\033[0;33m[WARNING]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi

# Fallback color definitions
: "${COLOR_GREEN:=\033[0;32m}"
: "${COLOR_YELLOW:=\033[0;33m}"
: "${COLOR_RED:=\033[0;31m}"
: "${COLOR_CYAN:=\033[0;36m}"
: "${COLOR_RESET:=\033[0m}"

# Fallback for show_command_header (if header.sh failed to load)
if ! declare -f show_command_header >/dev/null 2>&1; then
  show_command_header() { printf "\n%s - %s\n\n" "$1" "$2"; }
fi

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
  access [--check <env>]     Show your access level or test specific access
  load                       Load merged environment (respects hierarchy)

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

Environment File Hierarchy (cascading overrides):
  .env.dev        Base config for all developers (committed)
       ↓
  .env.local      Your machine overrides (gitignored)
       ↓
  .env.staging    Staging server only (SSH sync)
       ↓
  .env.prod       Production server only (SSH sync)
       ↓
  .secrets        Top-secret credentials (generated on server)

Access Levels:
  Dev        Local only (.env.dev + .env.local)
  Sr Dev     + staging access (SSH to staging server)
  Lead Dev   + prod + secrets (SSH to production server)

Notes:
  • Access is determined by SSH key authorization
  • Secrets are generated ON the server, never committed
  • Use 'nself env access' to check your current level
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
    env::show_status "$current" || true
  else
    printf "\n${COLOR_YELLOW}Environment directory not found${COLOR_RESET}\n"
    printf "Create it with: nself env create %s\n" "$current"
  fi

  # Always return success - status display completed
  return 0
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
  local project=false
  local env_file=""

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
      --project|-p)
        project=true
        shift
        ;;
      --file|-f)
        env_file="$2"
        project=true
        shift 2
        ;;
      -h|--help)
        printf "Usage: nself env validate [name] [--all] [--strict] [--project] [--file <path>]\n\n"
        printf "Options:\n"
        printf "  --all       Validate all environments\n"
        printf "  --strict    Treat warnings as errors\n"
        printf "  --project   Validate current project .env (not environment directory)\n"
        printf "  --file      Validate specific .env file\n\n"
        printf "Examples:\n"
        printf "  nself env validate             # Validate current environment\n"
        printf "  nself env validate --project   # Validate project .env file\n"
        printf "  nself env validate prod        # Validate production environment\n"
        printf "  nself env validate --all       # Validate all environments\n"
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

  # Project-level validation (validates .env file directly)
  if [[ "$project" == "true" ]] || [[ -n "$env_file" ]]; then
    if command -v env::validate_project >/dev/null 2>&1; then
      env::validate_project "${env_file:-.env}"
    else
      log_error "Project validation not available"
      return 1
    fi
  elif [[ "$all" == "true" ]]; then
    env::validate_all
  else
    # Default: check if running in a project without environments directory
    if [[ -z "$name" ]] && [[ ! -d ".environments" ]] && [[ -f ".env" ]]; then
      # No environments dir but has .env - validate project
      if command -v env::validate_project >/dev/null 2>&1; then
        env::validate_project ".env"
      else
        log_warning "No environments directory found"
        log_info "Run: nself env validate --project  to validate .env file"
        return 1
      fi
    else
      if [[ -z "$name" ]]; then
        name=$(env::get_current 2>/dev/null || echo "local")
      fi
      env::validate "$name" "$strict"
    fi
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

# Access subcommand - check user's access level
cmd_env_access() {
  local check_env=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --check)
        check_env="$2"
        shift 2
        ;;
      -h|--help)
        printf "Usage: nself env access [--check <env>]\n\n"
        printf "Check your environment access level based on SSH keys.\n\n"
        printf "Options:\n"
        printf "  --check <env>  Test access to specific environment (staging/prod)\n"
        return 0
        ;;
      *)
        shift
        ;;
    esac
  done

  # Get server configs
  local staging_server=""
  local prod_server=""

  if [[ -f ".environments/staging/server.json" ]]; then
    staging_server=$(grep -o '"host"[[:space:]]*:[[:space:]]*"[^"]*"' .environments/staging/server.json 2>/dev/null | cut -d'"' -f4)
  fi

  if [[ -f ".environments/prod/server.json" ]]; then
    prod_server=$(grep -o '"host"[[:space:]]*:[[:space:]]*"[^"]*"' .environments/prod/server.json 2>/dev/null | cut -d'"' -f4)
  fi

  # If checking specific environment
  if [[ -n "$check_env" ]]; then
    case "$check_env" in
      staging)
        if [[ -z "$staging_server" ]]; then
          log_error "No staging server configured"
          printf "Configure with: nself sync profile add staging\n"
          return 1
        fi
        printf "Testing SSH access to staging (%s)...\n" "$staging_server"
        if ssh -o BatchMode=yes -o ConnectTimeout=5 "$staging_server" exit 2>/dev/null; then
          log_success "You have staging access"
          return 0
        else
          log_error "No staging access (SSH key not authorized)"
          return 1
        fi
        ;;
      prod|production)
        if [[ -z "$prod_server" ]]; then
          log_error "No production server configured"
          printf "Configure with: nself sync profile add prod\n"
          return 1
        fi
        printf "Testing SSH access to production (%s)...\n" "$prod_server"
        if ssh -o BatchMode=yes -o ConnectTimeout=5 "$prod_server" exit 2>/dev/null; then
          log_success "You have production access"
          return 0
        else
          log_error "No production access (SSH key not authorized)"
          return 1
        fi
        ;;
      *)
        log_error "Unknown environment: $check_env"
        printf "Valid environments: staging, prod\n"
        return 1
        ;;
    esac
  fi

  # Show full access report
  printf "\n"
  printf "Your Environment Access Level\n"
  printf "==============================\n\n"

  local has_staging=false
  local has_prod=false
  local role="Dev"

  # Check local access (always yes)
  printf "  Local (.env.dev + .env.local)   %s\n" "${COLOR_GREEN}Yes${COLOR_RESET}"

  # Check staging access
  if [[ -n "$staging_server" ]]; then
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$staging_server" exit 2>/dev/null; then
      printf "  Staging (.env.staging)          %s\n" "${COLOR_GREEN}Yes${COLOR_RESET}"
      has_staging=true
      role="Sr Dev"
    else
      printf "  Staging (.env.staging)          %s\n" "${COLOR_RED}No${COLOR_RESET}"
    fi
  else
    printf "  Staging (.env.staging)          %s\n" "${COLOR_YELLOW}Not configured${COLOR_RESET}"
  fi

  # Check prod access
  if [[ -n "$prod_server" ]]; then
    if ssh -o BatchMode=yes -o ConnectTimeout=5 "$prod_server" exit 2>/dev/null; then
      printf "  Production (.env.prod)          %s\n" "${COLOR_GREEN}Yes${COLOR_RESET}"
      printf "  Secrets (.secrets)              %s\n" "${COLOR_GREEN}Yes${COLOR_RESET}"
      has_prod=true
      role="Lead Dev"
    else
      printf "  Production (.env.prod)          %s\n" "${COLOR_RED}No${COLOR_RESET}"
      printf "  Secrets (.secrets)              %s\n" "${COLOR_RED}No${COLOR_RESET}"
    fi
  else
    printf "  Production (.env.prod)          %s\n" "${COLOR_YELLOW}Not configured${COLOR_RESET}"
    printf "  Secrets (.secrets)              %s\n" "${COLOR_YELLOW}Not configured${COLOR_RESET}"
  fi

  printf "\n"
  printf "Your role: ${COLOR_CYAN}%s${COLOR_RESET}\n" "$role"
  printf "\n"

  case "$role" in
    "Dev")
      printf "You can:\n"
      printf "  - Edit .env.dev and .env.local\n"
      printf "  - Run 'nself env switch local'\n"
      printf "\n"
      printf "To get staging access, ask your tech lead to add your SSH key.\n"
      ;;
    "Sr Dev")
      printf "You can:\n"
      printf "  - Edit .env.dev and .env.local\n"
      printf "  - Run 'nself sync pull staging' to get staging config\n"
      printf "  - Run 'nself env switch staging'\n"
      printf "\n"
      printf "To get prod access, ask your tech lead to add your SSH key.\n"
      ;;
    "Lead Dev")
      printf "You have full access:\n"
      printf "  - Run 'nself sync pull staging' for staging config\n"
      printf "  - Run 'nself sync pull prod' for production config\n"
      printf "  - Run 'nself sync pull secrets' for production secrets\n"
      printf "  - Run 'nself env switch <any>' to switch environments\n"
      ;;
  esac
}

# Load subcommand - merge environment files respecting hierarchy
cmd_env_load() {
  local target="${1:-local}"
  local output="${2:-.env}"

  printf "Loading environment: %s\n" "$target"

  # Start with empty merged file
  local merged_file
  merged_file=$(mktemp)

  # Layer 1: .env.dev (base for all)
  if [[ -f ".env.dev" ]]; then
    cat ".env.dev" >> "$merged_file"
    printf "  + .env.dev (base)\n"
  fi

  case "$target" in
    local)
      # Layer 2: .env.local (developer overrides)
      if [[ -f ".env.local" ]]; then
        printf "\n# --- .env.local overrides ---\n" >> "$merged_file"
        cat ".env.local" >> "$merged_file"
        printf "  + .env.local (your overrides)\n"
      fi
      ;;
    staging)
      # Layer 2: .env.staging
      if [[ -f ".env.staging" ]]; then
        printf "\n# --- .env.staging overrides ---\n" >> "$merged_file"
        cat ".env.staging" >> "$merged_file"
        printf "  + .env.staging\n"
      elif [[ -f ".environments/staging/.env" ]]; then
        printf "\n# --- .env.staging overrides ---\n" >> "$merged_file"
        cat ".environments/staging/.env" >> "$merged_file"
        printf "  + .environments/staging/.env\n"
      else
        log_warning "No staging config found. Run: nself sync pull staging"
      fi
      ;;
    prod|production)
      # Layer 2: .env.prod
      if [[ -f ".env.prod" ]]; then
        printf "\n# --- .env.prod overrides ---\n" >> "$merged_file"
        cat ".env.prod" >> "$merged_file"
        printf "  + .env.prod\n"
      elif [[ -f ".environments/prod/.env" ]]; then
        printf "\n# --- .env.prod overrides ---\n" >> "$merged_file"
        cat ".environments/prod/.env" >> "$merged_file"
        printf "  + .environments/prod/.env\n"
      else
        log_warning "No prod config found. Run: nself sync pull prod"
      fi

      # Layer 3: .secrets (only for prod)
      if [[ -f ".secrets" ]]; then
        printf "\n# --- .secrets overrides ---\n" >> "$merged_file"
        cat ".secrets" >> "$merged_file"
        printf "  + .secrets (production secrets)\n"
      fi
      ;;
    *)
      log_error "Unknown target: $target"
      rm -f "$merged_file"
      return 1
      ;;
  esac

  # Write merged output
  mv "$merged_file" "$output"
  chmod 600 "$output"

  printf "\n"
  log_success "Created merged environment: $output"
  printf "Variables loaded: %d\n" "$(grep -c '^[A-Z]' "$output" 2>/dev/null || echo 0)"
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
    access|whoami|level)
      show_command_header "nself env access" "Your access level"
      cmd_env_access "$@"
      ;;
    load|merge)
      show_command_header "nself env load" "Merge environment files"
      cmd_env_load "$@"
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
