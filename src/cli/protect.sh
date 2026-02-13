#!/usr/bin/env bash
# protect.sh - Environment protection (staging safeguards)

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source utilities
source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"

# Colors
COLOR_RESET='\033[0m'
COLOR_GREEN='\033[0;32m'
COLOR_YELLOW='\033[0;33m'
COLOR_RED='\033[0;31m'
COLOR_BLUE='\033[0;34m'
COLOR_DIM='\033[2m'

# Show help
show_help() {
  cat << 'EOF'
Usage: nself protect <environment> [options]

Add password protection and IP whitelisting to staging/production environments.

Commands:
  nself protect staging --auth=basic         Enable basic auth on staging
  nself protect staging --ip=1.2.3.4         Add IP to whitelist
  nself protect staging --disable            Temporarily disable protection
  nself protect status                       Show protection status for all environments

Options:
  --auth=basic            Enable HTTP basic authentication
  --user=<username>       Set username (default: admin)
  --pass=<password>       Set password (generated if not provided)
  --ip=<ip-address>       Add IP to whitelist (comma-separated for multiple)
  --allow-ips=<ips>       Same as --ip
  --disable               Disable protection
  --duration=<time>       Disable for duration (e.g., 1h, 30m)
  -h, --help              Show this help message

Examples:
  # Enable basic auth with generated password
  nself protect staging --auth=basic

  # Enable basic auth with custom credentials
  nself protect staging --auth=basic --user=admin --pass=MySecurePass123

  # Add IP whitelist
  nself protect staging --ip=192.168.1.100,192.168.1.101

  # Show status
  nself protect status

  # Temporarily disable for 1 hour
  nself protect staging --disable --duration=1h

Security Notes:
  - Credentials are stored in .env.secrets (never committed)
  - Basic auth is configured via nginx
  - IP whitelist uses nginx allow/deny directives
  - Protection applies to ALL routes in the environment

EOF
}

# Generate random password
generate_password() {
  # Generate 16-character password with letters, numbers, and special chars
  local pass=$(LC_ALL=C tr -dc 'A-Za-z0-9!@#$%^&*' < /dev/urandom | head -c 16)
  echo "$pass"
}

# Create htpasswd entry
create_htpasswd() {
  local user="$1"
  local pass="$2"

  # Use openssl to create bcrypt hash (compatible with nginx)
  local hash=$(openssl passwd -apr1 "$pass" 2>/dev/null)

  echo "$user:$hash"
}

# Enable basic auth
enable_basic_auth() {
  local env="$1"
  local user="${2:-admin}"
  local pass="${3:-}"

  # Generate password if not provided
  if [[ -z "$pass" ]]; then
    pass=$(generate_password)
    printf "${COLOR_BLUE}⠿${COLOR_RESET} Generated password: ${COLOR_YELLOW}$pass${COLOR_RESET}\n"
  fi

  # Create htpasswd entry
  local htpasswd=$(create_htpasswd "$user" "$pass")

  # Store in .env.secrets
  local secrets_file=".env.secrets"

  if [[ ! -f "$secrets_file" ]]; then
    touch "$secrets_file"
    chmod 600 "$secrets_file"
  fi

  # Add or update credentials
  local key_user="${env}_BASIC_AUTH_USER"
  local key_pass="${env}_BASIC_AUTH_PASSWORD"
  local key_htpasswd="${env}_BASIC_AUTH_HTPASSWD"

  # Remove existing entries
  grep -v "^${key_user}=" "$secrets_file" > "$secrets_file.tmp" 2>/dev/null || true
  grep -v "^${key_pass}=" "$secrets_file.tmp" > "$secrets_file.tmp2" 2>/dev/null || true
  grep -v "^${key_htpasswd}=" "$secrets_file.tmp2" > "$secrets_file" 2>/dev/null || true
  rm -f "$secrets_file.tmp" "$secrets_file.tmp2"

  # Add new entries
  {
    echo ""
    echo "# Basic Auth for $env environment (generated $(date +%Y-%m-%d))"
    echo "${key_user}=$user"
    echo "${key_pass}=$pass"
    echo "${key_htpasswd}=$htpasswd"
  } >> "$secrets_file"

  # Create nginx htpasswd file
  local htpasswd_file="nginx/.htpasswd.$env"
  mkdir -p nginx
  echo "$htpasswd" > "$htpasswd_file"
  chmod 600 "$htpasswd_file"

  printf "${COLOR_GREEN}✓${COLOR_RESET} Basic auth enabled for ${COLOR_BLUE}$env${COLOR_RESET}\n"
  printf "  Username: ${COLOR_BLUE}$user${COLOR_RESET}\n"
  printf "  Password: ${COLOR_YELLOW}$pass${COLOR_RESET} ${COLOR_DIM}(saved to $secrets_file)${COLOR_RESET}\n"
  printf "  Htpasswd: ${COLOR_DIM}$htpasswd_file${COLOR_RESET}\n"
  printf "\n${COLOR_YELLOW}⚠${COLOR_RESET}  Save these credentials in a secure location!\n"
  printf "\n${COLOR_DIM}Next steps:${COLOR_RESET}\n"
  printf "  1. Run ${COLOR_BLUE}nself build${COLOR_RESET} to regenerate nginx config\n"
  printf "  2. Run ${COLOR_BLUE}nself restart nginx${COLOR_RESET} to apply changes\n"
}

# Add IP to whitelist
add_ip_whitelist() {
  local env="$1"
  local ips="$2"

  # Store in .env file
  local env_file=".env.$env"

  if [[ ! -f "$env_file" ]]; then
    touch "$env_file"
  fi

  # Add or update IP whitelist
  local key="${env}_ALLOWED_IPS"

  # Remove existing entry
  grep -v "^${key}=" "$env_file" > "$env_file.tmp" 2>/dev/null || true
  mv "$env_file.tmp" "$env_file"

  # Add new entry
  echo "${key}=$ips" >> "$env_file"

  # Count IPs
  local ip_count=$(echo "$ips" | tr ',' '\n' | wc -l | tr -d ' ')

  printf "${COLOR_GREEN}✓${COLOR_RESET} IP whitelist configured for ${COLOR_BLUE}$env${COLOR_RESET}\n"
  printf "  Allowed IPs ($ip_count): ${COLOR_BLUE}$ips${COLOR_RESET}\n"
  printf "\n${COLOR_DIM}Next steps:${COLOR_RESET}\n"
  printf "  1. Run ${COLOR_BLUE}nself build${COLOR_RESET} to regenerate nginx config\n"
  printf "  2. Run ${COLOR_BLUE}nself restart nginx${COLOR_RESET} to apply changes\n"
}

# Show protection status
show_status() {
  printf "${COLOR_BLUE}Environment Protection Status:${COLOR_RESET}\n\n"

  for env in dev staging prod; do
    printf "${COLOR_BLUE}${env}:${COLOR_RESET}\n"

    # Check basic auth
    local user_key="${env}_BASIC_AUTH_USER"
    local user=""

    if [[ -f ".env.secrets" ]]; then
      user=$(grep "^${user_key}=" .env.secrets 2>/dev/null | cut -d= -f2-)
    fi

    if [[ -n "$user" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} Basic auth: enabled (user: $user)\n"
    else
      printf "  ${COLOR_DIM}○${COLOR_RESET} Basic auth: disabled\n"
    fi

    # Check IP whitelist
    local ip_key="${env}_ALLOWED_IPS"
    local ips=""

    if [[ -f ".env.$env" ]]; then
      ips=$(grep "^${ip_key}=" ".env.$env" 2>/dev/null | cut -d= -f2-)
    fi

    if [[ -n "$ips" ]]; then
      local ip_count=$(echo "$ips" | tr ',' '\n' | wc -l | tr -d ' ')
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} IP whitelist: $ip_count IP(s) allowed\n"
    else
      printf "  ${COLOR_DIM}○${COLOR_RESET} IP whitelist: disabled (all IPs allowed)\n"
    fi

    printf "\n"
  done
}

# Main command dispatcher
main() {
  local command="${1:-}"
  shift || true

  case "$command" in
    status)
      show_status
      ;;
    dev|staging|prod|production)
      local env="$command"
      [[ "$env" == "production" ]] && env="prod"

      # Parse options
      local auth_enabled=false
      local user="admin"
      local pass=""
      local ips=""
      local disable=false
      local duration=""

      while [[ $# -gt 0 ]]; do
        case "$1" in
          --auth=basic)
            auth_enabled=true
            shift
            ;;
          --user=*)
            user="${1#*=}"
            shift
            ;;
          --pass=*)
            pass="${1#*=}"
            shift
            ;;
          --ip=*|--allow-ips=*)
            ips="${1#*=}"
            shift
            ;;
          --disable)
            disable=true
            shift
            ;;
          --duration=*)
            duration="${1#*=}"
            shift
            ;;
          *)
            shift
            ;;
        esac
      done

      if [[ "$auth_enabled" == "true" ]]; then
        enable_basic_auth "$env" "$user" "$pass"
      elif [[ -n "$ips" ]]; then
        add_ip_whitelist "$env" "$ips"
      elif [[ "$disable" == "true" ]]; then
        printf "${COLOR_YELLOW}⚠${COLOR_RESET} Disable functionality not yet implemented\n"
        printf "  Remove entries from .env.secrets and .env.$env manually\n"
      else
        printf "${COLOR_RED}✗${COLOR_RESET} No action specified\n" >&2
        printf "  Use --auth=basic or --ip=<address>\n" >&2
        return 1
      fi
      ;;
    -h|--help|help|"")
      show_help
      ;;
    *)
      printf "${COLOR_RED}✗${COLOR_RESET} Unknown command: $command\n" >&2
      printf "  Run ${COLOR_BLUE}nself protect --help${COLOR_RESET} for usage\n" >&2
      return 1
      ;;
  esac
}

main "$@"
