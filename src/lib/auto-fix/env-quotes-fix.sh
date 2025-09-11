#!/usr/bin/env bash
# env-quotes-fix.sh - Auto-fix unquoted environment values with spaces
# This prevents silent failures when sourcing env files with 'set -a'

# Function to check if a value needs quotes
needs_quotes() {
  local value="$1"
  
  # Skip if already quoted
  if [[ "$value" =~ ^\".*\"$ ]] || [[ "$value" =~ ^\'.*\'$ ]]; then
    return 1
  fi
  
  # Check if contains spaces or special characters that need quoting
  # Extended list to catch more problematic cases
  if [[ "$value" =~ [[:space:]] ]] || \
     [[ "$value" =~ [\*\?\[\]\{\}\(\)\!\|\&\;\<\>\`\$] ]] || \
     [[ "$value" =~ ^[0-9]+[[:space:]] ]]; then
    return 0
  fi
  
  return 1
}

# Function to fix unquoted values in an env file
fix_env_file_quotes() {
  local env_file="${1:-.env}"
  local fixed_count=0
  local temp_file=$(mktemp)
  
  if [[ ! -f "$env_file" ]]; then
    return 0
  fi
  
  # Process each line
  while IFS= read -r line || [[ -n "$line" ]]; do
    # Skip comments and empty lines
    if [[ "$line" =~ ^[[:space:]]*# ]] || [[ -z "${line// }" ]]; then
      echo "$line" >> "$temp_file"
      continue
    fi
    
    # Check if it's a variable assignment
    if [[ "$line" =~ ^([A-Z_][A-Z0-9_]*)=(.*)$ ]]; then
      local var_name="${BASH_REMATCH[1]}"
      local var_value="${BASH_REMATCH[2]}"
      
      # Check if value needs quotes
      if needs_quotes "$var_value"; then
        # Add quotes around the value
        echo "${var_name}=\"${var_value}\"" >> "$temp_file"
        ((fixed_count++))
        # Log the fix if verbose
        if [[ "${VERBOSE:-false}" == "true" ]] || [[ "${DEBUG:-false}" == "true" ]]; then
          echo "  Fixed: ${var_name}=${var_value} → ${var_name}=\"${var_value}\"" >&2
        fi
      else
        echo "$line" >> "$temp_file"
      fi
    else
      echo "$line" >> "$temp_file"
    fi
  done < "$env_file"
  
  # Replace original file if fixes were made
  if [[ $fixed_count -gt 0 ]]; then
    cp "$temp_file" "$env_file"
    rm -f "$temp_file"
    return $fixed_count
  else
    rm -f "$temp_file"
    return 0
  fi
}

# Function to validate env file before sourcing
validate_env_file() {
  local env_file="${1:-.env}"
  local has_issues=false
  local line_num=0
  
  if [[ ! -f "$env_file" ]]; then
    return 0
  fi
  
  while IFS= read -r line || [[ -n "$line" ]]; do
    ((line_num++))
    
    # Skip comments and empty lines
    if [[ "$line" =~ ^[[:space:]]*# ]] || [[ -z "${line// }" ]]; then
      continue
    fi
    
    # Check for variable assignments with unquoted spaces
    if [[ "$line" =~ ^([A-Z_][A-Z0-9_]*)=(.*)$ ]]; then
      local var_value="${BASH_REMATCH[2]}"
      
      if needs_quotes "$var_value"; then
        if [[ "$has_issues" == "false" ]]; then
          log_warning "Found unquoted values with spaces in $env_file:"
          has_issues=true
        fi
        log_info "  Line $line_num: $line"
      fi
    fi
  done < "$env_file"
  
  if [[ "$has_issues" == "true" ]]; then
    return 1
  fi
  
  return 0
}

# Main auto-fix function to be called from build.sh
auto_fix_env_quotes() {
  local auto_fix="${AUTO_FIX:-true}"
  local fixed_total=0
  
  # Check all env files
  for env_file in .env .env.dev .env.staging .env.prod .env.local; do
    if [[ -f "$env_file" ]]; then
      if ! validate_env_file "$env_file"; then
        if [[ "$auto_fix" == "true" ]]; then
          local fixed_count
          fix_env_file_quotes "$env_file"
          fixed_count=$?
          if [[ $fixed_count -gt 0 ]]; then
            log_success "Auto-fixed $fixed_count unquoted values in $env_file"
            ((fixed_total += fixed_count))
          fi
        else
          log_error "Unquoted values with spaces found in $env_file"
          log_info "Run with AUTO_FIX=true to fix automatically"
          return 1
        fi
      fi
    fi
  done
  
  if [[ $fixed_total -gt 0 ]]; then
    log_info "Total fixes applied: $fixed_total"
  fi
  
  return 0
}

# Export functions
export -f needs_quotes
export -f fix_env_file_quotes
export -f validate_env_file
export -f auto_fix_env_quotes