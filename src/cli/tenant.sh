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
source "$SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$SCRIPT_DIR/../lib/tenant/core.sh"

# Source billing libraries (lazy loaded when needed)
load_billing_libs() {
    [[ -n "${BILLING_LIBS_LOADED:-}" ]] && return 0
    source "$SCRIPT_DIR/../lib/utils/display.sh"
    source "$SCRIPT_DIR/../lib/utils/validation.sh"
    source "$SCRIPT_DIR/../lib/billing/core.sh"
    source "$SCRIPT_DIR/../lib/billing/usage.sh"
    source "$SCRIPT_DIR/../lib/billing/stripe.sh"
    source "$SCRIPT_DIR/../lib/billing/quotas.sh"
    BILLING_LIBS_LOADED=1
}

# Source whitelabel libraries (lazy loaded when needed)
load_whitelabel_libs() {
    [[ -n "${WHITELABEL_LIBS_LOADED:-}" ]] && return 0
    source "$SCRIPT_DIR/../lib/config/constants.sh"
    source "$SCRIPT_DIR/../lib/utils/platform-compat.sh"
    source "$SCRIPT_DIR/../lib/whitelabel/branding.sh"
    source "$SCRIPT_DIR/../lib/whitelabel/email-templates.sh"
    source "$SCRIPT_DIR/../lib/whitelabel/domains.sh"
    source "$SCRIPT_DIR/../lib/whitelabel/themes.sh"
    WHITELABEL_LIBS_LOADED=1
}

# ============================================================================
# Usage
# ============================================================================

usage() {
    cat << EOF
Usage: nself tenant <command> [options]

Multi-tenant management commands

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

BILLING:
  billing usage                        Show usage statistics
  billing invoice <subcommand>         Manage invoices (list, show, download, pay)
  billing subscription <subcommand>    Manage subscriptions (show, upgrade, downgrade)
  billing payment <subcommand>         Manage payment methods (list, add, remove)
  billing quota                        Check quota limits
  billing plan <subcommand>            Manage plans (list, show, compare, current)
  billing export                       Export billing data
  billing customer <subcommand>        Manage customer info (show, update, portal)

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

  # Manage custom domains
  nself tenant domains add app.example.com
  nself tenant domains verify app.example.com

  # Customize branding
  nself tenant branding set-colors --primary #0066cc

  # Manage email templates
  nself tenant email list

For detailed help on each category:
  nself tenant billing --help
  nself tenant branding --help
  nself tenant domains --help
  nself tenant email --help
  nself tenant themes --help

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

# ============================================================================
# Billing Command Router
# ============================================================================

tenant_billing_cmd() {
    load_billing_libs

    # Show help if requested
    if [[ $# -eq 0 ]] || [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
        bash "$SCRIPT_DIR/billing.sh" --help
        exit 0
    fi

    # Delegate to billing CLI
    bash "$SCRIPT_DIR/billing.sh" "$@"
}

# ============================================================================
# Branding Command Router
# ============================================================================

tenant_branding_cmd() {
    load_whitelabel_libs

    if [[ $# -eq 0 ]]; then
        error "branding command requires a subcommand"
        printf "\n"
        cat << EOF
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
                error "Brand name required"
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
                error "Logo path required"
                exit 1
            fi
            upload_brand_logo "$1"
            ;;
        set-css)
            if [[ $# -eq 0 ]]; then
                error "CSS file path required"
                exit 1
            fi
            set_custom_css "$1"
            ;;
        preview)
            preview_branding "$@"
            ;;
        --help|-h)
            bash "$SCRIPT_DIR/whitelabel.sh" branding --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
            ;;
        *)
            error "Unknown branding subcommand: $subcommand"
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
        error "domains command requires a subcommand"
        printf "\n"
        cat << EOF
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
                error "Domain name required"
                exit 1
            fi
            add_custom_domain "$1"
            ;;
        verify)
            if [[ $# -eq 0 ]]; then
                error "Domain name required"
                exit 1
            fi
            verify_domain "$1"
            ;;
        ssl)
            if [[ $# -eq 0 ]]; then
                error "Domain name required"
                exit 1
            fi
            provision_ssl "$@"
            ;;
        health)
            if [[ $# -eq 0 ]]; then
                error "Domain name required"
                exit 1
            fi
            check_domain_health "$1"
            ;;
        remove)
            if [[ $# -eq 0 ]]; then
                error "Domain name required"
                exit 1
            fi
            remove_custom_domain "$1"
            ;;
        --help|-h)
            bash "$SCRIPT_DIR/whitelabel.sh" domain --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
            ;;
        *)
            error "Unknown domains subcommand: $subcommand"
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
        error "email command requires a subcommand"
        printf "\n"
        cat << EOF
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
                error "Template name required"
                exit 1
            fi
            edit_email_template "$1"
            ;;
        preview)
            if [[ $# -eq 0 ]]; then
                error "Template name required"
                exit 1
            fi
            preview_email_template "$1"
            ;;
        test)
            if [[ $# -lt 2 ]]; then
                error "Template name and email address required"
                exit 1
            fi
            test_email_template "$1" "$2"
            ;;
        set-language)
            if [[ $# -eq 0 ]]; then
                error "Language code required"
                exit 1
            fi
            set_email_language "$1"
            ;;
        --help|-h)
            bash "$SCRIPT_DIR/whitelabel.sh" email --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
            ;;
        *)
            error "Unknown email subcommand: $subcommand"
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
        error "themes command requires a subcommand"
        printf "\n"
        cat << EOF
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
                error "Theme name required"
                exit 1
            fi
            create_theme "$1"
            ;;
        edit)
            if [[ $# -eq 0 ]]; then
                error "Theme name required"
                exit 1
            fi
            edit_theme "$1"
            ;;
        activate)
            if [[ $# -eq 0 ]]; then
                error "Theme name required"
                exit 1
            fi
            activate_theme "$1"
            ;;
        preview)
            if [[ $# -eq 0 ]]; then
                error "Theme name required"
                exit 1
            fi
            preview_theme "$1"
            ;;
        export)
            if [[ $# -eq 0 ]]; then
                error "Theme name required"
                exit 1
            fi
            export_theme "$1"
            ;;
        import)
            if [[ $# -eq 0 ]]; then
                error "Theme file path required"
                exit 1
            fi
            import_theme "$1"
            ;;
        --help|-h)
            bash "$SCRIPT_DIR/whitelabel.sh" theme --help 2>/dev/null || printf "See whitelabel CLI for detailed help\n"
            ;;
        *)
            error "Unknown themes subcommand: $subcommand"
            exit 1
            ;;
    esac
}

# Run main
main "$@"
