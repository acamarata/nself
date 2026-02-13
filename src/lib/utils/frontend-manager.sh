#!/usr/bin/env bash
# frontend-manager.sh - Frontend app detection and management for monorepo support
# Part of nself v0.9.9 - Monorepo orchestration
# POSIX-compliant, Bash 3.2+ compatible

# Detect all frontend apps in monorepo root
detect_frontend_apps() {
  # Directories to exclude from frontend detection
  local exclude_dirs=("backend" "node_modules" ".git" ".ai" "dist" "build" ".next")
  local apps=()
  
  # Find directories with package.json at maxdepth 1 (immediate children only)
  while IFS= read -r dir; do
    # Skip if directory itself
    if [[ "$dir" == "." ]]; then
      continue
    fi
    
    local dirname=$(basename "$dir")
    local skip=false
    
    # Check if directory should be excluded
    for excluded in "${exclude_dirs[@]}"; do
      if [[ "$dirname" == "$excluded" ]]; then
        skip=true
        break
      fi
    done
    
    # Check if package.json exists in this directory
    if [[ "$skip" == "false" ]] && [[ -f "$dir/package.json" ]]; then
      apps+=("$dir")
    fi
  done < <(find . -maxdepth 1 -type d)
  
  # Output array (one per line)
  printf "%s\n" "${apps[@]}"
}

# Detect package manager for a specific app directory
detect_package_manager() {
  local app_dir="$1"
  
  # Priority order: pnpm > yarn > bun > npm (based on performance/popularity)
  if [[ -f "$app_dir/pnpm-lock.yaml" ]]; then
    echo "pnpm"
  elif [[ -f "$app_dir/yarn.lock" ]]; then
    echo "yarn"
  elif [[ -f "$app_dir/bun.lockb" ]]; then
    echo "bun"
  elif [[ -f "$app_dir/package-lock.json" ]]; then
    echo "npm"
  else
    # Fallback: check which is installed on system
    if command -v pnpm >/dev/null 2>&1; then
      echo "pnpm"
    elif command -v yarn >/dev/null 2>&1; then
      echo "yarn"
    elif command -v bun >/dev/null 2>&1; then
      echo "bun"
    else
      echo "npm"  # npm is always available with Node.js
    fi
  fi
}

# Get dev command for package manager
get_dev_command() {
  local pkg_manager="$1"
  
  case "$pkg_manager" in
    pnpm|bun)
      echo "$pkg_manager dev"
      ;;
    npm|yarn)
      echo "$pkg_manager run dev"
      ;;
    *)
      echo "npm run dev"
      ;;
  esac
}

# Verify package.json has dev script
has_dev_script() {
  local app_dir="$1"
  local package_json="$app_dir/package.json"
  
  if [[ ! -f "$package_json" ]]; then
    return 1
  fi
  
  # Use grep to avoid requiring jq
  if grep -q '"dev"' "$package_json" 2>/dev/null; then
    return 0
  else
    return 1
  fi
}

# Start a single frontend app
start_frontend_app() {
  local app_dir="$1"
  local app_name=$(basename "$app_dir")
  local pkg_manager=$(detect_package_manager "$app_dir")
  local dev_cmd=$(get_dev_command "$pkg_manager")
  
  # Verify dev script exists
  if ! has_dev_script "$app_dir"; then
    return 1
  fi
  
  # Verify package manager is installed
  local pkg_cmd=$(echo "$pkg_manager" | awk '{print $1}')
  if ! command -v "$pkg_cmd" >/dev/null 2>&1; then
    printf "Warning: %s not installed. Please install it to run %s\n" "$pkg_cmd" "$app_name" >&2
    return 1
  fi
  
  # Create log directory if needed
  mkdir -p ".nself/frontend-logs"
  
  # Start in background with output redirect
  (cd "$app_dir" && $dev_cmd > "../.nself/frontend-logs/$app_name.log" 2>&1 &)
  local pid=$!
  
  # Give it a moment to fail if there's an immediate error
  sleep 0.5
  
  # Check if still running
  if kill -0 "$pid" 2>/dev/null; then
    echo "$pid"
    return 0
  else
    return 1
  fi
}

# Check if frontend app is healthy (listening on port)
check_frontend_health() {
  local app_dir="$1"
  local port="$2"
  
  if [[ -z "$port" ]]; then
    return 0  # No port specified, assume healthy
  fi
  
  # Simple port check
  if command -v lsof >/dev/null 2>&1; then
    lsof -i ":$port" -sTCP:LISTEN >/dev/null 2>&1
  elif command -v netstat >/dev/null 2>&1; then
    netstat -an | grep -q "LISTEN.*:$port"
  else
    # Fallback: assume healthy if process is running
    return 0
  fi
}

# Extract port from package.json or default
get_frontend_port() {
  local app_dir="$1"
  local package_json="$app_dir/package.json"
  
  # Try to extract from dev script (e.g., "dev": "next dev -p 3001")
  if [[ -f "$package_json" ]]; then
    local port=$(grep -o '\-p [0-9]*' "$package_json" 2>/dev/null | head -1 | awk '{print $2}')
    if [[ -n "$port" ]]; then
      echo "$port"
      return
    fi
    
    # Try --port variant
    port=$(grep -o '\--port[= ][0-9]*' "$package_json" 2>/dev/null | head -1 | grep -o '[0-9]*')
    if [[ -n "$port" ]]; then
      echo "$port"
      return
    fi
  fi
  
  # Default Next.js/React port
  echo "3000"
}

# Get framework type from package.json dependencies
detect_framework() {
  local app_dir="$1"
  local package_json="$app_dir/package.json"
  
  if [[ ! -f "$package_json" ]]; then
    echo "unknown"
    return
  fi
  
  # Check for framework-specific dependencies
  if grep -q '"next"' "$package_json" 2>/dev/null; then
    echo "nextjs"
  elif grep -q '"@vitejs/plugin-react"' "$package_json" 2>/dev/null; then
    echo "vite-react"
  elif grep -q '"vite"' "$package_json" 2>/dev/null; then
    echo "vite"
  elif grep -q '"react-scripts"' "$package_json" 2>/dev/null; then
    echo "cra"  # Create React App
  elif grep -q '"vue"' "$package_json" 2>/dev/null; then
    echo "vue"
  elif grep -q '"@angular/core"' "$package_json" 2>/dev/null; then
    echo "angular"
  elif grep -q '"svelte"' "$package_json" 2>/dev/null; then
    echo "svelte"
  else
    echo "unknown"
  fi
}

# Cleanup frontend processes
cleanup_frontend_processes() {
  local pids=("$@")
  
  for pid in "${pids[@]}"; do
    if kill -0 "$pid" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      
      # Give it a moment to exit gracefully
      sleep 0.5
      
      # Force kill if still running
      if kill -0 "$pid" 2>/dev/null; then
        kill -9 "$pid" 2>/dev/null || true
      fi
    fi
  done
}

# Functions are available when this file is sourced
