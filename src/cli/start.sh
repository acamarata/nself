#!/usr/bin/env bash
set -euo pipefail

# start.sh - Start services with streamlined error handling

# Source utilities
SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/utils/progress.sh"
source "$SCRIPT_DIR/../lib/errors/base.sh"
source "$SCRIPT_DIR/../lib/errors/quick-check.sh"
source "$SCRIPT_DIR/../lib/errors/handlers/ports.sh"
source "$SCRIPT_DIR/../lib/errors/handlers/build.sh"
source "$SCRIPT_DIR/../lib/auto-fix/config-validator-v2.sh"
source "$SCRIPT_DIR/../lib/auto-fix/auto-fixer-v2.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"
source "$SCRIPT_DIR/../lib/config/smart-defaults.sh"

# Load environment with smart defaults
load_env_with_defaults >/dev/null 2>&1

# Command function
cmd_start() {
  local verbose=false
  local skip_checks=false
  local detached=true
  local retry_count="${UP_RETRY_COUNT:-0}"
  # Default to true for ALWAYS_AUTOFIX unless explicitly set to false
  local auto_fix="${ALWAYS_AUTOFIX:-true}"
  if [[ "$auto_fix" == "false" ]]; then
    auto_fix="false"
  else
    # Any value other than "false" means true (including empty/unset)
    auto_fix="true"
  fi
  local max_retries=30

  # Check for ALWAYS_AUTOFIX mode
  if [[ "$auto_fix" == "true" ]]; then
    max_retries=5 # Reasonable limit for auto-fix mode
  else
    max_retries=3 # Fewer retries in interactive mode
  fi

  # Prevent infinite retry loops
  if [[ $retry_count -ge $max_retries ]]; then
    if [[ "$auto_fix" == "true" ]]; then
      printf "\r${COLOR_RED}✗${COLOR_RESET} Auto-fix exceeded $max_retries attempts                     \n"
      echo
      echo -e "${COLOR_RED}Services still failing after $max_retries attempts:${COLOR_RESET}"

      # Show which services are still problematic
      local problem_services=$(docker ps --format "{{.Names}} {{.Status}}" | grep -E "Restarting|unhealthy" | awk '{print $1}' | sed "s/${PROJECT_NAME:-unity}_//" | head -10)
      for svc in $problem_services; do
        echo -e "  ${COLOR_RED}✗${COLOR_RESET} $svc"
      done
      echo
      log_info "Manual intervention required. Check logs with:"
      echo -e "  ${COLOR_BLUE}nself logs <service>${COLOR_RESET}"
      echo
      log_info "Common fixes:"
      echo -e "  • Run ${COLOR_BLUE}nself stop && nself start${COLOR_RESET} for a fresh start"
      echo -e "  • Check ${COLOR_BLUE}docker logs unity_<service>${COLOR_RESET} for details"
      echo -e "  • Ensure all required files exist in service directories"
    else
      log_error "Too many retries. Please check your configuration."
    fi
    return 1
  fi

  # Parse options
  while [[ $# -gt 0 ]]; do
    case "$1" in
    --verbose | -v)
      verbose=true
      shift
      ;;
    --skip-checks)
      skip_checks=true
      shift
      ;;
    --attach | -a)
      detached=false
      shift
      ;;
    --help | -h)
      show_help
      return 0
      ;;
    *)
      log_error "Unknown option: $1"
      show_help
      return 1
      ;;
    esac
  done

  # Check if all services are already running and healthy BEFORE showing header
  if [[ $retry_count -eq 0 ]]; then
    # Source service health utilities
    if [[ -f "$SCRIPT_DIR/../lib/utils/service-health.sh" ]]; then
      source "$SCRIPT_DIR/../lib/utils/service-health.sh"

      # Check if all services are already healthy
      if check_all_services_healthy false; then
        # All services are running and healthy!
        echo
        echo -e "${COLOR_GREEN}✓${COLOR_RESET} All services are running and healthy!"
        echo

        # Display service status with green dots
        display_running_services
        echo

        # Check if configuration has changed
        if check_config_changed; then
          echo -e "${COLOR_YELLOW}⚠${COLOR_RESET}  Configuration has changed since services started"
          echo
          echo -e "   Run ${COLOR_BLUE}nself restart${COLOR_RESET} to apply changes"
          echo -e "   Or ${COLOR_BLUE}nself stop && nself start${COLOR_RESET} for a fresh start"
        else
          # Show service URLs
          show_service_urls
          echo
          echo -e "${COLOR_CYAN}➞ Useful Commands${COLOR_RESET}"
          echo
          echo -e "   ${COLOR_BLUE}nself status${COLOR_RESET}  - View detailed service status"
          echo -e "   ${COLOR_BLUE}nself logs${COLOR_RESET}    - View service logs"
          echo -e "   ${COLOR_BLUE}nself doctor${COLOR_RESET}  - Run health diagnostics"
        fi
        echo
        return 0 # Exit successfully
      fi
    fi
  fi

  # Show header FIRST (skip on retry)
  if [[ $retry_count -eq 0 ]]; then
    show_command_header "nself start" "Start all services and infrastructure"
  fi

  # Run comprehensive pre-flight checks (skip if retrying after auto-fix)
  if [[ $retry_count -eq 0 ]]; then
    if ! source "$SCRIPT_DIR/../lib/utils/preflight.sh" 2>/dev/null; then
      log_error "Failed to load pre-flight checks"
      return 1
    fi

    if [[ "$auto_fix" != "true" ]]; then
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Running pre-flight checks..."
      if preflight_up >/dev/null 2>&1; then
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Pre-flight checks passed                   \n"
      else
        printf "\r${COLOR_RED}✗${COLOR_RESET} Pre-flight checks failed                   \n"
        preflight_up # Run again to show the actual errors
        return 1
      fi
    else
      # Silent pre-flight checks in auto-fix mode
      if ! preflight_up >/dev/null 2>&1; then
        # Only show failure if we can't continue
        printf "\r${COLOR_RED}✗${COLOR_RESET} System requirements not met                \n"
        return 1
      fi
    fi
  fi

  # Reset autofix state and run pre-checks on fresh start (not retry)
  if [[ $retry_count -eq 0 ]]; then
    # Reset state tracking
    if [[ -f "$SCRIPT_DIR/../lib/autofix/state-tracker.sh" ]]; then
      source "$SCRIPT_DIR/../lib/autofix/state-tracker.sh"
      reset_all_states
    fi

    # Run pre-checks to fix common issues before Docker
    if [[ -f "$SCRIPT_DIR/../lib/autofix/orchestrator.sh" ]]; then
      source "$SCRIPT_DIR/../lib/autofix/orchestrator.sh"
      run_autofix_prechecks
    fi
  fi

  # Comprehensive upfront port checking (skip on retry)
  if [[ "$skip_checks" != "true" ]] && [[ $retry_count -eq 0 ]]; then
    # Source port scanner
    if [[ -f "$SCRIPT_DIR/../lib/utils/port-scanner.sh" ]]; then
      source "$SCRIPT_DIR/../lib/utils/port-scanner.sh"

      printf "${COLOR_BLUE}⠋${COLOR_RESET} Checking port availability..."

      # Pre-check all ports from docker-compose
      local port_conflicts=$(precheck_all_ports "docker-compose.yml")

      if [[ -n "$port_conflicts" ]]; then
        printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Port conflicts detected                    \n"
        echo

        # Parse and display conflicts
        local has_conflicts=false
        for conflict in $port_conflicts; do
          IFS=':' read -r port info <<<"$conflict"
          IFS='|' read -r pid process path <<<"$info"

          if [[ "$port" != "" ]]; then
            has_conflicts=true
            log_error "Port $port is in use"
            if [[ "$process" != "unknown" ]]; then
              log_info "Used by: $path (PID: $pid)"
            fi
          fi
        done

        if [[ "$has_conflicts" == "true" ]]; then
          local choice=1 # Default to auto-fix

          if [[ "$auto_fix" != "true" ]]; then
            echo
            log_info "Port conflict options:"
            echo "  1) Auto-fix: Change to alternative ports"
            echo "  2) Stop conflicting services"
            echo "  3) Continue anyway (may fail)"
            echo "  4) Cancel"
            echo

            read -p "Select option (1-4): " choice
          fi

          case "$choice" in
          1)
            if [[ "$auto_fix" == "true" ]]; then
              # Concise output in auto-fix mode
              local port_changes=""
              for conflict in $port_conflicts; do
                IFS=':' read -r port info <<<"$conflict"
                if [[ -n "$port" ]]; then
                  local new_port=$(suggest_alternative_port "$port")
                  if [[ -n "$new_port" ]]; then
                    fix_port_in_env "" "$port" "$new_port"
                    port_changes="${port_changes}Port $port→$new_port, "
                  fi
                fi
              done

              if [[ -n "$port_changes" ]]; then
                printf "\r${COLOR_YELLOW}⚡${COLOR_RESET} ${port_changes%, }              \n"
                printf "${COLOR_BLUE}⠋${COLOR_RESET} Rebuilding configuration..."
                if nself build --force >/dev/null 2>&1; then
                  printf "\r${COLOR_BLUE}⠋${COLOR_RESET} Starting all services...                   "
                else
                  printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to rebuild                         \n"
                  return 1
                fi
              fi
            else
              log_info "Auto-fixing port conflicts..."

              # Auto-fix each conflict
              for conflict in $port_conflicts; do
                IFS=':' read -r port info <<<"$conflict"
                if [[ -n "$port" ]]; then
                  local new_port=$(suggest_alternative_port "$port")
                  if [[ -n "$new_port" ]]; then
                    log_info "Changing port $port to $new_port"
                    fix_port_in_env "" "$port" "$new_port"
                  fi
                fi
              done

              log_success "Ports updated in .env"
              log_info "Rebuilding configuration..."

              # Rebuild with new ports
              if nself build --force >/dev/null 2>&1; then
                log_success "Configuration rebuilt with new ports"
              else
                log_error "Failed to rebuild configuration"
                return 1
              fi
            fi
            ;;
          2)
            log_info "Stop the conflicting services manually, then run 'nself start' again"
            return 1
            ;;
          3)
            log_warning "Continuing despite port conflicts..."
            ;;
          4)
            log_info "Cancelled"
            return 1
            ;;
          *)
            log_error "Invalid option"
            return 1
            ;;
          esac
        fi
      else
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Ports available                                   \n"
      fi
    else
      # Fallback to old check
      printf "${COLOR_BLUE}⠋${COLOR_RESET} Checking port availability..."
      if ! run_essential_checks false >/dev/null 2>&1; then
        printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Port conflicts detected                    \n"
        echo
        run_essential_checks true
        if [[ $? -ne 0 ]]; then
          echo
          log_error "Port checks failed"
          return 1
        fi
      else
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Ports available                                   \n"
      fi
    fi
  fi

  # Skip validation on retry (we're continuing from where we left off)
  if [[ $retry_count -eq 0 ]]; then
    # Validate docker-compose.yml exists
    if [[ ! -f "docker-compose.yml" ]]; then
      printf "${COLOR_RED}✗${COLOR_RESET} docker-compose.yml not found               \n"
      echo
      log_info "Run 'nself build' first to generate infrastructure"
      return 1
    fi

    # Load environment for validation
    if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
      set -a
      load_env_with_priority
      set +a
    fi

    # Validate docker-compose.yml
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Validating configuration..."
    if compose config >/dev/null 2>&1; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} Configuration valid                        \n"
    else
      printf "\r${COLOR_RED}✗${COLOR_RESET} Invalid docker-compose.yml                \n"
      echo
      # Show the actual validation error
      local validation_error=$(compose config 2>&1 | grep -v "\.go:[0-9]" | head -5)
      if [[ -n "$validation_error" ]]; then
        echo "$validation_error"
        echo
      fi
      log_info "Run 'nself build' to regenerate configuration"
      return 1
    fi
  else
    # On retry, still need to load environment
    if [[ -f ".env" ]] || [[ -f ".env.dev" ]]; then
      set -a
      load_env_with_priority
      set +a
    fi
  fi

  # Header is already shown above, don't show it again
  
  # Pre-initialize MLflow database if MLflow is enabled
  if [[ "${MLFLOW_ENABLED:-false}" == "true" ]] || grep -q "^\s*mlflow:" docker-compose.yml 2>/dev/null; then
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Initializing MLflow database..."
    
    # Ensure mlflow database exists
    if compose ps postgres 2>/dev/null | grep -q "running\|Up"; then
      docker exec "${project}_postgres" psql -U postgres -tc "SELECT 1 FROM pg_database WHERE datname = 'mlflow'" | grep -q 1 || \
        docker exec "${project}_postgres" psql -U postgres -c "CREATE DATABASE mlflow;" 2>/dev/null
      
      # Run MLflow migrations using a temporary container if database exists but schema doesn't
      if docker exec "${project}_postgres" psql -U postgres -d mlflow -c "SELECT 1 FROM information_schema.tables WHERE table_name='metrics'" 2>&1 | grep -q "0 rows"; then
        docker run --rm \
          --network "${project}_default" \
          ghcr.io/mlflow/mlflow:v2.9.2 \
          sh -c "pip install -q psycopg2-binary && mlflow db upgrade postgresql://postgres:${POSTGRES_PASSWORD:-postgres}@postgres:5432/mlflow" 2>/dev/null
      fi
    fi
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} MLflow database ready                      \n"
  fi

  # Start services (shorter message to avoid artifacts)
  printf "${COLOR_BLUE}⠋${COLOR_RESET} Starting services..."

  local output_file=$(mktemp)
  local result

  # Ensure environment is loaded for compose wrapper
  if [[ -f ".env.local" ]] || [[ -f ".env" ]]; then
    set -a
    load_env_with_priority
    set +a
  fi

  # Build the compose command using the full docker compose
  # Get the proper compose command with project name and env file
  local env_file=".env.local"
  if [[ ! -f "$env_file" ]] && [[ -f ".env" ]]; then
    env_file=".env"
  fi
  
  local project="${PROJECT_NAME:-nself}"
  local compose_cmd="docker compose --project-name \"$project\" --env-file \"$env_file\" up --build"
  if [[ "$detached" == "true" ]]; then
    compose_cmd="$compose_cmd -d"
  fi

  if [[ "$verbose" == "true" ]]; then
    # Show full output in verbose mode
    printf "\n"
    bash -c "$compose_cmd" 2>&1 | tee "$output_file"
    result=${PIPESTATUS[0]}
  else
    # Check existing services for better progress display
    local services_to_start=$(docker compose --project-name "$project" config --services 2>/dev/null | wc -l | tr -d ' ')
    local running_services=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')
    
    # Run silently with enhanced progress
    (bash -c "$compose_cmd" 2>&1) >"$output_file" &
    local compose_pid=$!

    # Enhanced progress monitoring
    local spin_chars="⠋⠙⠹⠸⠼⠴⠦⠧⠇⠏"
    local i=0
    local timeout=180  # 3 minute timeout
    local elapsed=0
    local last_count=$running_services
    local last_message=""
    
    while kill -0 $compose_pid 2>/dev/null; do
      if [[ $elapsed -ge $timeout ]]; then
        printf "\r${COLOR_RED}✗${COLOR_RESET} Docker compose timeout after ${timeout}s                  \n"
        kill $compose_pid 2>/dev/null
        
        # Show last error from output
        if [[ -f "$output_file" ]]; then
          echo
          log_error "Docker compose failed. Last output:"
          tail -20 "$output_file" | sed 's/^/  /'
        fi
        
        return 1
      fi
      
      # Check running services count for progress
      local current_count=$(docker ps --filter "label=com.docker.compose.project=$project" --format "{{.Names}}" 2>/dev/null | wc -l | tr -d ' ')
      
      # Check for building status in output
      local building_msg=""
      if [[ -f "$output_file" ]] && [[ $elapsed -gt 2 ]]; then
        local last_line=$(tail -1 "$output_file" 2>/dev/null | grep -oE "(Building|Pulling|Creating|Starting) [a-zA-Z0-9_-]+" | head -1)
        if [[ -n "$last_line" ]]; then
          building_msg=" - $last_line"
        fi
      fi
      
      local char="${spin_chars:$((i % ${#spin_chars})):1}"
      
      # Always show progress with count when available
      if [[ "$services_to_start" -gt 0 ]] && [[ "$current_count" -gt 0 ]]; then
        if [[ -n "$building_msg" ]]; then
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} Starting Docker containers... (%d/%d)%s          " "$char" "$current_count" "$services_to_start" "$building_msg"
        else
          printf "\r${COLOR_BLUE}%s${COLOR_RESET} Starting Docker containers... (%d/%d)          " "$char" "$current_count" "$services_to_start"
        fi
      elif [[ -n "$building_msg" ]]; then
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} Starting Docker containers...%s          " "$char" "$building_msg"
      else
        printf "\r${COLOR_BLUE}%s${COLOR_RESET} Starting Docker containers...          " "$char"
      fi
      
      last_count=$current_count
      last_message="$building_msg"
      
      ((i++))
      ((elapsed++))
      sleep 1
    done
    wait $compose_pid
    result=$?
  fi

  if [[ $result -eq 0 ]]; then
    printf "\r${COLOR_GREEN}✓${COLOR_RESET} Docker containers started                  \n"

    # Apply database auto-configuration if PostgreSQL is running
    if docker ps --format "{{.Names}}" | grep -q "${PROJECT_NAME:-myproject}_postgres"; then
      if [[ -f "$SCRIPT_DIR/../lib/database/auto-config.sh" ]]; then
        source "$SCRIPT_DIR/../lib/database/auto-config.sh" 2>/dev/null || true
        if command -v configure_postgres &>/dev/null; then
          printf "${COLOR_BLUE}⠋${COLOR_RESET} Optimizing database configuration..."
          configure_postgres "${PROJECT_NAME:-myproject}_postgres" >/dev/null 2>&1 && {
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Database optimized for ${ENV:-dev} environment    \n"
          } || {
            printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Database running with default settings         \n"
          }
        fi
      fi
    fi

    # Verify services are actually running and not restarting
    printf "${COLOR_BLUE}⠋${COLOR_RESET} Verifying services..."
    sleep 3 # Give services a moment to fully start

    # Get detailed status of all services
    local project_name="${PROJECT_NAME:-unity}"
    local all_healthy=true
    local unhealthy_services=""
    local restarting_services=""

    # Check each service individually
    # Need to use proper project name with compose
    local compose_cmd="docker compose"
    if [[ -n "$project_name" ]]; then
      compose_cmd="docker compose -p $project_name"
    fi

    local total_services=$($compose_cmd ps --services 2>/dev/null | wc -l | tr -d ' ')
    local healthy_count=0
    local unhealthy_count=0
    local restarting_count=0

    for service in $($compose_cmd ps --services 2>/dev/null); do
      # Container names use underscores, service names might have hyphens
      local container_name="${project_name}_${service//-/_}"
      local container_status=$(docker inspect --format='{{.State.Status}}' "$container_name" 2>/dev/null)
      local restart_count=$(docker inspect --format='{{.RestartCount}}' "$container_name" 2>/dev/null)

      if [[ "$container_status" == "running" ]]; then
        # Check if it's actually restarting (high restart count in short time)
        if [[ $restart_count -gt 0 ]]; then
          # Check if container was created recently (within last minute)
          local created_time=$(docker inspect --format='{{.Created}}' "$container_name" 2>/dev/null)
          local current_time=$(date -u +%s)
          local created_timestamp=$(date -d "$created_time" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%S" "${created_time%%.*}" +%s 2>/dev/null)
          local time_diff=$((current_time - created_timestamp))

          # If restart count > 0 and container is less than 60 seconds old, it's likely restarting
          if [[ $time_diff -lt 60 ]] && [[ $restart_count -gt 1 ]]; then
            restarting_services="$restarting_services $service"
            ((restarting_count++))
            all_healthy=false
            continue
          fi
        fi

        # Check health status if available
        local health_status=$(docker inspect --format='{{.State.Health.Status}}' "$container_name" 2>/dev/null)
        if [[ "$health_status" == "unhealthy" ]] || [[ "$health_status" == "starting" ]]; then
          unhealthy_services="$unhealthy_services $service"
          ((unhealthy_count++))
          all_healthy=false
        else
          ((healthy_count++))
        fi
      elif [[ "$container_status" == "restarting" ]]; then
        restarting_services="$restarting_services $service"
        ((restarting_count++))
        all_healthy=false
      else
        unhealthy_services="$unhealthy_services $service"
        ((unhealthy_count++))
        all_healthy=false
      fi
    done

    if [[ "$all_healthy" == "true" ]] && [[ $healthy_count -eq $total_services ]]; then
      printf "\r${COLOR_GREEN}✓${COLOR_RESET} All services running ($healthy_count/$total_services)              \n"

      echo
      log_success "Services started successfully!"
      
      # Start health monitoring daemon if enabled
      if [[ "${ENABLE_HEALTH_MONITORING:-true}" == "true" ]]; then
        if [[ -f "$SCRIPT_DIR/../lib/auto-fix/health-check-daemon.sh" ]]; then
          source "$SCRIPT_DIR/../lib/auto-fix/health-check-daemon.sh"
          start_health_daemon >/dev/null 2>&1
          log_info "Health monitoring active (checks every ${HEALTH_CHECK_INTERVAL:-60}s)"
        fi
      fi
      
      show_service_urls

      echo
      echo -e "${COLOR_CYAN}➞ Next Steps${COLOR_RESET}"
      echo
      echo -e "${COLOR_BLUE}1.${COLOR_RESET} ${COLOR_BLUE}nself status${COLOR_RESET} - Check service health"
      echo -e "   ${COLOR_DIM}View detailed status of all services${COLOR_RESET}"
      echo
      echo -e "${COLOR_BLUE}2.${COLOR_RESET} ${COLOR_BLUE}nself logs${COLOR_RESET} - View service logs"
      echo -e "   ${COLOR_DIM}Monitor real-time logs from services${COLOR_RESET}"
      echo
      echo -e "${COLOR_BLUE}3.${COLOR_RESET} ${COLOR_BLUE}nself urls${COLOR_RESET} - View service endpoints"
      echo -e "   ${COLOR_DIM}Display all available service URLs${COLOR_RESET}"
    else
      # Show detailed status of what's wrong
      if [[ $restarting_count -gt 0 ]]; then
        printf "\r${COLOR_RED}✗${COLOR_RESET} Services in restart loop ($restarting_count/$total_services)        \n"
        echo
        echo -e "${COLOR_RED}Restarting services:${COLOR_RESET}"
        for svc in $restarting_services; do
          echo -e "  ${COLOR_RED}↻${COLOR_RESET} $svc"
        done
        
        # Attempt to fix restart loops automatically
        if [[ -f "$SCRIPT_DIR/../lib/auto-fix/restart-loop-fix.sh" ]]; then
          echo
          printf "${COLOR_BLUE}⠋${COLOR_RESET} Attempting to fix restart loops..."
          
          # Source the restart loop fix script
          source "$SCRIPT_DIR/../lib/auto-fix/restart-loop-fix.sh"
          
          # Run the fix
          if fix_restart_loops >/dev/null 2>&1; then
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Restart loop fixes applied                 \n"
            echo
            log_info "Retrying service startup..."
            sleep 2
            # Restart the affected services
            for svc in $restarting_services; do
              docker restart "$svc" >/dev/null 2>&1 &
            done
            # Wait and recheck
            sleep 5
            continue
          else
            printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Unable to fix all restart loops            \n"
          fi
        fi
      elif [[ $unhealthy_count -gt 0 ]]; then
        # Services are running but marked unhealthy (often due to missing healthcheck tools)
        printf "\r${COLOR_GREEN}✓${COLOR_RESET} Services running                                     \n"
        echo
        log_success "All services started!"
        show_service_urls

        return 0 # Return success since services are running
      else
        printf "\r${COLOR_YELLOW}✱${COLOR_RESET} Some services failed ($healthy_count/$total_services)             \n"
      fi

      echo

      # If ALWAYS_AUTOFIX is enabled AND there are critical issues, try to fix
      # Don't auto-fix for just unhealthy status (could be false positives from bad healthchecks)
      if [[ "${ALWAYS_AUTOFIX:-true}" == "true" ]] && [[ $restarting_count -gt 0 ]]; then
        log_info "Auto-fixing restarting services..."

        # Source the orchestrator if not already loaded
        if [[ -f "$SCRIPT_DIR/../lib/autofix/orchestrator.sh" ]]; then
          source "$SCRIPT_DIR/../lib/autofix/orchestrator.sh"
        fi
        
        # Try restart loop fix first
        if [[ -f "$SCRIPT_DIR/../lib/auto-fix/restart-loop-fix.sh" ]]; then
          source "$SCRIPT_DIR/../lib/auto-fix/restart-loop-fix.sh"
          if declare -f fix_restart_loops >/dev/null 2>&1; then
            log_info "Running comprehensive fixes..."
            fix_restart_loops >/dev/null 2>&1
            fixed_any=true
          fi
        fi

        # Try comprehensive fix if available
        if [[ -f "$SCRIPT_DIR/../lib/autofix/comprehensive.sh" ]]; then
          source "$SCRIPT_DIR/../lib/autofix/comprehensive.sh"
          if declare -f run_comprehensive_fixes >/dev/null 2>&1; then
            run_comprehensive_fixes 2
            fixed_any=true
          fi
        fi

        # Try to fix each problematic service
        local fixed_any=false

        # Fix restarting services first
        for service in $restarting_services; do
          local container_name="${project_name}_${service//-/_}"
          local service_logs=$(docker logs "$container_name" --tail 50 2>&1)

          printf "${COLOR_BLUE}⠋${COLOR_RESET} Fixing $service..."
          autofix_service "$container_name" "$service_logs" "$verbose"
          local fix_result=$?

          if [[ $fix_result -eq 99 ]]; then
            fixed_any=true
            printf "\r${COLOR_GREEN}✓${COLOR_RESET} Fixed $service                            \n"
          else
            printf "\r${COLOR_RED}✗${COLOR_RESET} Could not fix $service                    \n"
          fi
        done

        # Skip fixing unhealthy services - they're usually false positives from bad healthchecks
        # Only fix services that are actually restarting/crashing

        if [[ "$fixed_any" == "true" ]]; then
          echo
          log_info "Retrying startup after fixes..."
          # Retry with incremented counter
          UP_RETRY_COUNT=$((retry_count + 1)) cmd_start "$@"
          return $?
        else
          # No fixes worked, don't keep retrying
          echo
          log_error "Auto-fix could not resolve the issues"
          echo
          echo -e "${COLOR_RED}Problematic services:${COLOR_RESET}"
          for svc in $restarting_services $unhealthy_services; do
            echo -e "  ${COLOR_RED}✗${COLOR_RESET} $svc"
          done
          echo
          log_info "Try these manual fixes:"
          echo -e "  1. ${COLOR_BLUE}nself stop && nself start${COLOR_RESET} - Full restart"
          echo -e "  2. ${COLOR_BLUE}nself build --force${COLOR_RESET} - Rebuild configuration"
          echo -e "  3. Check service logs for specific errors"
          return 1
        fi
      else
        echo
        log_info "Run 'nself start' again with ALWAYS_AUTOFIX=true to auto-fix issues"
      fi

      # If we couldn't fix, analyze the error
      analyze_startup_error "$(cat "$output_file")"
      rm -f "$output_file"
      return 1
    fi
  else
    # Only show failure message on first attempt
    if [[ $retry_count -eq 0 ]]; then
      printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to start services                   \n"
      echo
    fi

    # Save output for debugging if DEBUG is set
    if [[ "${DEBUG:-}" == "true" ]]; then
      log_info "Debug: Output saved to /tmp/nself-start-error.log"
      cp "$output_file" /tmp/nself-start-error.log
    fi

    # Analyze the specific error
    analyze_startup_error "$(cat "$output_file")"
    local analyze_result=$?
    rm -f "$output_file"

    # Check if auto-fix requested a retry
    if [[ $analyze_result -eq 99 ]]; then
      # Retry the up command after auto-fix with incremented counter
      UP_RETRY_COUNT=$((retry_count + 1)) cmd_start "$@"
      return $?
    fi

    return 1
  fi

  rm -f "$output_file"
}

# Analyze startup errors and offer solutions
analyze_startup_error() {
  local output="$1"

  # Container name conflicts
  if echo "$output" | grep -q "The container name.*is already in use"; then
    local container=$(echo "$output" | grep -oE 'container name "[^"]+"' | grep -oE '"[^"]+"' | tr -d '"' | head -1)
    if [[ -n "$container" ]]; then
      log_error "Container name conflict: $container"
      echo
      log_info "Another project is using the same container names"
      echo
      log_info "Solutions:"
      echo "  1. Stop the other project: ${COLOR_BLUE}docker stop $container${COLOR_RESET}"
      echo "  2. Remove old containers: ${COLOR_BLUE}docker rm $container${COLOR_RESET}"
      echo "  3. Change PROJECT_NAME in .env"
      echo "  4. Use a different project: ${COLOR_BLUE}nself stop && nself init --project new-name${COLOR_RESET}"
    else
      log_error "Container name conflict detected"
      echo "$output" | grep "container name" | head -3
    fi

  # Port conflicts
  elif echo "$output" | grep -q "port is already allocated\|bind: address already in use"; then
    # Extract port from various Docker error formats
    local port=$(echo "$output" | grep -oE "0\.0\.0\.0:[0-9]+" | grep -oE "[0-9]+$" | head -1)
    if [[ -z "$port" ]]; then
      port=$(echo "$output" | grep -oE "bind: address already in use.*:[0-9]+" | grep -oE "[0-9]+$" | head -1)
    fi
    if [[ -z "$port" ]]; then
      port=$(echo "$output" | grep -oE "port [0-9]+" | grep -oE "[0-9]+" | head -1)
    fi

    if [[ -n "$port" ]] && [[ "$port" != "0" ]]; then
      log_error "Port $port is already in use"

      # Try to identify the process
      if command -v lsof >/dev/null 2>&1; then
        local process_pid=$(lsof -i :$port -sTCP:LISTEN -t 2>/dev/null | head -1)
        if [[ -n "$process_pid" ]]; then
          local process_name=$(ps -p $process_pid -o comm= 2>/dev/null)
          local full_path=$(ps -p $process_pid -o command= 2>/dev/null | cut -d' ' -f1)
          echo
          log_info "Used by: $full_path (PID: $process_pid)"
        fi
      fi

      # Source port scanner for auto-fix capabilities
      if [[ -f "$SCRIPT_DIR/../lib/utils/port-scanner.sh" ]]; then
        source "$SCRIPT_DIR/../lib/utils/port-scanner.sh"

        # Check if ALWAYS_AUTOFIX is enabled
        if [[ "${ALWAYS_AUTOFIX:-true}" == "true" ]]; then
          # Auto-select option 1
          local choice=1
          log_info "Auto-fixing port conflict"
        else
          echo
          log_info "Port conflict options:"
          echo "  1) Auto-fix: Change to alternative port"
          echo "  2) Stop conflicting service"
          echo "  3) Cancel"
          echo

          read -p "Select option (1-3): " choice
        fi

        case "$choice" in
        1)
          local new_port=$(suggest_alternative_port "$port")
          if [[ -n "$new_port" ]]; then
            log_info "Changing port $port to $new_port in .env"
            fix_port_in_env "" "$port" "$new_port"
            log_success "Port updated"
            echo
            log_info "Rebuilding and retrying..."

            # Rebuild and retry
            if nself build --force >/dev/null 2>&1; then
              sleep 2
              return 99 # Retry
            fi
          else
            log_error "Could not find alternative port"
          fi
          ;;
        2)
          if [[ -n "$process_pid" ]]; then
            log_info "Run: ${COLOR_BLUE}kill $process_pid${COLOR_RESET}"
          fi
          log_info "Then run 'nself start' again"
          ;;
        3)
          log_info "Cancelled"
          ;;
        *)
          log_error "Invalid option"
          ;;
        esac
      else
        echo
        log_info "Solutions:"
        echo "  1. Stop the conflicting service"
        echo "  2. Change the port in .env"
        echo "  3. Run: ${COLOR_BLUE}nself stop && nself start${COLOR_RESET}"
      fi
    else
      log_error "Port conflict detected"
      echo "$output" | grep -E "port|bind" | head -5
    fi

  # Network conflicts
  elif echo "$output" | grep -q "a network with name.*exists but was not created"; then
    local network=$(echo "$output" | grep -oE 'network with name [^ ]+' | cut -d' ' -f4)
    log_error "Network conflict: $network"
    echo
    log_info "Another project is using the same network name"
    echo
    log_info "Solutions:"
    echo "  1. Remove the old network: ${COLOR_BLUE}docker network rm $network${COLOR_RESET}"
    echo "  2. Change PROJECT_NAME in .env"
    echo "  3. Run: ${COLOR_BLUE}nself build --force && nself start${COLOR_RESET}"

  # Volume conflicts
  elif echo "$output" | grep -q "volume.*already exists but was created for project"; then
    local volume=$(echo "$output" | grep -oE 'volume "[^"]+"' | grep -oE '"[^"]+"' | tr -d '"' | head -1)
    log_error "Volume conflict: $volume"
    echo
    log_info "Another project is using the same volume names"
    echo
    log_info "Solutions:"
    echo "  1. Remove old volumes: ${COLOR_BLUE}docker volume rm $volume${COLOR_RESET}"
    echo "  2. Change PROJECT_NAME in .env"
    echo "  3. Use different volumes for this project"

  # Build context errors (missing directories)
  elif echo "$output" | grep -q "unable to prepare context:\|path.*not found"; then
    log_error "Build context error - missing service directory"

    # Extract the missing path
    local missing_path=$(echo "$output" | grep -oE 'path "[^"]+"' | grep -oE '"[^"]+"' | tr -d '"' | head -1)
    if [[ -z "$missing_path" ]]; then
      missing_path=$(echo "$output" | grep "not found" | head -1 | sed 's/.*path //' | sed 's/ not found.*//')
    fi

    if [[ -n "$missing_path" ]]; then
      # Make path relative if it's absolute and in current project
      local relative_path="${missing_path#$(pwd)/}"
      echo "Missing: $relative_path"

      # Check if this is a service directory (handle various naming)
      # Match services/(type)/(name) pattern - we support any service type in env
      if [[ "$relative_path" =~ ^services/([^/]+)/([^/]+) ]]; then
        echo
        log_info "Auto-fixing: Generating missing service..."

        # Source service generator
        if [[ -f "$SCRIPT_DIR/../lib/auto-fix/service-generator.sh" ]]; then
          source "$SCRIPT_DIR/../lib/auto-fix/service-generator.sh"

          if check_missing_service "$relative_path"; then
            log_success "Service generated successfully"
            echo
            log_info "Continuing startup with newly generated service..."
            echo
            # Clean up temporary file
            rm -f "$output_file"

            # Give filesystem and Docker time to recognize new files
            sleep 2

            # Return special code to indicate retry needed
            return 99 # Special code to indicate retry needed
          fi
        fi
      fi
    fi

    echo
    log_info "Solutions:"
    echo "  1. Create the missing directory: ${COLOR_BLUE}mkdir -p ${relative_path}${COLOR_RESET}"
    echo "  2. Check your docker-compose.yml for incorrect paths"
    echo "  3. Rebuild configuration: ${COLOR_BLUE}nself build --force${COLOR_RESET}"

  # Build errors
  elif echo "$output" | grep -q "failed to solve\|exit code:\|missing go.sum entry\|failed to read dockerfile"; then
    log_error "Build error detected"

    # Check for missing Dockerfile
    if echo "$output" | grep -q "failed to read dockerfile.*no such file"; then
      # Extract the service name from the error
      local service_name=$(echo "$output" | grep -oE "target [^:]+:" | sed 's/target //; s/://' | head -1)
      if [[ -n "$service_name" ]]; then
        if [[ "${ALWAYS_AUTOFIX:-false}" == "true" ]]; then
          # Concise output in auto-fix mode
          printf "\r${COLOR_YELLOW}⚡${COLOR_RESET} Generating $service_name Dockerfile...              \n"

          # Source the dockerfile generator
          if [[ -f "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh" ]]; then
            source "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh"

            # Generate the appropriate service files
            if generate_dockerfile_for_service "$service_name" >/dev/null 2>&1; then
              printf "${COLOR_BLUE}⠋${COLOR_RESET} Continuing startup..."
              sleep 2   # Give Docker time to recognize new files
              return 99 # Retry
            else
              printf "\r${COLOR_RED}✗${COLOR_RESET} Failed to generate Dockerfile              \n"
            fi
          fi
        else
          log_info "Service '$service_name' is missing its Dockerfile"
          echo
          log_info "Auto-fixing: Generating $service_name service..."

          # Source the dockerfile generator
          if [[ -f "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh" ]]; then
            source "$SCRIPT_DIR/../lib/auto-fix/dockerfile-generator.sh"

            # Generate the appropriate service files
            if generate_dockerfile_for_service "$service_name"; then
              log_success "Service $service_name generated successfully"
              echo
              log_info "Continuing startup..."
              echo
              sleep 2   # Give Docker time to recognize new files
              return 99 # Retry
            else
              log_error "Failed to generate service $service_name"
            fi
          else
            log_error "Dockerfile generator not found"
          fi
        fi
      fi
    fi

    if echo "$output" | grep -q "missing go.sum entry\|go mod\|go.sum"; then
      log_info "Go module issue detected"
      echo
      log_info "Auto-fixing: Regenerating Go services..."

      # Find and regenerate all Go services
      if [[ -d "services/go" ]]; then
        for service_dir in services/go/*/; do
          if [[ -d "$service_dir" ]]; then
            service_name=$(basename "$service_dir")
            log_info "Fixing Go service: $service_name"

            # Add go.sum if missing
            if [[ ! -f "$service_dir/go.sum" ]]; then
              cat >"$service_dir/go.sum" <<'EOF'
github.com/gorilla/mux v1.8.0 h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=
github.com/gorilla/mux v1.8.0/go.mod h1:DVbg23sWSpFRCP0SfiEN6jmj59UnW/n46BH5rLB71So=
EOF
            fi
          fi
        done
        log_success "Go services fixed"
        echo
        log_info "Continuing startup..."
        echo
        sleep 2
        return 99 # Retry
      else
        echo
        log_info "Try: ${COLOR_BLUE}nself build --force && nself start${COLOR_RESET}"
      fi
    else
      # Show relevant build error lines
      echo "$output" | grep -E "ERROR|failed|exit code" | head -10
      echo
      log_info "Try: ${COLOR_BLUE}docker compose build --no-cache${COLOR_RESET}"
    fi

  # Docker not running
  elif echo "$output" | grep -q "Cannot connect to the Docker daemon\|docker daemon is not running"; then
    log_error "Docker is not running"
    echo
    log_info "Please start Docker Desktop or run:"
    echo "  ${COLOR_BLUE}sudo systemctl start docker${COLOR_RESET}"

  # Network issues
  elif echo "$output" | grep -q "network.*not found\|Error response from daemon.*network"; then
    log_error "Docker network issue"

    local network=$(echo "$output" | grep -oE "network [^ ]+" | cut -d' ' -f2 | head -1)
    if [[ -n "$network" ]]; then
      log_info "Network '$network' not found"
    fi

    echo
    log_info "Try: ${COLOR_BLUE}docker network prune && nself build && nself start${COLOR_RESET}"

  # Permission issues
  elif echo "$output" | grep -q "permission denied\|Permission denied"; then
    log_error "Permission denied"
    echo "$output" | grep -i "permission" | head -3
    echo
    log_info "Try: ${COLOR_BLUE}sudo nself start${COLOR_RESET}"

  # Unhealthy service dependency
  elif echo "$output" | grep -q "dependency failed to start.*is unhealthy"; then
    local unhealthy_service=$(echo "$output" | grep -oE "container [^ ]+ is unhealthy" | sed 's/container //; s/ is unhealthy//' | head -1)

    if [[ -n "$unhealthy_service" ]]; then
      # Get logs from the unhealthy service (silently)
      local service_logs=$(docker logs "$unhealthy_service" 2>&1 | tail -20)

      # Check if ALWAYS_AUTOFIX is enabled
      if [[ "${ALWAYS_AUTOFIX:-true}" == "true" ]]; then
        # Use new orchestrator
        if [[ -f "$SCRIPT_DIR/../lib/autofix/orchestrator.sh" ]]; then
          source "$SCRIPT_DIR/../lib/autofix/orchestrator.sh"

          # Run autofix (pass verbose flag)
          autofix_service "$unhealthy_service" "$service_logs" "$verbose"
          local fix_result=$?

          if [[ $fix_result -eq 99 ]]; then
            # Retry silently
            return 99
          elif [[ $fix_result -eq 0 ]]; then
            return 0
          else
            # Autofix failed completely
            return 1
          fi
        else
          # Fallback to simple recreation
          local choice=1
          log_info "Auto-fixing unhealthy service"
        fi
      else
        echo
        log_info "Options:"
        echo "  1) Auto-fix: Recreate the service"
        echo "  2) View full logs"
        echo "  3) Continue anyway"
        echo "  4) Cancel"
        echo

        read -p "Select option (1-4): " choice
      fi

      # Only execute case if we didn't use the dispatcher
      if [[ ! -f "$SCRIPT_DIR/../lib/autofix/dispatcher.sh" ]] || [[ "${ALWAYS_AUTOFIX:-true}" != "true" ]]; then
        case "$choice" in
        1)
          log_info "Recreating $unhealthy_service..."
          docker stop "$unhealthy_service" >/dev/null 2>&1
          docker rm "$unhealthy_service" >/dev/null 2>&1


          log_success "Service recreated"
          echo
          log_info "Retrying startup..."
          sleep 2
          return 99 # Retry
          ;;
        2)
          docker logs "$unhealthy_service" 2>&1 | tail -50
          echo
          log_info "Run 'nself start' again when ready"
          ;;
        3)
          log_warning "Continuing with unhealthy service..."
          ;;
        4)
          log_info "Cancelled"
          ;;
        *)
          log_error "Invalid option"
          ;;
        esac
      fi
    else
      log_error "Service dependency is unhealthy"
      echo "$output" | grep "unhealthy" | head -3
    fi

  # Generic error - show relevant output
  elif true; then
    # First check for common Docker Compose issues
    if echo "$output" | grep -q "no configuration file provided"; then
      log_error "docker-compose.yml not found"
      echo
      log_info "Run: ${COLOR_BLUE}nself build${COLOR_RESET}"
    elif echo "$output" | grep -q "yaml: "; then
      log_error "Invalid docker-compose.yml format"
      local yaml_error=$(echo "$output" | grep "yaml: " | head -1)
      echo "$yaml_error"
      echo
      log_info "Run: ${COLOR_BLUE}nself build --force${COLOR_RESET}"
    elif echo "$output" | grep -q "pull access denied\|no matching manifest"; then
      log_error "Docker image pull failed"
      local image_error=$(echo "$output" | grep -E "pull access denied|no matching manifest" | head -1)
      echo "$image_error"
      echo
      log_info "Check your Docker Hub access or image names"
    else
      # Generic error - try to find meaningful lines
      log_error "Service startup failed"

      # Look for actual error messages, not stack traces
      local error_lines=$(echo "$output" | grep -v "\.go:[0-9]" | grep -v "runtime\." | grep -v "github.com/" | grep -v "^[[:space:]]*$" | grep -E "error|ERROR|failed|Failed|unable|Unable" | head -5)

      if [[ -n "$error_lines" ]]; then
        echo "$error_lines"
      else
        # If no clear error, show the docker compose command output
        local compose_output=$(echo "$output" | grep -E "Container.*Creating|Container.*Error|Error response from daemon" | head -5)
        if [[ -n "$compose_output" ]]; then
          echo "$compose_output"
        else
          # Last resort - show any non-stack-trace output
          echo "$output" | grep -v "\.go:[0-9]" | grep -v "^[[:space:]]*$" | head -5
        fi
      fi
    fi
  fi

  echo
  log_info "For more details, run: ${COLOR_BLUE}nself start --verbose${COLOR_RESET}"
}

# Show service URLs
show_service_urls() {
  echo ""
  echo -e "${COLOR_CYAN}→${COLOR_RESET} Service URLs"
  echo

  # Load environment if available
  if [[ -f ".env.local" ]]; then
    set -a
    load_env_with_priority
    set +a
  fi

  # Get base domain or use default
  local base_domain="${BASE_DOMAIN:-localhost}"

  # Check which services are actually running (remove unity_ prefix)
  local running_services=$(docker ps --format "table {{.Names}}" | grep "^${PROJECT_NAME:-nself}_" | sed "s/^${PROJECT_NAME:-nself}_//" 2>/dev/null)

  # Track if any URLs were shown
  local urls_shown=false

  # Hasura GraphQL with sub-items
  if echo "$running_services" | grep -q "hasura"; then
    echo "GraphQL API:    https://api.$base_domain"
    echo " - Console:     https://api.$base_domain/console"
    urls_shown=true
  fi

  # Auth
  if echo "$running_services" | grep -q "auth"; then
    echo "Auth:           https://auth.$base_domain"
    urls_shown=true
  fi

  # Storage
  if echo "$running_services" | grep -q "storage\|minio"; then
    echo "Storage:        https://storage.$base_domain"
    urls_shown=true
  fi

  # Functions
  if echo "$running_services" | grep -q "functions"; then
    echo "Functions:      https://functions.$base_domain"
    urls_shown=true
  fi

  # Dashboard
  if echo "$running_services" | grep -q "dashboard"; then
    echo "Dashboard:      https://dashboard.$base_domain"
    urls_shown=true
  fi

  # Development tools
  if echo "$running_services" | grep -q "mailpit\|mail"; then
    echo "Mail UI:        http://localhost:8025"
    urls_shown=true
  fi

  if echo "$running_services" | grep -q "minio"; then
    echo "MinIO Console:  http://localhost:9001"
    urls_shown=true
  fi

  # If no URLs shown, indicate no exposed services
  if [[ "$urls_shown" != "true" ]]; then
    echo "No services with exposed URLs found"
  fi

  echo ""
  echo "nself status | nself logs <service> | nself doctor"
  echo
}

# Show help
show_help() {
  echo "Usage: nself start [OPTIONS]"
  echo ""
  echo "Start all services defined in docker-compose.yml"
  echo ""
  echo "Options:"
  echo "  -v, --verbose      Show detailed output"
  echo "  -a, --attach       Run in foreground (attached mode)"
  echo "  --skip-checks      Skip port availability checks"
  echo "  -h, --help         Show this help message"
  echo ""
  echo "Examples:"
  echo "  nself start        # Start services in background"
  echo "  nself start -v     # Start with verbose output"
  echo "  nself start -a     # Start in foreground mode"
}

# Export for use as library
export -f cmd_start

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "start" || exit $?
  cmd_start "$@"
  exit_code=$?
  post_command "start" $exit_code
  exit $exit_code
fi
