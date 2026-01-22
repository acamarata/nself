#!/usr/bin/env bash
# diff.sh - Environment comparison functionality
# POSIX-compliant, no Bash 4+ features

# Get the directory where this script is located
ENV_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_ROOT="$(dirname "$ENV_LIB_DIR")"

# Source dependencies
source "$LIB_ROOT/utils/display.sh" 2>/dev/null || true
source "$LIB_ROOT/utils/platform-compat.sh" 2>/dev/null || true

# Environment directory
ENVIRONMENTS_DIR="${ENVIRONMENTS_DIR:-./.environments}"

# Compare two environments
env::diff() {
  local env_a="$1"
  local env_b="$2"
  local show_values="${3:-false}"

  if [[ -z "$env_a" ]] || [[ -z "$env_b" ]]; then
    log_error "Two environment names are required"
    printf "Usage: nself env diff <env1> <env2>\n"
    return 1
  fi

  local dir_a="$ENVIRONMENTS_DIR/$env_a"
  local dir_b="$ENVIRONMENTS_DIR/$env_b"

  # Validate environments exist
  if [[ ! -d "$dir_a" ]]; then
    log_error "Environment '$env_a' does not exist"
    return 1
  fi

  if [[ ! -d "$dir_b" ]]; then
    log_error "Environment '$env_b' does not exist"
    return 1
  fi

  printf "Comparing: ${COLOR_BLUE}%s${COLOR_RESET} vs ${COLOR_BLUE}%s${COLOR_RESET}\n\n" "$env_a" "$env_b"

  # Compare .env files
  if [[ -f "$dir_a/.env" ]] && [[ -f "$dir_b/.env" ]]; then
    env::compare_env_files "$dir_a/.env" "$dir_b/.env" "$env_a" "$env_b" "$show_values"
  else
    if [[ ! -f "$dir_a/.env" ]]; then
      log_warning "No .env file in $env_a"
    fi
    if [[ ! -f "$dir_b/.env" ]]; then
      log_warning "No .env file in $env_b"
    fi
  fi

  # Compare server configurations
  if [[ -f "$dir_a/server.json" ]] || [[ -f "$dir_b/server.json" ]]; then
    printf "\n${COLOR_CYAN}Server Configuration:${COLOR_RESET}\n"
    env::compare_server_configs "$dir_a/server.json" "$dir_b/server.json" "$env_a" "$env_b"
  fi

  return 0
}

# Compare two .env files
env::compare_env_files() {
  local file_a="$1"
  local file_b="$2"
  local name_a="$3"
  local name_b="$4"
  local show_values="${5:-false}"

  # Extract keys from both files
  local keys_a keys_b
  keys_a=$(grep -E "^[A-Za-z_][A-Za-z0-9_]*=" "$file_a" 2>/dev/null | cut -d'=' -f1 | sort -u)
  keys_b=$(grep -E "^[A-Za-z_][A-Za-z0-9_]*=" "$file_b" 2>/dev/null | cut -d'=' -f1 | sort -u)

  # Find common keys, only in A, only in B
  local all_keys
  all_keys=$(printf "%s\n%s" "$keys_a" "$keys_b" | sort -u)

  local different=()
  local only_a=()
  local only_b=()

  while IFS= read -r key; do
    [[ -z "$key" ]] && continue

    local val_a val_b
    val_a=$(grep "^${key}=" "$file_a" 2>/dev/null | cut -d'=' -f2-)
    val_b=$(grep "^${key}=" "$file_b" 2>/dev/null | cut -d'=' -f2-)

    if [[ -z "$val_a" ]]; then
      only_b+=("$key")
    elif [[ -z "$val_b" ]]; then
      only_a+=("$key")
    elif [[ "$val_a" != "$val_b" ]]; then
      different+=("$key|$val_a|$val_b")
    fi
  done <<< "$all_keys"

  # Display differences
  if [[ ${#different[@]} -gt 0 ]]; then
    printf "${COLOR_YELLOW}Different values:${COLOR_RESET}\n"
    for item in "${different[@]}"; do
      local key val_a val_b
      key=$(printf "%s" "$item" | cut -d'|' -f1)
      val_a=$(printf "%s" "$item" | cut -d'|' -f2)
      val_b=$(printf "%s" "$item" | cut -d'|' -f3)

      printf "  ${COLOR_CYAN}%s${COLOR_RESET}\n" "$key"

      if [[ "$show_values" == "true" ]]; then
        # Mask sensitive values
        if [[ "$key" =~ (PASSWORD|SECRET|KEY|TOKEN) ]]; then
          printf "    %s: %s\n" "$name_a" "********"
          printf "    %s: %s\n" "$name_b" "********"
        else
          printf "    %s: %s\n" "$name_a" "$val_a"
          printf "    %s: %s\n" "$name_b" "$val_b"
        fi
      fi
    done
  fi

  # Display only in A
  if [[ ${#only_a[@]} -gt 0 ]]; then
    printf "\n${COLOR_GREEN}Only in %s:${COLOR_RESET}\n" "$name_a"
    for key in "${only_a[@]}"; do
      printf "  + %s\n" "$key"
    done
  fi

  # Display only in B
  if [[ ${#only_b[@]} -gt 0 ]]; then
    printf "\n${COLOR_RED}Only in %s:${COLOR_RESET}\n" "$name_b"
    for key in "${only_b[@]}"; do
      printf "  + %s\n" "$key"
    done
  fi

  # Summary
  local total_diff=$((${#different[@]} + ${#only_a[@]} + ${#only_b[@]}))
  if [[ $total_diff -eq 0 ]]; then
    printf "${COLOR_GREEN}✓ Environments have identical configuration${COLOR_RESET}\n"
  else
    printf "\n${COLOR_YELLOW}Summary: %d difference(s)${COLOR_RESET}\n" "$total_diff"
    printf "  %d different values\n" "${#different[@]}"
    printf "  %d only in %s\n" "${#only_a[@]}" "$name_a"
    printf "  %d only in %s\n" "${#only_b[@]}" "$name_b"
  fi
}

# Compare server configurations
env::compare_server_configs() {
  local file_a="$1"
  local file_b="$2"
  local name_a="$3"
  local name_b="$4"

  if [[ -f "$file_a" ]] && [[ -f "$file_b" ]]; then
    # Extract key fields
    local fields="host port user type"

    for field in $fields; do
      local val_a val_b
      val_a=$(grep "\"$field\"" "$file_a" 2>/dev/null | cut -d'"' -f4)
      val_b=$(grep "\"$field\"" "$file_b" 2>/dev/null | cut -d'"' -f4)

      if [[ "$val_a" != "$val_b" ]]; then
        printf "  ${COLOR_CYAN}%s${COLOR_RESET}: %s → %s\n" "$field" "${val_a:-<not set>}" "${val_b:-<not set>}"
      fi
    done
  elif [[ -f "$file_a" ]]; then
    printf "  ${COLOR_YELLOW}Server config only in %s${COLOR_RESET}\n" "$name_a"
  elif [[ -f "$file_b" ]]; then
    printf "  ${COLOR_YELLOW}Server config only in %s${COLOR_RESET}\n" "$name_b"
  fi
}

# Show what would change when switching environments
env::preview_switch() {
  local target_env="$1"
  local current_env="${2:-$(env::get_current 2>/dev/null || echo 'local')}"

  if [[ -z "$target_env" ]]; then
    log_error "Target environment name is required"
    return 1
  fi

  if [[ "$target_env" == "$current_env" ]]; then
    log_info "Already on environment: $target_env"
    return 0
  fi

  printf "Preview: Switch from ${COLOR_BLUE}%s${COLOR_RESET} to ${COLOR_BLUE}%s${COLOR_RESET}\n\n" "$current_env" "$target_env"

  env::diff "$current_env" "$target_env" "true"

  printf "\n${COLOR_YELLOW}Note: This is a preview. No changes have been made.${COLOR_RESET}\n"
  printf "Run ${COLOR_CYAN}nself env switch %s${COLOR_RESET} to apply these changes.\n" "$target_env"
}

# Get current environment (imported from create.sh but defined here for standalone use)
if ! command -v env::get_current >/dev/null 2>&1; then
  env::get_current() {
    local current_file=".current-env"
    if [[ -f "$current_file" ]]; then
      cat "$current_file"
    else
      echo "local"
    fi
  }
fi

# Export functions
export -f env::diff
export -f env::compare_env_files
export -f env::compare_server_configs
export -f env::preview_switch
