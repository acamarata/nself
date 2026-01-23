#!/usr/bin/env bash

# config.sh - Configuration management
# v0.4.6 - Feedback implementation

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Color fallbacks
: "${COLOR_GREEN:=\033[0;32m}"
: "${COLOR_YELLOW:=\033[0;33m}"
: "${COLOR_RED:=\033[0;31m}"
: "${COLOR_CYAN:=\033[0;36m}"
: "${COLOR_RESET:=\033[0m}"
: "${COLOR_DIM:=\033[2m}"
: "${COLOR_BOLD:=\033[1m}"

# Show help
show_config_help() {
  cat << 'EOF'
nself config - Configuration management

Usage: nself config <subcommand> [options]

Subcommands:
  show                  Show current configuration
  get <key>             Get specific configuration value
  set <key> <value>     Set configuration value
  list                  List all configuration keys
  edit                  Open .env in editor
  validate              Validate configuration
  diff <env1> <env2>    Compare configurations between environments
  export                Export configuration (redacted secrets)
  import <file>         Import configuration from file
  reset                 Reset to defaults

Options:
  --env NAME            Target environment (default: current)
  --reveal              Show secret values (use with caution)
  --json                Output in JSON format
  --no-backup           Don't create backup before changes
  -h, --help            Show this help message

Examples:
  nself config show                    # Show current config
  nself config get POSTGRES_HOST       # Get specific value
  nself config set REDIS_ENABLED true  # Enable Redis
  nself config diff local staging      # Compare envs
  nself config validate                # Check configuration
  nself config export --json           # Export as JSON
EOF
}

# Initialize config environment
init_config() {
  load_env_with_priority

  CONFIG_DIR="${CONFIG_DIR:-.nself/config}"
  mkdir -p "$CONFIG_DIR"

  # Determine current env file
  local env="${ENV:-local}"
  case "$env" in
    local|dev) ENV_FILE=".env" ;;
    staging) ENV_FILE=".env.staging" ;;
    prod|production) ENV_FILE=".env.prod" ;;
    *) ENV_FILE=".env" ;;
  esac

  # Fallback to .env if specific file doesn't exist
  [[ ! -f "$ENV_FILE" ]] && ENV_FILE=".env"
}

# List of secret keys that should be redacted
SECRET_KEYS="PASSWORD|SECRET|TOKEN|KEY|CREDENTIAL|AUTH|PRIVATE"

# Check if key is secret
is_secret_key() {
  local key="$1"
  echo "$key" | grep -qiE "$SECRET_KEYS"
}

# Redact secret value
redact_value() {
  local value="$1"
  local reveal="${REVEAL:-false}"

  if [[ "$reveal" == "true" ]]; then
    echo "$value"
  else
    echo "********"
  fi
}

# Show current configuration
cmd_show() {
  local json_mode="${JSON_OUTPUT:-false}"
  local reveal="${REVEAL:-false}"

  init_config

  if [[ ! -f "$ENV_FILE" ]]; then
    log_error "Configuration file not found: $ENV_FILE"
    return 1
  fi

  if [[ "$json_mode" != "true" ]]; then
    show_command_header "nself config" "Configuration"
    echo ""
    printf "${COLOR_CYAN}➞ Environment: %s${COLOR_RESET}\n" "${ENV:-local}"
    printf "${COLOR_CYAN}➞ File: %s${COLOR_RESET}\n" "$ENV_FILE"
    echo ""
  fi

  # Group configurations
  local core_config=""
  local services_config=""
  local monitoring_config=""
  local custom_config=""
  local other_config=""

  while IFS= read -r line; do
    # Skip comments and empty lines
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// }" ]] && continue

    # Parse key=value
    local key="${line%%=*}"
    local value="${line#*=}"

    # Remove quotes
    value="${value#\"}"
    value="${value%\"}"
    value="${value#\'}"
    value="${value%\'}"

    # Redact secrets
    if is_secret_key "$key"; then
      value=$(redact_value "$value")
    fi

    # Categorize
    if [[ "$key" =~ ^(PROJECT_|ENV|BASE_DOMAIN|POSTGRES_|HASURA_|AUTH_) ]]; then
      core_config+="  $key=$value\n"
    elif [[ "$key" =~ ^(REDIS_|MINIO_|MAILPIT_|MEILISEARCH_|MLFLOW_|FUNCTIONS_) ]]; then
      services_config+="  $key=$value\n"
    elif [[ "$key" =~ ^(MONITORING_|PROMETHEUS_|GRAFANA_|LOKI_|TEMPO_) ]]; then
      monitoring_config+="  $key=$value\n"
    elif [[ "$key" =~ ^(CS_|FRONTEND_APP_) ]]; then
      custom_config+="  $key=$value\n"
    else
      other_config+="  $key=$value\n"
    fi
  done < "$ENV_FILE"

  if [[ "$json_mode" == "true" ]]; then
    printf '{'
    printf '"env": "%s", ' "${ENV:-local}"
    printf '"file": "%s", ' "$ENV_FILE"
    printf '"config": {'
    # Simplified JSON output
    local first=true
    while IFS= read -r line; do
      [[ "$line" =~ ^[[:space:]]*# ]] && continue
      [[ -z "${line// }" ]] && continue
      local key="${line%%=*}"
      local value="${line#*=}"
      value="${value#\"}"
      value="${value%\"}"
      if is_secret_key "$key"; then
        value=$(redact_value "$value")
      fi
      [[ "$first" != "true" ]] && printf ", "
      first=false
      printf '"%s": "%s"' "$key" "$value"
    done < "$ENV_FILE"
    printf '}}\n'
  else
    # Display categorized config
    if [[ -n "$core_config" ]]; then
      printf "${COLOR_CYAN}Core Configuration${COLOR_RESET}\n"
      printf "$core_config"
      echo ""
    fi

    if [[ -n "$services_config" ]]; then
      printf "${COLOR_CYAN}Services${COLOR_RESET}\n"
      printf "$services_config"
      echo ""
    fi

    if [[ -n "$monitoring_config" ]]; then
      printf "${COLOR_CYAN}Monitoring${COLOR_RESET}\n"
      printf "$monitoring_config"
      echo ""
    fi

    if [[ -n "$custom_config" ]]; then
      printf "${COLOR_CYAN}Custom Services & Frontends${COLOR_RESET}\n"
      printf "$custom_config"
      echo ""
    fi

    if [[ -n "$other_config" ]]; then
      printf "${COLOR_CYAN}Other${COLOR_RESET}\n"
      printf "$other_config"
      echo ""
    fi

    if [[ "$reveal" != "true" ]]; then
      log_info "Secret values redacted. Use --reveal to show."
    fi
  fi
}

# Get specific configuration value
cmd_get() {
  local key="$1"
  local json_mode="${JSON_OUTPUT:-false}"
  local reveal="${REVEAL:-false}"

  if [[ -z "$key" ]]; then
    log_error "Configuration key required"
    return 1
  fi

  init_config

  if [[ ! -f "$ENV_FILE" ]]; then
    log_error "Configuration file not found: $ENV_FILE"
    return 1
  fi

  # Search for key
  local value=$(grep "^${key}=" "$ENV_FILE" 2>/dev/null | head -1 | cut -d'=' -f2-)

  if [[ -z "$value" ]]; then
    if [[ "$json_mode" == "true" ]]; then
      printf '{"key": "%s", "value": null, "found": false}\n' "$key"
    else
      log_error "Key not found: $key"
    fi
    return 1
  fi

  # Remove quotes
  value="${value#\"}"
  value="${value%\"}"
  value="${value#\'}"
  value="${value%\'}"

  # Redact if secret
  local display_value="$value"
  if is_secret_key "$key" && [[ "$reveal" != "true" ]]; then
    display_value=$(redact_value "$value")
  fi

  if [[ "$json_mode" == "true" ]]; then
    printf '{"key": "%s", "value": "%s", "found": true}\n' "$key" "$display_value"
  else
    echo "$display_value"
  fi
}

# Set configuration value
cmd_set() {
  local key="$1"
  local value="$2"
  local no_backup="${NO_BACKUP:-false}"

  if [[ -z "$key" ]] || [[ -z "$value" ]]; then
    log_error "Both key and value required"
    log_info "Usage: nself config set <key> <value>"
    return 1
  fi

  init_config

  if [[ ! -f "$ENV_FILE" ]]; then
    log_error "Configuration file not found: $ENV_FILE"
    return 1
  fi

  # Create backup
  if [[ "$no_backup" != "true" ]]; then
    cp "$ENV_FILE" "${ENV_FILE}.bak"
  fi

  # Check if key exists
  if grep -q "^${key}=" "$ENV_FILE"; then
    # Update existing
    if [[ "$OSTYPE" == "darwin"* ]]; then
      sed -i '' "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
    else
      sed -i "s|^${key}=.*|${key}=${value}|" "$ENV_FILE"
    fi
    log_success "Updated: $key"
  else
    # Add new
    echo "${key}=${value}" >> "$ENV_FILE"
    log_success "Added: $key"
  fi

  if [[ "$no_backup" != "true" ]]; then
    log_info "Backup: ${ENV_FILE}.bak"
  fi

  log_info "Run 'nself build && nself restart' to apply changes"
}

# List all configuration keys
cmd_list() {
  local json_mode="${JSON_OUTPUT:-false}"

  init_config

  if [[ ! -f "$ENV_FILE" ]]; then
    log_error "Configuration file not found: $ENV_FILE"
    return 1
  fi

  if [[ "$json_mode" == "true" ]]; then
    printf '{"keys": ['
    local first=true
    while IFS= read -r line; do
      [[ "$line" =~ ^[[:space:]]*# ]] && continue
      [[ -z "${line// }" ]] && continue
      local key="${line%%=*}"
      [[ "$first" != "true" ]] && printf ", "
      first=false
      printf '"%s"' "$key"
    done < "$ENV_FILE"
    printf ']}\n'
  else
    while IFS= read -r line; do
      [[ "$line" =~ ^[[:space:]]*# ]] && continue
      [[ -z "${line// }" ]] && continue
      echo "${line%%=*}"
    done < "$ENV_FILE"
  fi
}

# Open configuration in editor
cmd_edit() {
  init_config

  if [[ ! -f "$ENV_FILE" ]]; then
    log_error "Configuration file not found: $ENV_FILE"
    return 1
  fi

  local editor="${EDITOR:-${VISUAL:-nano}}"

  log_info "Opening $ENV_FILE in $editor"
  "$editor" "$ENV_FILE"
}

# Validate configuration
cmd_validate() {
  local json_mode="${JSON_OUTPUT:-false}"

  init_config

  if [[ ! -f "$ENV_FILE" ]]; then
    log_error "Configuration file not found: $ENV_FILE"
    return 1
  fi

  if [[ "$json_mode" != "true" ]]; then
    show_command_header "nself config" "Validating Configuration"
    echo ""
  fi

  local errors=0
  local warnings=0
  local issues="["

  # Required keys
  local required_keys=(
    "PROJECT_NAME"
    "BASE_DOMAIN"
    "POSTGRES_PASSWORD"
    "HASURA_GRAPHQL_ADMIN_SECRET"
  )

  for key in "${required_keys[@]}"; do
    local value=$(grep "^${key}=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2-)
    if [[ -z "$value" ]]; then
      if [[ "$json_mode" != "true" ]]; then
        printf "  ${COLOR_RED}✗${COLOR_RESET} Missing required: %s\n" "$key"
      fi
      issues+="{\"type\": \"error\", \"key\": \"$key\", \"message\": \"Missing required key\"},"
      errors=$((errors + 1))
    fi
  done

  # Check for weak passwords
  local password_keys=$(grep -E "PASSWORD|SECRET" "$ENV_FILE" 2>/dev/null | cut -d'=' -f1)
  for key in $password_keys; do
    local value=$(grep "^${key}=" "$ENV_FILE" | cut -d'=' -f2-)
    if [[ ${#value} -lt 12 ]]; then
      if [[ "$json_mode" != "true" ]]; then
        printf "  ${COLOR_YELLOW}!${COLOR_RESET} Weak password: %s (< 12 chars)\n" "$key"
      fi
      issues+="{\"type\": \"warning\", \"key\": \"$key\", \"message\": \"Password too short\"},"
      warnings=$((warnings + 1))
    fi
  done

  # Check for default values
  local default_checks=(
    "POSTGRES_PASSWORD:postgres"
    "HASURA_GRAPHQL_ADMIN_SECRET:admin"
    "HASURA_GRAPHQL_ADMIN_SECRET:secret"
  )

  for check in "${default_checks[@]}"; do
    local key="${check%%:*}"
    local bad_value="${check#*:}"
    local value=$(grep "^${key}=" "$ENV_FILE" 2>/dev/null | cut -d'=' -f2-)
    if [[ "$value" == "$bad_value" ]]; then
      if [[ "$json_mode" != "true" ]]; then
        printf "  ${COLOR_YELLOW}!${COLOR_RESET} Default value: %s\n" "$key"
      fi
      issues+="{\"type\": \"warning\", \"key\": \"$key\", \"message\": \"Using default value\"},"
      warnings=$((warnings + 1))
    fi
  done

  # Check for duplicate keys
  local duplicates=$(grep -v "^#" "$ENV_FILE" | cut -d'=' -f1 | sort | uniq -d)
  for dup in $duplicates; do
    if [[ "$json_mode" != "true" ]]; then
      printf "  ${COLOR_YELLOW}!${COLOR_RESET} Duplicate key: %s\n" "$dup"
    fi
    issues+="{\"type\": \"warning\", \"key\": \"$dup\", \"message\": \"Duplicate key\"},"
    warnings=$((warnings + 1))
  done

  # Remove trailing comma and close array
  issues="${issues%,}]"

  if [[ "$json_mode" == "true" ]]; then
    printf '{"valid": %s, "errors": %d, "warnings": %d, "issues": %s}\n' \
      "$([[ $errors -eq 0 ]] && echo "true" || echo "false")" \
      "$errors" "$warnings" "$issues"
  else
    echo ""
    if [[ $errors -eq 0 ]] && [[ $warnings -eq 0 ]]; then
      log_success "Configuration valid"
    elif [[ $errors -eq 0 ]]; then
      log_warning "Configuration valid with $warnings warning(s)"
    else
      log_error "Configuration invalid: $errors error(s), $warnings warning(s)"
    fi
  fi

  [[ $errors -eq 0 ]]
}

# Compare configurations
cmd_diff() {
  local env1="${1:-local}"
  local env2="${2:-staging}"

  init_config

  # Determine files
  local file1 file2

  case "$env1" in
    local|dev) file1=".env" ;;
    staging) file1=".env.staging" ;;
    prod|production) file1=".env.prod" ;;
    *) file1="$env1" ;;
  esac

  case "$env2" in
    local|dev) file2=".env" ;;
    staging) file2=".env.staging" ;;
    prod|production) file2=".env.prod" ;;
    *) file2="$env2" ;;
  esac

  if [[ ! -f "$file1" ]]; then
    log_error "File not found: $file1"
    return 1
  fi

  if [[ ! -f "$file2" ]]; then
    log_error "File not found: $file2"
    return 1
  fi

  show_command_header "nself config" "Comparing $env1 ↔ $env2"
  echo ""

  # Create temp files with sorted, non-comment lines
  local tmp1=$(mktemp)
  local tmp2=$(mktemp)

  grep -v "^#" "$file1" | grep -v "^$" | sort > "$tmp1"
  grep -v "^#" "$file2" | grep -v "^$" | sort > "$tmp2"

  # Show diff
  local diff_output=$(diff -u "$tmp1" "$tmp2" 2>/dev/null || true)

  if [[ -z "$diff_output" ]]; then
    log_success "Configurations are identical"
  else
    printf "${COLOR_CYAN}➞ Differences${COLOR_RESET}\n"
    echo ""

    # Parse diff output
    echo "$diff_output" | while read -r line; do
      case "$line" in
        ---*|+++*|@@*) continue ;;
        -*)
          # Redact secrets
          local key="${line#-}"
          key="${key%%=*}"
          if is_secret_key "$key"; then
            printf "${COLOR_RED}-%s=********${COLOR_RESET}\n" "$key"
          else
            printf "${COLOR_RED}%s${COLOR_RESET}\n" "$line"
          fi
          ;;
        +*)
          local key="${line#+}"
          key="${key%%=*}"
          if is_secret_key "$key"; then
            printf "${COLOR_GREEN}+%s=********${COLOR_RESET}\n" "$key"
          else
            printf "${COLOR_GREEN}%s${COLOR_RESET}\n" "$line"
          fi
          ;;
        *)
          echo "$line"
          ;;
      esac
    done
  fi

  rm -f "$tmp1" "$tmp2"
}

# Export configuration
cmd_export() {
  local json_mode="${JSON_OUTPUT:-false}"
  local output_file="${OUTPUT_FILE:-}"
  local reveal="${REVEAL:-false}"

  init_config

  if [[ ! -f "$ENV_FILE" ]]; then
    log_error "Configuration file not found: $ENV_FILE"
    return 1
  fi

  local timestamp=$(date +%Y%m%d_%H%M%S)
  if [[ -z "$output_file" ]]; then
    output_file="${CONFIG_DIR}/export_${timestamp}.json"
  fi

  if [[ "$json_mode" != "true" ]]; then
    show_command_header "nself config" "Exporting Configuration"
    echo ""
  fi

  # Build JSON export
  printf '{\n  "exported": "%s",\n  "env": "%s",\n  "config": {\n' \
    "$(date -Iseconds)" "${ENV:-local}" > "$output_file"

  local first=true
  while IFS= read -r line; do
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// }" ]] && continue

    local key="${line%%=*}"
    local value="${line#*=}"
    value="${value#\"}"
    value="${value%\"}"

    if is_secret_key "$key" && [[ "$reveal" != "true" ]]; then
      value="********"
    fi

    [[ "$first" != "true" ]] && printf ",\n" >> "$output_file"
    first=false
    printf '    "%s": "%s"' "$key" "$value" >> "$output_file"
  done < "$ENV_FILE"

  printf '\n  }\n}\n' >> "$output_file"

  if [[ "$json_mode" == "true" ]]; then
    cat "$output_file"
  else
    log_success "Exported to: $output_file"
    [[ "$reveal" != "true" ]] && log_info "Secret values redacted. Use --reveal to include."
  fi
}

# Import configuration
cmd_import() {
  local import_file="$1"
  local no_backup="${NO_BACKUP:-false}"

  if [[ -z "$import_file" ]]; then
    log_error "Import file required"
    return 1
  fi

  if [[ ! -f "$import_file" ]]; then
    log_error "File not found: $import_file"
    return 1
  fi

  init_config

  show_command_header "nself config" "Importing Configuration"
  echo ""

  log_warning "This will overwrite current configuration"
  read -p "Continue? (y/N) " -n 1 -r
  echo
  if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    log_info "Import cancelled"
    return 1
  fi

  # Backup current
  if [[ -f "$ENV_FILE" ]] && [[ "$no_backup" != "true" ]]; then
    cp "$ENV_FILE" "${ENV_FILE}.bak"
    log_info "Backup: ${ENV_FILE}.bak"
  fi

  # Import based on file type
  if [[ "$import_file" == *.json ]]; then
    # Parse JSON (simplified)
    grep -o '"[^"]*": *"[^"]*"' "$import_file" | while read -r pair; do
      local key=$(echo "$pair" | cut -d'"' -f2)
      local value=$(echo "$pair" | cut -d'"' -f4)

      # Skip metadata keys
      [[ "$key" == "exported" || "$key" == "env" || "$key" == "config" ]] && continue
      [[ "$value" == "********" ]] && continue

      echo "${key}=${value}"
    done > "$ENV_FILE.new"

    if [[ -s "$ENV_FILE.new" ]]; then
      mv "$ENV_FILE.new" "$ENV_FILE"
      log_success "Configuration imported"
    else
      rm -f "$ENV_FILE.new"
      log_error "No valid configuration found in import file"
      return 1
    fi
  else
    # Assume .env format
    cp "$import_file" "$ENV_FILE"
    log_success "Configuration imported"
  fi

  log_info "Run 'nself build && nself restart' to apply changes"
}

# Reset to defaults
cmd_reset() {
  local force="${FORCE:-false}"

  init_config

  show_command_header "nself config" "Reset Configuration"
  echo ""

  log_warning "This will reset configuration to defaults"
  log_warning "Current configuration will be backed up"
  echo ""

  if [[ "$force" != "true" ]]; then
    read -p "Type 'RESET' to confirm: " confirm
    if [[ "$confirm" != "RESET" ]]; then
      log_info "Reset cancelled"
      return 1
    fi
  fi

  # Backup current
  if [[ -f "$ENV_FILE" ]]; then
    local backup="${ENV_FILE}.$(date +%Y%m%d_%H%M%S).bak"
    cp "$ENV_FILE" "$backup"
    log_info "Backup: $backup"
  fi

  # Check for .env.example
  if [[ -f ".env.example" ]]; then
    cp ".env.example" "$ENV_FILE"
    log_success "Reset from .env.example"
  else
    log_warning "No .env.example found"
    log_info "Run 'nself init' to create fresh configuration"
  fi
}

# Main command handler
cmd_config() {
  local subcommand="${1:-show}"

  # Check for help first
  if [[ "$subcommand" == "-h" ]] || [[ "$subcommand" == "--help" ]]; then
    show_config_help
    return 0
  fi

  # Parse global options
  local args=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --env)
        ENV="$2"
        shift 2
        ;;
      --reveal)
        REVEAL=true
        shift
        ;;
      --output)
        OUTPUT_FILE="$2"
        shift 2
        ;;
      --no-backup)
        NO_BACKUP=true
        shift
        ;;
      --force)
        FORCE=true
        shift
        ;;
      --json)
        JSON_OUTPUT=true
        shift
        ;;
      -h|--help)
        show_config_help
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
  subcommand="${1:-show}"

  case "$subcommand" in
    show)
      cmd_show
      ;;
    get)
      shift
      cmd_get "$@"
      ;;
    set)
      shift
      cmd_set "$@"
      ;;
    list)
      cmd_list
      ;;
    edit)
      cmd_edit
      ;;
    validate)
      cmd_validate
      ;;
    diff)
      shift
      cmd_diff "$@"
      ;;
    export)
      cmd_export
      ;;
    import)
      shift
      cmd_import "$@"
      ;;
    reset)
      cmd_reset
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      show_config_help
      return 1
      ;;
  esac
}

# Export for use as library
export -f cmd_config

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "config" || exit $?
  cmd_config "$@"
  exit_code=$?
  post_command "config" $exit_code
  exit $exit_code
fi
