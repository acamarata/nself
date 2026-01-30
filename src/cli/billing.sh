#!/usr/bin/env bash
#
# nself billing.sh - Billing Management CLI
# Part of nself v0.9.0 - Sprint 13: Billing Integration & Usage Tracking
#
# Comprehensive billing management with Stripe integration, usage metering,
# and quota enforcement.
#
# Usage: nself billing <command> [options]
# PREFERRED: nself tenant billing <command> [options]
#
# This command can be called directly or via 'nself tenant billing'.
# Both forms work identically.
#

set -euo pipefail

# Script directory resolution
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies (use relative paths from src/cli)
source "${SCRIPT_DIR}/../lib/utils/display.sh"
source "${SCRIPT_DIR}/../lib/utils/validation.sh"
source "${SCRIPT_DIR}/../lib/billing/core.sh"
source "${SCRIPT_DIR}/../lib/billing/usage.sh"
source "${SCRIPT_DIR}/../lib/billing/stripe.sh"
source "${SCRIPT_DIR}/../lib/billing/quotas.sh"

# Display help information
show_help() {
    cat << 'EOF'
nself billing - Billing Management CLI

USAGE:
    nself billing <command> [options]
    nself tenant billing <command> [options]  (Preferred)

NOTE: These commands are now part of nself tenant. Use 'nself tenant billing'
      for consistency with other multi-tenant features.

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
    nself billing usage                    Show current period usage
    nself billing usage --service=api      Show usage for specific service
    nself billing usage --detailed         Show detailed usage breakdown
    nself billing usage --period=last-month

INVOICE COMMANDS:
    nself billing invoice list             List all invoices
    nself billing invoice show <id>        Show invoice details
    nself billing invoice download <id>    Download invoice PDF
    nself billing invoice pay <id>         Pay unpaid invoice

SUBSCRIPTION COMMANDS:
    nself billing subscription show        Show current subscription
    nself billing subscription plans       List available plans
    nself billing subscription upgrade <plan>
    nself billing subscription downgrade <plan>
    nself billing subscription cancel
    nself billing subscription reactivate

PAYMENT COMMANDS:
    nself billing payment list             List payment methods
    nself billing payment add              Add new payment method
    nself billing payment remove <id>      Remove payment method
    nself billing payment default <id>     Set default payment method

QUOTA COMMANDS:
    nself billing quota                    Show all quota limits
    nself billing quota --service=api      Show specific service quota
    nself billing quota --usage            Show quota with current usage
    nself billing quota --alerts           Show quota alerts

PLAN COMMANDS:
    nself billing plan list                List all available plans
    nself billing plan show <name>         Show plan details
    nself billing plan compare             Compare plans
    nself billing plan current             Show current plan

EXPORT COMMANDS:
    nself billing export usage --format=csv
    nself billing export invoices --year=2026
    nself billing export --all --format=json

OPTIONS:
    --service=<name>    Filter by service (api, storage, compute, bandwidth)
    --period=<period>   Time period (current, last-month, custom)
    --start=<date>      Start date (YYYY-MM-DD)
    --end=<date>        End date (YYYY-MM-DD)
    --format=<format>   Output format (table, json, csv)
    --detailed          Show detailed breakdown
    --help, -h          Show this help message

EXAMPLES:
    # Check current usage
    nself billing usage

    # Check API usage for last month
    nself billing usage --service=api --period=last-month

    # List all invoices
    nself billing invoice list

    # Check quota limits with usage
    nself billing quota --usage

    # Upgrade subscription
    nself billing subscription upgrade pro

    # Export usage data as CSV
    nself billing export usage --format=csv

    # Add payment method
    nself billing payment add

SERVICES TRACKED:
    - API Requests (per request)
    - Storage (GB-hours)
    - Bandwidth (GB transferred)
    - Compute (CPU-hours)
    - Database (connections, queries)
    - Functions (invocations, duration)

BILLING PLANS:
    - free: Development and testing
    - starter: Small projects
    - pro: Production applications
    - enterprise: Large-scale deployments

For more information: https://docs.nself.org/billing
EOF
}

# Command: usage - Show usage statistics
cmd_usage() {
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
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    # Initialize billing system
    billing_init || {
        error "Failed to initialize billing system"
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
                error "Custom period requires --start and --end dates"
                exit 1
            fi
            ;;
        *)
            error "Invalid period: $period"
            exit 1
            ;;
    esac

    info "Usage Report: ${start} to ${end}"
    printf "\n"

    # Get usage data
    if [[ -n "$service" ]]; then
        usage_get_service "$service" "$start" "$end" "$format" "$detailed"
    else
        usage_get_all "$start" "$end" "$format" "$detailed"
    fi
}

# Command: invoice - Manage invoices
cmd_invoice() {
    local subcommand="${1:-list}"
    shift || true

    billing_init || {
        error "Failed to initialize billing system"
        exit 1
    }

    case "$subcommand" in
        list)
            stripe_invoice_list "$@"
            ;;
        show)
            if [[ $# -eq 0 ]]; then
                error "Invoice ID required"
                exit 1
            fi
            stripe_invoice_show "$1"
            ;;
        download)
            if [[ $# -eq 0 ]]; then
                error "Invoice ID required"
                exit 1
            fi
            stripe_invoice_download "$1"
            ;;
        pay)
            if [[ $# -eq 0 ]]; then
                error "Invoice ID required"
                exit 1
            fi
            stripe_invoice_pay "$1"
            ;;
        *)
            error "Unknown invoice command: $subcommand"
            printf "Valid commands: list, show, download, pay\n"
            exit 1
            ;;
    esac
}

# Command: subscription - Manage subscriptions
cmd_subscription() {
    local subcommand="${1:-show}"
    shift || true

    billing_init || {
        error "Failed to initialize billing system"
        exit 1
    }

    case "$subcommand" in
        show|current)
            stripe_subscription_show
            ;;
        plans)
            stripe_plans_list
            ;;
        upgrade)
            if [[ $# -eq 0 ]]; then
                error "Plan name required"
                exit 1
            fi
            stripe_subscription_upgrade "$1"
            ;;
        downgrade)
            if [[ $# -eq 0 ]]; then
                error "Plan name required"
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
            error "Unknown subscription command: $subcommand"
            printf "Valid commands: show, plans, upgrade, downgrade, cancel, reactivate\n"
            exit 1
            ;;
    esac
}

# Command: payment - Manage payment methods
cmd_payment() {
    local subcommand="${1:-list}"
    shift || true

    billing_init || {
        error "Failed to initialize billing system"
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
                error "Payment method ID required"
                exit 1
            fi
            stripe_payment_remove "$1"
            ;;
        default)
            if [[ $# -eq 0 ]]; then
                error "Payment method ID required"
                exit 1
            fi
            stripe_payment_set_default "$1"
            ;;
        *)
            error "Unknown payment command: $subcommand"
            printf "Valid commands: list, add, remove, default\n"
            exit 1
            ;;
    esac
}

# Command: quota - Check quota limits
cmd_quota() {
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
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    billing_init || {
        error "Failed to initialize billing system"
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

# Command: plan - Manage billing plans
cmd_plan() {
    local subcommand="${1:-list}"
    shift || true

    billing_init || {
        error "Failed to initialize billing system"
        exit 1
    }

    case "$subcommand" in
        list)
            stripe_plans_list "$@"
            ;;
        show)
            if [[ $# -eq 0 ]]; then
                error "Plan name required"
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
            error "Unknown plan command: $subcommand"
            printf "Valid commands: list, show, compare, current\n"
            exit 1
            ;;
    esac
}

# Command: export - Export billing data
cmd_export() {
    local export_type="all"
    local format="json"
    local output_file=""
    local year=""

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            usage|invoices|subscriptions|payments)
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
            --help|-h)
                show_help
                exit 0
                ;;
            *)
                error "Unknown option: $1"
                exit 1
                ;;
        esac
    done

    billing_init || {
        error "Failed to initialize billing system"
        exit 1
    }

    # Generate default output filename if not provided
    if [[ -z "$output_file" ]]; then
        local timestamp
        timestamp=$(date +"%Y%m%d_%H%M%S")
        output_file="nself_billing_${export_type}_${timestamp}.${format}"
    fi

    info "Exporting billing data to: ${output_file}"

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
            error "Unknown export type: $export_type"
            exit 1
            ;;
    esac

    success "Export complete: ${output_file}"
}

# Command: customer - Manage customer information
cmd_customer() {
    local subcommand="${1:-show}"
    shift || true

    billing_init || {
        error "Failed to initialize billing system"
        exit 1
    }

    case "$subcommand" in
        show|info)
            stripe_customer_show
            ;;
        update)
            stripe_customer_update "$@"
            ;;
        portal)
            stripe_customer_portal
            ;;
        *)
            error "Unknown customer command: $subcommand"
            printf "Valid commands: show, update, portal\n"
            exit 1
            ;;
    esac
}

# Command: webhook - Test webhook endpoints
cmd_webhook() {
    local subcommand="${1:-test}"
    shift || true

    billing_init || {
        error "Failed to initialize billing system"
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
            error "Unknown webhook command: $subcommand"
            printf "Valid commands: test, list, events\n"
            exit 1
            ;;
    esac
}

# Main function
main() {
    # Check for help flag
    if [[ $# -eq 0 ]] || [[ "$1" == "--help" ]] || [[ "$1" == "-h" ]]; then
        show_help
        exit 0
    fi

    local command="$1"
    shift

    case "$command" in
        usage)
            cmd_usage "$@"
            ;;
        invoice)
            cmd_invoice "$@"
            ;;
        subscription)
            cmd_subscription "$@"
            ;;
        payment)
            cmd_payment "$@"
            ;;
        quota)
            cmd_quota "$@"
            ;;
        plan)
            cmd_plan "$@"
            ;;
        export)
            cmd_export "$@"
            ;;
        customer)
            cmd_customer "$@"
            ;;
        webhook)
            cmd_webhook "$@"
            ;;
        --help|-h)
            show_help
            exit 0
            ;;
        *)
            error "Unknown command: $command"
            printf "\nRun 'nself billing --help' for usage information.\n"
            exit 1
            ;;
    esac
}

# Execute main function
main "$@"
