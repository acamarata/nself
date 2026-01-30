#!/usr/bin/env bash
#
# nself billing/core.sh - Billing System Core Functions
# Part of nself v0.9.0 - Sprint 13: Billing Integration & Usage Tracking
#
# Core billing system initialization, configuration, and foundational functions.
#

# Prevent multiple sourcing
[[ -n "${NSELF_BILLING_CORE_LOADED:-}" ]] && return 0
NSELF_BILLING_CORE_LOADED=1

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
NSELF_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

source "${NSELF_ROOT}/src/lib/utils/output.sh"
source "${NSELF_ROOT}/src/lib/utils/validation.sh"

# Billing configuration
BILLING_DB_HOST="${BILLING_DB_HOST:-localhost}"
BILLING_DB_PORT="${BILLING_DB_PORT:-5432}"
BILLING_DB_NAME="${BILLING_DB_NAME:-nself}"
BILLING_DB_USER="${BILLING_DB_USER:-postgres}"
BILLING_DB_PASSWORD="${BILLING_DB_PASSWORD:-}"

# Stripe configuration
STRIPE_SECRET_KEY="${STRIPE_SECRET_KEY:-}"
STRIPE_PUBLISHABLE_KEY="${STRIPE_PUBLISHABLE_KEY:-}"
STRIPE_WEBHOOK_SECRET="${STRIPE_WEBHOOK_SECRET:-}"
STRIPE_API_VERSION="${STRIPE_API_VERSION:-2023-10-16}"

# Billing paths
BILLING_DATA_DIR="${NSELF_ROOT}/.nself/billing"
BILLING_CACHE_DIR="${BILLING_DATA_DIR}/cache"
BILLING_EXPORT_DIR="${BILLING_DATA_DIR}/exports"
BILLING_LOG_FILE="${BILLING_DATA_DIR}/billing.log"

# Initialize billing system
billing_init() {
    local quiet="${1:-false}"

    # Create required directories
    mkdir -p "${BILLING_DATA_DIR}" "${BILLING_CACHE_DIR}" "${BILLING_EXPORT_DIR}"

    # Validate configuration
    if ! billing_validate_config; then
        if [[ "$quiet" != "true" ]]; then
            error "Billing configuration validation failed"
        fi
        return 1
    fi

    # Test database connection
    if ! billing_test_db_connection; then
        if [[ "$quiet" != "true" ]]; then
            error "Database connection failed"
        fi
        return 1
    fi

    # Test Stripe API if configured
    if [[ -n "$STRIPE_SECRET_KEY" ]]; then
        if ! billing_test_stripe_connection; then
            if [[ "$quiet" != "true" ]]; then
                warn "Stripe API connection failed (continuing with limited functionality)"
            fi
        fi
    fi

    if [[ "$quiet" != "true" ]]; then
        success "Billing system initialized"
    fi

    return 0
}

# Validate billing configuration
billing_validate_config() {
    local errors=0

    # Check required database configuration
    if [[ -z "$BILLING_DB_HOST" ]]; then
        error "BILLING_DB_HOST not set"
        ((errors++))
    fi

    if [[ -z "$BILLING_DB_NAME" ]]; then
        error "BILLING_DB_NAME not set"
        ((errors++))
    fi

    # Check Stripe configuration (optional but recommended)
    if [[ -z "$STRIPE_SECRET_KEY" ]]; then
        warn "STRIPE_SECRET_KEY not set - Stripe features disabled"
    fi

    if [[ -z "$STRIPE_PUBLISHABLE_KEY" ]]; then
        warn "STRIPE_PUBLISHABLE_KEY not set - Stripe features disabled"
    fi

    # Check directories are writable
    if [[ ! -w "$BILLING_DATA_DIR" ]]; then
        error "Billing data directory not writable: ${BILLING_DATA_DIR}"
        ((errors++))
    fi

    return $errors
}

# Test database connection
billing_test_db_connection() {
    local result

    result=$(PGPASSWORD="$BILLING_DB_PASSWORD" psql -h "$BILLING_DB_HOST" \
        -p "$BILLING_DB_PORT" -U "$BILLING_DB_USER" -d "$BILLING_DB_NAME" \
        -t -c "SELECT 1;" 2>/dev/null || echo "")

    if [[ "$result" =~ 1 ]]; then
        return 0
    else
        return 1
    fi
}

# Test Stripe API connection
billing_test_stripe_connection() {
    if [[ -z "$STRIPE_SECRET_KEY" ]]; then
        return 1
    fi

    local response
    response=$(curl -s -u "${STRIPE_SECRET_KEY}:" \
        "https://api.stripe.com/v1/balance" 2>/dev/null || echo "")

    if [[ -n "$response" ]] && [[ ! "$response" =~ "error" ]]; then
        return 0
    else
        return 1
    fi
}

# Execute database query with parameterized query support
# Usage: billing_db_query "SELECT * FROM table WHERE id = :'id' AND name = :'name'" "tuples" "id" "123" "name" "John"
billing_db_query() {
    local query="$1"
    local format="${2:-tuples}"  # tuples, csv, json
    shift 2

    local psql_opts="-h ${BILLING_DB_HOST} -p ${BILLING_DB_PORT} -U ${BILLING_DB_USER} -d ${BILLING_DB_NAME}"

    # Build variable bindings from remaining arguments (key-value pairs)
    local var_opts=""
    while (( $# >= 2 )); do
        local var_name="$1"
        local var_value="$2"
        shift 2
        var_opts="${var_opts} -v ${var_name}='${var_value}'"
    done

    case "$format" in
        csv)
            psql_opts="${psql_opts} --csv"
            ;;
        json)
            # PostgreSQL 9.2+ supports JSON output
            query="SELECT row_to_json(t) FROM (${query}) t;"
            ;;
        *)
            psql_opts="${psql_opts} -t"  # tuples only
            ;;
    esac

    PGPASSWORD="$BILLING_DB_PASSWORD" psql $psql_opts $var_opts -c "$query" 2>/dev/null
}

# Get current customer ID from environment or config
billing_get_customer_id() {
    # Try environment variable first
    if [[ -n "${NSELF_CUSTOMER_ID:-}" ]]; then
        printf "%s" "$NSELF_CUSTOMER_ID"
        return 0
    fi

    # Try project config
    local project_config="${NSELF_ROOT}/.env"
    if [[ -f "$project_config" ]]; then
        local customer_id
        customer_id=$(grep "^NSELF_CUSTOMER_ID=" "$project_config" 2>/dev/null | cut -d= -f2-)
        if [[ -n "$customer_id" ]]; then
            printf "%s" "$customer_id"
            return 0
        fi
    fi

    # Try database
    local db_customer_id
    db_customer_id=$(billing_db_query "SELECT customer_id FROM billing_customers WHERE project_name=:'project_name' LIMIT 1;" "tuples" "project_name" "${PROJECT_NAME:-default}" 2>/dev/null | tr -d ' ')

    if [[ -n "$db_customer_id" ]]; then
        printf "%s" "$db_customer_id"
        return 0
    fi

    return 1
}

# Get current subscription
billing_get_subscription() {
    local customer_id
    customer_id=$(billing_get_customer_id) || {
        error "No customer ID found"
        return 1
    }

    billing_db_query "
        SELECT
            subscription_id,
            plan_name,
            status,
            current_period_start,
            current_period_end,
            cancel_at_period_end
        FROM billing_subscriptions
        WHERE customer_id = :'customer_id'
        AND status IN ('active', 'trialing')
        ORDER BY created_at DESC
        LIMIT 1;
    " "tuples" "customer_id" "$customer_id"
}

# Record usage event
billing_record_usage() {
    local service="$1"
    local quantity="$2"
    local metadata="${3:-{}}"

    local customer_id
    customer_id=$(billing_get_customer_id) || {
        warn "No customer ID - usage not recorded"
        return 1
    }

    local timestamp
    timestamp=$(date -u +"%Y-%m-%d %H:%M:%S")

    billing_db_query "
        INSERT INTO billing_usage_records
            (customer_id, service_name, quantity, metadata, recorded_at)
        VALUES
            (:'customer_id', :'service_name', :'quantity', :'metadata', :'recorded_at');
    " "tuples" "customer_id" "$customer_id" "service_name" "$service" "quantity" "$quantity" "metadata" "$metadata" "recorded_at" "$timestamp" >/dev/null

    billing_log "USAGE" "$service" "$quantity" "$metadata"
}

# Check quota for service
billing_check_quota() {
    local service="$1"
    local requested="${2:-1}"

    local customer_id
    customer_id=$(billing_get_customer_id) || {
        warn "No customer ID - quota check skipped"
        return 0  # Allow if no billing setup
    }

    # Get current plan's quota
    local quota
    quota=$(billing_db_query "
        SELECT q.limit_value
        FROM billing_quotas q
        JOIN billing_subscriptions s ON s.plan_name = q.plan_name
        WHERE s.customer_id = :'customer_id'
        AND s.status = 'active'
        AND q.service_name = :'service_name'
        LIMIT 1;
    " "tuples" "customer_id" "$customer_id" "service_name" "$service" | tr -d ' ')

    # If no quota set, allow unlimited
    if [[ -z "$quota" ]] || [[ "$quota" == "-1" ]]; then
        return 0
    fi

    # Get current usage for this billing period
    local usage
    usage=$(billing_db_query "
        SELECT COALESCE(SUM(quantity), 0)
        FROM billing_usage_records ur
        JOIN billing_subscriptions s ON s.customer_id = ur.customer_id
        WHERE ur.customer_id = :'customer_id'
        AND ur.service_name = :'service_name'
        AND ur.recorded_at >= s.current_period_start
        AND ur.recorded_at <= s.current_period_end;
    " "tuples" "customer_id" "$customer_id" "service_name" "$service" | tr -d ' ')

    # Check if adding requested amount would exceed quota
    local total=$((usage + requested))
    if [[ $total -gt $quota ]]; then
        return 1  # Quota exceeded
    fi

    return 0  # Quota available
}

# Get quota status
billing_get_quota_status() {
    local service="$1"

    local customer_id
    customer_id=$(billing_get_customer_id) || {
        printf "unknown"
        return 1
    }

    local quota usage

    # Get quota limit
    quota=$(billing_db_query "
        SELECT q.limit_value
        FROM billing_quotas q
        JOIN billing_subscriptions s ON s.plan_name = q.plan_name
        WHERE s.customer_id = :'customer_id'
        AND s.status = 'active'
        AND q.service_name = :'service_name'
        LIMIT 1;
    " "tuples" "customer_id" "$customer_id" "service_name" "$service" | tr -d ' ')

    # Get current usage
    usage=$(billing_db_query "
        SELECT COALESCE(SUM(quantity), 0)
        FROM billing_usage_records ur
        JOIN billing_subscriptions s ON s.customer_id = ur.customer_id
        WHERE ur.customer_id = :'customer_id'
        AND ur.service_name = :'service_name'
        AND ur.recorded_at >= s.current_period_start
        AND ur.recorded_at <= s.current_period_end;
    " "tuples" "customer_id" "$customer_id" "service_name" "$service" | tr -d ' ')

    # Calculate percentage
    local percent=0
    if [[ -n "$quota" ]] && [[ "$quota" != "-1" ]] && [[ $quota -gt 0 ]]; then
        percent=$((usage * 100 / quota))
    fi

    # Output status JSON
    printf '{"service":"%s","usage":%d,"quota":%s,"percent":%d}\n' \
        "$service" "${usage:-0}" "${quota:--1}" "$percent"
}

# Generate invoice
billing_generate_invoice() {
    local customer_id="$1"
    local period_start="$2"
    local period_end="$3"

    local invoice_id
    invoice_id="inv_$(date +%s)_$(openssl rand -hex 4)"

    # Calculate usage-based charges
    local total_amount=0
    local line_items=""

    # API usage
    local api_usage
    api_usage=$(billing_db_query "
        SELECT COALESCE(SUM(quantity), 0)
        FROM billing_usage_records
        WHERE customer_id = :'customer_id'
        AND service_name = 'api'
        AND recorded_at >= :'period_start'
        AND recorded_at <= :'period_end';
    " "tuples" "customer_id" "$customer_id" "period_start" "$period_start" "period_end" "$period_end" | tr -d ' ')

    # Storage usage (GB-hours)
    local storage_usage
    storage_usage=$(billing_db_query "
        SELECT COALESCE(SUM(quantity), 0)
        FROM billing_usage_records
        WHERE customer_id = :'customer_id'
        AND service_name = 'storage'
        AND recorded_at >= :'period_start'
        AND recorded_at <= :'period_end';
    " "tuples" "customer_id" "$customer_id" "period_start" "$period_start" "period_end" "$period_end" | tr -d ' ')

    # Get pricing from plan
    local plan_name
    plan_name=$(billing_db_query "
        SELECT plan_name FROM billing_subscriptions
        WHERE customer_id = :'customer_id'
        AND status = 'active'
        LIMIT 1;
    " "tuples" "customer_id" "$customer_id" | tr -d ' ')

    # Insert invoice
    billing_db_query "
        INSERT INTO billing_invoices
            (invoice_id, customer_id, period_start, period_end, total_amount, status)
        VALUES
            (:'invoice_id', :'customer_id', :'period_start', :'period_end', :'total_amount', 'draft');
    " "tuples" "invoice_id" "$invoice_id" "customer_id" "$customer_id" "period_start" "$period_start" "period_end" "$period_end" "total_amount" "$total_amount" >/dev/null

    printf "%s" "$invoice_id"
}

# Export billing data
billing_export_all() {
    local format="$1"
    local output_file="$2"
    local year="${3:-}"

    local customer_id
    customer_id=$(billing_get_customer_id) || {
        error "No customer ID found"
        return 1
    }

    case "$format" in
        json)
            billing_db_query "
                SELECT json_build_object(
                    'customer', (SELECT row_to_json(c) FROM billing_customers c WHERE c.customer_id = :'customer_id'),
                    'subscription', (SELECT row_to_json(s) FROM billing_subscriptions s WHERE s.customer_id = :'customer_id' AND s.status = 'active'),
                    'invoices', (SELECT json_agg(row_to_json(i)) FROM billing_invoices i WHERE i.customer_id = :'customer_id'),
                    'usage', (SELECT json_agg(row_to_json(u)) FROM billing_usage_records u WHERE u.customer_id = :'customer_id')
                );
            " "tuples" "customer_id" "$customer_id" > "$output_file"
            ;;
        csv)
            # Export multiple CSV files
            local base="${output_file%.csv}"

            billing_db_query "SELECT * FROM billing_customers WHERE customer_id = :'customer_id';" "csv" "customer_id" "$customer_id" > "${base}_customer.csv"
            billing_db_query "SELECT * FROM billing_subscriptions WHERE customer_id = :'customer_id';" "csv" "customer_id" "$customer_id" > "${base}_subscriptions.csv"
            billing_db_query "SELECT * FROM billing_invoices WHERE customer_id = :'customer_id';" "csv" "customer_id" "$customer_id" > "${base}_invoices.csv"
            billing_db_query "SELECT * FROM billing_usage_records WHERE customer_id = :'customer_id';" "csv" "customer_id" "$customer_id" > "${base}_usage.csv"
            ;;
        *)
            error "Unsupported format: $format"
            return 1
            ;;
    esac

    return 0
}

# Log billing event
billing_log() {
    local event_type="$1"
    local service="$2"
    local value="$3"
    local metadata="${4:-}"

    local timestamp
    timestamp=$(date -u +"%Y-%m-%d %H:%M:%S")

    printf "[%s] %s | %s | %s | %s\n" \
        "$timestamp" "$event_type" "$service" "$value" "$metadata" \
        >> "$BILLING_LOG_FILE"
}

# Get billing summary
billing_get_summary() {
    local customer_id
    customer_id=$(billing_get_customer_id) || {
        error "No customer ID found"
        return 1
    }

    billing_db_query "
        SELECT
            s.plan_name,
            s.status,
            COUNT(DISTINCT i.invoice_id) as invoice_count,
            COALESCE(SUM(i.total_amount), 0) as total_billed,
            COUNT(DISTINCT ur.service_name) as services_used
        FROM billing_subscriptions s
        LEFT JOIN billing_invoices i ON i.customer_id = s.customer_id
        LEFT JOIN billing_usage_records ur ON ur.customer_id = s.customer_id
        WHERE s.customer_id = :'customer_id'
        AND s.status = 'active'
        GROUP BY s.plan_name, s.status;
    " "tuples" "customer_id" "$customer_id"
}

# Export individual functions
export -f billing_init
export -f billing_validate_config
export -f billing_test_db_connection
export -f billing_test_stripe_connection
export -f billing_db_query
export -f billing_get_customer_id
export -f billing_get_subscription
export -f billing_record_usage
export -f billing_check_quota
export -f billing_get_quota_status
export -f billing_generate_invoice
export -f billing_export_all
export -f billing_log
export -f billing_get_summary
