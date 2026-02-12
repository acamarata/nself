#!/usr/bin/env bash

# project-detection.sh - Smart detection of nself projects and auto-navigation
# Prevents users from running commands in the wrong directory

# Check if current directory is an nself project
is_nself_project() {
  local dir="${1:-$PWD}"

  # Check for nself project indicators
  if [[ -f "$dir/.env" ]] || \
     [[ -f "$dir/.env.dev" ]] || \
     [[ -f "$dir/docker-compose.yml" ]] || \
     [[ -d "$dir/.nself" ]] || \
     [[ -d "$dir/nself" ]]; then
    return 0  # Is an nself project
  fi

  return 1  # Not an nself project
}

# Find nself project directory (current or backend/)
find_nself_project_dir() {
  local current_dir="$PWD"

  # Check current directory first
  if is_nself_project "$current_dir"; then
    echo "$current_dir"
    return 0
  fi

  # Check backend/ subdirectory
  if [[ -d "$current_dir/backend" ]]; then
    if is_nself_project "$current_dir/backend"; then
      echo "$current_dir/backend"
      return 0
    fi
  fi

  # Check if we're IN a backend/ directory but need to go up
  if [[ "$(basename "$current_dir")" == "backend" ]]; then
    if is_nself_project "$current_dir"; then
      echo "$current_dir"
      return 0
    fi
  fi

  # Not found
  return 1
}

# Auto-navigate to nself project directory
# Returns 0 if we're in or moved to an nself project
# Returns 1 if no nself project found
auto_navigate_to_project() {
  local command_name="${1:-nself}"
  local require_project="${2:-true}"  # Most commands require a project
  local allow_destructive="${3:-false}"  # Some commands are destructive (init, build)

  local project_dir
  if project_dir=$(find_nself_project_dir); then
    if [[ "$project_dir" != "$PWD" ]]; then
      # Found project in different directory
      local relative_path="${project_dir#$PWD/}"

      # For destructive commands, ask confirmation before switching
      if [[ "$allow_destructive" == "true" ]]; then
        printf "\n⚠️  Not in an nself project directory!\n" >&2
        printf "   Current: %s\n" "$PWD" >&2
        printf "   Found nself project: %s\n" "$project_dir" >&2
        printf "\n   Run command in %s instead? [y/N] " "$relative_path" >&2
        read -r response
        response=$(echo "$response" | tr '[:upper:]' '[:lower:]')
        if [[ "$response" != "y" ]] && [[ "$response" != "yes" ]]; then
          printf "\nAborted. Not running in wrong directory.\n" >&2
          return 1
        fi
      fi

      # Auto-navigate
      printf "→ Auto-detected nself project in: %s\n" "$relative_path" >&2
      cd "$project_dir" || return 1
    fi
    return 0
  else
    # No nself project found
    if [[ "$require_project" == "true" ]]; then
      printf "\nError: Not in an nself project directory\n" >&2
      printf "\n" >&2
      printf "This command requires an nself project.\n" >&2
      printf "\n" >&2
      printf "Current directory: %s\n" "$PWD" >&2
      printf "\n" >&2
      printf "To initialize an nself project here:\n" >&2
      printf "  nself init\n" >&2
      printf "\n" >&2
      printf "Or navigate to an existing nself project:\n" >&2
      printf "  cd path/to/your/project\n" >&2
      printf "  cd backend  # if nself is in backend/\n" >&2
      printf "\n" >&2
      return 1
    fi
    return 0
  fi
}

# Check if we're about to create files in the wrong place
# Returns 0 if safe, 1 if potentially wrong directory
check_destructive_command_safety() {
  local command_name="$1"

  # Check if we're in a frontend directory (indicators: package.json, src/, app/, etc.)
  local frontend_indicators=0
  [[ -f "package.json" ]] && ((frontend_indicators++))
  [[ -d "src" ]] && ((frontend_indicators++))
  [[ -d "app" ]] && ((frontend_indicators++))
  [[ -d "components" ]] && ((frontend_indicators++))
  [[ -d "pages" ]] && ((frontend_indicators++))
  [[ -f "next.config.js" ]] || [[ -f "next.config.mjs" ]] && ((frontend_indicators++))
  [[ -f "vite.config.js" ]] || [[ -f "vite.config.ts" ]] && ((frontend_indicators++))

  # Check if backend/ exists
  local backend_exists=false
  [[ -d "backend" ]] && backend_exists=true

  # If we have frontend indicators AND a backend directory exists, warn
  if [[ $frontend_indicators -ge 2 ]] && [[ "$backend_exists" == "true" ]]; then
    printf "\n⚠️  WARNING: This looks like a frontend directory!\n" >&2
    printf "\n" >&2
    printf "Detected: " >&2
    [[ -f "package.json" ]] && printf "package.json " >&2
    [[ -d "src" ]] && printf "src/ " >&2
    [[ -d "app" ]] && printf "app/ " >&2
    [[ -d "components" ]] && printf "components/ " >&2
    printf "\n" >&2
    printf "\n" >&2
    printf "You probably want to run this in: ./backend/\n" >&2
    printf "\n" >&2
    printf "Continue anyway? [y/N] " >&2
    read -r response
    response=$(echo "$response" | tr '[:upper:]' '[:lower:]')
    if [[ "$response" != "y" ]] && [[ "$response" != "yes" ]]; then
      printf "\nAborted. Run from ./backend/ instead:\n" >&2
      printf "  cd backend && nself %s\n" "$command_name" >&2
      printf "\n" >&2
      return 1
    fi
  fi

  return 0
}

# Export functions
export -f is_nself_project
export -f find_nself_project_dir
export -f auto_navigate_to_project
export -f check_destructive_command_safety
