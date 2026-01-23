#!/usr/bin/env bash

# servers.sh - Server and infrastructure management
# v0.4.6 - Feedback implementation

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/header.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"

# Color fallbacks
: "${COLOR_GREEN:=\033[0;32m}"
: "${COLOR_YELLOW:=\033[0;33m}"
: "${COLOR_RED:=\033[0;31m}"
: "${COLOR_CYAN:=\033[0;36m}"
: "${COLOR_RESET:=\033[0m}"
: "${COLOR_DIM:=\033[2m}"
: "${COLOR_BOLD:=\033[1m}"

# Show help
show_servers_help() {
  cat << 'EOF'
nself servers - Server and infrastructure management

Usage: nself servers <subcommand> [options]

Subcommands:
  list                  List all configured servers
  add <name>            Add a new server
  remove <name>         Remove a server
  status [name]         Check server status
  ssh <name>            SSH into a server
  logs <name>           View server logs
  update <name>         Update server configuration
  reboot <name>         Reboot a server
  info <name>           Show detailed server info

Options:
  --provider NAME       Cloud provider (aws, hetzner, do, etc.)
  --region REGION       Server region
  --type TYPE           Server type/size
  --ip IP               Server IP address
  --user USER           SSH user (default: root)
  --env NAME            Environment (staging, prod)
  --json                Output in JSON format
  -h, --help            Show this help message

Examples:
  nself servers list                      # List all servers
  nself servers add web1 --ip 1.2.3.4     # Add server
  nself servers status                    # Check all status
  nself servers ssh prod-web              # SSH to server
  nself servers info staging-db           # Server details
EOF
}

# Initialize servers environment
init_servers() {
  load_env_with_priority

  SERVERS_DIR="${SERVERS_DIR:-.nself/servers}"
  SERVERS_FILE="${SERVERS_DIR}/servers.json"
  mkdir -p "$SERVERS_DIR"

  # Initialize servers file if not exists
  if [[ ! -f "$SERVERS_FILE" ]]; then
    echo '{"servers": []}' > "$SERVERS_FILE"
  fi
}

# Get server by name
get_server() {
  local name="$1"

  if command -v jq >/dev/null 2>&1; then
    jq -r ".servers[] | select(.name == \"$name\")" "$SERVERS_FILE" 2>/dev/null
  else
    # Fallback to grep
    grep -o "{[^}]*\"name\": *\"$name\"[^}]*}" "$SERVERS_FILE" 2>/dev/null | head -1
  fi
}

# List all servers
cmd_list() {
  local json_mode="${JSON_OUTPUT:-false}"
  local filter_env="${FILTER_ENV:-}"
  local filter_provider="${FILTER_PROVIDER:-}"

  init_servers

  if [[ "$json_mode" == "true" ]]; then
    cat "$SERVERS_FILE"
    return 0
  fi

  show_command_header "nself servers" "Server List"
  echo ""

  # Check if any servers exist
  local server_count=0
  if command -v jq >/dev/null 2>&1; then
    server_count=$(jq '.servers | length' "$SERVERS_FILE" 2>/dev/null || echo 0)
  else
    server_count=$(grep -c '"name":' "$SERVERS_FILE" 2>/dev/null || echo 0)
  fi

  if [[ "$server_count" -eq 0 ]]; then
    log_info "No servers configured"
    echo ""
    log_info "Add a server with: nself servers add <name> --ip <ip>"
    return 0
  fi

  printf "  %-20s %-15s %-15s %-12s %s\n" "Name" "IP" "Provider" "Environment" "Status"
  printf "  %-20s %-15s %-15s %-12s %s\n" "----" "--" "--------" "-----------" "------"

  # Parse servers (with or without jq)
  if command -v jq >/dev/null 2>&1; then
    jq -r '.servers[] | "\(.name)|\(.ip)|\(.provider // "unknown")|\(.env // "unknown")|\(.status // "unknown")"' "$SERVERS_FILE" 2>/dev/null | while IFS='|' read -r name ip provider env status; do
      # Apply filters
      [[ -n "$filter_env" && "$env" != "$filter_env" ]] && continue
      [[ -n "$filter_provider" && "$provider" != "$filter_provider" ]] && continue

      local status_color="$COLOR_GREEN"
      [[ "$status" != "active" && "$status" != "running" ]] && status_color="$COLOR_YELLOW"
      [[ "$status" == "error" || "$status" == "unreachable" ]] && status_color="$COLOR_RED"

      printf "  %-20s %-15s %-15s %-12s ${status_color}%s${COLOR_RESET}\n" \
        "$name" "$ip" "$provider" "$env" "$status"
    done
  else
    # Fallback parsing
    grep '"name":' "$SERVERS_FILE" | while read -r line; do
      printf "  (jq not installed - raw view)\n"
      cat "$SERVERS_FILE"
      break
    done
  fi

  echo ""
  printf "  ${COLOR_DIM}Total: %d server(s)${COLOR_RESET}\n" "$server_count"
}

# Add a new server
cmd_add() {
  local name="$1"
  local ip="${SERVER_IP:-}"
  local provider="${SERVER_PROVIDER:-manual}"
  local region="${SERVER_REGION:-}"
  local server_type="${SERVER_TYPE:-}"
  local user="${SERVER_USER:-root}"
  local env="${SERVER_ENV:-production}"

  if [[ -z "$name" ]]; then
    log_error "Server name required"
    return 1
  fi

  if [[ -z "$ip" ]]; then
    log_error "Server IP required (--ip)"
    return 1
  fi

  init_servers

  # Check if server already exists
  local existing=$(get_server "$name")
  if [[ -n "$existing" ]]; then
    log_error "Server already exists: $name"
    log_info "Use 'nself servers update $name' to modify"
    return 1
  fi

  show_command_header "nself servers" "Adding Server"
  echo ""

  printf "${COLOR_CYAN}➞ Server Configuration${COLOR_RESET}\n"
  printf "  Name: %s\n" "$name"
  printf "  IP: %s\n" "$ip"
  printf "  Provider: %s\n" "$provider"
  printf "  Region: %s\n" "${region:-not specified}"
  printf "  Type: %s\n" "${server_type:-not specified}"
  printf "  User: %s\n" "$user"
  printf "  Environment: %s\n" "$env"
  echo ""

  # Test SSH connectivity
  printf "Testing SSH connectivity..."
  if ssh -o ConnectTimeout=5 -o BatchMode=yes "${user}@${ip}" "echo ok" >/dev/null 2>&1; then
    printf " ${COLOR_GREEN}✓${COLOR_RESET}\n"
  else
    printf " ${COLOR_YELLOW}⚠ (may require key setup)${COLOR_RESET}\n"
  fi
  echo ""

  # Add to servers file
  local timestamp=$(date -Iseconds)
  local new_server=$(cat << EOF
{
  "name": "$name",
  "ip": "$ip",
  "provider": "$provider",
  "region": "$region",
  "type": "$server_type",
  "user": "$user",
  "env": "$env",
  "status": "active",
  "added": "$timestamp"
}
EOF
)

  if command -v jq >/dev/null 2>&1; then
    jq ".servers += [$new_server]" "$SERVERS_FILE" > "${SERVERS_FILE}.tmp" && mv "${SERVERS_FILE}.tmp" "$SERVERS_FILE"
  else
    # Fallback: simple append
    local current=$(cat "$SERVERS_FILE")
    if [[ "$current" == '{"servers": []}' ]]; then
      echo "{\"servers\": [$new_server]}" > "$SERVERS_FILE"
    else
      # Remove closing brackets and append
      sed 's/]}$//' "$SERVERS_FILE" > "${SERVERS_FILE}.tmp"
      echo ", $new_server]}" >> "${SERVERS_FILE}.tmp"
      mv "${SERVERS_FILE}.tmp" "$SERVERS_FILE"
    fi
  fi

  log_success "Server added: $name"
  log_info "SSH: nself servers ssh $name"
}

# Remove a server
cmd_remove() {
  local name="$1"
  local force="${FORCE:-false}"

  if [[ -z "$name" ]]; then
    log_error "Server name required"
    return 1
  fi

  init_servers

  local server=$(get_server "$name")
  if [[ -z "$server" ]]; then
    log_error "Server not found: $name"
    return 1
  fi

  show_command_header "nself servers" "Removing Server"
  echo ""

  log_warning "This will remove server configuration for: $name"
  log_info "The actual server will NOT be deleted"
  echo ""

  if [[ "$force" != "true" ]]; then
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      log_info "Removal cancelled"
      return 1
    fi
  fi

  if command -v jq >/dev/null 2>&1; then
    jq "del(.servers[] | select(.name == \"$name\"))" "$SERVERS_FILE" > "${SERVERS_FILE}.tmp" && mv "${SERVERS_FILE}.tmp" "$SERVERS_FILE"
    log_success "Server removed: $name"
  else
    log_error "jq required for this operation"
    return 1
  fi
}

# Check server status
cmd_status() {
  local name="$1"
  local json_mode="${JSON_OUTPUT:-false}"

  init_servers

  if [[ "$json_mode" != "true" ]]; then
    show_command_header "nself servers" "Server Status"
    echo ""
  fi

  # Get servers to check
  local servers_to_check=()
  if [[ -n "$name" ]]; then
    servers_to_check+=("$name")
  else
    if command -v jq >/dev/null 2>&1; then
      while IFS= read -r s; do
        servers_to_check+=("$s")
      done < <(jq -r '.servers[].name' "$SERVERS_FILE" 2>/dev/null)
    fi
  fi

  if [[ ${#servers_to_check[@]} -eq 0 ]]; then
    log_info "No servers to check"
    return 0
  fi

  if [[ "$json_mode" != "true" ]]; then
    printf "  %-20s %-15s %-10s %-10s %s\n" "Name" "IP" "SSH" "Ping" "Load"
    printf "  %-20s %-15s %-10s %-10s %s\n" "----" "--" "---" "----" "----"
  fi

  local results="["
  local first=true

  for server_name in "${servers_to_check[@]}"; do
    local server=$(get_server "$server_name")
    if [[ -z "$server" ]]; then
      continue
    fi

    local ip=$(echo "$server" | grep -o '"ip": *"[^"]*"' | sed 's/"ip": *"\([^"]*\)"/\1/')
    local user=$(echo "$server" | grep -o '"user": *"[^"]*"' | sed 's/"user": *"\([^"]*\)"/\1/')
    user="${user:-root}"

    # Check ping
    local ping_status="timeout"
    if ping -c 1 -W 2 "$ip" >/dev/null 2>&1; then
      ping_status="ok"
    fi

    # Check SSH
    local ssh_status="timeout"
    local load="N/A"
    if ssh -o ConnectTimeout=5 -o BatchMode=yes "${user}@${ip}" "true" >/dev/null 2>&1; then
      ssh_status="ok"
      # Get load average
      load=$(ssh -o ConnectTimeout=5 "${user}@${ip}" "uptime" 2>/dev/null | grep -oE 'load average: [0-9.]+' | cut -d' ' -f3 || echo "N/A")
    fi

    if [[ "$first" != "true" ]]; then
      results+=","
    fi
    first=false
    results+="{\"name\": \"$server_name\", \"ip\": \"$ip\", \"ssh\": \"$ssh_status\", \"ping\": \"$ping_status\", \"load\": \"$load\"}"

    if [[ "$json_mode" != "true" ]]; then
      local ping_color="$COLOR_GREEN"
      [[ "$ping_status" != "ok" ]] && ping_color="$COLOR_RED"

      local ssh_color="$COLOR_GREEN"
      [[ "$ssh_status" != "ok" ]] && ssh_color="$COLOR_RED"

      printf "  %-20s %-15s ${ssh_color}%-10s${COLOR_RESET} ${ping_color}%-10s${COLOR_RESET} %s\n" \
        "$server_name" "$ip" "$ssh_status" "$ping_status" "$load"
    fi
  done

  results+="]"

  if [[ "$json_mode" == "true" ]]; then
    printf '{"status": %s}\n' "$results"
  fi
}

# SSH to server
cmd_ssh() {
  local name="$1"
  shift

  if [[ -z "$name" ]]; then
    log_error "Server name required"
    return 1
  fi

  init_servers

  local server=$(get_server "$name")
  if [[ -z "$server" ]]; then
    log_error "Server not found: $name"
    return 1
  fi

  local ip=$(echo "$server" | grep -o '"ip": *"[^"]*"' | sed 's/"ip": *"\([^"]*\)"/\1/')
  local user=$(echo "$server" | grep -o '"user": *"[^"]*"' | sed 's/"user": *"\([^"]*\)"/\1/')
  user="${user:-root}"

  log_info "Connecting to ${user}@${ip}..."
  exec ssh "${user}@${ip}" "$@"
}

# View server logs
cmd_logs() {
  local name="$1"
  local lines="${LOG_LINES:-100}"

  if [[ -z "$name" ]]; then
    log_error "Server name required"
    return 1
  fi

  init_servers

  local server=$(get_server "$name")
  if [[ -z "$server" ]]; then
    log_error "Server not found: $name"
    return 1
  fi

  local ip=$(echo "$server" | grep -o '"ip": *"[^"]*"' | sed 's/"ip": *"\([^"]*\)"/\1/')
  local user=$(echo "$server" | grep -o '"user": *"[^"]*"' | sed 's/"user": *"\([^"]*\)"/\1/')
  user="${user:-root}"

  show_command_header "nself servers" "Logs: $name"
  echo ""

  log_info "Fetching last $lines lines from syslog..."
  echo ""

  ssh "${user}@${ip}" "tail -n $lines /var/log/syslog 2>/dev/null || journalctl -n $lines --no-pager 2>/dev/null || echo 'Could not fetch logs'"
}

# Update server configuration
cmd_update() {
  local name="$1"

  if [[ -z "$name" ]]; then
    log_error "Server name required"
    return 1
  fi

  init_servers

  local server=$(get_server "$name")
  if [[ -z "$server" ]]; then
    log_error "Server not found: $name"
    return 1
  fi

  show_command_header "nself servers" "Updating: $name"
  echo ""

  # Update fields if provided
  local updated=false

  if [[ -n "${SERVER_IP:-}" ]]; then
    if command -v jq >/dev/null 2>&1; then
      jq "(.servers[] | select(.name == \"$name\") | .ip) = \"$SERVER_IP\"" "$SERVERS_FILE" > "${SERVERS_FILE}.tmp" && mv "${SERVERS_FILE}.tmp" "$SERVERS_FILE"
      printf "  IP: %s\n" "$SERVER_IP"
      updated=true
    fi
  fi

  if [[ -n "${SERVER_USER:-}" ]]; then
    if command -v jq >/dev/null 2>&1; then
      jq "(.servers[] | select(.name == \"$name\") | .user) = \"$SERVER_USER\"" "$SERVERS_FILE" > "${SERVERS_FILE}.tmp" && mv "${SERVERS_FILE}.tmp" "$SERVERS_FILE"
      printf "  User: %s\n" "$SERVER_USER"
      updated=true
    fi
  fi

  if [[ -n "${SERVER_ENV:-}" ]]; then
    if command -v jq >/dev/null 2>&1; then
      jq "(.servers[] | select(.name == \"$name\") | .env) = \"$SERVER_ENV\"" "$SERVERS_FILE" > "${SERVERS_FILE}.tmp" && mv "${SERVERS_FILE}.tmp" "$SERVERS_FILE"
      printf "  Environment: %s\n" "$SERVER_ENV"
      updated=true
    fi
  fi

  if [[ "$updated" == "true" ]]; then
    echo ""
    log_success "Server updated"
  else
    log_info "No changes specified"
    log_info "Use --ip, --user, or --env to update fields"
  fi
}

# Reboot server
cmd_reboot() {
  local name="$1"
  local force="${FORCE:-false}"

  if [[ -z "$name" ]]; then
    log_error "Server name required"
    return 1
  fi

  init_servers

  local server=$(get_server "$name")
  if [[ -z "$server" ]]; then
    log_error "Server not found: $name"
    return 1
  fi

  local ip=$(echo "$server" | grep -o '"ip": *"[^"]*"' | sed 's/"ip": *"\([^"]*\)"/\1/')
  local user=$(echo "$server" | grep -o '"user": *"[^"]*"' | sed 's/"user": *"\([^"]*\)"/\1/')
  user="${user:-root}"

  show_command_header "nself servers" "Rebooting: $name"
  echo ""

  log_warning "This will reboot server: $name ($ip)"
  echo ""

  if [[ "$force" != "true" ]]; then
    read -p "Continue? (y/N) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
      log_info "Reboot cancelled"
      return 1
    fi
  fi

  log_info "Sending reboot command..."
  ssh "${user}@${ip}" "sudo reboot" 2>/dev/null || true

  log_success "Reboot command sent"
  log_info "Server should be back online in 1-2 minutes"
}

# Show detailed server info
cmd_info() {
  local name="$1"
  local json_mode="${JSON_OUTPUT:-false}"

  if [[ -z "$name" ]]; then
    log_error "Server name required"
    return 1
  fi

  init_servers

  local server=$(get_server "$name")
  if [[ -z "$server" ]]; then
    log_error "Server not found: $name"
    return 1
  fi

  if [[ "$json_mode" == "true" ]]; then
    echo "$server"
    return 0
  fi

  local ip=$(echo "$server" | grep -o '"ip": *"[^"]*"' | sed 's/"ip": *"\([^"]*\)"/\1/')
  local provider=$(echo "$server" | grep -o '"provider": *"[^"]*"' | sed 's/"provider": *"\([^"]*\)"/\1/')
  local region=$(echo "$server" | grep -o '"region": *"[^"]*"' | sed 's/"region": *"\([^"]*\)"/\1/')
  local server_type=$(echo "$server" | grep -o '"type": *"[^"]*"' | sed 's/"type": *"\([^"]*\)"/\1/')
  local user=$(echo "$server" | grep -o '"user": *"[^"]*"' | sed 's/"user": *"\([^"]*\)"/\1/')
  local env=$(echo "$server" | grep -o '"env": *"[^"]*"' | sed 's/"env": *"\([^"]*\)"/\1/')
  local added=$(echo "$server" | grep -o '"added": *"[^"]*"' | sed 's/"added": *"\([^"]*\)"/\1/')

  show_command_header "nself servers" "Server Info: $name"
  echo ""

  printf "${COLOR_CYAN}➞ Configuration${COLOR_RESET}\n"
  printf "  Name: %s\n" "$name"
  printf "  IP: %s\n" "$ip"
  printf "  User: %s\n" "${user:-root}"
  printf "  Provider: %s\n" "${provider:-unknown}"
  printf "  Region: %s\n" "${region:-not specified}"
  printf "  Type: %s\n" "${server_type:-not specified}"
  printf "  Environment: %s\n" "${env:-unknown}"
  printf "  Added: %s\n" "${added:-unknown}"
  echo ""

  # Get live info via SSH
  printf "${COLOR_CYAN}➞ Live Status${COLOR_RESET}\n"
  if ssh -o ConnectTimeout=5 -o BatchMode=yes "${user:-root}@${ip}" "true" >/dev/null 2>&1; then
    printf "  SSH: ${COLOR_GREEN}connected${COLOR_RESET}\n"

    # Get uptime
    local uptime=$(ssh -o ConnectTimeout=5 "${user:-root}@${ip}" "uptime -p 2>/dev/null || uptime | sed 's/.*up /up /' | cut -d',' -f1,2" 2>/dev/null)
    printf "  Uptime: %s\n" "$uptime"

    # Get memory
    local mem=$(ssh -o ConnectTimeout=5 "${user:-root}@${ip}" "free -h 2>/dev/null | grep Mem | awk '{print \$3\"/\"\$2}'" 2>/dev/null)
    printf "  Memory: %s\n" "${mem:-N/A}"

    # Get disk
    local disk=$(ssh -o ConnectTimeout=5 "${user:-root}@${ip}" "df -h / 2>/dev/null | tail -1 | awk '{print \$3\"/\"\$2\" (\"\$5\" used)\"}'" 2>/dev/null)
    printf "  Disk: %s\n" "${disk:-N/A}"

    # Get load
    local load=$(ssh -o ConnectTimeout=5 "${user:-root}@${ip}" "cat /proc/loadavg 2>/dev/null | cut -d' ' -f1-3" 2>/dev/null)
    printf "  Load: %s\n" "${load:-N/A}"
  else
    printf "  SSH: ${COLOR_RED}unreachable${COLOR_RESET}\n"
  fi
}

# Main command handler
cmd_servers() {
  local subcommand="${1:-list}"

  # Check for help first
  if [[ "$subcommand" == "-h" ]] || [[ "$subcommand" == "--help" ]]; then
    show_servers_help
    return 0
  fi

  # Parse global options
  local args=()
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --provider)
        SERVER_PROVIDER="$2"
        FILTER_PROVIDER="$2"
        shift 2
        ;;
      --region)
        SERVER_REGION="$2"
        shift 2
        ;;
      --type)
        SERVER_TYPE="$2"
        shift 2
        ;;
      --ip)
        SERVER_IP="$2"
        shift 2
        ;;
      --user)
        SERVER_USER="$2"
        shift 2
        ;;
      --env)
        SERVER_ENV="$2"
        FILTER_ENV="$2"
        shift 2
        ;;
      --lines)
        LOG_LINES="$2"
        shift 2
        ;;
      --force)
        FORCE=true
        shift
        ;;
      --json)
        JSON_OUTPUT=true
        shift
        ;;
      -h|--help)
        show_servers_help
        return 0
        ;;
      *)
        args+=("$1")
        shift
        ;;
    esac
  done

  # Restore positional arguments
  set -- "${args[@]}"
  subcommand="${1:-list}"

  case "$subcommand" in
    list)
      cmd_list
      ;;
    add)
      shift
      cmd_add "$@"
      ;;
    remove|rm)
      shift
      cmd_remove "$@"
      ;;
    status)
      shift
      cmd_status "$@"
      ;;
    ssh)
      shift
      cmd_ssh "$@"
      ;;
    logs)
      shift
      cmd_logs "$@"
      ;;
    update)
      shift
      cmd_update "$@"
      ;;
    reboot)
      shift
      cmd_reboot "$@"
      ;;
    info)
      shift
      cmd_info "$@"
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      show_servers_help
      return 1
      ;;
  esac
}

# Export for use as library
export -f cmd_servers

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "servers" || exit $?
  cmd_servers "$@"
  exit_code=$?
  post_command "servers" $exit_code
  exit $exit_code
fi
