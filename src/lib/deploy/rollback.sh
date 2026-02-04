#!/usr/bin/env bash
# rollback.sh - Automated health-check based rollback system
# Part of nself v0.9.8 - Production Features

set -euo pipefail

# Source utilities
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/../utils/cli-output.sh" 2>/dev/null || true
source "${SCRIPT_DIR}/../utils/env.sh" 2>/dev/null || true

# Rollback configuration
ROLLBACK_DIR="${ROLLBACK_DIR:-.nself/rollback}"
ROLLBACK_RETENTION="${ROLLBACK_RETENTION:-3}"
HEALTH_CHECK_TIMEOUT="${HEALTH_CHECK_TIMEOUT:-120}"
HEALTH_CHECK_INTERVAL="${HEALTH_CHECK_INTERVAL:-5}"
HEALTH_CHECK_REQUIRED_PERCENT="${HEALTH_CHECK_REQUIRED_PERCENT:-80}"

# =============================================================================
# DEPLOYMENT SNAPSHOT
# =============================================================================

# Create deployment snapshot before deploying
create_deployment_snapshot() {
  local snapshot_name="${1:-deployment_$(date +%Y%m%d_%H%M%S)}"

  mkdir -p "$ROLLBACK_DIR"

  cli_info "Creating deployment snapshot: $snapshot_name"

  local snapshot_dir="$ROLLBACK_DIR/$snapshot_name"
  mkdir -p "$snapshot_dir"

  # Save current docker-compose.yml
  if [[ -f "docker-compose.yml" ]]; then
    cp docker-compose.yml "$snapshot_dir/docker-compose.yml"
  fi

  # Save environment files
  for env_file in .env .env.production .env.local; do
    [[ -f "$env_file" ]] && cp "$env_file" "$snapshot_dir/"
  done

  # Save nginx configuration
  if [[ -d "./nginx" ]]; then
    cp -r ./nginx "$snapshot_dir/"
  fi

  # Save service versions
  docker ps --format "{{.Names}}:{{.Image}}" >"$snapshot_dir/running_containers.txt" 2>/dev/null || true

  # Save metadata
  cat >"$snapshot_dir/metadata.json" <<EOF
{
  "snapshot_name": "$snapshot_name",
  "created_at": "$(date -Iseconds)",
  "git_commit": "$(git rev-parse HEAD 2>/dev/null || echo 'unknown')",
  "git_branch": "$(git branch --show-current 2>/dev/null || echo 'unknown')"
}
EOF

  echo "$snapshot_name"
}

# =============================================================================
# HEALTH CHECKS
# =============================================================================

# Run comprehensive health checks
run_health_checks() {
  local timeout="${1:-$HEALTH_CHECK_TIMEOUT}"
  local required_percent="${2:-$HEALTH_CHECK_REQUIRED_PERCENT}"

  cli_info "Running health checks (timeout: ${timeout}s, required: ${required_percent}%)"

  load_env_with_priority 2>/dev/null || true
  local project_name="${PROJECT_NAME:-nself}"

  # Get list of services
  local services=()
  while IFS= read -r service; do
    services+=("$service")
  done < <(docker ps --filter "name=${project_name}_" --format "{{.Names}}" 2>/dev/null)

  if [[ ${#services[@]} -eq 0 ]]; then
    cli_error "No services found running"
    return 1
  fi

  local start_time=$(date +%s)
  local healthy_count=0
  local total_count=${#services[@]}

  while true; do
    healthy_count=0

    for service in "${services[@]}"; do
      # Check Docker health status
      local health=$(docker inspect --format='{{.State.Health.Status}}' "$service" 2>/dev/null || echo "none")

      case "$health" in
        healthy)
          healthy_count=$((healthy_count + 1))
          ;;
        none)
          # No healthcheck defined, check if running
          if docker inspect --format='{{.State.Running}}' "$service" 2>/dev/null | grep -q "true"; then
            healthy_count=$((healthy_count + 1))
          fi
          ;;
      esac
    done

    local healthy_percent=$((healthy_count * 100 / total_count))
    local elapsed=$(($(date +%s) - start_time))

    printf "Health: %d/%d (%d%%) - Elapsed: %ds\r" \
      "$healthy_count" "$total_count" "$healthy_percent" "$elapsed"

    # Check if we've met the threshold
    if [[ $healthy_percent -ge $required_percent ]]; then
      printf "\n"
      cli_success "Health checks passed: $healthy_count/$total_count services healthy ($healthy_percent%)"
      return 0
    fi

    # Check timeout
    if [[ $elapsed -ge $timeout ]]; then
      printf "\n"
      cli_error "Health check timeout: Only $healthy_count/$total_count services healthy ($healthy_percent%)"
      return 1
    fi

    sleep "$HEALTH_CHECK_INTERVAL"
  done
}

# =============================================================================
# ROLLBACK OPERATIONS
# =============================================================================

# Rollback to a specific snapshot
rollback_to_snapshot() {
  local snapshot_name="$1"
  local snapshot_dir="$ROLLBACK_DIR/$snapshot_name"

  if [[ ! -d "$snapshot_dir" ]]; then
    cli_error "Snapshot not found: $snapshot_name"
    return 1
  fi

  cli_warning "Rolling back to snapshot: $snapshot_name"

  # Stop current services
  cli_info "Stopping current services..."
  docker compose down 2>/dev/null || true

  # Restore files
  cli_info "Restoring configuration files..."

  if [[ -f "$snapshot_dir/docker-compose.yml" ]]; then
    cp "$snapshot_dir/docker-compose.yml" docker-compose.yml
  fi

  for env_file in .env .env.production .env.local; do
    if [[ -f "$snapshot_dir/$env_file" ]]; then
      cp "$snapshot_dir/$env_file" "$env_file"
    fi
  done

  if [[ -d "$snapshot_dir/nginx" ]]; then
    rm -rf ./nginx
    cp -r "$snapshot_dir/nginx" ./nginx
  fi

  # Restart services with old configuration
  cli_info "Starting services from snapshot..."
  docker compose up -d

  # Wait for services to stabilize
  sleep 10

  # Verify health
  if run_health_checks 60 80; then
    cli_success "Rollback successful!"
    return 0
  else
    cli_error "Rollback completed but health checks failed"
    return 1
  fi
}

# Rollback to previous deployment
rollback_to_previous() {
  local snapshots=($(ls -1t "$ROLLBACK_DIR" 2>/dev/null | head -2))

  if [[ ${#snapshots[@]} -lt 2 ]]; then
    cli_error "No previous deployment found to rollback to"
    return 1
  fi

  # Skip the most recent (current deployment) and use the previous one
  local previous_snapshot="${snapshots[1]}"

  rollback_to_snapshot "$previous_snapshot"
}

# =============================================================================
# AUTOMATED DEPLOYMENT WITH ROLLBACK
# =============================================================================

# Deploy with automatic rollback on failure
deploy_with_rollback() {
  local deployment_command="$1"
  local no_rollback="${2:-false}"

  cli_section "Automated Deployment with Health-Based Rollback"
  printf "\n"

  # Create pre-deployment snapshot
  local snapshot_name=$(create_deployment_snapshot)

  printf "\n"
  cli_info "Executing deployment..."

  # Execute deployment command
  if eval "$deployment_command"; then
    cli_success "Deployment command completed"
  else
    cli_error "Deployment command failed"

    if [[ "$no_rollback" == "true" ]]; then
      cli_warning "Rollback disabled. Manual intervention required."
      return 1
    fi

    cli_warning "Initiating automatic rollback..."
    rollback_to_snapshot "$snapshot_name"
    return 1
  fi

  printf "\n"
  cli_info "Waiting for services to stabilize..."
  sleep 10

  # Run health checks
  printf "\n"
  if run_health_checks "$HEALTH_CHECK_TIMEOUT" "$HEALTH_CHECK_REQUIRED_PERCENT"; then
    printf "\n"
    cli_success "Deployment successful! All health checks passed."

    # Cleanup old snapshots
    cleanup_old_snapshots

    return 0
  else
    printf "\n"
    cli_error "Health checks failed after deployment"

    if [[ "$no_rollback" == "true" ]]; then
      cli_warning "Rollback disabled. Manual intervention required."
      return 1
    fi

    cli_warning "Initiating automatic rollback..."
    rollback_to_snapshot "$snapshot_name"
    return 1
  fi
}

# =============================================================================
# SNAPSHOT MANAGEMENT
# =============================================================================

# List available snapshots
list_snapshots() {
  if [[ ! -d "$ROLLBACK_DIR" ]] || [[ -z "$(ls -A "$ROLLBACK_DIR" 2>/dev/null)" ]]; then
    cli_info "No deployment snapshots found"
    return 0
  fi

  cli_section "Available Deployment Snapshots"
  printf "\n"
  printf "  %-30s %-20s %-15s\n" "Snapshot" "Created" "Git Commit"
  printf "  %-30s %-20s %-15s\n" "--------" "-------" "----------"

  for snapshot_dir in "$ROLLBACK_DIR"/*; do
    if [[ -d "$snapshot_dir" ]]; then
      local snapshot_name=$(basename "$snapshot_dir")
      local created="Unknown"
      local git_commit="Unknown"

      if [[ -f "$snapshot_dir/metadata.json" ]]; then
        created=$(grep -o '"created_at": *"[^"]*"' "$snapshot_dir/metadata.json" | sed 's/"created_at": *"\([^"]*\)"/\1/' | cut -d'T' -f1,2 | tr 'T' ' ')
        git_commit=$(grep -o '"git_commit": *"[^"]*"' "$snapshot_dir/metadata.json" | sed 's/"git_commit": *"\([^"]*\)"/\1/' | cut -c1-12)
      fi

      printf "  %-30s %-20s %-15s\n" "$snapshot_name" "$created" "$git_commit"
    fi
  done
}

# Cleanup old snapshots (keep only N most recent)
cleanup_old_snapshots() {
  local retention="${1:-$ROLLBACK_RETENTION}"

  cli_info "Cleaning up old snapshots (keeping $retention most recent)"

  local snapshots=($(ls -1t "$ROLLBACK_DIR" 2>/dev/null))
  local count=${#snapshots[@]}

  if [[ $count -le $retention ]]; then
    cli_info "No cleanup needed ($count snapshots, retention: $retention)"
    return 0
  fi

  local to_remove=$((count - retention))
  local removed=0

  for ((i = retention; i < count; i++)); do
    local snapshot="${snapshots[$i]}"
    rm -rf "$ROLLBACK_DIR/$snapshot"
    removed=$((removed + 1))
    cli_info "Removed old snapshot: $snapshot"
  done

  cli_success "Removed $removed old snapshot(s)"
}

# Delete specific snapshot
delete_snapshot() {
  local snapshot_name="$1"

  if [[ -z "$snapshot_name" ]]; then
    cli_error "Snapshot name required"
    return 1
  fi

  if [[ ! -d "$ROLLBACK_DIR/$snapshot_name" ]]; then
    cli_error "Snapshot not found: $snapshot_name"
    return 1
  fi

  rm -rf "$ROLLBACK_DIR/$snapshot_name"
  cli_success "Deleted snapshot: $snapshot_name"
}

# =============================================================================
# EXPORTS
# =============================================================================

export -f create_deployment_snapshot run_health_checks
export -f rollback_to_snapshot rollback_to_previous deploy_with_rollback
export -f list_snapshots cleanup_old_snapshots delete_snapshot

# If run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  case "${1:-help}" in
    create-snapshot)
      create_deployment_snapshot "${2:-}"
      ;;
    rollback)
      if [[ -n "${2:-}" ]]; then
        rollback_to_snapshot "$2"
      else
        rollback_to_previous
      fi
      ;;
    health-check)
      run_health_checks "${2:-120}" "${3:-80}"
      ;;
    list)
      list_snapshots
      ;;
    cleanup)
      cleanup_old_snapshots "${2:-3}"
      ;;
    delete)
      delete_snapshot "$2"
      ;;
    deploy)
      deploy_with_rollback "${2:-docker compose up -d}" "${3:-false}"
      ;;
    *)
      echo "Usage: $0 {create-snapshot|rollback|health-check|list|cleanup|delete|deploy}"
      echo ""
      echo "Commands:"
      echo "  create-snapshot [name]           Create deployment snapshot"
      echo "  rollback [snapshot]              Rollback to snapshot (or previous if not specified)"
      echo "  health-check [timeout] [percent] Run health checks"
      echo "  list                             List available snapshots"
      echo "  cleanup [retention]              Remove old snapshots"
      echo "  delete <snapshot>                Delete specific snapshot"
      echo "  deploy <command> [no-rollback]   Deploy with automatic rollback"
      exit 1
      ;;
  esac
fi
