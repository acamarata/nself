#!/usr/bin/env bash
# init-wizard-refactored.sh - Refactored configuration wizard using modules
# POSIX-compliant, no Bash 4+ features

# Determine directories
WIZARD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
INIT_LIB_DIR="$(dirname "$WIZARD_DIR")"
ROOT_DIR="$(dirname "$(dirname "$(dirname "$INIT_LIB_DIR")")")"

# Source required modules with existence checks
for module in "$WIZARD_DIR/prompts.sh" "$WIZARD_DIR/detection.sh" "$WIZARD_DIR/templates.sh" "$WIZARD_DIR/hosts-helper.sh"; do
  if [[ ! -f "$module" ]]; then
    echo "Error: Required wizard module not found: $module" >&2
    exit 1
  fi
  source "$module"
done

# Source from lib/utils and lib/wizard
for lib in "$INIT_LIB_DIR/../utils/display.sh" "$INIT_LIB_DIR/../utils/env.sh" "$INIT_LIB_DIR/../wizard/environment-manager.sh"; do
  if [[ ! -f "$lib" ]]; then
    echo "Error: Required library not found: $lib" >&2
    exit 1
  fi
  source "$lib"
done

# Source the wizard core module
if [[ -f "$WIZARD_DIR/wizard-core.sh" ]]; then
  source "$WIZARD_DIR/wizard-core.sh"
fi

# Helper functions for backward compatibility
show_wizard_header() {
  local title="$1"
  local subtitle="$2"

  echo "╔══════════════════════════════════════════════════════════╗"
  echo "║ $title"
  echo "║ $subtitle"
  echo "╚══════════════════════════════════════════════════════════╝"
  echo ""
}

show_wizard_step() {
  local current="$1"
  local total="$2"
  local title="$3"

  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo " Step $current of $total: $title"
  echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
  echo ""
}

press_any_key() {
  echo "Press any key to continue..."
  read -n 1 -s -r
}

prompt_input() {
  local prompt="$1"
  local default="$2"
  local var_name="$3"
  local pattern="${4:-.*}"

  while true; do
    echo -n "$prompt"
    if [[ -n "$default" ]]; then
      echo -n " [$default]: "
    else
      echo -n ": "
    fi

    read -r input_value

    # Use default if empty
    if [[ -z "$input_value" ]] && [[ -n "$default" ]]; then
      input_value="$default"
    fi

    # Validate pattern
    if [[ "$input_value" =~ $pattern ]]; then
      eval "$var_name='$input_value'"
      break
    else
      echo "Invalid input. Please try again."
    fi
  done
}

prompt_password() {
  local prompt="$1"
  local var_name="$2"

  echo -n "$prompt: "
  read -s -r password_value
  echo ""
  eval "$var_name='$password_value'"
}

confirm_action() {
  local prompt="$1"
  echo -n "$prompt [y/N]: "
  read -r response
  case "$response" in
    [yY][eE][sS]|[yY]) return 0 ;;
    *) return 1 ;;
  esac
}

select_option() {
  local prompt="$1"
  local -n options=$2
  local var_name="$3"

  echo "$prompt:"
  local i=0
  for option in "${options[@]}"; do
    echo "  $((i+1)). $option"
    i=$((i+1))
  done

  while true; do
    echo -n "Select [1-${#options[@]}]: "
    read -r selection

    if [[ "$selection" =~ ^[0-9]+$ ]] && [[ $selection -ge 1 ]] && [[ $selection -le ${#options[@]} ]]; then
      eval "$var_name=$((selection-1))"
      break
    else
      echo "Invalid selection. Please try again."
    fi
  done
}

log_info() {
  echo "ℹ $1"
}

# Main wizard function
run_config_wizard() {
  # Use the modular wizard if available
  if command -v run_modular_wizard >/dev/null 2>&1; then
    run_modular_wizard "$@"
  else
    echo "Error: Modular wizard not available" >&2
    exit 1
  fi
}

# Quick start templates for backward compatibility
apply_template() {
  local template="$1"
  local env_file="${2:-.env}"

  case "$template" in
    demo)
      cp "$ROOT_DIR/templates/demo/.env.demo" "$env_file"
      echo "Demo template applied to $env_file"
      ;;
    minimal)
      cat > "$env_file" <<'EOF'
PROJECT_NAME=myproject
BASE_DOMAIN=localhost
ENV=dev

# Database
POSTGRES_ENABLED=true
POSTGRES_DB=myproject
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres

# Services
NGINX_ENABLED=true
SSL_ENABLED=false
EOF
      echo "Minimal template applied to $env_file"
      ;;
    production)
      cat > "$env_file" <<'EOF'
PROJECT_NAME=myproject
BASE_DOMAIN=example.com
ENV=prod

# Database
POSTGRES_ENABLED=true
POSTGRES_DB=myproject
POSTGRES_USER=postgres
POSTGRES_PASSWORD=CHANGE_ME

# Core Services
NGINX_ENABLED=true
SSL_ENABLED=true
SSL_PROVIDER=letsencrypt
LETSENCRYPT_EMAIL=admin@example.com

# Security
HASURA_ADMIN_SECRET=CHANGE_ME
JWT_SECRET=CHANGE_ME

# Services
HASURA_ENABLED=true
AUTH_ENABLED=true
STORAGE_ENABLED=true
REDIS_ENABLED=true
EOF
      echo "Production template applied to $env_file"
      ;;
    *)
      echo "Unknown template: $template"
      return 1
      ;;
  esac
}

# Main entry point
main() {
  local mode="${1:-wizard}"

  case "$mode" in
    wizard)
      run_config_wizard
      ;;
    template)
      local template="${2:-demo}"
      apply_template "$template"
      ;;
    *)
      echo "Usage: $0 [wizard|template <name>]"
      exit 1
      ;;
  esac
}

# Export functions
export -f show_wizard_header
export -f show_wizard_step
export -f press_any_key
export -f prompt_input
export -f prompt_password
export -f confirm_action
export -f select_option
export -f log_info
export -f run_config_wizard
export -f apply_template

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi