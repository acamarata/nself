#!/usr/bin/env bash
# audit.sh - Audit log CLI
# Part of nself v0.6.0 - Phase 2

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
[[ -f "$SCRIPT_DIR/../lib/auth/audit-log.sh" ]] && source "$SCRIPT_DIR/../lib/auth/audit-log.sh"

cmd_audit() {
  case "${1:-help}" in
    init) audit_init && printf "✓ Audit logging initialized\n" ;;
    query)
      shift
      local filters="${1:-{}}"
      local limit="${2:-100}"
      audit_query "$filters" "$limit" | jq '.'
      ;;
    help|--help|-h)
      cat <<'HELP'
nself audit - Audit log management

COMMANDS:
  init                Initialize audit logging
  query [filters]     Query audit logs

EXAMPLES:
  nself audit init
  nself audit query
  nself audit query '{"event_type":"user.login"}' 50
  nself audit query '{"actor_id":"<user-uuid>"}'
HELP
      ;;
    *) echo "ERROR: Unknown command" >&2; return 1 ;;
  esac
}

export -f cmd_audit
[[ "${BASH_SOURCE[0]}" == "${0}" ]] && cmd_audit "$@"
