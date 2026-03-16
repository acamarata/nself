#!/usr/bin/env bash
# security.sh — nself security command
# Routes subcommands to the appropriate security modules.
# Bash 3.2+ compatible.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

cmd_security() {
  local subcommand="${1:-help}"
  shift || true

  case "$subcommand" in
    audit)
      # Full on-demand security scan (8 checks)
      if [[ -f "$SCRIPT_DIR/security_audit.sh" ]]; then
        source "$SCRIPT_DIR/security_audit.sh"
        cmd_security_audit "$@"
      else
        printf "\033[0;31m[ERROR]\033[0m Security audit module not found\n" >&2
        return 1
      fi
      ;;
    help | --help | -h)
      _security_usage
      ;;
    -*)
      printf "\033[0;31m[ERROR]\033[0m Unknown option: %s\n" "$subcommand" >&2
      _security_usage
      return 1
      ;;
    *)
      printf "\033[0;31m[ERROR]\033[0m Unknown subcommand: %s\n" "$subcommand" >&2
      _security_usage
      return 1
      ;;
  esac
}

_security_usage() {
  printf "Usage: nself security <subcommand> [options]\n\n"
  printf "SUBCOMMANDS:\n"
  printf "  audit         Run full on-demand security scan (8 checks)\n"
  printf "  audit --fix   Auto-remediate safe issues (fail2ban, UFW, sshd)\n"
  printf "  help          Show this help\n\n"
  printf "EXAMPLES:\n"
  printf "  nself security audit          # Full scan\n"
  printf "  nself security audit --fix    # Scan + auto-fix\n\n"
  printf "CHECKS (audit):\n"
  printf "  1. Port exposure — no non-nginx service on 0.0.0.0\n"
  printf "  2. SSH configuration — PasswordAuthentication, PermitRootLogin\n"
  printf "  3. fail2ban — installed, running, SSH jail active\n"
  printf "  4. Docker iptables — daemon.json not disabling iptables\n"
  printf "  5. UFW firewall — active with HTTP/HTTPS rules\n"
  printf "  6. Grafana — not using default credentials\n"
  printf "  7. Nginx TLS — no TLS 1.0/1.1\n"
  printf "  8. Nginx rate limiting — limit_req_zone defined\n"
}

export -f cmd_security

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_security "$@"
fi
