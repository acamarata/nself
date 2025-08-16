#!/usr/bin/env bash
# diff.sh - Show configuration differences

# Source shared utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/header.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"

# Create a file with default values for comparison
create_defaults_file() {
  local output_file="$1"

  cat >"$output_file" <<'EOF'
# Default nself configuration
PROJECT_NAME=myproject
POSTGRES_PASSWORD=changeme
HASURA_GRAPHQL_ADMIN_SECRET=changeme
HASURA_JWT_KEY=changeme-minimum-32-characters-long

# Core services (enabled by default)
POSTGRES_ENABLED=true
HASURA_ENABLED=true
NGINX_ENABLED=true
CONFIG_SERVER_ENABLED=true
MINIO_ENABLED=true

# Optional services (disabled by default)
REDIS_ENABLED=false
FUNCTIONS_ENABLED=false
DASHBOARD_ENABLED=false
SERVICES_ENABLED=false
NESTJS_ENABLED=false
BULLMQ_ENABLED=false
GOLANG_ENABLED=false
PYTHON_ENABLED=false

# Service configurations
NESTJS_SERVICES=
BULLMQ_WORKERS=
BULLMQ_DASHBOARD_ENABLED=false
GOLANG_SERVICES=
PYTHON_SERVICES=

# Network and security
BASE_DOMAIN=localhost
SSL_ENABLED=false
EOF
}

# Command function
cmd_diff() {
  local file1=""
  local file2=""

  # Parse arguments
  if [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
    show_diff_help
    return 0
  fi

  # Determine files to compare
  if [[ $# -eq 0 ]]; then
    # No arguments - try to auto-detect files to compare

    # Priority 1: Check for .old versions (after reset/backup)
    if [[ -f ".env.local" ]] && [[ -f ".env.local.old" ]]; then
      file1=".env.local.old"
      file2=".env.local"
    elif [[ -f ".env" ]] && [[ -f ".env.old" ]]; then
      file1=".env.old"
      file2=".env"
    # Priority 2: Compare different env files
    elif [[ -f ".env.local" ]] && [[ -f ".env.prod" ]]; then
      file1=".env.local"
      file2=".env.prod"
    elif [[ -f ".env.local" ]] && [[ -f ".env.dev" ]]; then
      file1=".env.local"
      file2=".env.dev"
    elif [[ -f ".env.local" ]] && [[ -f ".env" ]]; then
      file1=".env"
      file2=".env.local"
    elif [[ -f ".env.dev" ]] && [[ -f ".env.prod" ]]; then
      file1=".env.dev"
      file2=".env.prod"
    # Priority 3: Compare against defaults if only one env exists
    elif [[ -f ".env.local" ]] || [[ -f ".env" ]] || [[ -f ".env.dev" ]] || [[ -f ".env.prod" ]]; then
      # Find which env file exists
      local env_file=""
      [[ -f ".env.local" ]] && env_file=".env.local"
      [[ -f ".env" ]] && env_file=".env"
      [[ -f ".env.dev" ]] && env_file=".env.dev"
      [[ -f ".env.prod" ]] && env_file=".env.prod"

      # Create temporary defaults file
      local defaults_file="/tmp/nself-defaults-$$.env"
      create_defaults_file "$defaults_file"

      file1="$defaults_file"
      file2="$env_file"

      # Flag to clean up temp file later
      local cleanup_temp=true
    else
      # No env files at all
      show_command_header "nself diff" "Compare configuration files"
      echo
      log_error "No configuration files found"
      echo
      echo "This command compares configuration files to show what has changed."
      echo
      echo "Usage:"
      echo "  ${COLOR_BLUE}nself init${COLOR_RESET}                    # Create initial configuration"
      echo "  ${COLOR_BLUE}nself diff file1 file2${COLOR_RESET}        # Compare two specific files"
      echo
      return 1
    fi
  elif [[ $# -eq 1 ]]; then
    # One argument - look for its .old version
    file1="$1"
    if [[ -f "${file1}.old" ]]; then
      file2="${file1}.old"
      # Swap so old is on left, new on right
      local temp="$file1"
      file1="$file2"
      file2="$temp"
    else
      show_command_header "nself diff" "Compare configuration files"
      echo
      log_error "No .old version found for: $file1"
      echo
      echo "Usage:"
      echo "  ${COLOR_BLUE}nself diff${COLOR_RESET}                    # Auto-detect .env.local vs .env.local.old"
      echo "  ${COLOR_BLUE}nself diff file1 file2${COLOR_RESET}        # Compare two specific files"
      echo
      return 1
    fi
  else
    # Two arguments - use them as-is
    file1="$1"
    file2="$2"
  fi

  # Show header with subtitle
  show_command_header "nself diff" "Compare configuration files"

  # Check if files exist
  if [[ ! -f "$file1" ]]; then
    echo
    log_error "File not found: $file1"
    return 1
  fi

  if [[ ! -f "$file2" ]]; then
    echo
    log_error "File not found: $file2"
    return 1
  fi

  # Show what we're comparing
  echo
  printf "${COLOR_CYAN}➞ Comparison${COLOR_RESET}\n"
  printf "  ${COLOR_DIM}Old:${COLOR_RESET} %s\n" "$file1"
  printf "  ${COLOR_DIM}New:${COLOR_RESET} %s\n" "$file2"
  echo

  # Check if files are identical
  if cmp -s "$file1" "$file2"; then
    log_success "Files are identical - no changes detected"
    return 0
  fi

  # Show differences with better formatting
  printf "${COLOR_CYAN}➞ Changes${COLOR_RESET}\n"
  echo

  if command -v diff >/dev/null 2>&1; then
    # Use colored diff if available
    if command -v colordiff >/dev/null 2>&1; then
      diff -u "$file1" "$file2" | colordiff | tail -n +4
    else
      # Standard diff with manual coloring
      diff -u "$file1" "$file2" | tail -n +4 | while IFS= read -r line; do
        case "$line" in
        +*)
          printf "${COLOR_GREEN}%s${COLOR_RESET}\n" "$line"
          ;;
        -*)
          printf "${COLOR_RED}%s${COLOR_RESET}\n" "$line"
          ;;
        @*)
          printf "${COLOR_CYAN}%s${COLOR_RESET}\n" "$line"
          ;;
        *)
          echo "$line"
          ;;
        esac
      done
    fi
  else
    log_error "diff command not available"
    return 1
  fi

  echo

  # Enhanced summary
  printf "${COLOR_CYAN}➞ Summary${COLOR_RESET}\n"

  local vars1=$(grep -c '^[A-Z_][A-Z0-9_]*=' "$file1" 2>/dev/null || echo 0)
  local vars2=$(grep -c '^[A-Z_][A-Z0-9_]*=' "$file2" 2>/dev/null || echo 0)

  # Count actual changes more accurately
  local temp_diff="/tmp/nself-diff-$$.txt"
  diff "$file1" "$file2" >"$temp_diff" 2>/dev/null || true

  local added_vars=0
  local removed_vars=0
  local modified_vars=0

  # Get all variables from both files
  local all_vars=$(cat "$file1" "$file2" | grep '^[A-Z_][A-Z0-9_]*=' | cut -d= -f1 | sort -u)

  while IFS= read -r var; do
    [[ -z "$var" ]] && continue

    local in_file1=$(grep -c "^${var}=" "$file1" 2>/dev/null | tr -d '\n' || echo 0)
    local in_file2=$(grep -c "^${var}=" "$file2" 2>/dev/null | tr -d '\n' || echo 0)

    if [[ $in_file1 -eq 0 ]] && [[ $in_file2 -gt 0 ]]; then
      ((added_vars++))
    elif [[ $in_file1 -gt 0 ]] && [[ $in_file2 -eq 0 ]]; then
      ((removed_vars++))
    elif [[ $in_file1 -gt 0 ]] && [[ $in_file2 -gt 0 ]]; then
      local val1=$(grep "^${var}=" "$file1" | cut -d= -f2-)
      local val2=$(grep "^${var}=" "$file2" | cut -d= -f2-)
      if [[ "$val1" != "$val2" ]]; then
        ((modified_vars++))
      fi
    fi
  done <<<"$all_vars"

  rm -f "$temp_diff"

  echo

  # Display file names nicely
  local display_file1=$(basename "$file1")
  local display_file2=$(basename "$file2")

  # Special handling for temp defaults file
  [[ "$file1" == /tmp/nself-defaults-* ]] && display_file1="defaults"
  [[ "$file2" == /tmp/nself-defaults-* ]] && display_file2="defaults"

  printf "  ${COLOR_DIM}Configuration files:${COLOR_RESET}\n"
  printf "    %-20s %d variables\n" "${display_file1}:" "$vars1"
  printf "    %-20s %d variables\n" "${display_file2}:" "$vars2"

  if [[ $added_vars -gt 0 ]] || [[ $removed_vars -gt 0 ]] || [[ $modified_vars -gt 0 ]]; then
    echo
    printf "  ${COLOR_DIM}Changes detected:${COLOR_RESET}\n"
    [[ $added_vars -gt 0 ]] && printf "    ${COLOR_GREEN}+ %2d${COLOR_RESET} new variables added\n" "$added_vars"
    [[ $removed_vars -gt 0 ]] && printf "    ${COLOR_RED}- %2d${COLOR_RESET} variables removed\n" "$removed_vars"
    [[ $modified_vars -gt 0 ]] && printf "    ${COLOR_YELLOW}~ %2d${COLOR_RESET} variables modified\n" "$modified_vars"
  else
    echo
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} No changes detected\n"
  fi

  # Show important changes
  local important_vars="PROJECT_NAME|POSTGRES_PASSWORD|HASURA_GRAPHQL_ADMIN_SECRET|HASURA_JWT_KEY|SSL_ENABLED|BASE_DOMAIN"
  local important_changes=""

  while IFS= read -r var; do
    [[ -z "$var" ]] && continue

    if echo "$var" | grep -qE "^($important_vars)$"; then
      local in_file1=$(grep -c "^${var}=" "$file1" 2>/dev/null | tr -d '\n' || echo 0)
      local in_file2=$(grep -c "^${var}=" "$file2" 2>/dev/null | tr -d '\n' || echo 0)

      if [[ $in_file1 -eq 0 ]] && [[ $in_file2 -gt 0 ]]; then
        important_changes="${important_changes}    • ${var} ${COLOR_GREEN}(added)${COLOR_RESET}\n"
      elif [[ $in_file1 -gt 0 ]] && [[ $in_file2 -eq 0 ]]; then
        important_changes="${important_changes}    • ${var} ${COLOR_RED}(removed)${COLOR_RESET}\n"
      elif [[ $in_file1 -gt 0 ]] && [[ $in_file2 -gt 0 ]]; then
        local val1=$(grep "^${var}=" "$file1" | cut -d= -f2-)
        local val2=$(grep "^${var}=" "$file2" | cut -d= -f2-)
        if [[ "$val1" != "$val2" ]]; then
          # Check if it's a sensitive value that changed
          if [[ "$var" =~ (PASSWORD|SECRET|JWT_KEY) ]]; then
            important_changes="${important_changes}    • ${var} ${COLOR_YELLOW}(modified - security)${COLOR_RESET}\n"
          else
            important_changes="${important_changes}    • ${var} ${COLOR_YELLOW}(modified: ${val1} → ${val2})${COLOR_RESET}\n"
          fi
        fi
      fi
    fi
  done <<<"$all_vars"

  if [[ -n "$important_changes" ]]; then
    echo
    printf "  ${COLOR_YELLOW}⚠${COLOR_RESET}  ${COLOR_DIM}Critical variables:${COLOR_RESET}\n"
    printf "$important_changes"
  fi

  # Clean up temp file if we created one
  if [[ "${cleanup_temp:-false}" == "true" ]] && [[ -f "$file1" ]]; then
    rm -f "$file1"
  fi

  echo
}

# Show help
show_diff_help() {
  echo "Usage: nself diff [file1] [file2]"
  echo
  echo "Compare configuration files to see what has changed"
  echo
  echo "Arguments:"
  echo "  file1    First file to compare (optional)"
  echo "  file2    Second file to compare (optional)"
  echo
  echo "Automatic detection (when no arguments):"
  echo "  1. Compares .env.local vs .env.local.old (after reset/backup)"
  echo "  2. Compares different env files (.env.local vs .env.prod, etc.)"
  echo "  3. Compares single env against defaults if only one exists"
  echo "  4. Shows error if no configuration files found"
  echo
  echo "Examples:"
  echo "  nself diff                          # Auto-detect files to compare"
  echo "  nself diff .env.local               # Compare with .env.local.old"
  echo "  nself diff .env.local .env.prod     # Compare local vs production"
  echo "  nself diff .env.dev .env            # Compare specific files"
  echo
  echo "Features:"
  echo "  • Color-coded diff output (+ added, - removed, ~ modified)"
  echo "  • Summary with change counts"
  echo "  • Highlights critical security variables"
  echo "  • Compares against defaults when appropriate"
}

# Export for use as library
export -f cmd_diff

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "diff" || exit $?
  cmd_diff "$@"
  exit_code=$?
  post_command "diff" $exit_code
  exit $exit_code
fi
