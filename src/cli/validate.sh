#!/usr/bin/env bash

# validate.sh - Configuration validation with schema checking

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Validation rules
REQUIRED_VARS=(
  "PROJECT_NAME"
  "BASE_DOMAIN"
  "POSTGRES_PASSWORD"
  "HASURA_GRAPHQL_ADMIN_SECRET"
)

OPTIONAL_VARS=(
  "ENV"
  "SSL_ENABLED"
  "MONITORING_ENABLED"
  "BACKUP_ENABLED"
  "SERVICES_ENABLED"
)

# Dangerous combinations
DANGEROUS_COMBOS=(
  "ENV=production:SSL_ENABLED=false:Production requires SSL"
  "ENV=production:DEBUG=true:Debug should be disabled in production"
  "POSTGRES_PASSWORD=postgres:Default password detected"
  "HASURA_GRAPHQL_ADMIN_SECRET=admin:Default admin secret detected"
)

# Show help
show_validate_help() {
  echo "nself validate - Validate configuration files"
  echo ""
  echo "Usage: nself validate [options] [file]"
  echo ""
  echo "Options:"
  echo "  --profile <name>   Validate against profile (dev, staging, prod)"
  echo "  --strict           Fail on warnings"
  echo "  --fix              Attempt to fix issues"
  echo "  -q, --quiet        Suppress output except errors"
  echo "  -h, --help         Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself validate                    # Validate .env"
  echo "  nself validate --profile prod     # Validate for production"
  echo "  nself validate --fix              # Fix issues automatically"
}

# Validate required variables
validate_required_vars() {
  local file="$1"
  local errors=0
  
  for var in "${REQUIRED_VARS[@]}"; do
    if ! grep -q "^${var}=" "$file" 2>/dev/null; then
      log_error "Missing required variable: $var"
      errors=$((errors + 1))
    else
      local value=$(grep "^${var}=" "$file" | cut -d'=' -f2- | tr -d '"' | tr -d "'")
      if [[ -z "$value" ]]; then
        log_error "Empty value for required variable: $var"
        errors=$((errors + 1))
      fi
    fi
  done
  
  return $errors
}

# Check for dangerous combinations
check_dangerous_combos() {
  local file="$1"
  local warnings=0
  
  # Load environment
  set -a
  source "$file" 2>/dev/null || true
  set +a
  
  for combo in "${DANGEROUS_COMBOS[@]}"; do
    IFS=':' read -r var1 var2 message <<< "$combo"
    
    # Parse variable and value
    local check_var="${var1%=*}"
    local check_val="${var1#*=}"
    
    # Get actual value
    local actual_val="${!check_var}"
    
    if [[ "$actual_val" == "$check_val" ]]; then
      if [[ -n "$var2" ]] && [[ "$var2" != "$message" ]]; then
        # Check second condition
        local check_var2="${var2%=*}"
        local check_val2="${var2#*=}"
        local actual_val2="${!check_var2}"
        
        if [[ "$actual_val2" == "$check_val2" ]]; then
          log_warning "Dangerous combination: $message"
          warnings=$((warnings + 1))
        fi
      else
        log_warning "Security issue: $message"
        warnings=$((warnings + 1))
      fi
    fi
  done
  
  return $warnings
}

# Validate for specific profile
validate_profile() {
  local file="$1"
  local profile="$2"
  local errors=0
  
  # Load environment
  set -a
  source "$file" 2>/dev/null || true
  set +a
  
  case "$profile" in
    prod|production)
      # Production checks
      if [[ "${SSL_ENABLED:-true}" != "true" ]]; then
        log_error "Production requires SSL_ENABLED=true"
        errors=$((errors + 1))
      fi
      
      if [[ "${DEBUG:-false}" == "true" ]]; then
        log_error "Production requires DEBUG=false"
        errors=$((errors + 1))
      fi
      
      if [[ "${MONITORING_ENABLED:-false}" != "true" ]]; then
        log_warning "Production should have MONITORING_ENABLED=true"
      fi
      
      if [[ "${BACKUP_ENABLED:-false}" != "true" ]]; then
        log_warning "Production should have BACKUP_ENABLED=true"
      fi
      
      # Check password strength
      if [[ ${#POSTGRES_PASSWORD} -lt 16 ]]; then
        log_error "Production requires strong passwords (16+ characters)"
        errors=$((errors + 1))
      fi
      ;;
    
    staging)
      # Staging checks
      if [[ "${SSL_ENABLED:-true}" != "true" ]]; then
        log_warning "Staging should have SSL_ENABLED=true"
      fi
      
      if [[ "${MONITORING_ENABLED:-false}" != "true" ]]; then
        log_info "Consider enabling monitoring for staging"
      fi
      ;;
    
    dev|development)
      # Development checks
      if [[ "${DEBUG:-true}" != "true" ]]; then
        log_info "Development typically has DEBUG=true"
      fi
      ;;
    
    *)
      log_error "Unknown profile: $profile"
      return 1
      ;;
  esac
  
  return $errors
}

# Validate syntax
validate_syntax() {
  local file="$1"
  local errors=0
  
  # Check for basic syntax issues
  while IFS= read -r line; do
    # Skip comments and empty lines
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "$line" ]] && continue
    
    # Check for valid KEY=VALUE format
    if ! [[ "$line" =~ ^[A-Z_][A-Z0-9_]*= ]]; then
      log_error "Invalid syntax: $line"
      errors=$((errors + 1))
    fi
    
    # Check for unescaped quotes
    if [[ "$line" =~ [^\\][\"\'] ]]; then
      local key="${line%%=*}"
      local value="${line#*=}"
      
      # Count quotes
      local quote_count=$(echo "$value" | tr -cd '"' | wc -c)
      local squote_count=$(echo "$value" | tr -cd "'" | wc -c)
      
      if [[ $((quote_count % 2)) -ne 0 ]] || [[ $((squote_count % 2)) -ne 0 ]]; then
        log_warning "Unmatched quotes in: $key"
      fi
    fi
  done < "$file"
  
  return $errors
}

# Fix common issues
fix_issues() {
  local file="$1"
  local fixed=0
  
  # Create backup with verification
  local backup_file="${file}.backup-$(date +%Y%m%d-%H%M%S)"
  if ! cp "$file" "$backup_file"; then
    log_error "Failed to create backup"
    return 1
  fi
  
  # Work on temp file for safety
  local temp_file="${file}.fixing.$$"
  cp "$file" "$temp_file" || { log_error "Failed to create temp file"; return 1; }
  
  # Fix missing required variables
  for var in "${REQUIRED_VARS[@]}"; do
    if ! grep -q "^${var}=" "$file" 2>/dev/null; then
      case "$var" in
        PROJECT_NAME)
          echo "PROJECT_NAME=my-project" >> "$file"
          log_info "Added missing PROJECT_NAME"
          fixed=$((fixed + 1))
          ;;
        BASE_DOMAIN)
          echo "BASE_DOMAIN=local.nself.org" >> "$file"
          log_info "Added missing BASE_DOMAIN"
          fixed=$((fixed + 1))
          ;;
        POSTGRES_PASSWORD)
          if command -v openssl >/dev/null 2>&1; then
            local password
            password="$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)"
          else
            # Fallback to /dev/urandom if openssl not available
            local password
            password="$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 25)"
          fi
          echo "POSTGRES_PASSWORD=$password" >> "$file"
          log_info "Generated secure POSTGRES_PASSWORD"
          fixed=$((fixed + 1))
          ;;
        HASURA_GRAPHQL_ADMIN_SECRET)
          if command -v openssl >/dev/null 2>&1; then
            local secret
            secret="$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)"
          else
            # Fallback to /dev/urandom if openssl not available
            local secret
            secret="$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 25)"
          fi
          echo "HASURA_GRAPHQL_ADMIN_SECRET=$secret" >> "$file"
          log_info "Generated secure HASURA_GRAPHQL_ADMIN_SECRET"
          fixed=$((fixed + 1))
          ;;
      esac
    fi
  done
  
  # Fix dangerous defaults
  local new_password
  if command -v openssl >/dev/null 2>&1; then
    new_password="$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)"
  else
    new_password="$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 25)"
  fi
  
  if ! sed -i.tmp "s/^POSTGRES_PASSWORD=postgres$/POSTGRES_PASSWORD=${new_password}/" "$file"; then
    log_error "Failed to update POSTGRES_PASSWORD"
    return 1
  fi
  
  local new_secret
  if command -v openssl >/dev/null 2>&1; then
    new_secret="$(openssl rand -base64 32 | tr -d "=+/" | cut -c1-25)"
  else
    new_secret="$(LC_ALL=C tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 25)"
  fi
  
  if ! sed -i.tmp "s/^HASURA_GRAPHQL_ADMIN_SECRET=admin$/HASURA_GRAPHQL_ADMIN_SECRET=${new_secret}/" "$file"; then
    log_error "Failed to update HASURA_GRAPHQL_ADMIN_SECRET"
    return 1
  fi
  
  rm -f "${file}.tmp"
  
  # Replace original with fixed temp file only if successful
  if [[ -f "$temp_file" ]]; then
    if mv "$temp_file" "$file"; then
      log_success "Applied $fixed fixes"
    else
      log_error "Failed to apply fixes"
      mv "$backup_file" "$file"  # Restore backup
      return 1
    fi
  fi
  
  return $fixed
}

# Main validation function
cmd_validate() {
  local file=".env"
  local profile=""
  local strict=false
  local fix=false
  local quiet=false
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --profile)
        profile="$2"
        shift 2
        ;;
      --strict)
        strict=true
        shift
        ;;
      --fix)
        fix=true
        shift
        ;;
      -q|--quiet)
        quiet=true
        shift
        ;;
      -h|--help)
        show_validate_help
        return 0
        ;;
      *)
        if [[ -z "$file" ]] && [[ -f "$1" ]]; then
          file="$1"
        elif [[ -z "$file" ]]; then
          file="$1"  # Will be checked later if it exists
        fi
        shift
        ;;
    esac
  done
  
  if [[ "$quiet" != "true" ]]; then
    show_command_header "nself validate" "Configuration validation"
    echo "Validating: $file"
    echo ""
  fi
  
  # Check if file exists
  if [[ ! -f "$file" ]]; then
    log_error "File not found: $file"
    return 1
  fi
  
  local total_errors=0
  local total_warnings=0
  
  # Run validations
  if [[ "$quiet" != "true" ]]; then
    echo "Checking syntax..."
  fi
  validate_syntax "$file" || total_errors=$((total_errors + $?))
  
  if [[ "$quiet" != "true" ]]; then
    echo "Checking required variables..."
  fi
  validate_required_vars "$file" || total_errors=$((total_errors + $?))
  
  if [[ "$quiet" != "true" ]]; then
    echo "Checking for security issues..."
  fi
  check_dangerous_combos "$file" || total_warnings=$((total_warnings + $?))
  
  # Profile validation if specified
  if [[ -n "$profile" ]]; then
    if [[ "$quiet" != "true" ]]; then
      echo "Validating for profile: $profile..."
    fi
    validate_profile "$file" "$profile" || total_errors=$((total_errors + $?))
  fi
  
  # Fix issues if requested
  if [[ "$fix" == "true" ]] && [[ $total_errors -gt 0 ]]; then
    echo ""
    log_info "Attempting to fix issues..."
    fix_issues "$file" || true
    
    # Re-validate after fixes
    echo ""
    log_info "Re-validating after fixes..."
    total_errors=0
    total_warnings=0
    validate_syntax "$file" || total_errors=$((total_errors + $?))
    validate_required_vars "$file" || total_errors=$((total_errors + $?))
    check_dangerous_combos "$file" || total_warnings=$((total_warnings + $?))
  fi
  
  # Summary
  if [[ "$quiet" != "true" ]]; then
    echo ""
    echo "Validation Summary"
    echo "────────────────────────────────────────────────"
    
    if [[ $total_errors -eq 0 ]] && [[ $total_warnings -eq 0 ]]; then
      log_success "Configuration is valid!"
    elif [[ $total_errors -eq 0 ]]; then
      log_warning "Configuration has $total_warnings warning(s)"
      if [[ "$strict" == "true" ]]; then
        return 1
      fi
    else
      log_error "Configuration has $total_errors error(s) and $total_warnings warning(s)"
      
      if [[ "$fix" != "true" ]]; then
        echo ""
        log_info "Run 'nself validate --fix' to attempt automatic fixes"
      fi
      
      return 1
    fi
  else
    # Quiet mode - just return status
    if [[ $total_errors -gt 0 ]]; then
      return 1
    elif [[ "$strict" == "true" ]] && [[ $total_warnings -gt 0 ]]; then
      return 1
    fi
  fi
  
  return 0
}

# Export for use as library
export -f cmd_validate

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "validate" || exit $?
  cmd_validate "$@"
  exit_code=$?
  post_command "validate" $exit_code
  exit $exit_code
fi