#!/usr/bin/env bash
#
# nself org - Organization management
#
# Manages organizations, teams, and RBAC
#

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source libraries
source "$SCRIPT_DIR/../lib/utils/output.sh"
source "$SCRIPT_DIR/../lib/utils/docker.sh"
source "$SCRIPT_DIR/../lib/config/env.sh"
source "$SCRIPT_DIR/../lib/org/core.sh"

# ============================================================================
# Usage
# ============================================================================

usage() {
    cat << EOF
Usage: nself org <command> [options]

Organization and team management commands

COMMANDS:
  # Organization management
  init                    Initialize organization system
  create <name>           Create a new organization
  list                    List all organizations
  show <id>               Show organization details
  delete <id>             Delete an organization

  # Member management
  member add <org> <user> [role]     Add member to organization
  member remove <org> <user>         Remove member from organization
  member list <org>                  List organization members
  member role <org> <user> <role>    Change member role

  # Team management
  team create <org> <name>           Create a team
  team list <org>                    List teams in organization
  team show <team>                   Show team details
  team delete <team>                 Delete a team
  team add <team> <user> [role]      Add user to team
  team remove <team> <user>          Remove user from team

  # Role & Permission management
  role create <org> <name>           Create custom role
  role list <org>                    List roles
  role assign <org> <user> <role>    Assign role to user
  role revoke <org> <user> <role>    Revoke role from user
  permission grant <role> <perm>     Grant permission to role
  permission revoke <role> <perm>    Revoke permission from role
  permission list                    List all permissions

OPTIONS:
  -h, --help           Show this help message
  --json               Output in JSON format
  --slug <slug>        Organization slug (for create)
  --owner <user_id>    Owner user ID

EXAMPLES:
  # Initialize organization system
  nself org init

  # Create organization
  nself org create "Acme Corp" --slug acme

  # Manage members
  nself org member add acme user123 admin
  nself org member list acme

  # Create and manage teams
  nself org team create acme "Engineering"
  nself org team add engineering user456 lead

  # Role management
  nself org role create acme "Developer"
  nself org role assign acme user789 Developer
  nself org permission grant Developer tenant.create

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
            org_init "$@"
            ;;
        create)
            org_create "$@"
            ;;
        list)
            org_list "$@"
            ;;
        show)
            org_show "$@"
            ;;
        delete)
            org_delete "$@"
            ;;
        member)
            org_member_cmd "$@"
            ;;
        team)
            org_team_cmd "$@"
            ;;
        role)
            org_role_cmd "$@"
            ;;
        permission)
            org_permission_cmd "$@"
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

org_member_cmd() {
    if [[ $# -eq 0 ]]; then
        error "member command requires a subcommand"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        add)
            org_member_add "$@"
            ;;
        remove)
            org_member_remove "$@"
            ;;
        list)
            org_member_list "$@"
            ;;
        role)
            org_member_change_role "$@"
            ;;
        *)
            error "Unknown member subcommand: $subcmd"
            exit 1
            ;;
    esac
}

org_team_cmd() {
    if [[ $# -eq 0 ]]; then
        error "team command requires a subcommand"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        create)
            org_team_create "$@"
            ;;
        list)
            org_team_list "$@"
            ;;
        show)
            org_team_show "$@"
            ;;
        delete)
            org_team_delete "$@"
            ;;
        add)
            org_team_add_member "$@"
            ;;
        remove)
            org_team_remove_member "$@"
            ;;
        *)
            error "Unknown team subcommand: $subcmd"
            exit 1
            ;;
    esac
}

org_role_cmd() {
    if [[ $# -eq 0 ]]; then
        error "role command requires a subcommand"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        create)
            org_role_create "$@"
            ;;
        list)
            org_role_list "$@"
            ;;
        assign)
            org_role_assign "$@"
            ;;
        revoke)
            org_role_revoke "$@"
            ;;
        *)
            error "Unknown role subcommand: $subcmd"
            exit 1
            ;;
    esac
}

org_permission_cmd() {
    if [[ $# -eq 0 ]]; then
        error "permission command requires a subcommand"
        usage
        exit 1
    fi

    local subcmd="$1"
    shift

    case "$subcmd" in
        grant)
            org_permission_grant "$@"
            ;;
        revoke)
            org_permission_revoke "$@"
            ;;
        list)
            org_permission_list "$@"
            ;;
        *)
            error "Unknown permission subcommand: $subcmd"
            exit 1
            ;;
    esac
}

# Run main
main "$@"
