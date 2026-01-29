#!/usr/bin/env bash
#
# nself tenant - Multi-tenant management
#
# Manages tenants, organizations, and tenant isolation
#

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source libraries
source "$SCRIPT_DIR/../lib/utils/output.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/config/env.sh"
source "$SCRIPT_DIR/../lib/tenant/core.sh"

# ============================================================================
# Usage
# ============================================================================

usage() {
    cat << EOF
Usage: nself tenant <command> [options]

Multi-tenant management commands

COMMANDS:
  init                 Initialize multi-tenancy system
  create <name>        Create a new tenant
  list                 List all tenants
  show <id>            Show tenant details
  suspend <id>         Suspend a tenant
  activate <id>        Activate a suspended tenant
  delete <id>          Delete a tenant (with confirmation)
  stats                Show tenant statistics

  # Member management
  member add <tenant> <user> [role]    Add user to tenant
  member remove <tenant> <user>        Remove user from tenant
  member list <tenant>                 List tenant members

  # Domain management
  domain add <tenant> <domain>         Add custom domain
  domain verify <tenant> <domain>      Verify domain ownership
  domain remove <tenant> <domain>      Remove domain
  domain list <tenant>                 List tenant domains

  # Settings
  setting set <tenant> <key> <value>   Set tenant setting
  setting get <tenant> <key>           Get tenant setting
  setting list <tenant>                List tenant settings

OPTIONS:
  -h, --help           Show this help message
  --json               Output in JSON format
  --slug <slug>        Tenant slug (for create)
  --plan <plan>        Plan ID (free, pro, enterprise)
  --owner <user_id>    Owner user ID

EXAMPLES:
  # Initialize multi-tenancy
  nself tenant init

  # Create tenant
  nself tenant create "Acme Corp" --slug acme --plan pro

  # List tenants
  nself tenant list
  nself tenant list --json

  # Manage members
  nself tenant member add acme user123 admin
  nself tenant member list acme

  # Manage domains
  nself tenant domain add acme acme.example.com
  nself tenant domain verify acme acme.example.com

  # Manage settings
  nself tenant setting set acme branding.logo_url "https://..."
  nself tenant setting get acme branding.logo_url

EOF
}

# ============================================================================
# Main Command Router
# ============================================================================

main() {
    if [[ $# -eq 0 ]]; then
        usage
        exit 1
    fi

    local command="$1"
    shift

    case "$command" in
        init)
            tenant_init "$@"
            ;;
        create)
            tenant_create "$@"
            ;;
        list)
            tenant_list "$@"
            ;;
        show)
            tenant_show "$@"
            ;;
        suspend)
            tenant_suspend "$@"
            ;;
        activate)
            tenant_activate "$@"
            ;;
        delete)
            tenant_delete "$@"
            ;;
        stats)
            tenant_stats "$@"
            ;;
        member)
            tenant_member_cmd "$@"
            ;;
        domain)
            tenant_domain_cmd "$@"
            ;;
        setting)
            tenant_setting_cmd "$@"
            ;;
        -h|--help|help)
            usage
            exit 0
            ;;
        *)
            error "Unknown command: $command"
            printf "\n"
            usage
            exit 1
            ;;
    esac
}

# ============================================================================
# Subcommand Routers
# ============================================================================

tenant_member_cmd() {
    if [[ $# -eq 0 ]]; then
        error "member command requires a subcommand"
        printf "\n"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        add)
            tenant_member_add "$@"
            ;;
        remove)
            tenant_member_remove "$@"
            ;;
        list)
            tenant_member_list "$@"
            ;;
        *)
            error "Unknown member subcommand: $subcmd"
            exit 1
            ;;
    esac
}

tenant_domain_cmd() {
    if [[ $# -eq 0 ]]; then
        error "domain command requires a subcommand"
        printf "\n"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        add)
            tenant_domain_add "$@"
            ;;
        verify)
            tenant_domain_verify "$@"
            ;;
        remove)
            tenant_domain_remove "$@"
            ;;
        list)
            tenant_domain_list "$@"
            ;;
        *)
            error "Unknown domain subcommand: $subcmd"
            exit 1
            ;;
    esac
}

tenant_setting_cmd() {
    if [[ $# -eq 0 ]]; then
        error "setting command requires a subcommand"
        printf "\n"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        set)
            tenant_setting_set "$@"
            ;;
        get)
            tenant_setting_get "$@"
            ;;
        list)
            tenant_setting_list "$@"
            ;;
        *)
            error "Unknown setting subcommand: $subcmd"
            exit 1
            ;;
    esac
}

# Run main
main "$@"
