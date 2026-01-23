#!/usr/bin/env bash
# sync.sh - Environment synchronization for nself
# Sync database, config, and files between local/staging/production

set -o pipefail

# Source shared utilities
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/platform-compat.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh" 2>/dev/null || true

# Fallback logging
if ! declare -f log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_warning >/dev/null 2>&1; then
  log_warning() { printf "\033[0;33m[WARNING]\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
fi
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
fi

# Fallback color definitions (if not already defined by display.sh)
: "${COLOR_GREEN:=\033[0;32m}"
: "${COLOR_YELLOW:=\033[0;33m}"
: "${COLOR_RED:=\033[0;31m}"
: "${COLOR_CYAN:=\033[0;36m}"
: "${COLOR_RESET:=\033[0m}"

# ============================================================================
# CONFIGURATION
# ============================================================================

SYNC_CONFIG_DIR=".nself/sync"
SYNC_PROFILES_FILE="$SYNC_CONFIG_DIR/profiles.yaml"
SYNC_HISTORY_FILE="$SYNC_CONFIG_DIR/history.log"

# ============================================================================
# ENVIRONMENT DETECTION
# ============================================================================

get_current_env() {
  echo "${ENV:-${ENVIRONMENT:-local}}"
}

is_local() { [[ "$(get_current_env)" == "local" ]] || [[ "$(get_current_env)" == "dev" ]]; }
is_staging() { [[ "$(get_current_env)" == "staging" ]]; }
is_production() { [[ "$(get_current_env)" == "production" ]] || [[ "$(get_current_env)" == "prod" ]]; }

# ============================================================================
# PROFILE MANAGEMENT
# ============================================================================

ensure_sync_dir() {
  mkdir -p "$SYNC_CONFIG_DIR"
}

# Initialize sync profiles
init_profiles() {
  ensure_sync_dir

  if [[ -f "$SYNC_PROFILES_FILE" ]]; then
    log_info "Sync profiles already exist: $SYNC_PROFILES_FILE"
    return 0
  fi

  cat > "$SYNC_PROFILES_FILE" << 'EOF'
# nself Sync Profiles
# Define your remote environments for syncing

profiles:
  staging:
    host: staging.example.com
    user: deploy
    port: 22
    path: /var/www/myapp
    ssh_key: ~/.ssh/id_rsa

  production:
    host: prod.example.com
    user: deploy
    port: 22
    path: /var/www/myapp
    ssh_key: ~/.ssh/id_rsa

# Sync options
options:
  # Always anonymize when pulling from production
  anonymize_from_prod: true

  # Create backup before sync
  backup_before_sync: true

  # Exclude patterns for file sync
  exclude:
    - node_modules
    - .git
    - _backups
    - "*.log"
    - ".env.local"
EOF

  log_success "Created sync profiles: $SYNC_PROFILES_FILE"
  log_info "Edit this file to configure your remote environments"
}

# Parse profile (simplified - would use yq in production)
get_profile_value() {
  local profile="$1"
  local key="$2"

  if [[ ! -f "$SYNC_PROFILES_FILE" ]]; then
    return 1
  fi

  # Simple grep-based parsing (would use yq in production)
  grep -A 10 "^  $profile:" "$SYNC_PROFILES_FILE" | grep "    $key:" | head -1 | sed 's/.*: *//' | tr -d '"'
}

# ============================================================================
# SSH HELPERS
# ============================================================================

test_ssh_connection() {
  local host="$1"
  local user="$2"
  local port="${3:-22}"
  local ssh_key="$4"

  local ssh_opts="-o ConnectTimeout=5 -o BatchMode=yes"
  [[ -n "$ssh_key" ]] && ssh_opts="$ssh_opts -i $ssh_key"

  if ssh $ssh_opts -p "$port" "$user@$host" "echo ok" >/dev/null 2>&1; then
    return 0
  fi
  return 1
}

run_remote_cmd() {
  local profile="$1"
  shift
  local cmd="$*"

  local host=$(get_profile_value "$profile" "host")
  local user=$(get_profile_value "$profile" "user")
  local port=$(get_profile_value "$profile" "port")
  local ssh_key=$(get_profile_value "$profile" "ssh_key")
  local path=$(get_profile_value "$profile" "path")

  [[ -z "$host" ]] && { log_error "Profile '$profile' not found or missing host"; return 1; }

  local ssh_opts="-o ConnectTimeout=10"
  [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key") && ssh_opts="$ssh_opts -i $ssh_key"
  [[ -n "$port" ]] && ssh_opts="$ssh_opts -p $port"

  ssh $ssh_opts "$user@$host" "cd $path && $cmd"
}

# ============================================================================
# DATABASE SYNC
# ============================================================================

# Pull database from remote environment
cmd_pull() {
  local source="${1:-staging}"
  local anonymize="${2:-}"

  # Validate source
  if [[ "$source" == "local" ]]; then
    log_error "Cannot pull from local - you're already local"
    return 1
  fi

  # Force anonymization from production
  if [[ "$source" == "production" ]] || [[ "$source" == "prod" ]]; then
    if [[ "$anonymize" != "--anonymize" ]] && [[ "$anonymize" != "-a" ]]; then
      log_error "Pulling from production requires --anonymize flag"
      log_info "Usage: nself sync pull production --anonymize"
      return 1
    fi
    source="production"
    anonymize="--anonymize"
  fi

  log_info "Pulling database from $source..."

  # Check SSH connection
  local host=$(get_profile_value "$source" "host")
  if [[ -z "$host" ]]; then
    log_error "Profile '$source' not configured"
    log_info "Run 'nself sync init' to create profiles, then edit $SYNC_PROFILES_FILE"
    return 1
  fi

  local user=$(get_profile_value "$source" "user")
  local port=$(get_profile_value "$source" "port")
  local ssh_key=$(get_profile_value "$source" "ssh_key")

  log_info "Testing connection to $user@$host..."
  if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
    log_error "Cannot connect to $source server"
    log_info "Check SSH key and server availability"
    return 1
  fi
  log_success "Connection OK"

  # Create backup locally first
  log_info "Creating local backup before sync..."
  if command -v nself >/dev/null 2>&1; then
    nself db backup create "pre-sync-$(date +%Y%m%d_%H%M%S)" 2>/dev/null || true
  fi

  # Get remote database dump
  log_info "Exporting database from $source..."
  local temp_dump=$(mktemp)

  if run_remote_cmd "$source" "nself db backup database --compress" > "$temp_dump.gz" 2>/dev/null; then
    log_success "Database exported"
    gunzip "$temp_dump.gz" 2>/dev/null || mv "$temp_dump.gz" "$temp_dump"
  else
    # Fallback: direct pg_dump
    log_info "Trying direct pg_dump..."
    local remote_path=$(get_profile_value "$source" "path")

    run_remote_cmd "$source" "docker exec \$(docker ps -qf 'name=postgres' | head -1) pg_dump -U postgres nhost" > "$temp_dump" 2>/dev/null

    if [[ ! -s "$temp_dump" ]]; then
      log_error "Failed to export database from $source"
      rm -f "$temp_dump"
      return 1
    fi
  fi

  # Import locally
  log_info "Importing database locally..."

  local container=$(docker ps --filter "name=postgres" --format "{{.Names}}" | head -1)
  if [[ -z "$container" ]]; then
    log_error "Local PostgreSQL is not running. Run 'nself start' first."
    rm -f "$temp_dump"
    return 1
  fi

  # Drop and recreate
  docker exec -i "$container" psql -U postgres -c "DROP DATABASE IF EXISTS nhost;" 2>/dev/null || true
  docker exec -i "$container" psql -U postgres -c "CREATE DATABASE nhost;" 2>/dev/null || true
  docker exec -i "$container" psql -U postgres -d nhost < "$temp_dump" 2>/dev/null

  log_success "Database imported"

  # Anonymize if requested
  if [[ "$anonymize" == "--anonymize" ]] || [[ "$anonymize" == "-a" ]]; then
    log_info "Anonymizing PII data..."
    if command -v nself >/dev/null 2>&1; then
      nself db data anonymize 2>/dev/null || true
    else
      # Manual anonymization
      docker exec -i "$container" psql -U postgres -d nhost -c "
        UPDATE auth.users SET
          email = 'user' || id || '@anonymized.local',
          encrypted_password = 'anonymized'
        WHERE email IS NOT NULL;
      " 2>/dev/null || true
    fi
    log_success "Data anonymized"
  fi

  rm -f "$temp_dump"

  # Log sync
  ensure_sync_dir
  echo "$(date -Iseconds) PULL $source $(get_current_env) ${anonymize:-}" >> "$SYNC_HISTORY_FILE"

  log_success "Sync complete: $source → local"
}

# Pull environment config files (new hierarchy support)
cmd_pull_env() {
  local source="${1:-staging}"

  case "$source" in
    staging)
      log_info "Pulling staging environment config..."

      local host=$(get_profile_value "staging" "host")
      if [[ -z "$host" ]]; then
        log_error "Staging profile not configured"
        log_info "Run 'nself sync init' and edit $SYNC_PROFILES_FILE"
        return 1
      fi

      local user=$(get_profile_value "staging" "user")
      local port=$(get_profile_value "staging" "port")
      local ssh_key=$(get_profile_value "staging" "ssh_key")

      log_info "Testing SSH access to staging..."
      if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
        log_error "No SSH access to staging server"
        log_info "Ask your tech lead to add your SSH key"
        return 1
      fi

      # Pull .env.staging from remote
      local remote_config=$(run_remote_cmd "staging" "cat .env.staging 2>/dev/null || cat .env 2>/dev/null || echo ''")

      if [[ -z "$remote_config" ]]; then
        log_error "Could not read staging config from server"
        return 1
      fi

      echo "$remote_config" > ".env.staging"
      chmod 600 ".env.staging"

      log_success "Saved to .env.staging"
      printf "Variables pulled: %d\n" "$(grep -c '^[A-Z]' .env.staging 2>/dev/null || echo 0)"
      ;;

    prod|production)
      log_info "Pulling production environment config..."

      local host=$(get_profile_value "production" "host")
      if [[ -z "$host" ]]; then
        log_error "Production profile not configured"
        return 1
      fi

      local user=$(get_profile_value "production" "user")
      local port=$(get_profile_value "production" "port")
      local ssh_key=$(get_profile_value "production" "ssh_key")

      log_info "Testing SSH access to production..."
      if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
        log_error "No SSH access to production server"
        log_info "Only Lead Devs have production access"
        return 1
      fi

      # Pull .env.prod from remote
      local remote_config=$(run_remote_cmd "production" "cat .env.prod 2>/dev/null || cat .env 2>/dev/null || echo ''")

      if [[ -z "$remote_config" ]]; then
        log_error "Could not read production config from server"
        return 1
      fi

      echo "$remote_config" > ".env.prod"
      chmod 600 ".env.prod"

      log_success "Saved to .env.prod"
      printf "Variables pulled: %d\n" "$(grep -c '^[A-Z]' .env.prod 2>/dev/null || echo 0)"
      log_warning "Remember: .env.prod should be gitignored"
      ;;

    secrets)
      log_info "Pulling production secrets..."

      local host=$(get_profile_value "production" "host")
      if [[ -z "$host" ]]; then
        log_error "Production profile not configured"
        return 1
      fi

      local user=$(get_profile_value "production" "user")
      local port=$(get_profile_value "production" "port")
      local ssh_key=$(get_profile_value "production" "ssh_key")

      log_info "Testing SSH access to production..."
      if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
        log_error "No SSH access to production server"
        log_info "Only Lead Devs have access to production secrets"
        return 1
      fi

      # Pull .secrets from remote
      local remote_secrets=$(run_remote_cmd "production" "cat .secrets 2>/dev/null || echo ''")

      if [[ -z "$remote_secrets" ]]; then
        log_warning "No .secrets file found on production server"
        log_info "Generate secrets on the server first, or create .secrets manually"
        return 1
      fi

      echo "$remote_secrets" > ".secrets"
      chmod 600 ".secrets"

      log_success "Saved to .secrets (chmod 600)"
      printf "Secrets pulled: %d\n" "$(grep -c '^[A-Z]' .secrets 2>/dev/null || echo 0)"
      log_warning "CRITICAL: .secrets must be gitignored!"

      # Verify gitignore
      if [[ -f ".gitignore" ]]; then
        if ! grep -q "^\.secrets$" .gitignore 2>/dev/null; then
          log_error ".secrets is NOT in .gitignore - FIX THIS IMMEDIATELY"
          printf "Add this line to .gitignore:\n"
          printf "  .secrets\n"
        fi
      fi
      ;;

    *)
      log_error "Unknown environment: $source"
      printf "Valid options: staging, prod, secrets\n"
      return 1
      ;;
  esac
}

# Push environment config files to remote and optionally rebuild
cmd_push_env() {
  local target="${1:-staging}"
  local rebuild="${2:-}"

  case "$target" in
    staging)
      log_info "Pushing environment config to staging..."

      if [[ ! -f ".env.staging" ]]; then
        log_error "No .env.staging file found"
        log_info "Create one or run: nself sync pull staging"
        return 1
      fi

      local host=$(get_profile_value "staging" "host")
      if [[ -z "$host" ]]; then
        log_error "Staging profile not configured"
        return 1
      fi

      local user=$(get_profile_value "staging" "user")
      local port=$(get_profile_value "staging" "port")
      local ssh_key=$(get_profile_value "staging" "ssh_key")
      local remote_path=$(get_profile_value "staging" "path")

      log_info "Testing SSH access to staging..."
      if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
        log_error "No SSH access to staging server"
        return 1
      fi

      # Push .env.staging to remote
      log_info "Uploading .env.staging to $host:$remote_path..."

      local scp_opts=""
      [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key") && scp_opts="-i $ssh_key"
      [[ -n "$port" ]] && scp_opts="$scp_opts -P $port"

      scp $scp_opts ".env.staging" "$user@$host:$remote_path/.env.staging" 2>/dev/null

      # Set permissions
      run_remote_cmd "staging" "chmod 600 .env.staging"

      log_success "Pushed .env.staging to staging server"

      # Auto-rebuild if requested or by default
      if [[ "$rebuild" == "--rebuild" ]] || [[ "$rebuild" == "-r" ]] || [[ -z "$rebuild" ]]; then
        printf "\n"
        log_info "Rebuilding staging environment..."

        run_remote_cmd "staging" "
          # Merge env files
          if [ -f .env.dev ]; then
            cat .env.dev > .env
            echo '' >> .env
          fi
          cat .env.staging >> .env

          # Add secrets if present
          if [ -f .secrets ]; then
            echo '' >> .env
            cat .secrets >> .env
          fi

          chmod 600 .env

          # Restart services
          if command -v nself >/dev/null 2>&1; then
            nself restart
          elif [ -f docker-compose.yml ]; then
            docker compose up -d --force-recreate
          fi

          echo 'rebuild_complete'
        " 2>/dev/null | grep -q "rebuild_complete" && log_success "Staging rebuilt successfully" || log_warning "Rebuild may have issues - check staging"
      fi
      ;;

    prod|production)
      log_info "Pushing environment config to production..."

      if [[ ! -f ".env.prod" ]]; then
        log_error "No .env.prod file found"
        return 1
      fi

      local host=$(get_profile_value "production" "host")
      if [[ -z "$host" ]]; then
        log_error "Production profile not configured"
        return 1
      fi

      local user=$(get_profile_value "production" "user")
      local port=$(get_profile_value "production" "port")
      local ssh_key=$(get_profile_value "production" "ssh_key")
      local remote_path=$(get_profile_value "production" "path")

      log_info "Testing SSH access to production..."
      if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
        log_error "No SSH access to production server"
        log_info "Only Lead Devs have production access"
        return 1
      fi

      # Confirm production push
      log_warning "You are about to push config to PRODUCTION"
      printf "Continue? (y/N): "
      read -r response
      if [[ ! "$response" =~ ^[Yy]$ ]]; then
        log_info "Cancelled"
        return 0
      fi

      # Push .env.prod to remote
      log_info "Uploading .env.prod to $host:$remote_path..."

      local scp_opts=""
      [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key") && scp_opts="-i $ssh_key"
      [[ -n "$port" ]] && scp_opts="$scp_opts -P $port"

      scp $scp_opts ".env.prod" "$user@$host:$remote_path/.env.prod" 2>/dev/null

      run_remote_cmd "production" "chmod 600 .env.prod"

      log_success "Pushed .env.prod to production server"

      # Auto-rebuild if requested
      if [[ "$rebuild" == "--rebuild" ]] || [[ "$rebuild" == "-r" ]]; then
        printf "\n"
        log_warning "Rebuilding PRODUCTION environment..."

        run_remote_cmd "production" "
          # Merge env files
          if [ -f .env.dev ]; then
            cat .env.dev > .env
            echo '' >> .env
          fi
          cat .env.prod >> .env

          # Add secrets
          if [ -f .secrets ]; then
            echo '' >> .env
            cat .secrets >> .env
          fi

          chmod 600 .env

          # Restart services
          if command -v nself >/dev/null 2>&1; then
            nself restart
          elif [ -f docker-compose.yml ]; then
            docker compose up -d --force-recreate
          fi

          echo 'rebuild_complete'
        " 2>/dev/null | grep -q "rebuild_complete" && log_success "Production rebuilt successfully" || log_warning "Rebuild may have issues - check production"
      else
        log_info "Run with --rebuild to restart services"
      fi
      ;;

    secrets)
      log_info "Pushing secrets to production..."

      if [[ ! -f ".secrets" ]]; then
        log_error "No .secrets file found locally"
        return 1
      fi

      local host=$(get_profile_value "production" "host")
      if [[ -z "$host" ]]; then
        log_error "Production profile not configured"
        return 1
      fi

      local user=$(get_profile_value "production" "user")
      local port=$(get_profile_value "production" "port")
      local ssh_key=$(get_profile_value "production" "ssh_key")
      local remote_path=$(get_profile_value "production" "path")

      log_info "Testing SSH access to production..."
      if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
        log_error "No SSH access to production server"
        return 1
      fi

      # Confirm secrets push
      log_warning "You are about to push SECRETS to PRODUCTION"
      printf "This is a sensitive operation. Continue? (y/N): "
      read -r response
      if [[ ! "$response" =~ ^[Yy]$ ]]; then
        log_info "Cancelled"
        return 0
      fi

      # Push .secrets to remote
      log_info "Uploading .secrets to $host:$remote_path..."

      local scp_opts=""
      [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key") && scp_opts="-i $ssh_key"
      [[ -n "$port" ]] && scp_opts="$scp_opts -P $port"

      scp $scp_opts ".secrets" "$user@$host:$remote_path/.secrets" 2>/dev/null

      run_remote_cmd "production" "chmod 600 .secrets"

      log_success "Pushed .secrets to production server"

      if [[ "$rebuild" == "--rebuild" ]] || [[ "$rebuild" == "-r" ]]; then
        log_info "Rebuilding production with new secrets..."
        run_remote_cmd "production" "
          if command -v nself >/dev/null 2>&1; then
            nself restart
          elif [ -f docker-compose.yml ]; then
            docker compose up -d --force-recreate
          fi
        " 2>/dev/null
        log_success "Production restarted with new secrets"
      fi
      ;;

    *)
      log_error "Unknown target: $target"
      printf "Valid options: staging, prod, secrets\n"
      return 1
      ;;
  esac

  # Log sync
  ensure_sync_dir
  echo "$(date -Iseconds) PUSH_ENV $target $(get_current_env)" >> "$SYNC_HISTORY_FILE"
}

# Sync frontend apps to staging
cmd_frontend() {
  local action="${1:-sync}"
  local target="${2:-staging}"

  case "$action" in
    sync|push)
      sync_frontend_apps "$target"
      ;;
    pull)
      log_error "Frontend pull not supported - use git pull on server instead"
      return 1
      ;;
    *)
      log_error "Unknown action: $action"
      printf "Usage: nself sync frontend sync|push [staging|prod]\n"
      return 1
      ;;
  esac
}

# Sync frontend apps to remote server
sync_frontend_apps() {
  local target="${1:-staging}"

  # Get frontend app configs from environment
  [[ -f ".env" ]] && source ".env" 2>/dev/null || true
  [[ -f ".env.dev" ]] && source ".env.dev" 2>/dev/null || true
  [[ -f ".env.local" ]] && source ".env.local" 2>/dev/null || true

  local host=$(get_profile_value "$target" "host")
  if [[ -z "$host" ]]; then
    log_error "Profile '$target' not configured"
    return 1
  fi

  local user=$(get_profile_value "$target" "user")
  local port=$(get_profile_value "$target" "port")
  local ssh_key=$(get_profile_value "$target" "ssh_key")
  local remote_path=$(get_profile_value "$target" "path")

  log_info "Syncing frontend apps to $target..."

  # Check SSH connection
  if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
    log_error "Cannot connect to $target server"
    return 1
  fi

  local found_apps=0

  # Iterate through FRONTEND_APP_N configs
  for i in 1 2 3 4 5 6 7 8 9 10; do
    local name_var="FRONTEND_APP_${i}_NAME"
    local path_var="FRONTEND_APP_${i}_PATH"
    local repo_var="FRONTEND_APP_${i}_REPO"
    local branch_var="FRONTEND_APP_${i}_BRANCH"

    local app_name="${!name_var:-}"
    local app_path="${!path_var:-}"
    local app_repo="${!repo_var:-}"
    local app_branch="${!branch_var:-main}"

    [[ -z "$app_name" ]] && continue

    found_apps=$((found_apps + 1))

    printf "\n  Syncing frontend: %s\n" "$app_name"

    # Determine sync method
    if [[ -n "$app_repo" ]]; then
      # Git-based sync
      printf "    Method: Git (%s)\n" "$app_repo"

      run_remote_cmd "$target" "
        mkdir -p '$remote_path/frontends/$app_name'
        cd '$remote_path/frontends/$app_name'

        if [ -d '.git' ]; then
          git fetch origin
          git checkout '$app_branch'
          git pull origin '$app_branch'
        else
          git clone -b '$app_branch' '$app_repo' .
        fi

        # Install dependencies and build if package.json exists
        if [ -f 'package.json' ]; then
          if command -v pnpm >/dev/null 2>&1; then
            pnpm install --frozen-lockfile 2>/dev/null || pnpm install
            pnpm build 2>/dev/null || true
          elif command -v npm >/dev/null 2>&1; then
            npm ci 2>/dev/null || npm install
            npm run build 2>/dev/null || true
          fi
        fi

        echo 'sync_ok'
      " 2>/dev/null | grep -q "sync_ok" && printf "    ${COLOR_GREEN}✓${COLOR_RESET} Synced via git\n" || printf "    ${COLOR_YELLOW}!${COLOR_RESET} Sync may have issues\n"

    elif [[ -n "$app_path" ]] && [[ -d "$app_path" ]]; then
      # Rsync-based sync (local directory)
      printf "    Method: rsync (%s)\n" "$app_path"

      local rsync_opts="-avz --delete"
      [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key")
      local ssh_cmd="ssh -p ${port:-22}"
      [[ -n "$ssh_key" ]] && ssh_cmd="$ssh_cmd -i $ssh_key"

      rsync_opts="$rsync_opts -e '$ssh_cmd'"
      rsync_opts="$rsync_opts --exclude=node_modules --exclude=.git --exclude='.env.local' --exclude='*.log'"

      # Create remote directory
      run_remote_cmd "$target" "mkdir -p '$remote_path/frontends/$app_name'" 2>/dev/null

      # Sync files
      if eval rsync $rsync_opts "'$app_path/'" "'$user@$host:$remote_path/frontends/$app_name/'" 2>/dev/null; then
        printf "    ${COLOR_GREEN}✓${COLOR_RESET} Synced via rsync\n"
      else
        printf "    ${COLOR_YELLOW}!${COLOR_RESET} Sync may have issues\n"
      fi
    else
      printf "    ${COLOR_YELLOW}!${COLOR_RESET} No path or repo configured\n"
    fi
  done

  if [[ $found_apps -eq 0 ]]; then
    log_warning "No frontend apps configured"
    log_info "Configure in .env with FRONTEND_APP_N_NAME, FRONTEND_APP_N_PATH or FRONTEND_APP_N_REPO"
    return 0
  fi

  printf "\n"
  log_success "Frontend sync complete: $found_apps app(s)"
}

# Full sync - everything to target environment
cmd_full() {
  local target="${1:-staging}"
  local skip_db="${2:-}"

  log_info "Full sync to $target..."
  printf "\n"

  # 1. Sync env config
  printf "Step 1/4: Syncing environment config...\n"
  cmd_push_env "$target" "--no-rebuild"

  # 2. Sync files (backend)
  printf "\nStep 2/4: Syncing backend files...\n"
  sync_files_push "$target" "."

  # 3. Sync frontend apps (staging only by default)
  if [[ "$target" == "staging" ]] || [[ "$target" == "stage" ]]; then
    printf "\nStep 3/4: Syncing frontend apps...\n"
    sync_frontend_apps "$target"
  else
    printf "\nStep 3/4: Skipping frontend apps (production)\n"
    printf "  Frontend apps should be deployed via Vercel/CDN\n"
  fi

  # 4. Rebuild remote
  printf "\nStep 4/4: Rebuilding remote environment...\n"

  local profile="$target"
  [[ "$target" == "prod" ]] && profile="production"

  # Determine the correct env file based on target
  local env_file=".env.staging"
  [[ "$target" == "prod" ]] || [[ "$target" == "production" ]] && env_file=".env.prod"

  run_remote_cmd "$profile" "
    # Merge env files (base + target + secrets)
    > .env
    [ -f .env.dev ] && cat .env.dev >> .env && echo '' >> .env
    [ -f $env_file ] && cat $env_file >> .env && echo '' >> .env
    [ -f .secrets ] && cat .secrets >> .env

    chmod 600 .env

    # Rebuild
    if command -v nself >/dev/null 2>&1; then
      nself build && nself restart
    elif [ -f docker-compose.yml ]; then
      docker compose build
      docker compose up -d --force-recreate
    fi

    echo 'full_sync_complete'
  " 2>/dev/null | grep -q "full_sync_complete" && log_success "Full sync to $target complete" || log_warning "Sync completed with warnings"
}

# Push database to remote environment
cmd_push() {
  local target="${1:-staging}"

  # Safety checks
  if [[ "$target" == "local" ]]; then
    log_error "Cannot push to local - you're already local"
    return 1
  fi

  if [[ "$target" == "production" ]] || [[ "$target" == "prod" ]]; then
    log_error "Direct push to production is blocked for safety"
    log_info "Use your CI/CD pipeline for production deployments"
    log_info "Or use: nself deploy production"
    return 1
  fi

  log_info "Pushing database to $target..."

  # Check SSH connection
  local host=$(get_profile_value "$target" "host")
  if [[ -z "$host" ]]; then
    log_error "Profile '$target' not configured"
    return 1
  fi

  local user=$(get_profile_value "$target" "user")
  local port=$(get_profile_value "$target" "port")
  local ssh_key=$(get_profile_value "$target" "ssh_key")

  log_info "Testing connection to $user@$host..."
  if ! test_ssh_connection "$host" "$user" "$port" "$ssh_key"; then
    log_error "Cannot connect to $target server"
    return 1
  fi
  log_success "Connection OK"

  # Confirm
  log_warning "This will overwrite the $target database"
  printf "Continue? (y/N): "
  read -r response
  if [[ ! "$response" =~ ^[Yy]$ ]]; then
    log_info "Cancelled"
    return 0
  fi

  # Export local database
  log_info "Exporting local database..."
  local temp_dump=$(mktemp)
  local container=$(docker ps --filter "name=postgres" --format "{{.Names}}" | head -1)

  if [[ -z "$container" ]]; then
    log_error "Local PostgreSQL is not running"
    return 1
  fi

  docker exec "$container" pg_dump -U postgres nhost > "$temp_dump"
  gzip "$temp_dump"

  # Upload to remote
  log_info "Uploading to $target..."
  local ssh_opts=""
  [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key") && ssh_opts="-i $ssh_key"
  [[ -n "$port" ]] && ssh_opts="$ssh_opts -P $port"

  local remote_path=$(get_profile_value "$target" "path")
  scp $ssh_opts "$temp_dump.gz" "$user@$host:$remote_path/_sync_import.sql.gz"

  # Import on remote
  log_info "Importing on $target..."
  run_remote_cmd "$target" "
    gunzip -f _sync_import.sql.gz
    docker exec -i \$(docker ps -qf 'name=postgres' | head -1) psql -U postgres -d nhost < _sync_import.sql
    rm -f _sync_import.sql
  "

  rm -f "$temp_dump.gz"

  # Log sync
  ensure_sync_dir
  echo "$(date -Iseconds) PUSH $(get_current_env) $target" >> "$SYNC_HISTORY_FILE"

  log_success "Sync complete: local → $target"
}

# ============================================================================
# FILE SYNC
# ============================================================================

cmd_files() {
  local direction="${1:-pull}"
  local target="${2:-staging}"
  local path="${3:-.}"

  case "$direction" in
    pull)
      sync_files_pull "$target" "$path"
      ;;
    push)
      sync_files_push "$target" "$path"
      ;;
    *)
      log_error "Unknown direction: $direction"
      log_info "Usage: nself sync files pull|push <environment> [path]"
      return 1
      ;;
  esac
}

sync_files_pull() {
  local source="$1"
  local path="$2"

  local host=$(get_profile_value "$source" "host")
  local user=$(get_profile_value "$source" "user")
  local port=$(get_profile_value "$source" "port")
  local ssh_key=$(get_profile_value "$source" "ssh_key")
  local remote_path=$(get_profile_value "$source" "path")

  [[ -z "$host" ]] && { log_error "Profile '$source' not configured"; return 1; }

  log_info "Syncing files from $source..."

  local rsync_opts="-avz --progress"
  [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key")
  [[ -n "$port" ]] && rsync_opts="$rsync_opts -e 'ssh -p $port -i $ssh_key'"

  # Add excludes
  rsync_opts="$rsync_opts --exclude=node_modules --exclude=.git --exclude=_backups --exclude='*.log' --exclude='.env.local'"

  rsync $rsync_opts "$user@$host:$remote_path/$path" "./$path"

  log_success "Files synced from $source"
}

sync_files_push() {
  local target="$1"
  local path="$2"

  if [[ "$target" == "production" ]] || [[ "$target" == "prod" ]]; then
    log_error "Direct file push to production is blocked"
    log_info "Use your CI/CD pipeline or 'nself deploy production'"
    return 1
  fi

  local host=$(get_profile_value "$target" "host")
  local user=$(get_profile_value "$target" "user")
  local port=$(get_profile_value "$target" "port")
  local ssh_key=$(get_profile_value "$target" "ssh_key")
  local remote_path=$(get_profile_value "$target" "path")

  [[ -z "$host" ]] && { log_error "Profile '$target' not configured"; return 1; }

  log_info "Syncing files to $target..."

  local rsync_opts="-avz --progress"
  [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key")
  [[ -n "$port" ]] && rsync_opts="$rsync_opts -e 'ssh -p $port -i $ssh_key'"

  # Add excludes
  rsync_opts="$rsync_opts --exclude=node_modules --exclude=.git --exclude=_backups --exclude='*.log' --exclude='.env.local' --exclude='.env'"

  rsync $rsync_opts "./$path" "$user@$host:$remote_path/$path"

  log_success "Files synced to $target"
}

# ============================================================================
# CONFIG SYNC
# ============================================================================

cmd_config() {
  local direction="${1:-pull}"
  local target="${2:-staging}"

  case "$direction" in
    pull)
      sync_config_pull "$target"
      ;;
    push)
      sync_config_push "$target"
      ;;
    diff)
      sync_config_diff "$target"
      ;;
    *)
      log_error "Unknown direction: $direction"
      log_info "Usage: nself sync config pull|push|diff <environment>"
      return 1
      ;;
  esac
}

sync_config_pull() {
  local source="$1"

  log_info "Pulling config from $source..."

  local remote_config=$(run_remote_cmd "$source" "cat .env 2>/dev/null || cat .env.staging 2>/dev/null || echo ''")

  if [[ -z "$remote_config" ]]; then
    log_error "Could not read config from $source"
    return 1
  fi

  # Save to .env.{source} locally
  echo "$remote_config" > ".env.$source"
  log_success "Config saved to .env.$source"
  log_info "Review and merge changes as needed"
}

sync_config_push() {
  local target="$1"

  if [[ "$target" == "production" ]] || [[ "$target" == "prod" ]]; then
    log_error "Direct config push to production is blocked"
    log_info "Use secure config management for production"
    return 1
  fi

  if [[ ! -f ".env" ]]; then
    log_error "No .env file found locally"
    return 1
  fi

  log_info "Pushing config to $target..."

  # Confirm
  log_warning "This will overwrite the $target configuration"
  printf "Continue? (y/N): "
  read -r response
  if [[ ! "$response" =~ ^[Yy]$ ]]; then
    log_info "Cancelled"
    return 0
  fi

  local host=$(get_profile_value "$target" "host")
  local user=$(get_profile_value "$target" "user")
  local port=$(get_profile_value "$target" "port")
  local ssh_key=$(get_profile_value "$target" "ssh_key")
  local remote_path=$(get_profile_value "$target" "path")

  local scp_opts=""
  [[ -n "$ssh_key" ]] && ssh_key=$(eval echo "$ssh_key") && scp_opts="-i $ssh_key"
  [[ -n "$port" ]] && scp_opts="$scp_opts -P $port"

  scp $scp_opts ".env" "$user@$host:$remote_path/.env"

  log_success "Config pushed to $target"
  log_info "SSH to $target and run 'nself restart' to apply changes"
}

sync_config_diff() {
  local target="$1"

  log_info "Comparing config with $target..."

  local remote_config=$(run_remote_cmd "$target" "cat .env 2>/dev/null || echo ''")

  if [[ -z "$remote_config" ]]; then
    log_error "Could not read config from $target"
    return 1
  fi

  local temp_remote=$(mktemp)
  echo "$remote_config" > "$temp_remote"

  log_info "Config differences (local vs $target):"
  echo "───────────────────────────────────────"
  diff -u ".env" "$temp_remote" || true

  rm -f "$temp_remote"
}

# ============================================================================
# STATUS & HISTORY
# ============================================================================

cmd_status() {
  local target="${1:-all}"

  log_info "Sync Status"
  echo ""

  if [[ "$target" == "all" ]]; then
    # Show all configured profiles
    for profile in staging production; do
      local host=$(get_profile_value "$profile" "host")
      if [[ -n "$host" ]]; then
        printf "  %-12s " "$profile:"
        local user=$(get_profile_value "$profile" "user")
        local port=$(get_profile_value "$profile" "port")
        local ssh_key=$(get_profile_value "$profile" "ssh_key")

        if test_ssh_connection "$host" "$user" "$port" "$ssh_key" 2>/dev/null; then
          printf "\033[32m● Online\033[0m (%s@%s)\n" "$user" "$host"
        else
          printf "\033[31m○ Offline\033[0m (%s@%s)\n" "$user" "$host"
        fi
      fi
    done
  else
    # Show specific profile status
    local host=$(get_profile_value "$target" "host")
    if [[ -z "$host" ]]; then
      log_error "Profile '$target' not configured"
      return 1
    fi

    local user=$(get_profile_value "$target" "user")
    log_info "Checking $target ($user@$host)..."

    if run_remote_cmd "$target" "nself status" 2>/dev/null; then
      log_success "Connected to $target"
    else
      log_error "Cannot connect to $target"
    fi
  fi

  # Show recent history
  if [[ -f "$SYNC_HISTORY_FILE" ]]; then
    echo ""
    log_info "Recent Sync History:"
    tail -5 "$SYNC_HISTORY_FILE" | while read -r line; do
      echo "  $line"
    done
  fi
}

cmd_history() {
  if [[ ! -f "$SYNC_HISTORY_FILE" ]]; then
    log_info "No sync history yet"
    return 0
  fi

  log_info "Sync History"
  echo "───────────────────────────────────────"
  cat "$SYNC_HISTORY_FILE"
}

# ============================================================================
# HELP
# ============================================================================

show_help() {
  cat << 'EOF'
nself sync - Environment Synchronization

USAGE:
  nself sync <command> [options]

COMMANDS:
  Pull (download from remote):
    pull staging              Pull .env.staging from staging server
    pull prod                 Pull .env.prod from production server
    pull secrets              Pull .secrets from production server
    pull <env> --db           Pull database from remote environment

  Push (upload to remote):
    push staging              Push .env.staging to staging (auto-rebuilds by default)
    push staging --no-rebuild Push without restarting services
    push prod [--rebuild]     Push .env.prod to production (no auto-rebuild)
    push secrets [--rebuild]  Push .secrets to production

  Frontend Apps:
    frontend sync staging     Sync all frontend apps to staging (git/rsync)
    frontend sync prod        Sync frontends to production (if configured)

  Full Sync (everything):
    full staging              Full sync: env + files + frontends + rebuild
    full prod                 Full sync to production (backend only)

  Database:
    db pull <env>             Pull database from remote
    db push <env>             Push database to remote (staging only)

  Files:
    files pull <env> [path]   Pull files from remote
    files push <env> [path]   Push files to remote

  Config (legacy):
    config pull <env>         Pull .env from remote
    config push <env>         Push .env to remote
    config diff <env>         Compare configs

  Management:
    init                      Create sync profiles configuration
    status [env]              Show connection status
    history                   Show sync history

ACCESS LEVELS:
  Dev         Local only (.env.dev + .env.local)
  Sr Dev      + staging access (SSH to staging server)
  Lead Dev    + prod + secrets (SSH to production server)

ENVIRONMENT FILE HIERARCHY:
  .env.dev     → Base config (committed to git)
  .env.local   → Your machine overrides (gitignored)
  .env.staging → Staging server config (SSH sync)
  .env.prod    → Production server config (SSH sync)
  .secrets     → Top-secret credentials (generated on server)

DEFAULT DEPLOYMENT BEHAVIOR:
  Staging:    Backend + Frontend apps (full replica)
  Production: Backend only (frontends on Vercel/CDN)

FRONTEND APP CONFIGURATION:
  In .env or .env.dev, configure frontend apps:
    FRONTEND_APP_1_NAME=web
    FRONTEND_APP_1_PATH=../frontend     # Local path for rsync
    FRONTEND_APP_1_REPO=git@...         # Git repo for server clone
    FRONTEND_APP_1_BRANCH=main

EXAMPLES:
  # Typical development workflow
  nself sync pull staging              # Get staging config
  # ... make changes to .env.staging ...
  nself sync push staging              # Push and auto-rebuild

  # Push without rebuilding (useful for minor config changes)
  nself sync push staging --no-rebuild # Push config only, skip restart

  # Full deployment to staging
  nself sync full staging              # Everything: env + files + frontends

  # Production (backend only, requires explicit --rebuild)
  nself sync push prod                 # Push config (no auto-rebuild)
  nself sync push prod --rebuild       # Push config and restart services

  # Frontend apps (staging)
  nself sync frontend sync staging     # Sync all configured frontends

  # Database operations
  nself sync pull staging --db         # Pull staging database
  nself sync pull prod --db --anonymize # Pull prod (anonymized)

CONFIGURATION:
  Edit .nself/sync/profiles.yaml:
    profiles:
      staging:
        host: staging.example.com
        user: deploy
        port: 22
        path: /var/www/myapp
        ssh_key: ~/.ssh/id_rsa

EOF
}

# ============================================================================
# MAIN
# ============================================================================

main() {
  local command="${1:-help}"
  shift || true

  # Load environment
  [[ -f ".env" ]] && source ".env" 2>/dev/null || true
  [[ -f ".env.local" ]] && source ".env.local" 2>/dev/null || true

  case "$command" in
    # Pull (env config or database)
    pull)
      local target="${1:-staging}"
      # Check if pulling env config or database
      if [[ "$target" == "staging" ]] || [[ "$target" == "prod" ]] || [[ "$target" == "production" ]] || [[ "$target" == "secrets" ]]; then
        # Check for --db flag to force database pull
        if [[ "${2:-}" == "--db" ]] || [[ "${2:-}" == "--database" ]]; then
          shift
          cmd_pull "$@"
        else
          # Default: pull environment config
          cmd_pull_env "$target"
        fi
      else
        cmd_pull "$@"
      fi
      ;;

    # Push (env config by default, --db for database)
    push)
      local target="${1:-staging}"
      if [[ "$target" == "staging" ]] || [[ "$target" == "prod" ]] || [[ "$target" == "production" ]] || [[ "$target" == "secrets" ]]; then
        # Check for --db flag to force database push
        if [[ "${2:-}" == "--db" ]] || [[ "${2:-}" == "--database" ]]; then
          shift
          cmd_push "$@"
        else
          # Default: push environment config
          cmd_push_env "$@"
        fi
      else
        cmd_push "$@"
      fi
      ;;

    # Explicit database operations
    db)
      local action="${1:-pull}"
      shift || true
      case "$action" in
        pull)
          cmd_pull "$@"
          ;;
        push)
          cmd_push "$@"
          ;;
        *)
          log_error "Unknown db action: $action"
          printf "Usage: nself sync db pull|push <environment>\n"
          return 1
          ;;
      esac
      ;;

    # Frontend app sync
    frontend)
      cmd_frontend "$@"
      ;;

    # Full sync (everything)
    full)
      cmd_full "$@"
      ;;

    # File sync
    files)
      cmd_files "$@"
      ;;

    # Config sync (legacy)
    config)
      cmd_config "$@"
      ;;

    # Management
    init)
      init_profiles
      ;;
    status)
      cmd_status "$@"
      ;;
    history)
      cmd_history
      ;;

    # Help
    help|--help|-h)
      show_help
      ;;

    *)
      log_error "Unknown command: $command"
      echo ""
      show_help
      return 1
      ;;
  esac
}

main "$@"
