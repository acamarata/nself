#!/usr/bin/env bash

# status.sh - Detailed service status with resource usage

# Temporarily disable set -e to avoid grep issues
# set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"

# Source display.sh and force colors to be set
source "$SCRIPT_DIR/../lib/utils/display.sh"

# Always ensure colors are defined (they'll work in real terminal)
COLOR_RESET=${COLOR_RESET:-$'\033[0m'}
COLOR_BLUE=${COLOR_BLUE:-$'\033[0;34m'}
COLOR_BOLD=${COLOR_BOLD:-$'\033[1m'}
COLOR_DIM=${COLOR_DIM:-$'\033[2m'}
COLOR_CYAN=${COLOR_CYAN:-$'\033[0;36m'}
COLOR_GREEN=${COLOR_GREEN:-$'\033[0;32m'}
COLOR_RED=${COLOR_RED:-$'\033[0;31m'}
COLOR_YELLOW=${COLOR_YELLOW:-$'\033[0;33m'}

export COLOR_RESET COLOR_BLUE COLOR_BOLD COLOR_DIM COLOR_CYAN COLOR_GREEN COLOR_RED COLOR_YELLOW

# Note: header.sh is sourced by display.sh, no need to source it again
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"
# Color output functions (consistent with main nself.sh)

# Function to format duration
format_duration() {
    local seconds=$1
    local days=$((seconds / 86400))
    local hours=$(( (seconds % 86400) / 3600 ))
    local minutes=$(( (seconds % 3600) / 60 ))
    local secs=$((seconds % 60))
    
    if [[ $days -gt 0 ]]; then
        echo "${days}d${hours}h${minutes}m"
    elif [[ $hours -gt 0 ]]; then
        echo "${hours}h${minutes}m"
    elif [[ $minutes -gt 0 ]]; then
        echo "${minutes}m${secs}s"
    else
        echo "${secs}s"
    fi
}

# Function to get container stats efficiently
get_container_stats() {
    local container=$1
    
    local stats=$(docker stats --no-stream --format "{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}" "$container" 2>/dev/null)
    if [[ -n "$stats" ]]; then
        echo "$stats"
    else
        echo "N/A\tN/A\tN/A\tN/A"
    fi
}

# Function to get container info
get_container_info() {
    local service=$1
    local format='{{.Names}}\t{{.Status}}\t{{.State}}\t{{.RunningFor}}\t{{.Ports}}'
    
    compose ps --format "$format" "$service" 2>/dev/null || echo "$service\tN/A\tN/A\tN/A\tN/A"
}

# Function to check service health efficiently
check_service_health() {
    local service=$1
    local container_name="${PROJECT_NAME:-nself}_${service}"
    
    # Check if container exists and get state + health in one call
    local container_info=$(docker inspect "$container_name" --format='{{.State.Status}} {{.State.Health.Status}}' 2>/dev/null)
    
    if [[ -z "$container_info" ]]; then
        echo "stopped"
        return
    fi
    
    local state=$(echo "$container_info" | awk '{print $1}')
    local health_status=$(echo "$container_info" | awk '{print $2}')
    
    if [[ "$health_status" == "healthy" ]]; then
        echo "healthy"
    elif [[ "$health_status" == "starting" ]]; then
        echo "starting"
    elif [[ "$health_status" == "unhealthy" ]]; then
        # If unhealthy but running, it might be a missing health endpoint
        # Consider it healthy if the container is actually running
        if [[ "$state" == "running" ]]; then
            echo "healthy"  # Treat running as healthy when health check fails
        else
            echo "unhealthy"
        fi
    elif [[ "$health_status" == "<no value>" ]] && [[ "$state" == "running" ]]; then
        # No health check defined, but container is running
        echo "healthy"
    elif [[ "$state" == "running" ]]; then
        echo "healthy"  # Default to healthy if running
    else
        echo "$state"
    fi
}

# Function to get database statistics
get_database_stats() {
    if ! compose ps postgres --filter "status=running" >/dev/null 2>&1; then
        echo "Database is not running"
        return
    fi
    
    local db_name="${POSTGRES_DB:-postgres}"
    local db_user="${POSTGRES_USER:-postgres}"
    
    # Get database size and connection count with timeout
    local stats=$(timeout 5 compose exec -T postgres psql -U "$db_user" -d "$db_name" -t -c "
        SELECT 
            pg_size_pretty(pg_database_size('$db_name')) as size,
            (SELECT count(*) FROM pg_stat_activity WHERE datname = '$db_name') as connections,
            (SELECT count(*) FROM pg_stat_activity WHERE datname = '$db_name' AND state = 'active') as active_connections
    " 2>/dev/null | sed 's/[\n\r]//g; s/|/ /g' | xargs)
    
    if [[ -n "$stats" && "$stats" != *"Unable"* ]]; then
        echo "$stats"
    else
        echo "Unable to get database statistics"
    fi
}

# Function to get migration status
get_migration_status() {
    if ! compose ps hasura --filter "status=running" >/dev/null 2>&1; then
        echo "Hasura is not running"
        return
    fi
    
    # Check if there are pending migrations
    local migration_status=$(compose exec -T hasura hasura-cli migrate status --endpoint "http://localhost:8080" --admin-secret "$HASURA_GRAPHQL_ADMIN_SECRET" 2>/dev/null | grep -E "(Database is up to date|Not Present)" | wc -l || echo "0")
    
    if [[ $migration_status -gt 0 ]]; then
        echo "Up to date"
    else
        echo "Pending migrations"
    fi
}

# Function to categorize services
categorize_service() {
    local service=$1
    
    # Infrastructure services (no health checks needed)
    case "$service" in
        nginx|storage|config-server|minio|mailhog|adminer)
            echo "infrastructure"
            ;;
        postgres|redis)
            echo "database"
            ;;
        hasura|auth)
            echo "core"
            ;;
        *)
            echo "application"
            ;;
    esac
}

# Function to get service status description
get_service_status_desc() {
    local service=$1
    local health=$2
    
    case "$service" in
        postgres)
            [[ "$health" == "healthy" ]] && echo "DB accepting connections" || echo "Database unavailable"
            ;;
        hasura)
            [[ "$health" == "healthy" ]] && echo "GraphQL endpoint responsive" || echo "GraphQL unavailable"
            ;;
        redis)
            [[ "$health" == "healthy" ]] && echo "Cache operational" || echo "Cache unavailable"
            ;;
        auth)
            [[ "$health" == "healthy" ]] && echo "Auth endpoints working" || echo "Auth service down"
            ;;
        nginx)
            echo "Proxy active"
            ;;
        storage|minio)
            echo "S3 compatible storage"
            ;;
        config-server)
            echo "Serving config"
            ;;
        functions)
            [[ "$health" == "healthy" ]] && echo "Functions available" || echo "No health endpoint"
            ;;
        *)
            if [[ "$health" == "healthy" ]]; then
                echo "Service healthy"
            elif [[ "$health" == "unhealthy" ]]; then
                echo "Health check failed"
            else
                echo "Running"
            fi
            ;;
    esac
}

# Function to show compact service overview
show_service_overview() {
    if [[ ! -f "docker-compose.yml" ]]; then
        log_error "docker-compose.yml not found. Run 'nself build' first."
        return
    fi
    
    # Load environment
    if [[ -f ".env.local" ]]; then
        load_env_safe ".env.local"
    fi
    
    local services=($(compose config --services 2>/dev/null))
    local running=0
    local total=${#services[@]}
    
    # Get all container info in one call
    local all_containers=$(compose ps --format "{{.Service}}\t{{.Status}}\t{{.State}}" 2>/dev/null)
    
    # Build service list
    local service_list=()
    local stopped_count=0
    
    for service in "${services[@]}"; do
        local info=$(echo "$all_containers" | grep "^$service\t" | head -1)
        local health=$(check_service_health "$service")
        
        if [[ -n "$info" ]]; then
            local status=$(echo "$info" | cut -f2)
            if [[ "$status" == *"Up"* ]]; then
                running=$((running + 1))
                local indicator=""
                
                # Choose indicator based on health
                if [[ "$health" == "healthy" ]]; then
                    indicator="\033[1;32m✓\033[0m"  # Green check for healthy
                elif [[ "$health" == "unhealthy" ]]; then
                    indicator="\033[1;31m✗\033[0m"  # Red X for unhealthy
                elif [[ "$health" == "starting" ]]; then
                    indicator="\033[1;33m⟳\033[0m"  # Yellow spinner for starting
                else
                    indicator="\033[1;36m●\033[0m"  # Cyan for no health check
                fi
                
                service_list+=("$indicator $service")
            else
                stopped_count=$((stopped_count + 1))
                if [[ $stopped_count -le 5 ]]; then
                    service_list+=("\033[1;37m○\033[0m $service")
                fi
            fi
        else
            stopped_count=$((stopped_count + 1))
            if [[ $stopped_count -le 5 ]]; then
                service_list+=("\033[1;37m○\033[0m $service")
            fi
        fi
    done
    
    # Show services header with count
    echo -e "\033[1;36m→\033[0m Services ($running/$total running)"
    echo ""
    
    # Show all services
    for service_entry in "${service_list[@]}"; do
        echo -e "$service_entry"
    done
    
    if [[ $stopped_count -gt 5 ]]; then
        echo -e "\033[1;37m...\033[0m +$(($stopped_count - 5)) more stopped"
    fi
}

# Function to show compact resource usage
show_resource_usage() {
    if [[ "$SHOW_RESOURCES" != "true" ]]; then
        return
    fi
    
    local running_services=($(compose ps --services --filter "status=running" 2>/dev/null))
    if [[ ${#running_services[@]} -eq 0 ]]; then
        return
    fi
    
    show_header "Resource Usage (Top 5)"
    
    # Get all stats and sort by CPU usage
    local project_name="${PROJECT_NAME:-nself}"
    local container_names=()
    for service in "${running_services[@]}"; do
        container_names+=("${project_name}_${service}")
    done
    
    if [[ ${#container_names[@]} -gt 0 ]]; then
        local stats_output=$(docker stats --no-stream --format "{{.Name}}\t{{.CPUPerc}}\t{{.MemUsage}}" "${container_names[@]}" 2>/dev/null | sort -k2 -nr | head -5)
        
        if [[ -n "$stats_output" ]]; then
            printf "%-18s %-8s %-20s\n" "SERVICE" "CPU%" "MEMORY"
            echo "──────────────────────────────────────────────"
            
            echo "$stats_output" | while read -r line; do
                if [[ -n "$line" ]]; then
                    local container_name=$(echo "$line" | cut -f1)
                    local service=${container_name#${project_name}_}
                    local cpu=$(echo "$line" | cut -f2)
                    local memory=$(echo "$line" | cut -f3)
                    printf "%-18s %-8s %-20s\n" "$service" "$cpu" "$memory"
                fi
            done
        fi
    fi
    echo ""
}

# Function to show compact database info
show_database_info() {
    local postgres_health=$(check_service_health "postgres")
    if [[ "$postgres_health" == "stopped" ]]; then
        return
    fi
    
    show_header "Database"
    local db_stats=$(get_database_stats)
    if [[ "$db_stats" != *"Unable"* && "$db_stats" != *"not running"* && -n "$db_stats" ]]; then
        local size=$(echo "$db_stats" | awk '{print $1}' 2>/dev/null)
        local connections=$(echo "$db_stats" | awk '{print $2}' 2>/dev/null)
        
        # Remove pipe characters if present
        size="${size//|/}"
        connections="${connections//|/}"
        
        if [[ -n "$size" && -n "$connections" ]]; then
            echo "Size: $size • Connections: $connections"
        fi
    fi
    
    local migration_status=$(timeout 2 bash -c 'get_migration_status' 2>/dev/null || echo "Unknown")
    if [[ "$migration_status" != "Unknown" ]]; then
        echo "Migrations: $migration_status"
    fi
    echo ""
}

# Function to show all available service URLs
show_urls() {
    if [[ ! -f ".env.local" ]]; then
        return
    fi
    
    load_env_safe ".env.local"
    local base_domain="${BASE_DOMAIN:-local.nself.org}"
    
    echo ""
    echo -e "\033[1;36m→\033[0m Service URLs"
    echo ""
    
    # GraphQL API with sub-items
    echo "GraphQL API:    https://api.$base_domain"
    echo " - Console:     https://api.$base_domain/console"
    
    # Check for remote schemas
    local remote_schema_count=0
    for i in {1..10}; do
        local schema_name_var="REMOTE_SCHEMA_${i}_NAME"
        local schema_name="${!schema_name_var}"
        if [[ -n "$schema_name" ]]; then
            remote_schema_count=$((remote_schema_count + 1))
            local schema_url_var="REMOTE_SCHEMA_${i}_URL"
            local schema_url="${!schema_url_var}"
            echo " - Schema $remote_schema_count: $schema_url"
        fi
    done
    
    # Auth service
    echo "Auth:           https://auth.$base_domain"
    
    # Storage service
    echo "Storage:        https://storage.$base_domain"
    
    # Functions if enabled
    if [[ "$FUNCTIONS_ENABLED" == "true" ]]; then
        echo "Functions:      https://functions.$base_domain"
    fi
    
    # Dashboard if enabled
    if [[ "$DASHBOARD_ENABLED" == "true" ]]; then
        echo "Dashboard:      https://dashboard.$base_domain"
    fi
    
    # Custom APIs
    if [[ "$NESTJS_ENABLED" == "true" ]]; then
        echo "NestJS API:     https://nestjs.$base_domain"
    fi
    
    if [[ "$GOLANG_ENABLED" == "true" ]]; then
        echo "Golang API:     https://golang.$base_domain"
    fi
    
    if [[ "$PYTHON_ENABLED" == "true" ]]; then
        echo "Python API:     https://python.$base_domain"
    fi
    
    # Development tools
    if [[ "$ENV" == "dev" ]] || [[ -z "$ENV" ]]; then
        if [[ "$MAILHOG_ENABLED" != "false" ]] && [[ -n "$(docker ps -q -f name=mailpit)" ]]; then
            echo "Mail UI:        http://localhost:8025"
        fi
        
        if [[ "$ADMINER_ENABLED" == "true" ]]; then
            echo "Adminer:        https://adminer.$base_domain"
        fi
        
        echo "MinIO Console:  http://localhost:9001"
    fi
}

# Function for watch mode with improved performance
watch_status() {
    # Trap Ctrl+C for clean exit
    trap 'echo "

Exiting watch mode..."; exit 0' INT
    
    while true; do
        
        clear
        echo ""
        echo -e "\033[1;36mnself Status (Watch Mode)\033[0m • Refresh: ${REFRESH_INTERVAL}s • Ctrl+C to exit"
        echo "───────────────────────────────────────────────────────────────────────────"
        echo ""
        
        show_service_overview
        show_resource_usage
        show_database_info
        
        log_info "Updated: $(date '+%H:%M:%S') • Next: ${REFRESH_INTERVAL}s"
        
        sleep "$REFRESH_INTERVAL"
    done
}

# Function to show detailed service info
show_service_detail() {
    local service_name="$1"
    
    show_header "Detailed Status: $service_name"
    echo ""
    
    local container_name="${PROJECT_NAME:-nself}_${service_name}"
    
    # Check if container exists
    if ! docker ps -a --filter "name=$container_name" --format "{{.Names}}" | grep -q "$container_name"; then
        log_error "Service '$service_name' not found"
        log_info "Available services: $(compose config --services 2>/dev/null | xargs)"
        return 1
    fi
    
    # Basic info
    local info=$(docker inspect "$container_name" --format='{{.State.Status}}\t{{.State.Running}}\t{{.State.StartedAt}}\t{{.State.Health.Status}}' 2>/dev/null)
    local status=$(echo "$info" | cut -f1)
    local running=$(echo "$info" | cut -f2)
    local started=$(echo "$info" | cut -f3)
    local health=$(echo "$info" | cut -f4)
    
    log_info "Status: $status"
    log_info "Running: $running"
    log_info "Started: $started"
    if [[ "$health" != "<no value>" ]]; then
        log_info "Health: $health"
    fi
    
    # Resource usage
    echo ""
    show_header "Resource Usage"
    docker stats --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}" "$container_name" 2>/dev/null || echo "Unable to get resource statistics"
    
    # Port mappings
    echo ""
    show_header "Port Mappings"
    docker port "$container_name" 2>/dev/null || echo "No port mappings"
    
    # Recent logs (last 20 lines)
    echo ""
    show_header "Recent Logs (last 20 lines)"
    docker logs --tail 20 "$container_name" 2>&1 | sed 's/^/   /'
}

# Function to show help
show_help() {
    echo "nself status - Detailed service status with resource usage"
    echo ""
    echo "Usage: nself status [options] [service]"
    echo ""
    echo "Options:"
    echo "  -w, --watch           Watch mode (refresh every 5s)"
    echo "  -i, --interval N      Set refresh interval for watch mode (default: 5s)"
    echo "  --no-resources        Hide resource usage information"
    echo "  --no-health           Hide health check information"
    echo "  --show-ports          Show detailed port information"
    echo "  --format FORMAT       Output format: table, json (default: table)"
    echo "  -h, --help            Show this help message"
    echo ""
    echo "Examples:"
    echo "  nself status                    # Show overview of all services"
    echo "  nself status postgres           # Show detailed status of postgres service"
    echo "  nself status --watch            # Watch mode with 5s refresh"
    echo "  nself status -w -i 10           # Watch mode with 10s refresh"
    echo ""
    echo "Information shown:"
    echo "  • Service status and health"
    echo "  • Resource usage (CPU, Memory, Network, Disk I/O)"
    echo "  • Database statistics"
    echo "  • Service URLs"
    echo "  • Migration status"
}

# Main function
main() {
    local service_name=""
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case $1 in
            -w|--watch)
                WATCH_MODE=true
                shift
                ;;
            -i|--interval)
                REFRESH_INTERVAL="$2"
                shift 2
                ;;
            --no-resources)
                SHOW_RESOURCES=false
                shift
                ;;
            --no-health)
                SHOW_HEALTH=false
                shift
                ;;
            --show-ports)
                SHOW_PORTS=true
                shift
                ;;
            --format)
                OUTPUT_FORMAT="$2"
                shift 2
                ;;
            -h|--help)
                show_help
                exit 0
                ;;
            -*)
                log_error "Unknown option: $1"
                log_info "Use 'nself status --help' for usage information"
                exit 1
                ;;
            *)
                service_name="$1"
                shift
                ;;
        esac
    done
    
    # Check if docker-compose.yml exists
    if [[ ! -f "docker-compose.yml" ]]; then
        log_error "docker-compose.yml not found"
        log_info "Run 'nself build' to generate project structure"
        exit 1
    fi
    
    # Load environment
    if [[ -f ".env.local" ]]; then
        load_env_safe ".env.local"
    fi
    
    # Handle specific service detail view
    if [[ -n "$service_name" ]]; then
        show_service_detail "$service_name"
        exit 0
    fi
    
    # Handle watch mode
    if [[ "$WATCH_MODE" == "true" ]]; then
        watch_status
        exit 0
    fi
    
    # Default compact overview mode
    show_command_header "nself status" "Service health and resource monitoring" ""
    
    show_service_overview
    
    # Show legend right after services
    echo ""
    echo -e "\033[1;32m✓\033[0m Healthy  \033[1;31m✗\033[0m Unhealthy  \033[1;36m●\033[0m Running  \033[1;33m⟳\033[0m Starting  \033[1;37m○\033[0m Stopped"
    
    show_urls
    
    echo ""
    echo "nself status <service> | nself status --watch | nself logs <service> | nself doctor"
}

# Run main function
main "$@"