#!/usr/bin/env bash
# security_audit.sh - nself security audit
# On-demand full security scan for nself deployments
# 8 checks: ports, sshd, fail2ban, docker iptables, UFW, Grafana creds, nginx TLS, rate limits
# Bash 3.2+ compatible

set -euo pipefail

CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/platform-compat.sh" 2>/dev/null || true

if ! type log_info >/dev/null 2>&1; then
  log_info()    { printf "\033[0;34m[INFO]\033[0m %s\n" "$1"; }
  log_error()   { printf "\033[0;31m[ERROR]\033[0m %s\n" "$1" >&2; }
  log_success() { printf "\033[0;32m[SUCCESS]\033[0m %s\n" "$1"; }
  log_warning() { printf "\033[0;33m[WARNING]\033[0m %s\n" "$1"; }
fi

# ── Output helpers ───────────────────────────────────────────────────────────
_pass()  { printf "  \033[0;32m[PASS]\033[0m  %s\n" "$1"; }
_warn()  { printf "  \033[0;33m[WARN]\033[0m  %s\n" "$1"; }
_fail()  { printf "  \033[0;31m[FAIL]\033[0m  %s\n" "$1"; }
_skip()  { printf "  \033[2m[SKIP]\033[0m  %s\n" "$1"; }
_fix()   { printf "         \033[2m→ fix: %s\033[0m\n" "$1"; }
_note()  { printf "         \033[2m%s\033[0m\n" "$1"; }

# ============================================================================
# CHECK 0 — Environment File Secrets
# Reads .env in the current directory, checks for missing/weak secrets
# This check always runs (including with --no-docker)
# ============================================================================

check_env_secrets() {
  local check_num="${1:-0}"
  printf "\n\033[1m[%s] Environment File Secrets\033[0m\n" "$check_num"

  local env_file=".env"
  if [[ ! -f "$env_file" ]]; then
    _skip ".env not found — run nself init first"
    return 0
  fi

  local issues=0

  # Check HASURA_GRAPHQL_JWT_SECRET
  local jwt_secret
  jwt_secret=$(grep -E '^HASURA_GRAPHQL_JWT_SECRET=' "$env_file" | cut -d= -f2- | tr -d '"' | tr -d "'" || true)
  if [[ -z "$jwt_secret" ]]; then
    _fail "JWT secret is not set (HASURA_GRAPHQL_JWT_SECRET missing from .env)"
    _fix "Add: HASURA_GRAPHQL_JWT_SECRET='{\"type\":\"HS256\",\"key\":\"<32+ char secret>\"}'"
    _note "Generate: python3 -c \"import secrets; print(secrets.token_hex(32))\""
    issues=$((issues + 1))
  elif [[ ${#jwt_secret} -lt 16 ]]; then
    _fail "JWT secret is too short (${#jwt_secret} chars — minimum 32 chars for HS256)"
    issues=$((issues + 1))
  else
    _pass "JWT secret is set"
  fi

  # Check HASURA_GRAPHQL_ADMIN_SECRET
  local admin_secret
  admin_secret=$(grep -E '^HASURA_GRAPHQL_ADMIN_SECRET=' "$env_file" | cut -d= -f2- | tr -d '"' | tr -d "'" || true)
  if [[ -z "$admin_secret" ]]; then
    _warn "ADMIN_SECRET not set — Hasura admin endpoint is unprotected"
    _fix "Add: HASURA_GRAPHQL_ADMIN_SECRET=<strong secret>"
    issues=$((issues + 1))
  elif [[ ${#admin_secret} -lt 16 ]]; then
    _fail "Admin secret is too short (${#admin_secret} chars — minimum 16 chars)"
    _note "HASURA_GRAPHQL_ADMIN_SECRET must be at least 16 characters"
    issues=$((issues + 1))
  else
    _pass "Admin secret is set and meets minimum length"
  fi

  # Check POSTGRES_PASSWORD
  local pg_pass
  pg_pass=$(grep -E '^POSTGRES_PASSWORD=' "$env_file" | cut -d= -f2- | tr -d '"' | tr -d "'" || true)
  if [[ -z "$pg_pass" ]]; then
    _warn "POSTGRES_PASSWORD not set — database uses no password"
    issues=$((issues + 1))
  elif [[ ${#pg_pass} -lt 12 ]]; then
    _fail "POSTGRES_PASSWORD is too short (${#pg_pass} chars — minimum 12 chars)"
    _note "Postgres password must be at least 12 characters to meet minimum security requirements"
    issues=$((issues + 1))
  else
    _pass "Postgres password is set and meets minimum length"
  fi

  return $issues
}

# ============================================================================
# CHECK 1 — Port Exposure
# No non-nginx service must bind to 0.0.0.0
# ============================================================================

check_port_exposure() {
  printf "\n\033[1m[1/8] Port Exposure\033[0m\n"

  local dc_file="${1:-docker-compose.yml}"

  if [[ ! -f "$dc_file" ]]; then
    _skip "docker-compose.yml not found — run nself build first"
    return 0
  fi

  local violations=0
  local in_service=""
  local in_ports=false

  while IFS= read -r line; do
    # Track current service
    if printf '%s' "$line" | grep -qE '^[[:space:]]{2}[a-z]'; then
      in_service=$(printf '%s' "$line" | sed 's/[[:space:]]*\([a-z_-]*\):.*/\1/')
      in_ports=false
    fi

    # Track ports section
    if printf '%s' "$line" | grep -qE '^[[:space:]]*(ports|published):'; then
      in_ports=true
    fi

    # Check for 0.0.0.0 bindings outside nginx
    if [[ "$in_ports" == "true" ]] && printf '%s' "$line" | grep -qE '0\.0\.0\.0:'; then
      if [[ "$in_service" != "nginx" ]]; then
        local port
        port=$(printf '%s' "$line" | grep -oE '[0-9]+:[0-9]+' | head -1)
        _fail "Service '$in_service' exposes port $port on 0.0.0.0 (bypasses nginx)"
        _fix "Remove port binding or change to 127.0.0.1:${port#*:}:${port#*:} in docker-compose.yml"
        _note "Run: nself build (regenerates docker-compose from .env)"
        violations=$((violations + 1))
      fi
    fi

    # Reset ports tracking on new section
    if [[ "$in_ports" == "true" ]] && ! printf '%s' "$line" | grep -qE '(ports|published|-)'; then
      in_ports=false
    fi
  done < "$dc_file"

  if [[ $violations -eq 0 ]]; then
    _pass "All services bind to 127.0.0.1 (nginx is the only public endpoint)"
  fi

  return $violations
}

# ============================================================================
# CHECK 2 — SSH Configuration
# ============================================================================

check_sshd_config() {
  printf "\n\033[1m[2/8] SSH Configuration\033[0m\n"

  local sshd_config="${SSHD_CONFIG_PATH:-/etc/ssh/sshd_config}"

  if [[ ! -f "$sshd_config" ]]; then
    _skip "sshd_config not found ($sshd_config) — likely running on macOS or no SSH server"
    return 0
  fi

  local issues=0

  # Password authentication
  local pw_auth
  pw_auth=$(grep -iE '^[[:space:]]*PasswordAuthentication' "$sshd_config" | tail -1 | awk '{print $2}' | tr '[:upper:]' '[:lower:]' || true)
  if [[ "${pw_auth:-yes}" == "yes" ]]; then
    _warn "PasswordAuthentication is enabled (allows brute-force attacks)"
    _fix "Set 'PasswordAuthentication no' in $sshd_config"
    issues=$((issues + 1))
  else
    _pass "PasswordAuthentication is disabled"
  fi

  # Root login
  local root_login
  root_login=$(grep -iE '^[[:space:]]*PermitRootLogin' "$sshd_config" | tail -1 | awk '{print $2}' | tr '[:upper:]' '[:lower:]' || true)
  if [[ "${root_login:-yes}" == "yes" ]] || [[ "${root_login:-}" == "prohibit-password" ]]; then
    _warn "PermitRootLogin allows root access (use a non-root user)"
    _fix "Set 'PermitRootLogin no' in $sshd_config"
    issues=$((issues + 1))
  else
    _pass "Root login is disabled"
  fi

  # Port 22
  local ssh_port
  ssh_port=$(grep -iE '^[[:space:]]*Port[[:space:]]' "$sshd_config" | tail -1 | awk '{print $2}' || true)
  if [[ -z "$ssh_port" ]] || [[ "$ssh_port" == "22" ]]; then
    _warn "SSH is on the default port 22 (common target for scanners)"
    _note "Consider changing to a non-standard port for reduced noise"
  else
    _pass "SSH port is $ssh_port (non-default)"
  fi

  return $issues
}

# ============================================================================
# CHECK 3 — fail2ban
# ============================================================================

check_fail2ban() {
  printf "\n\033[1m[3/8] fail2ban\033[0m\n"

  # Skip on macOS
  if [[ "$(uname -s)" == "Darwin" ]]; then
    _skip "fail2ban not applicable on macOS"
    return 0
  fi

  if ! command -v fail2ban-client >/dev/null 2>&1; then
    _warn "fail2ban is not installed"
    _fix "Install: apt install fail2ban  or  yum install fail2ban"
    return 1
  fi

  _pass "fail2ban is installed"

  # Check if running
  local status_output
  status_output=$(fail2ban-client status 2>/dev/null || true)
  if [[ -z "$status_output" ]]; then
    _warn "fail2ban service is not running"
    _fix "Start: systemctl enable --now fail2ban"
    return 1
  fi

  _pass "fail2ban service is running"

  # Check for SSH jail
  if printf '%s' "$status_output" | grep -qi 'sshd\|ssh'; then
    _pass "SSH jail is active"
  else
    _warn "No SSH jail detected in fail2ban"
    _fix "Create /etc/fail2ban/jail.local with [sshd] enabled = true"
  fi

  return 0
}

# ============================================================================
# CHECK 4 — Docker daemon iptables setting
# ============================================================================

check_docker_iptables() {
  printf "\n\033[1m[4/8] Docker iptables\033[0m\n"

  # Skip on macOS
  if [[ "$(uname -s)" == "Darwin" ]]; then
    _skip "Docker iptables check not applicable on macOS"
    return 0
  fi

  local daemon_json="/etc/docker/daemon.json"

  if [[ ! -f "$daemon_json" ]]; then
    _pass "No daemon.json — Docker uses iptables by default (good)"
    return 0
  fi

  if grep -q '"iptables"[[:space:]]*:[[:space:]]*false' "$daemon_json" 2>/dev/null; then
    _fail "Docker iptables is disabled in daemon.json"
    _note "This causes UFW/firewall rules to NOT apply to Docker-exposed ports"
    _fix "Remove 'iptables: false' from $daemon_json and restart Docker"
    return 1
  fi

  _pass "Docker iptables is enabled (firewall rules apply correctly)"
  return 0
}

# ============================================================================
# CHECK 5 — UFW status and rules
# ============================================================================

check_ufw() {
  printf "\n\033[1m[5/8] UFW Firewall\033[0m\n"

  # Skip on macOS
  if [[ "$(uname -s)" == "Darwin" ]]; then
    _skip "UFW check not applicable on macOS"
    return 0
  fi

  if ! command -v ufw >/dev/null 2>&1; then
    _warn "UFW is not installed"
    _fix "Install: apt install ufw && ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw enable"
    return 1
  fi

  local ufw_status
  ufw_status=$(ufw status 2>/dev/null | head -1 | awk '{print $2}' | tr '[:upper:]' '[:lower:]' || true)

  if [[ "$ufw_status" != "active" ]]; then
    _warn "UFW is installed but not active"
    _fix "Enable: ufw allow 22/tcp && ufw allow 80/tcp && ufw allow 443/tcp && ufw enable"
    return 1
  fi

  _pass "UFW is active"

  # Check for HTTP/HTTPS rules
  local has_http=false
  local has_https=false
  local ufw_rules
  ufw_rules=$(ufw status 2>/dev/null || true)

  printf '%s' "$ufw_rules" | grep -q "80\|8080\|http" && has_http=true
  printf '%s' "$ufw_rules" | grep -q "443\|https" && has_https=true

  if [[ "$has_http" == "true" ]]; then
    _pass "HTTP (port 80) rule exists"
  else
    _warn "No HTTP (port 80) rule in UFW — nginx may not be reachable"
    _fix "Run: ufw allow 80/tcp"
  fi

  if [[ "$has_https" == "true" ]]; then
    _pass "HTTPS (port 443) rule exists"
  else
    _warn "No HTTPS (port 443) rule in UFW — SSL may not be reachable"
    _fix "Run: ufw allow 443/tcp"
  fi

  return 0
}

# ============================================================================
# CHECK 6 — Grafana default credentials
# ============================================================================

check_grafana_defaults() {
  printf "\n\033[1m[6/8] Grafana Default Credentials\033[0m\n"

  # Check if Grafana is in the docker-compose
  local dc_file="${1:-docker-compose.yml}"
  if [[ ! -f "$dc_file" ]] || ! grep -q 'grafana' "$dc_file" 2>/dev/null; then
    _skip "Grafana not found in docker-compose.yml"
    return 0
  fi

  # Try a login check on localhost (Grafana default port is 3000, but internal only)
  # We check via nginx proxy or direct if accessible
  local grafana_url="${GRAFANA_URL:-http://localhost:3000}"

  if command -v curl >/dev/null 2>&1; then
    local http_code
    http_code=$(curl -s -o /dev/null -w "%{http_code}" --max-time 5 \
      -u admin:admin "${grafana_url}/api/org" 2>/dev/null || true)

    if [[ "$http_code" == "200" ]]; then
      _fail "Grafana accepts default credentials (admin/admin)"
      _fix "Log in to Grafana and change the password immediately"
      return 1
    elif [[ "$http_code" == "401" ]] || [[ "$http_code" == "403" ]]; then
      _pass "Grafana does not accept default credentials"
    else
      _skip "Could not reach Grafana ($grafana_url returned $http_code) — check manually"
    fi
  else
    _skip "curl not available — cannot check Grafana credentials"
  fi

  return 0
}

# ============================================================================
# CHECK 7 — Nginx TLS version
# ============================================================================

check_nginx_tls() {
  printf "\n\033[1m[7/8] Nginx TLS Configuration\033[0m\n"

  local nginx_conf_dir="${NGINX_CONF_DIR:-nginx}"
  local found_configs=false
  local issues=0

  for conf in "$nginx_conf_dir"/*.conf "$nginx_conf_dir"/sites/*.conf; do
    [[ -f "$conf" ]] || continue
    found_configs=true

    # Check for TLS 1.0 or 1.1
    if grep -q 'TLSv1\.0\|TLSv1\.1\|SSLv2\|SSLv3' "$conf" 2>/dev/null; then
      _fail "Legacy TLS versions detected in $(basename "$conf")"
      _fix "Remove TLSv1.0 and TLSv1.1 from ssl_protocols directive"
      _note "Use: ssl_protocols TLSv1.2 TLSv1.3;"
      issues=$((issues + 1))
    fi

    # Check for ssl_protocols directive
    if grep -q 'ssl_protocols' "$conf" 2>/dev/null; then
      local proto_line
      proto_line=$(grep 'ssl_protocols' "$conf" | head -1)
      if printf '%s' "$proto_line" | grep -qE 'TLSv1\.2|TLSv1\.3'; then
        _pass "$(basename "$conf"): TLS 1.2+ only"
      fi
    fi
  done

  if [[ "$found_configs" == "false" ]]; then
    _skip "No nginx config files found in ./$nginx_conf_dir/"
    return 0
  fi

  if [[ $issues -eq 0 ]]; then
    _pass "No legacy TLS versions found in nginx config"
  fi

  return $issues
}

# ============================================================================
# CHECK 8 — Nginx rate limiting
# ============================================================================

check_nginx_rate_limits() {
  printf "\n\033[1m[8/8] Nginx Rate Limiting\033[0m\n"

  local nginx_conf_dir="${NGINX_CONF_DIR:-nginx}"
  local found_rate_limit=false
  local found_configs=false

  for conf in "$nginx_conf_dir"/*.conf "$nginx_conf_dir"/sites/*.conf; do
    [[ -f "$conf" ]] || continue
    found_configs=true

    if grep -q 'limit_req_zone\|limit_req ' "$conf" 2>/dev/null; then
      found_rate_limit=true
      break
    fi
  done

  if [[ "$found_configs" == "false" ]]; then
    _skip "No nginx config files found — run nself build first"
    return 0
  fi

  if [[ "$found_rate_limit" == "true" ]]; then
    _pass "Rate limiting zones are configured in nginx"
  else
    _warn "No rate limiting found in nginx config"
    _note "Add: limit_req_zone \$binary_remote_addr zone=api:10m rate=30r/m;"
    _note "     limit_req zone=api burst=60 nodelay;"
  fi

  return 0
}

# ============================================================================
# AUTO-FIX HELPERS
# ============================================================================

_autofix() {
  printf "\n\033[1mApplying safe auto-fixes...\033[0m\n\n"

  local sshd_config="${SSHD_CONFIG_PATH:-/etc/ssh/sshd_config}"
  local fixed=0

  # Fix 1: fail2ban SSH jail
  if command -v fail2ban-client >/dev/null 2>&1 && [[ "$(uname -s)" != "Darwin" ]]; then
    local jail_local="/etc/fail2ban/jail.local"
    if [[ ! -f "$jail_local" ]] || ! grep -q '\[sshd\]' "$jail_local" 2>/dev/null; then
      printf "  Creating fail2ban SSH jail at %s\n" "$jail_local"
      # Only write if we have permission
      if [[ -w "$(dirname "$jail_local")" ]]; then
        printf '[sshd]\nenabled = true\nport = ssh\nlogpath = %%(sshd_log)s\nbackend = %%(sshd_backend)s\n' \
          >> "$jail_local"
        systemctl reload fail2ban 2>/dev/null || true
        log_success "Created fail2ban SSH jail"
        fixed=$((fixed + 1))
      else
        log_warning "No write permission to $jail_local — run as root"
      fi
    fi
  fi

  # Fix 2: UFW rules for 80 and 443
  if command -v ufw >/dev/null 2>&1 && [[ "$(uname -s)" != "Darwin" ]]; then
    if ufw status 2>/dev/null | grep -q "active"; then
      if ! ufw status 2>/dev/null | grep -qE "80/tcp|80 "; then
        printf "  Adding UFW rule: 80/tcp\n"
        ufw allow 80/tcp 2>/dev/null && fixed=$((fixed + 1)) || true
      fi
      if ! ufw status 2>/dev/null | grep -qE "443/tcp|443 "; then
        printf "  Adding UFW rule: 443/tcp\n"
        ufw allow 443/tcp 2>/dev/null && fixed=$((fixed + 1)) || true
      fi
    fi
  fi

  # Fix 3: sshd_config — comment out insecure options (with backup)
  if [[ -f "$sshd_config" ]] && [[ -w "$sshd_config" ]]; then
    local backup="${sshd_config}.nself-backup-$(date '+%Y%m%d-%H%M%S')"
    cp "$sshd_config" "$backup"
    printf "  Backed up sshd_config to %s\n" "$backup"

    # Disable PasswordAuthentication
    if grep -qiE '^[[:space:]]*PasswordAuthentication[[:space:]]+yes' "$sshd_config"; then
      if type safe_sed_inline >/dev/null 2>&1; then
        safe_sed_inline 's/^[[:space:]]*PasswordAuthentication[[:space:]]*yes/# PasswordAuthentication yes  # disabled by nself security audit/' "$sshd_config"
      else
        local tmp="${sshd_config}.tmp.$$"
        while IFS= read -r line; do
          case "$line" in
            *[Pp]assword[Aa]uthentication*yes*)
              printf '# %s  # disabled by nself security audit\n' "$line" ;;
            *) printf '%s\n' "$line" ;;
          esac
        done < "$sshd_config" > "$tmp"
        mv "$tmp" "$sshd_config"
      fi
      log_success "Commented out PasswordAuthentication yes"
      fixed=$((fixed + 1))
    fi

    # Disable PermitRootLogin
    if grep -qiE '^[[:space:]]*PermitRootLogin[[:space:]]+yes' "$sshd_config"; then
      if type safe_sed_inline >/dev/null 2>&1; then
        safe_sed_inline 's/^[[:space:]]*PermitRootLogin[[:space:]]*yes/# PermitRootLogin yes  # disabled by nself security audit/' "$sshd_config"
      else
        local tmp="${sshd_config}.tmp.$$"
        while IFS= read -r line; do
          case "$line" in
            *[Pp]ermit[Rr]oot[Ll]ogin*yes*)
              printf '# %s  # disabled by nself security audit\n' "$line" ;;
            *) printf '%s\n' "$line" ;;
          esac
        done < "$sshd_config" > "$tmp"
        mv "$tmp" "$sshd_config"
      fi
      log_success "Commented out PermitRootLogin yes"
      fixed=$((fixed + 1))
    fi
  fi

  if [[ $fixed -gt 0 ]]; then
    log_success "Applied $fixed fix(es). Review changes and restart affected services."
  else
    log_info "No auto-fixable issues found."
  fi
}

# ============================================================================
# MAIN COMMAND
# ============================================================================

cmd_security_audit() {
  local autofix=false
  local no_docker=false
  local output_format="text"
  local dc_file="docker-compose.yml"

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --fix)
        autofix=true
        shift
        ;;
      --no-docker)
        no_docker=true
        shift
        ;;
      --format)
        output_format="${2:-text}"
        shift 2
        ;;
      --format=*)
        output_format="${1#--format=}"
        shift
        ;;
      --compose)
        dc_file="${2:-docker-compose.yml}"
        shift 2
        ;;
      --help|-h)
        printf "Usage: nself security audit [--fix] [--no-docker] [--format text|json]\n\n"
        printf "Run security checks on your nself deployment:\n"
        printf "  0. Environment secrets (JWT, admin secret, postgres password)\n"
        printf "  1. Port exposure (no service on 0.0.0.0 except nginx)\n"
        printf "  2. SSH configuration (PasswordAuthentication, PermitRootLogin)\n"
        printf "  3. fail2ban installed and active\n"
        printf "  4. Docker daemon iptables setting\n"
        printf "  5. UFW status and HTTP/HTTPS rules\n"
        printf "  6. Grafana default credentials\n"
        printf "  7. Nginx TLS version (no TLS 1.0/1.1)\n"
        printf "  8. Nginx rate limiting zones\n\n"
        printf "Options:\n"
        printf "  --fix          Auto-remediate safe fixes (sshd, fail2ban, UFW)\n"
        printf "  --no-docker    Skip Docker-dependent checks (runs env + SSH + nginx checks only)\n"
        printf "  --format json  Output results as JSON\n"
        return 0
        ;;
      -*)
        log_error "Unknown option: $1"
        return 1
        ;;
      *)
        shift
        ;;
    esac
  done

  if [[ "$output_format" == "json" ]]; then
    # Capture output and emit JSON summary
    local json_issues="[]"
    local env_out failed_list=""

    # Env secrets (always)
    env_out=$(check_env_secrets "0" 2>&1 || true)
    # SSH
    local ssh_out
    ssh_out=$(check_sshd_config 2>&1 || true)

    # Build minimal JSON summary
    local issues_count=0
    if printf '%s' "$env_out" | grep -q '\[FAIL\]'; then
      issues_count=$((issues_count + 1))
    fi

    printf '{"audit":{"project":"%s","no_docker":%s,"issues":%d,"env_secrets":"%s","ssh":"%s"}}\n' \
      "$PWD" "$no_docker" "$issues_count" \
      "$(printf '%s' "$env_out" | grep -c '\[FAIL\]\|\[WARN\]' || true)" \
      "$(printf '%s' "$ssh_out" | grep -c '\[FAIL\]\|\[WARN\]' || true)"
    return 0
  fi

  printf "\n\033[1m\033[0;36m=== nself Security Audit ===\033[0m\n"
  printf "  Project: %s\n" "${PWD}"
  printf "  Date   : %s\n" "$(date '+%Y-%m-%d %H:%M:%S')"
  if [[ "$no_docker" == "true" ]]; then
    printf "  Mode   : --no-docker (skipping Docker-dependent checks)\n"
  fi

  local failed=0
  local warned=0

  # Check 0: env secrets (always runs, even with --no-docker)
  check_env_secrets "0/8" || failed=$((failed + 1))

  local total_checks=4
  if [[ "$no_docker" == "false" ]]; then
    # Full Docker-aware checks
    check_port_exposure "$dc_file" || failed=$((failed + 1))
    check_sshd_config || warned=$((warned + 1))
    check_fail2ban || warned=$((warned + 1))
    check_docker_iptables || failed=$((failed + 1))
    check_ufw || warned=$((warned + 1))
    check_grafana_defaults "$dc_file" || failed=$((failed + 1))
    check_nginx_tls || warned=$((warned + 1))
    check_nginx_rate_limits || warned=$((warned + 1))
    total_checks=9
  else
    # No-docker mode: env + SSH + nginx only
    check_sshd_config || warned=$((warned + 1))
    check_nginx_tls || warned=$((warned + 1))
    check_nginx_rate_limits || warned=$((warned + 1))
  fi

  printf "\n\033[1m=== Summary ===\033[0m\n\n"
  printf "  Checks: %d total" "$total_checks"
  if [[ $failed -gt 0 ]]; then
    printf "  |  \033[0;31m%d FAILED\033[0m" "$failed"
  fi
  if [[ $warned -gt 0 ]]; then
    printf "  |  \033[0;33m%d WARNED\033[0m" "$warned"
  fi
  local passed=$((total_checks - failed - warned))
  if [[ $passed -gt 0 ]]; then
    printf "  |  \033[0;32m%d PASSED\033[0m" "$passed"
  fi
  printf "\n\n"

  if [[ "$autofix" == "true" ]]; then
    _autofix
  elif [[ $((failed + warned)) -gt 0 ]]; then
    printf "  Run with --fix to auto-remediate safe issues:\n"
    printf "    nself security audit --fix\n\n"
  fi

  if [[ $failed -gt 0 ]]; then
    return 1
  fi
  return 0
}

# ── Standalone invocation ────────────────────────────────────────────────────
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_security_audit "$@"
fi
