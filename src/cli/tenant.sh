#!/usr/bin/env bash
#
# nself tenant - Multi-tenant management
#
# Manages tenants, organizations, billing, and tenant isolation
#

set -euo pipefail

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/../.." && pwd)"

# Source libraries - store paths before sourcing
_OUTPUT_PATH="$SCRIPT_DIR/../lib/utils/output.sh"
_CLI_OUTPUT_PATH="$SCRIPT_DIR/../lib/utils/cli-output.sh"
_DOCKER_PATH="$SCRIPT_DIR/../lib/utils/docker.sh"
_ENV_PATH="$SCRIPT_DIR/../lib/utils/env.sh"
_TENANT_CORE_PATH="$SCRIPT_DIR/../lib/tenant/core.sh"

source "$_OUTPUT_PATH"
source "$_CLI_OUTPUT_PATH"
source "$_DOCKER_PATH"
source "$_ENV_PATH" 2>/dev/null || true
source "$_TENANT_CORE_PATH"

# Source billing libraries (lazy loaded when needed)
load_billing_libs() {
  [[ -n "${BILLING_LIBS_LOADED:-}" ]] && return 0
  local LIB_DIR="$ROOT_DIR/src/lib"
  source "$LIB_DIR/utils/display.sh"
  source "$LIB_DIR/utils/validation.sh"
  source "$LIB_DIR/billing/core.sh"
  source "$LIB_DIR/billing/usage.sh"
  source "$LIB_DIR/billing/stripe.sh"
  source "$LIB_DIR/billing/quotas.sh"
  BILLING_LIBS_LOADED=1
}

# Source org libraries (lazy loaded when needed)
load_org_libs() {
  [[ -n "${ORG_LIBS_LOADED:-}" ]] && return 0
  local LIB_DIR="$ROOT_DIR/src/lib"
  source "$LIB_DIR/org/core.sh"
  ORG_LIBS_LOADED=1
}

# Source whitelabel libraries (lazy loaded when needed)
load_whitelabel_libs() {
  [[ -n "${WHITELABEL_LIBS_LOADED:-}" ]] && return 0
  local LIB_DIR="$ROOT_DIR/src/lib"
  source "$LIB_DIR/config/constants.sh"
  source "$LIB_DIR/utils/platform-compat.sh"
  source "$LIB_DIR/whitelabel/branding.sh"
  source "$LIB_DIR/whitelabel/email-templates.sh"
  source "$LIB_DIR/whitelabel/domains.sh"
  source "$LIB_DIR/whitelabel/themes.sh"
  WHITELABEL_LIBS_LOADED=1
}

# ============================================================================
# Usage
# ============================================================================

usage() {
  cat <<EOF
Usage: nself tenant <command> [options]

Multi-tenant management with billing and organization support

TENANT MANAGEMENT:
  init                 Initialize multi-tenancy system
  create <name>        Create a new tenant
  list                 List all tenants
  show <id>            Show tenant details
  suspend <id>         Suspend a tenant
  activate <id>        Activate a suspended tenant
  delete <id>          Delete a tenant (with confirmation)
  stats                Show tenant statistics

MEMBER MANAGEMENT:
  member add <tenant> <user> [role]    Add user to tenant
  member remove <tenant> <user>        Remove user from tenant
  member list <tenant>                 List tenant members

BILLING MANAGEMENT:
  billing usage                        Show usage statistics
  billing invoice <subcommand>         Manage invoices (list, show, download, pay)
  billing subscription <subcommand>    Manage subscriptions (show, upgrade, downgrade)
  billing payment <subcommand>         Manage payment methods (list, add, remove)
  billing quota                        Check quota limits
  billing plan <subcommand>            Manage plans (list, show, compare, current)
  billing export                       Export billing data
  billing customer <subcommand>        Manage customer info (show, update, portal)
  billing webhook                      Test webhook endpoints

ORGANIZATION MANAGEMENT:
  org init                             Initialize organization system
  org create <name>                    Create a new organization
  org list                             List all organizations
  org show <id>                        Show organization details
  org delete <id>                      Delete an organization
  org member <subcommand>              Manage organization members
  org team <subcommand>                Manage teams within organizations
  org role <subcommand>                Manage roles and RBAC
  org permission <subcommand>          Manage permissions

BRANDING & WHITE-LABEL:
  branding <subcommand>                Manage brand customization
  domains <subcommand>                 Configure custom domains and SSL
  email <subcommand>                   Customize email templates
  themes <subcommand>                  Create and manage themes

SETTINGS:
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

  # Check billing usage
  nself tenant billing usage

  # Manage organizations
  nself tenant org create "Engineering"
  nself tenant org member add eng-org user123 admin

  # Manage custom domains
  nself tenant domains add app.example.com
  nself tenant domains verify app.example.com

  # Customize branding
  nself tenant branding set-colors --primary #0066cc

  # Manage email templates
  nself tenant email list

For detailed help on each category:
  nself tenant billing --help
  nself tenant org --help
  nself tenant branding --help
  nself tenant domains --help
  nself tenant email --help
  nself tenant themes --help

NOTE: 'nself billing' and 'nself org' are deprecated.
      Use 'nself tenant billing' and 'nself tenant org' instead.

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
    billing)
      tenant_billing_cmd "$@"
      ;;
    org)
      tenant_org_cmd "$@"
      ;;
    branding)
      tenant_branding_cmd "$@"
      ;;
    domains)
      tenant_domains_cmd "$@"
      ;;
    email)
      tenant_email_cmd "$@"
      ;;
    themes)
      tenant_themes_cmd "$@"
      ;;
    -h | --help | help)
      usage
      exit 0
      ;;
    *)
      cli_error "Unknown command: $command"
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
    cli_error "member command requires a subcommand"
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
      cli_error "Unknown member subcommand: $subcmd"
      exit 1
      ;;
  esac
}

tenant_domain_cmd() {
  if [[ $# -eq 0 ]]; then
    cli_error "domain command requires a subcommand"
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
      cli_error "Unknown domain subcommand: $subcmd"
      exit 1
      ;;
  esac
}

tenant_setting_cmd() {
  if [[ $# -eq 0 ]]; then
    cli_error "setting command requires a subcommand"
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
      cli_error "Unknown setting subcommand: $subcmd"
      exit 1
      ;;
  esac
}

# ============================================================================
# Billing Command Router
# ============================================================================

tenant_billing_cmd() {
  load_billing_libs

  # Show help if requested
  if [[ $# -eq 0 ]] || [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    show_billing_help
    exit 0
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    usage)
      billing_cmd_usage "$@"
      ;;
    invoice)
      billing_cmd_invoice "$@"
      ;;
    subscription)
      billing_cmd_subscription "$@"
      ;;
    payment)
      billing_cmd_payment "$@"
      ;;
    quota)
      billing_cmd_quota "$@"
      ;;
    plan)
      billing_cmd_plan "$@"
      ;;
    export)
      billing_cmd_export "$@"
      ;;
    customer)
      billing_cmd_customer "$@"
      ;;
    webhook)
      billing_cmd_webhook "$@"
      ;;
    *)
      cli_error "Unknown billing subcommand: $subcommand"
      printf "\n"
      show_billing_help
      exit 1
      ;;
  esac
}

# Show billing help
show_billing_help() {
  cat <<'EOF'
nself tenant billing - Billing Management

USAGE:
    nself tenant billing <command> [options]

COMMANDS:
    usage               Show current usage statistics
    invoice             Manage invoices
    subscription        Manage subscriptions
    payment             Manage payment methods
    quota               Check quota limits and usage
    plan                Manage billing plans
    export              Export billing data
    customer            Manage customer information
    webhook             Test webhook endpoints

USAGE COMMANDS:
    nself tenant billing usage                    Show current period usage
    nself tenant billing usage --service=api      Show usage for specific service
    nself tenant billing usage --detailed         Show detailed usage breakdown
    nself tenant billing usage --period=last-month

INVOICE COMMANDS:
    nself tenant billing invoice list             List all invoices
    nself tenant billing invoice show <id>        Show invoice details
    nself tenant billing invoice download <id>    Download invoice PDF
    nself tenant billing invoice pay <id>         Pay unpaid invoice

SUBSCRIPTION COMMANDS:
    nself tenant billing subscription show        Show current subscription
    nself tenant billing subscription plans       List available plans
    nself tenant billing subscription upgrade <plan>
    nself tenant billing subscription downgrade <plan>
    nself tenant billing subscription cancel
    nself tenant billing subscription reactivate

PAYMENT COMMANDS:
    nself tenant billing payment list             List payment methods
    nself tenant billing payment add              Add new payment method
    nself tenant billing payment remove <id>      Remove payment method
    nself tenant billing payment default <id>     Set default payment method

QUOTA COMMANDS:
    nself tenant billing quota                    Show all quota limits
    nself tenant billing quota --service=api      Show specific service quota
    nself tenant billing quota --usage            Show quota with current usage
    nself tenant billing quota --alerts           Show quota alerts

PLAN COMMANDS:
    nself tenant billing plan list                List all available plans
    nself tenant billing plan show <name>         Show plan details
    nself tenant billing plan compare             Compare plans
    nself tenant billing plan current             Show current plan

EXPORT COMMANDS:
    nself tenant billing export usage --format=csv
    nself tenant billing export invoices --year=2026
    nself tenant billing export --all --format=json

For more information: https://docs.nself.org/billing
EOF
}

# Billing command implementations (delegating to billing lib functions)

billing_cmd_usage() {
  local service=""
  local period="current"
  local detailed=false
  local format="table"
  local start_date=""
  local end_date=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --service=*)
        service="${1#*=}"
        shift
        ;;
      --period=*)
        period="${1#*=}"
        shift
        ;;
      --detailed)
        detailed=true
        shift
        ;;
      --format=*)
        format="${1#*=}"
        shift
        ;;
      --start=*)
        start_date="${1#*=}"
        shift
        ;;
      --end=*)
        end_date="${1#*=}"
        shift
        ;;
      --help | -h)
        show_billing_help
        exit 0
        ;;
      *)
        cli_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  # Initialize billing system
  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  # Calculate date range
  local start end
  case "$period" in
    current)
      start=$(date -u +"%Y-%m-01")
      end=$(date -u +"%Y-%m-%d")
      ;;
    last-month)
      if [[ "$(uname)" == "Darwin" ]]; then
        start=$(date -v-1m -u +"%Y-%m-01")
        end=$(date -v-1m -u +"%Y-%m-%d")
      else
        start=$(date -d "last month" -u +"%Y-%m-01")
        end=$(date -d "last month" -u +"%Y-%m-%d")
      fi
      ;;
    custom)
      start="${start_date}"
      end="${end_date}"
      if [[ -z "$start" ]] || [[ -z "$end" ]]; then
        cli_error "Custom period requires --start and --end dates"
        exit 1
      fi
      ;;
    *)
      cli_error "Invalid period: $period"
      exit 1
      ;;
  esac

  cli_info "Usage Report: ${start} to ${end}"
  printf "\n"

  # Get usage data
  if [[ -n "$service" ]]; then
    usage_get_service "$service" "$start" "$end" "$format" "$detailed"
  else
    usage_get_all "$start" "$end" "$format" "$detailed"
  fi
}

billing_cmd_invoice() {
  local subcommand="${1:-list}"
  shift || true

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  case "$subcommand" in
    list)
      stripe_invoice_list "$@"
      ;;
    show)
      if [[ $# -eq 0 ]]; then
        cli_error "Invoice ID required"
        exit 1
      fi
      stripe_invoice_show "$1"
      ;;
    download)
      if [[ $# -eq 0 ]]; then
        cli_error "Invoice ID required"
        exit 1
      fi
      stripe_invoice_download "$1"
      ;;
    pay)
      if [[ $# -eq 0 ]]; then
        cli_error "Invoice ID required"
        exit 1
      fi
      stripe_invoice_pay "$1"
      ;;
    *)
      cli_error "Unknown invoice command: $subcommand"
      printf "Valid commands: list, show, download, pay\n"
      exit 1
      ;;
  esac
}

billing_cmd_subscription() {
  local subcommand="${1:-show}"
  shift || true

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  case "$subcommand" in
    show | current)
      stripe_subscription_show
      ;;
    plans)
      stripe_plans_list
      ;;
    upgrade)
      if [[ $# -eq 0 ]]; then
        cli_error "Plan name required"
        exit 1
      fi
      stripe_subscription_upgrade "$1"
      ;;
    downgrade)
      if [[ $# -eq 0 ]]; then
        cli_error "Plan name required"
        exit 1
      fi
      stripe_subscription_downgrade "$1"
      ;;
    cancel)
      stripe_subscription_cancel "$@"
      ;;
    reactivate)
      stripe_subscription_reactivate
      ;;
    *)
      cli_error "Unknown subscription command: $subcommand"
      printf "Valid commands: show, plans, upgrade, downgrade, cancel, reactivate\n"
      exit 1
      ;;
  esac
}

billing_cmd_payment() {
  local subcommand="${1:-list}"
  shift || true

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  case "$subcommand" in
    list)
      stripe_payment_list
      ;;
    add)
      stripe_payment_add "$@"
      ;;
    remove)
      if [[ $# -eq 0 ]]; then
        cli_error "Payment method ID required"
        exit 1
      fi
      stripe_payment_remove "$1"
      ;;
    default)
      if [[ $# -eq 0 ]]; then
        cli_error "Payment method ID required"
        exit 1
      fi
      stripe_payment_set_default "$1"
      ;;
    *)
      cli_error "Unknown payment command: $subcommand"
      printf "Valid commands: list, add, remove, default\n"
      exit 1
      ;;
  esac
}

billing_cmd_quota() {
  local service=""
  local show_usage=false
  local show_alerts=false
  local format="table"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --service=*)
        service="${1#*=}"
        shift
        ;;
      --usage)
        show_usage=true
        shift
        ;;
      --alerts)
        show_alerts=true
        shift
        ;;
      --format=*)
        format="${1#*=}"
        shift
        ;;
      --help | -h)
        show_billing_help
        exit 0
        ;;
      *)
        cli_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  if [[ "$show_alerts" == "true" ]]; then
    quota_get_alerts "$format"
  elif [[ -n "$service" ]]; then
    quota_get_service "$service" "$show_usage" "$format"
  else
    quota_get_all "$show_usage" "$format"
  fi
}

billing_cmd_plan() {
  local subcommand="${1:-list}"
  shift || true

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  case "$subcommand" in
    list)
      stripe_plans_list "$@"
      ;;
    show)
      if [[ $# -eq 0 ]]; then
        cli_error "Plan name required"
        exit 1
      fi
      stripe_plan_show "$1"
      ;;
    compare)
      stripe_plans_compare "$@"
      ;;
    current)
      stripe_plan_current
      ;;
    *)
      cli_error "Unknown plan command: $subcommand"
      printf "Valid commands: list, show, compare, current\n"
      exit 1
      ;;
  esac
}

billing_cmd_export() {
  local export_type="all"
  local format="json"
  local output_file=""
  local year=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      usage | invoices | subscriptions | payments)
        export_type="$1"
        shift
        ;;
      --all)
        export_type="all"
        shift
        ;;
      --format=*)
        format="${1#*=}"
        shift
        ;;
      --output=*)
        output_file="${1#*=}"
        shift
        ;;
      --year=*)
        year="${1#*=}"
        shift
        ;;
      --help | -h)
        show_billing_help
        exit 0
        ;;
      *)
        cli_error "Unknown option: $1"
        exit 1
        ;;
    esac
  done

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  # Generate default output filename if not provided
  if [[ -z "$output_file" ]]; then
    local timestamp
    timestamp=$(date +"%Y%m%d_%H%M%S")
    output_file="nself_billing_${export_type}_${timestamp}.${format}"
  fi

  cli_info "Exporting billing data to: ${output_file}"

  case "$export_type" in
    usage)
      billing_export_usage "$format" "$output_file" "${year:-}"
      ;;
    invoices)
      billing_export_invoices "$format" "$output_file" "${year:-}"
      ;;
    subscriptions)
      billing_export_subscriptions "$format" "$output_file"
      ;;
    payments)
      billing_export_payments "$format" "$output_file"
      ;;
    all)
      billing_export_all "$format" "$output_file" "${year:-}"
      ;;
    *)
      cli_error "Unknown export type: $export_type"
      exit 1
      ;;
  esac

  cli_success "Export complete: ${output_file}"
}

billing_cmd_customer() {
  local subcommand="${1:-show}"
  shift || true

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  case "$subcommand" in
    show | info)
      stripe_customer_show
      ;;
    update)
      stripe_customer_update "$@"
      ;;
    portal)
      stripe_customer_portal
      ;;
    *)
      cli_error "Unknown customer command: $subcommand"
      printf "Valid commands: show, update, portal\n"
      exit 1
      ;;
  esac
}

billing_cmd_webhook() {
  local subcommand="${1:-test}"
  shift || true

  billing_init || {
    cli_error "Failed to initialize billing system"
    exit 1
  }

  case "$subcommand" in
    test)
      stripe_webhook_test "$@"
      ;;
    list)
      stripe_webhook_list
      ;;
    events)
      stripe_webhook_events "$@"
      ;;
    *)
      cli_error "Unknown webhook command: $subcommand"
      printf "Valid commands: test, list, events\n"
      exit 1
      ;;
  esac
}

# ============================================================================
# Organization Command Router
# ============================================================================

tenant_org_cmd() {
  load_org_libs

  # Show help if requested
  if [[ $# -eq 0 ]] || [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    show_org_help
    exit 0
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
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
      org_member_subcmd "$@"
      ;;
    team)
      org_team_subcmd "$@"
      ;;
    role)
      org_role_subcmd "$@"
      ;;
    permission)
      org_permission_subcmd "$@"
      ;;
    *)
      cli_error "Unknown org subcommand: $subcommand"
      printf "\n"
      show_org_help
      exit 1
      ;;
  esac
}

# Show org help
show_org_help() {
  cat <<EOF
nself tenant org - Organization Management

USAGE:
    nself tenant org <command> [options]

ORGANIZATION MANAGEMENT:
  init                    Initialize organization system
  create <name>           Create a new organization
  list                    List all organizations
  show <id>               Show organization details
  delete <id>             Delete an organization

MEMBER MANAGEMENT:
  member add <org> <user> [role]     Add member to organization
  member remove <org> <user>         Remove member from organization
  member list <org>                  List organization members
  member role <org> <user> <role>    Change member role

TEAM MANAGEMENT:
  team create <org> <name>           Create a team
  team list <org>                    List teams in organization
  team show <team>                   Show team details
  team delete <team>                 Delete a team
  team add <team> <user> [role]      Add user to team
  team remove <team> <user>          Remove user from team

ROLE & PERMISSION MANAGEMENT:
  role create <org> <name>           Create custom role
  role list <org>                    List roles
  role assign <org> <user> <role>    Assign role to user
  role revoke <org> <user> <role>    Revoke role from user
  permission grant <role> <perm>     Grant permission to role
  permission revoke <role> <perm>    Revoke permission from role
  permission list                    List all permissions

EXAMPLES:
  # Initialize organization system
  nself tenant org init

  # Create organization
  nself tenant org create "Acme Corp" --slug acme

  # Manage members
  nself tenant org member add acme user123 admin
  nself tenant org member list acme

  # Create and manage teams
  nself tenant org team create acme "Engineering"
  nself tenant org team add engineering user456 lead

  # Role management
  nself tenant org role create acme "Developer"
  nself tenant org role assign acme user789 Developer
  nself tenant org permission grant Developer tenant.create

EOF
}

# Organization subcommand routers

org_member_subcmd() {
  if [[ $# -eq 0 ]]; then
    cli_error "member command requires a subcommand"
    show_org_help
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
      cli_error "Unknown member subcommand: $subcmd"
      exit 1
      ;;
  esac
}

org_team_subcmd() {
  if [[ $# -eq 0 ]]; then
    cli_error "team command requires a subcommand"
    show_org_help
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
      cli_error "Unknown team subcommand: $subcmd"
      exit 1
      ;;
  esac
}

org_role_subcmd() {
  if [[ $# -eq 0 ]]; then
    cli_error "role command requires a subcommand"
    show_org_help
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
      cli_error "Unknown role subcommand: $subcmd"
      exit 1
      ;;
  esac
}

org_permission_subcmd() {
  if [[ $# -eq 0 ]]; then
    cli_error "permission command requires a subcommand"
    show_org_help
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
      cli_error "Unknown permission subcommand: $subcmd"
      exit 1
      ;;
  esac
}

# ============================================================================
# Branding Command Router
# ============================================================================

tenant_branding_cmd() {
  load_whitelabel_libs

  if [[ $# -eq 0 ]]; then
    cli_error "branding command requires a subcommand"
    printf "\n"
    cat <<EOF
Branding commands:
  create <name>              Create new brand
  set-colors                 Set brand colors
  set-fonts                  Set brand fonts
  upload-logo <path>         Upload brand logo
  set-css <path>             Set custom CSS
  preview                    Preview branding

Run 'nself tenant branding <subcommand> --help' for more information.
EOF
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    create)
      if [[ $# -eq 0 ]]; then
        cli_error "Brand name required"
        exit 1
      fi
      create_brand "$1"
      ;;
    set-colors)
      set_brand_colors "$@"
      ;;
    set-fonts)
      set_brand_fonts "$@"
      ;;
    upload-logo)
      if [[ $# -eq 0 ]]; then
        cli_error "Logo path required"
        exit 1
      fi
      upload_brand_logo "$1"
      ;;
    set-css)
      if [[ $# -eq 0 ]]; then
        cli_error "CSS file path required"
        exit 1
      fi
      set_custom_css "$1"
      ;;
    preview)
      preview_branding "$@"
      ;;
    --help | -h)
      bash "$SCRIPT_DIR/whitelabel.sh" branding --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
      ;;
    *)
      cli_error "Unknown branding subcommand: $subcommand"
      exit 1
      ;;
  esac
}

# ============================================================================
# Domains Command Router
# ============================================================================

tenant_domains_cmd() {
  load_whitelabel_libs

  if [[ $# -eq 0 ]]; then
    cli_error "domains command requires a subcommand"
    printf "\n"
    cat <<EOF
Domain commands:
  add <domain>               Add custom domain
  verify <domain>            Verify domain ownership
  ssl <domain>               Provision SSL certificate
  health <domain>            Check domain health
  remove <domain>            Remove custom domain

Run 'nself tenant domains <subcommand> --help' for more information.
EOF
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    add)
      if [[ $# -eq 0 ]]; then
        cli_error "Domain name required"
        exit 1
      fi
      add_custom_domain "$1"
      ;;
    verify)
      if [[ $# -eq 0 ]]; then
        cli_error "Domain name required"
        exit 1
      fi
      verify_domain "$1"
      ;;
    ssl)
      if [[ $# -eq 0 ]]; then
        cli_error "Domain name required"
        exit 1
      fi
      provision_ssl "$@"
      ;;
    health)
      if [[ $# -eq 0 ]]; then
        cli_error "Domain name required"
        exit 1
      fi
      check_domain_health "$1"
      ;;
    remove)
      if [[ $# -eq 0 ]]; then
        cli_error "Domain name required"
        exit 1
      fi
      remove_custom_domain "$1"
      ;;
    --help | -h)
      bash "$SCRIPT_DIR/whitelabel.sh" domain --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
      ;;
    *)
      cli_error "Unknown domains subcommand: $subcommand"
      exit 1
      ;;
  esac
}

# ============================================================================
# Email Command Router
# ============================================================================

tenant_email_cmd() {
  load_whitelabel_libs

  if [[ $# -eq 0 ]]; then
    cli_error "email command requires a subcommand"
    printf "\n"
    cat <<EOF
Email template commands:
  list                       List all email templates
  edit <template>            Edit email template
  preview <template>         Preview email template
  test <template> <email>    Send test email
  set-language <code>        Set email language

Run 'nself tenant email <subcommand> --help' for more information.
EOF
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    list)
      list_email_templates "$@"
      ;;
    edit)
      if [[ $# -eq 0 ]]; then
        cli_error "Template name required"
        exit 1
      fi
      edit_email_template "$1"
      ;;
    preview)
      if [[ $# -eq 0 ]]; then
        cli_error "Template name required"
        exit 1
      fi
      preview_email_template "$1"
      ;;
    test)
      if [[ $# -lt 2 ]]; then
        cli_error "Template name and email address required"
        exit 1
      fi
      test_email_template "$1" "$2"
      ;;
    set-language)
      if [[ $# -eq 0 ]]; then
        cli_error "Language code required"
        exit 1
      fi
      set_email_language "$1"
      ;;
    --help | -h)
      bash "$SCRIPT_DIR/whitelabel.sh" email --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
      ;;
    *)
      cli_error "Unknown email subcommand: $subcommand"
      exit 1
      ;;
  esac
}

# ============================================================================
# Themes Command Router
# ============================================================================

tenant_themes_cmd() {
  load_whitelabel_libs

  if [[ $# -eq 0 ]]; then
    cli_error "themes command requires a subcommand"
    printf "\n"
    cat <<EOF
Theme commands:
  create <name>              Create new theme
  edit <name>                Edit theme
  activate <name>            Activate theme
  preview <name>             Preview theme
  export <name>              Export theme
  import <path>              Import theme

Run 'nself tenant themes <subcommand> --help' for more information.
EOF
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    create)
      if [[ $# -eq 0 ]]; then
        cli_error "Theme name required"
        exit 1
      fi
      create_theme "$1"
      ;;
    edit)
      if [[ $# -eq 0 ]]; then
        cli_error "Theme name required"
        exit 1
      fi
      edit_theme "$1"
      ;;
    activate)
      if [[ $# -eq 0 ]]; then
        cli_error "Theme name required"
        exit 1
      fi
      activate_theme "$1"
      ;;
    preview)
      if [[ $# -eq 0 ]]; then
        cli_error "Theme name required"
        exit 1
      fi
      preview_theme "$1"
      ;;
    export)
      if [[ $# -eq 0 ]]; then
        cli_error "Theme name required"
        exit 1
      fi
      export_theme "$1"
      ;;
    import)
      if [[ $# -eq 0 ]]; then
        cli_error "Theme file path required"
        exit 1
      fi
      import_theme "$1"
      ;;
    --help | -h)
      bash "$SCRIPT_DIR/whitelabel.sh" theme --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
      ;;
    *)
      cli_error "Unknown themes subcommand: $subcommand"
      exit 1
      ;;
  esac
}

# Run main
main "$@"
