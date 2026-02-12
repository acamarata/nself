#!/usr/bin/env bash

# env-validator.sh - Validate environment files for undefined variable references
# Prevents silent failures from ${UNDEFINED_VAR} expansions

# Validate that all ${VAR} references in a file are defined
# Returns 0 if valid, 1 if undefined variables found
validate_env_file_references() {
  local file="$1"
  local show_warnings="${2:-true}"

  if [[ ! -f "$file" ]]; then
    return 0  # File doesn't exist, nothing to validate
  fi

  # Extract all ${VAR} references from the file
  local undefined_vars=()
  local line_num=0

  while IFS= read -r line; do
    ((line_num++))

    # Skip comments and empty lines
    [[ "$line" =~ ^[[:space:]]*# ]] && continue
    [[ -z "${line// /}" ]] && continue

    # Find all ${VAR} patterns in this line
    while [[ "$line" =~ \$\{([A-Z_][A-Z0-9_]*)\} ]]; do
      local var_name="${BASH_REMATCH[1]}"
      local var_value="${!var_name:-}"

      # Check if variable is defined
      if [[ -z "$var_value" ]]; then
        undefined_vars+=("$var_name (line $line_num)")
      fi

      # Remove this match and continue searching
      line="${line/${BASH_REMATCH[0]}/}"
    done
  done < "$file"

  # Report undefined variables
  if [[ ${#undefined_vars[@]} -gt 0 ]]; then
    if [[ "$show_warnings" == "true" ]]; then
      printf "\n⚠️  WARNING: Undefined variables in %s:\n" "$file" >&2
      for var in "${undefined_vars[@]}"; do
        printf "   ❌ \${%s}\n" "$var" >&2
      done
      printf "\n" >&2
      printf "These variables are referenced but not defined.\n" >&2
      printf "This will cause silent failures when using 'set -u' (nounset).\n" >&2
      printf "\n" >&2
      printf "Solutions:\n" >&2
      printf "  1. Define these variables in .env.secrets or .env\n" >&2
      printf "  2. Remove the variable references if not needed\n" >&2
      printf "  3. Use different variable names that are actually defined\n" >&2
      printf "\n" >&2
    fi
    return 1
  fi

  return 0
}

# Validate all environment files in cascade order
# Returns 0 if all valid, 1 if any have undefined references
validate_env_cascade() {
  local env="${1:-${ENV:-dev}}"
  local show_warnings="${2:-true}"
  local errors=0

  # Check .env.dev
  if ! validate_env_file_references ".env.dev" "$show_warnings"; then
    ((errors++))
  fi

  # Check environment-specific files
  case "$env" in
    staging)
      if ! validate_env_file_references ".env.staging" "$show_warnings"; then
        ((errors++))
      fi
      ;;
    prod|production)
      if ! validate_env_file_references ".env.staging" "$show_warnings"; then
        ((errors++))
      fi
      if ! validate_env_file_references ".env.prod" "$show_warnings"; then
        ((errors++))
      fi
      ;;
  esac

  # Check .env (final override)
  if ! validate_env_file_references ".env" "$show_warnings"; then
    ((errors++))
  fi

  if [[ $errors -gt 0 ]]; then
    if [[ "$show_warnings" == "true" ]]; then
      printf "Found undefined variable references in %d file(s)\n" "$errors" >&2
      printf "Run 'nself config validate' for detailed analysis\n" >&2
    fi
    return 1
  fi

  return 0
}

# Validate before sourcing (safer wrapper)
# This validates and then sources if valid
safe_source_env_file() {
  local file="$1"

  if [[ ! -f "$file" ]]; then
    return 1  # File doesn't exist
  fi

  # Validate first
  if ! validate_env_file_references "$file" true; then
    printf "ERROR: Cannot source %s due to undefined variable references\n" "$file" >&2
    printf "Fix the issues above before continuing\n" >&2
    return 1
  fi

  # If valid, source it
  set -a
  source "$file" 2>/dev/null
  set +a

  return 0
}

# Export functions
export -f validate_env_file_references
export -f validate_env_cascade
export -f safe_source_env_file
