#!/usr/bin/env bash
# runtime.sh - Plugin runtime management (start/stop/logs)
# POSIX-compliant, no Bash 4+ features

# ============================================================================
# Plugin & Runtime Directories
# ============================================================================

PLUGIN_DIR="${PLUGIN_DIR:-${NSELF_PLUGIN_DIR:-$HOME/.nself/plugins}}"
PLUGIN_RUNTIME_DIR="${NSELF_PLUGIN_RUNTIME:-$HOME/.nself/runtime}"
PLUGIN_LOGS_DIR="$PLUGIN_RUNTIME_DIR/logs"
PLUGIN_PIDS_DIR="$PLUGIN_RUNTIME_DIR/pids"

# ============================================================================
# Setup & Prerequisites
# ============================================================================

# Ensure runtime directories exist
ensure_runtime_dirs() {
  mkdir -p "$PLUGIN_LOGS_DIR"
  mkdir -p "$PLUGIN_PIDS_DIR"
}

# Setup shared utilities (one-time)
setup_shared_utilities() {
  local shared_link="$HOME/.nself/shared"
  local shared_target="$HOME/.nself/plugins/_shared"

  # Check if symlink already exists
  if [[ -L "$shared_link" ]]; then
    return 0
  fi

  # Check if _shared directory exists
  if [[ ! -d "$shared_target" ]]; then
    log_warning "Shared utilities not found at $shared_target"
    printf "Install plugins first with: nself plugin install <name>\n"
    return 1
  fi

  # Check if _shared is built
  if [[ ! -d "$shared_target/dist" ]]; then
    log_info "Building shared utilities..."

    # Try to build from source repo if available
    if [[ -d "$HOME/Sites/nself-plugins/shared" ]]; then
      (cd "$HOME/Sites/nself-plugins/shared" && pnpm install --silent && pnpm build --silent)
      cp -r "$HOME/Sites/nself-plugins/shared/dist" "$shared_target/"
    else
      log_error "Shared utilities not built and source not found"
      printf "\nRun: cd ~/Sites/nself-plugins/shared && pnpm install && pnpm build\n"
      return 1
    fi
  fi

  # Create symlink
  ln -s "$shared_target" "$shared_link"
  log_success "Shared utilities ready"
}

# Get DATABASE_URL from project .env
get_database_url() {
  local db_url=""

  # Use project directory (where nself command was run)
  local project_dir="${NSELF_PROJECT_DIR:-$(pwd)}"

  # Try to load from project .env files
  for env_file in "$project_dir/.env.dev" "$project_dir/.env.local" "$project_dir/.env"; do
    if [[ -f "$env_file" ]]; then
      db_url=$(grep "^DATABASE_URL=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'")
      if [[ -n "$db_url" ]]; then
        break
      fi

      # Try building from POSTGRES_* variables
      local pg_user=$(grep "^POSTGRES_USER=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'")
      local pg_pass=$(grep "^POSTGRES_PASSWORD=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'")
      local pg_db=$(grep "^POSTGRES_DB=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'")
      local pg_host=$(grep "^POSTGRES_HOST=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'")
      local pg_port=$(grep "^POSTGRES_PORT=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'")

      pg_host="${pg_host:-localhost}"
      pg_port="${pg_port:-5432}"

      if [[ -n "$pg_user" ]] && [[ -n "$pg_pass" ]] && [[ -n "$pg_db" ]]; then
        db_url="postgresql://${pg_user}:${pg_pass}@${pg_host}:${pg_port}/${pg_db}"
        break
      fi
    fi
  done

  if [[ -z "$db_url" ]]; then
    log_error "Could not find DATABASE_URL in .env files"
    return 1
  fi

  printf '%s' "$db_url"
}

# Get or generate encryption key
get_encryption_key() {
  local key_file="$HOME/.nself/encryption.key"

  if [[ ! -f "$key_file" ]]; then
    log_info "Generating encryption key..."
    openssl rand -base64 32 > "$key_file"
    chmod 600 "$key_file"
  fi

  cat "$key_file"
}

# Get MinIO configuration from project .env (if enabled)
get_minio_config() {
  local project_dir="${NSELF_PROJECT_DIR:-$(pwd)}"
  local minio_enabled=""
  local minio_port=""
  local minio_user=""
  local minio_pass=""
  local minio_bucket=""

  # Try to load from project .env files
  for env_file in "$project_dir/.env.dev" "$project_dir/.env.local" "$project_dir/.env"; do
    if [[ -f "$env_file" ]]; then
      minio_enabled=$(grep "^MINIO_ENABLED=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')

      if [[ "$minio_enabled" == "true" ]]; then
        minio_port=$(grep "^MINIO_PORT=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')
        minio_user=$(grep "^MINIO_ROOT_USER=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')
        minio_pass=$(grep "^MINIO_ROOT_PASSWORD=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')

        # Try to find a default bucket (prefer MINIO_BUCKET_RAW or first bucket found)
        minio_bucket=$(grep "^MINIO_BUCKET_RAW=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')
        if [[ -z "$minio_bucket" ]]; then
          minio_bucket=$(grep "^MINIO_BUCKET_" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')
        fi

        break
      fi
    fi
  done

  # Return empty if MinIO not enabled
  if [[ "$minio_enabled" != "true" ]]; then
    return 0
  fi

  # Set defaults
  minio_port="${minio_port:-9000}"
  minio_user="${minio_user:-minioadmin}"
  minio_pass="${minio_pass:-minioadmin}"
  minio_bucket="${minio_bucket:-default}"

  # Output as key=value pairs (will be parsed by caller)
  printf "FILE_STORAGE_PROVIDER=minio\n"
  printf "FILE_STORAGE_ENDPOINT=http://127.0.0.1:%s\n" "$minio_port"
  printf "FILE_STORAGE_BUCKET=%s\n" "$minio_bucket"
  printf "FILE_STORAGE_ACCESS_KEY=%s\n" "$minio_user"
  printf "FILE_STORAGE_SECRET_KEY=%s\n" "$minio_pass"
}

# Get Redis configuration from project .env (if enabled)
get_redis_config() {
  local project_dir="${NSELF_PROJECT_DIR:-$(pwd)}"
  local redis_enabled=""
  local redis_host=""
  local redis_port=""
  local redis_pass=""

  # Try to load from project .env files
  for env_file in "$project_dir/.env.dev" "$project_dir/.env.local" "$project_dir/.env"; do
    if [[ -f "$env_file" ]]; then
      redis_enabled=$(grep "^REDIS_ENABLED=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')

      if [[ "$redis_enabled" == "true" ]]; then
        redis_host=$(grep "^REDIS_HOST=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')
        redis_port=$(grep "^REDIS_PORT=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')
        redis_pass=$(grep "^REDIS_PASSWORD=" "$env_file" | head -1 | cut -d= -f2- | tr -d '"' | tr -d "'" | tr -d ' ')
        break
      fi
    fi
  done

  # Return empty if Redis not enabled
  if [[ "$redis_enabled" != "true" ]]; then
    return 0
  fi

  # Set defaults
  redis_host="${redis_host:-127.0.0.1}"
  redis_port="${redis_port:-6379}"

  # Build Redis URL
  local redis_url="redis://"
  if [[ -n "$redis_pass" ]]; then
    redis_url="${redis_url}:${redis_pass}@"
  fi
  redis_url="${redis_url}${redis_host}:${redis_port}"

  printf "JOBS_REDIS_URL=%s\n" "$redis_url"
  printf "REDIS_URL=%s\n" "$redis_url"
}

# ============================================================================
# Plugin Process Management
# ============================================================================

# Check if plugin is running
is_plugin_running() {
  local plugin_name="$1"
  local pid_file="$PLUGIN_PIDS_DIR/${plugin_name}.pid"

  if [[ ! -f "$pid_file" ]]; then
    return 1
  fi

  local pid=$(cat "$pid_file")
  if kill -0 "$pid" 2>/dev/null; then
    return 0
  else
    # PID file exists but process is dead
    rm -f "$pid_file"
    return 1
  fi
}

# Get plugin PID
get_plugin_pid() {
  local plugin_name="$1"
  local pid_file="$PLUGIN_PIDS_DIR/${plugin_name}.pid"

  if [[ -f "$pid_file" ]]; then
    cat "$pid_file"
  fi
}

# Prepare plugin for startup
prepare_plugin() {
  local plugin_name="$1"
  local plugin_dir="$PLUGIN_DIR/$plugin_name/ts"

  if [[ ! -d "$plugin_dir" ]]; then
    log_error "Plugin '$plugin_name' not installed"
    return 1
  fi

  # Install dependencies if needed
  if [[ ! -d "$plugin_dir/node_modules" ]]; then
    log_info "Installing dependencies for $plugin_name..."
    (cd "$plugin_dir" && pnpm install --silent) || return 1
  fi

  # Build if needed
  if [[ ! -d "$plugin_dir/dist" ]]; then
    log_info "Building $plugin_name..."
    (cd "$plugin_dir" && pnpm build 2>&1 >/dev/null) || return 1
  fi

  return 0
}

# Create plugin .env file
create_plugin_env() {
  local plugin_name="$1"
  local plugin_dir="$PLUGIN_DIR/$plugin_name/ts"
  local port="$2"

  # Get plugin manifest for default port
  local manifest="$PLUGIN_DIR/$plugin_name/plugin.json"
  if [[ -z "$port" ]] && [[ -f "$manifest" ]]; then
    if command -v jq >/dev/null 2>&1; then
      port=$(jq -r '.port // 3000' "$manifest" 2>/dev/null)
    else
      port=$(grep '"port"' "$manifest" | head -1 | sed 's/[^0-9]//g')
    fi
  fi
  port="${port:-3000}"

  local env_file="$plugin_dir/.env"

  # Don't overwrite existing .env
  if [[ -f "$env_file" ]]; then
    return 0
  fi

  local db_url=$(get_database_url) || return 1
  local encryption_key=$(get_encryption_key) || return 1

  # Get optional service configurations
  local minio_config=$(get_minio_config)
  local redis_config=$(get_redis_config)

  # Start building .env file
  cat > "$env_file" <<EOF
# Auto-generated by nself plugin start
# Edit as needed for plugin-specific configuration

# Core Configuration
DATABASE_URL=$db_url
ENCRYPTION_KEY=$encryption_key
PORT=$port
LOG_LEVEL=info
EOF

  # Add MinIO configuration if enabled
  if [[ -n "$minio_config" ]]; then
    printf "\n# File Storage (MinIO)\n" >> "$env_file"
    printf "%s\n" "$minio_config" >> "$env_file"
  fi

  # Add Redis configuration if enabled
  if [[ -n "$redis_config" ]]; then
    printf "\n# Cache & Jobs (Redis)\n" >> "$env_file"
    printf "%s\n" "$redis_config" >> "$env_file"
  fi

  log_success "Created .env for $plugin_name"
}

# Start a plugin
start_plugin() {
  local plugin_name="$1"
  local port="$2"

  ensure_runtime_dirs

  if is_plugin_running "$plugin_name"; then
    log_warning "Plugin '$plugin_name' is already running (PID: $(get_plugin_pid "$plugin_name"))"
    return 0
  fi

  # Setup shared utilities if needed
  if [[ ! -L "$HOME/.nself/shared" ]]; then
    setup_shared_utilities || return 1
  fi

  # Prepare plugin (install deps, build)
  prepare_plugin "$plugin_name" || return 1

  # Create .env if needed
  create_plugin_env "$plugin_name" "$port" || return 1

  local plugin_dir="$PLUGIN_DIR/$plugin_name/ts"
  local log_file="$PLUGIN_LOGS_DIR/${plugin_name}.log"
  local pid_file="$PLUGIN_PIDS_DIR/${plugin_name}.pid"

  log_info "Starting $plugin_name..."

  # Start plugin in background
  (
    cd "$plugin_dir" && \
    pnpm start > "$log_file" 2>&1 &
    echo $! > "$pid_file"
  )

  sleep 0.5

  if is_plugin_running "$plugin_name"; then
    log_success "$plugin_name started (PID: $(get_plugin_pid "$plugin_name"))"
    printf "Logs: nself plugin logs %s\n" "$plugin_name"
    return 0
  else
    log_error "$plugin_name failed to start (check logs: $log_file)"
    return 1
  fi
}

# Stop a plugin
stop_plugin() {
  local plugin_name="$1"

  if ! is_plugin_running "$plugin_name"; then
    log_warning "Plugin '$plugin_name' is not running"
    return 0
  fi

  local pid=$(get_plugin_pid "$plugin_name")
  log_info "Stopping $plugin_name (PID: $pid)..."

  if kill "$pid" 2>/dev/null; then
    # Wait for graceful shutdown
    local timeout=5
    while kill -0 "$pid" 2>/dev/null && ((timeout > 0)); do
      sleep 1
      ((timeout--))
    done

    # Force kill if still running
    if kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null
    fi

    rm -f "$PLUGIN_PIDS_DIR/${plugin_name}.pid"
    log_success "$plugin_name stopped"
    return 0
  else
    log_error "Failed to stop $plugin_name"
    return 1
  fi
}

# Restart a plugin
restart_plugin() {
  local plugin_name="$1"
  stop_plugin "$plugin_name"
  sleep 1
  start_plugin "$plugin_name"
}

# ============================================================================
# Batch Operations
# ============================================================================

# Start all installed plugins
start_all_plugins() {
  ensure_runtime_dirs

  local count=0
  local failed=0

  for plugin_dir in "$PLUGIN_DIR"/*/; do
    if [[ -f "$plugin_dir/plugin.json" ]]; then
      local name=$(basename "$plugin_dir")
      # Skip shared utilities directory
      [[ "$name" == "_shared" ]] && continue
      if start_plugin "$name"; then
        ((count++))
      else
        ((failed++))
      fi
    fi
  done

  if [[ $count -eq 0 ]]; then
    log_info "No plugins installed"
    return 0
  fi

  printf "\n"
  log_success "Started $count plugins"
  if [[ $failed -gt 0 ]]; then
    log_warning "$failed plugins failed to start"
  fi
}

# Stop all running plugins
stop_all_plugins() {
  local count=0

  for pid_file in "$PLUGIN_PIDS_DIR"/*.pid; do
    [[ -f "$pid_file" ]] || continue
    local name=$(basename "$pid_file" .pid)
    if stop_plugin "$name"; then
      ((count++))
    fi
  done

  if [[ $count -eq 0 ]]; then
    log_info "No plugins were running"
  else
    log_success "Stopped $count plugins"
  fi
}

# List running plugins
list_running_plugins() {
  printf "\n=== Running Plugins ===\n\n"

  local count=0
  for pid_file in "$PLUGIN_PIDS_DIR"/*.pid; do
    [[ -f "$pid_file" ]] || continue
    local name=$(basename "$pid_file" .pid)

    if is_plugin_running "$name"; then
      local pid=$(get_plugin_pid "$name")

      # Get port from .env
      local port=""
      local env_file="$PLUGIN_DIR/$name/ts/.env"
      if [[ -f "$env_file" ]]; then
        port=$(grep "^PORT=" "$env_file" | cut -d= -f2)
      fi

      printf "%-20s PID: %-8s Port: %s\n" "$name" "$pid" "$port"
      ((count++))
    fi
  done

  if [[ $count -eq 0 ]]; then
    log_info "No plugins currently running"
    printf "\nStart plugins with: nself plugin start <name>\n"
  else
    printf "\nTotal: %d running\n" "$count"
  fi
}

# ============================================================================
# Health Checks
# ============================================================================

# Check plugin health
check_plugin_health() {
  local plugin_name="$1"

  if ! is_plugin_running "$plugin_name"; then
    printf "❌ %s - Not running\n" "$plugin_name"
    return 1
  fi

  # Get port
  local env_file="$PLUGIN_DIR/$plugin_name/ts/.env"
  local port=""
  if [[ -f "$env_file" ]]; then
    port=$(grep "^PORT=" "$env_file" | cut -d= -f2)
  fi

  if [[ -z "$port" ]]; then
    printf "⚠️  %s - Running but port unknown\n" "$plugin_name"
    return 1
  fi

  # Try health endpoint
  if command -v curl >/dev/null 2>&1; then
    if curl -sf "http://localhost:$port/health" >/dev/null 2>&1; then
      printf "✅ %s - Healthy (port %s)\n" "$plugin_name" "$port"
      return 0
    else
      printf "⚠️  %s - Running but not responding (port %s)\n" "$plugin_name" "$port"
      return 1
    fi
  else
    printf "⚠️  %s - Running (port %s, curl not available for health check)\n" "$plugin_name" "$port"
    return 0
  fi
}

# Health check all running plugins
health_check_all() {
  printf "\n=== Plugin Health Check ===\n\n"

  local count=0
  local healthy=0

  for pid_file in "$PLUGIN_PIDS_DIR"/*.pid; do
    [[ -f "$pid_file" ]] || continue
    local name=$(basename "$pid_file" .pid)

    if is_plugin_running "$name"; then
      ((count++))
      if check_plugin_health "$name"; then
        ((healthy++))
      fi
    fi
  done

  if [[ $count -eq 0 ]]; then
    log_info "No plugins running"
  else
    printf "\n%d/%d plugins healthy\n" "$healthy" "$count"
  fi
}

# ============================================================================
# Logs
# ============================================================================

# Show plugin logs
show_plugin_logs() {
  local plugin_name="$1"
  local follow="${2:-false}"

  local log_file="$PLUGIN_LOGS_DIR/${plugin_name}.log"

  if [[ ! -f "$log_file" ]]; then
    log_error "No logs found for $plugin_name"
    printf "Log file: %s\n" "$log_file"
    return 1
  fi

  if [[ "$follow" == "true" ]]; then
    tail -f "$log_file"
  else
    tail -50 "$log_file"
  fi
}

# Export functions
export -f ensure_runtime_dirs
export -f setup_shared_utilities
export -f get_database_url
export -f get_encryption_key
export -f get_minio_config
export -f get_redis_config
export -f is_plugin_running
export -f get_plugin_pid
export -f prepare_plugin
export -f create_plugin_env
export -f start_plugin
export -f stop_plugin
export -f restart_plugin
export -f start_all_plugins
export -f stop_all_plugins
export -f list_running_plugins
export -f check_plugin_health
export -f health_check_all
export -f show_plugin_logs
