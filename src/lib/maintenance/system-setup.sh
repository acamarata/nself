#!/usr/bin/env bash
# system-setup.sh - System-level maintenance setup (logrotate, journald)
# Part of nself SysOps - Bash 3.2 compatible
# These functions configure host-level log management to prevent disk exhaustion.

set -euo pipefail

# Get the directory where this script is located
_SYS_SETUP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
_SYS_SETUP_LIB_ROOT="$(dirname "$_SYS_SETUP_DIR")"

# Source dependencies
source "$_SYS_SETUP_LIB_ROOT/utils/display.sh" 2>/dev/null || true
source "$_SYS_SETUP_LIB_ROOT/utils/platform-compat.sh" 2>/dev/null || true

# T-2462: Install nginx log rotation config
# Creates /etc/logrotate.d/nself-nginx for rotating nginx access/error logs
setup_nginx_logrotate() {
  local nginx_log_dir="${1:-./logs/nginx}"
  local dry_run="${2:-false}"
  local rotate_count="${3:-14}"
  local max_size="${4:-50M}"

  # Only run on Linux (logrotate is Linux-specific)
  if [[ "$(uname)" == "Darwin" ]]; then
    printf "\033[33m%s\033[0m Skipping nginx logrotate setup (macOS, not applicable)\n" "WARNING"
    return 0
  fi

  if ! command -v logrotate >/dev/null 2>&1; then
    printf "\033[33m%s\033[0m logrotate not installed. Install with: sudo apt install logrotate\n" "WARNING"
    return 1
  fi

  local config_path="/etc/logrotate.d/nself-nginx"
  local config_content
  config_content=$(cat <<LOGROTATE
# nself nginx log rotation
# Installed by: nself setup
# Rotates nginx access and error logs to prevent disk exhaustion

${nginx_log_dir}/*.log {
    daily
    missingok
    rotate ${rotate_count}
    maxsize ${max_size}
    compress
    delaycompress
    notifempty
    create 0640 root adm
    sharedscripts
    postrotate
        # Signal nginx to reopen log files
        if [ -f /var/run/nginx.pid ]; then
            kill -USR1 \$(cat /var/run/nginx.pid) 2>/dev/null || true
        fi
        # Also signal Docker nginx container if running
        docker kill --signal=USR1 \$(docker ps --filter 'name=nginx' --format '{{.Names}}' | head -1) 2>/dev/null || true
    endscript
}
LOGROTATE
)

  if [[ "$dry_run" == "true" ]]; then
    printf "\033[33m%s\033[0m DRY RUN - would write to %s:\n" "WARNING" "$config_path"
    printf "%s\n" "$config_content"
    return 0
  fi

  # Requires root
  if [[ "$(id -u)" -ne 0 ]]; then
    printf "\033[34m%s\033[0m Writing nginx logrotate config (requires sudo)...\n" "INFO"
    printf '%s\n' "$config_content" | sudo tee "$config_path" > /dev/null
  else
    printf '%s\n' "$config_content" > "$config_path"
  fi

  sudo chmod 644 "$config_path" 2>/dev/null || true

  printf "\033[32m%s\033[0m Nginx logrotate config installed: %s\n" "SUCCESS" "$config_path"

  # Test the config
  if sudo logrotate --debug "$config_path" >/dev/null 2>&1; then
    printf "\033[32m%s\033[0m Config validated successfully\n" "SUCCESS"
  else
    printf "\033[33m%s\033[0m Config may have issues (run: sudo logrotate --debug %s)\n" "WARNING" "$config_path"
  fi
}

# T-2463: Configure journald size limit
# Writes /etc/systemd/journald.conf.d/nself.conf with SystemMaxUse
# Only runs on systemd systems.
setup_journald_limit() {
  local max_use="${1:-500M}"
  local dry_run="${2:-false}"

  # Only run on Linux with systemd
  if [[ "$(uname)" == "Darwin" ]]; then
    printf "\033[33m%s\033[0m Skipping journald setup (macOS, not applicable)\n" "WARNING"
    return 0
  fi

  if ! command -v systemctl >/dev/null 2>&1; then
    printf "\033[33m%s\033[0m systemd not detected, skipping journald configuration\n" "WARNING"
    return 0
  fi

  if ! systemctl is-active systemd-journald >/dev/null 2>&1; then
    printf "\033[33m%s\033[0m journald not active, skipping configuration\n" "WARNING"
    return 0
  fi

  local config_dir="/etc/systemd/journald.conf.d"
  local config_path="${config_dir}/nself.conf"

  local config_content
  config_content=$(cat <<JOURNALD
# nself journald configuration
# Installed by: nself setup
# Limits journal disk usage to prevent disk exhaustion

[Journal]
SystemMaxUse=${max_use}
SystemMaxFileSize=50M
MaxRetentionSec=1month
Compress=yes
JOURNALD
)

  if [[ "$dry_run" == "true" ]]; then
    printf "\033[33m%s\033[0m DRY RUN - would write to %s:\n" "WARNING" "$config_path"
    printf "%s\n" "$config_content"
    return 0
  fi

  # Create directory and write config
  if [[ "$(id -u)" -ne 0 ]]; then
    sudo mkdir -p "$config_dir"
    printf '%s\n' "$config_content" | sudo tee "$config_path" > /dev/null
    sudo chmod 644 "$config_path"
  else
    mkdir -p "$config_dir"
    printf '%s\n' "$config_content" > "$config_path"
    chmod 644 "$config_path"
  fi

  printf "\033[32m%s\033[0m Journald config installed: %s\n" "SUCCESS" "$config_path"

  # Restart journald to apply
  printf "\033[34m%s\033[0m Restarting journald to apply new limits...\n" "INFO"
  sudo systemctl restart systemd-journald 2>/dev/null || {
    printf "\033[33m%s\033[0m Could not restart journald (manual restart needed: sudo systemctl restart systemd-journald)\n" "WARNING"
  }

  # Verify current journal size
  if command -v journalctl >/dev/null 2>&1; then
    printf "\033[34m%s\033[0m Current journal disk usage:\n" "INFO"
    journalctl --disk-usage 2>/dev/null || true
  fi
}

# Run all system setup tasks
setup_system_maintenance() {
  local dry_run="${1:-false}"

  printf "\033[34m%s\033[0m Setting up system maintenance...\n\n" "INFO"

  setup_nginx_logrotate "./logs/nginx" "$dry_run"
  printf "\n"
  setup_journald_limit "500M" "$dry_run"

  printf "\n\033[32m%s\033[0m System maintenance setup complete\n" "SUCCESS"
}

# Export functions
export -f setup_nginx_logrotate 2>/dev/null || true
export -f setup_journald_limit 2>/dev/null || true
export -f setup_system_maintenance 2>/dev/null || true
