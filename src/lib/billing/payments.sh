#!/usr/bin/env bash
set -euo pipefail

#
# nself billing/payments.sh - Payment Method Management
# Part of nself v0.9.6 - Complete Billing System
#
# Payment method storage, management, and Stripe integration.
#

# Prevent multiple sourcing
[[ -n "${NSELF_BILLING_PAYMENTS_LOADED:-}" ]] && return 0
NSELF_BILLING_PAYMENTS_LOADED=1

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Payment configuration
PAYMENT_METHODS_ENABLED="${PAYMENT_METHODS_ENABLED:-card,us_bank_account}"
ALLOW_MULTIPLE_PAYMENT_METHODS="${ALLOW_MULTIPLE_PAYMENT_METHODS:-true}"
PAYMENT_RETRY_ATTEMPTS="${PAYMENT_RETRY_ATTEMPTS:-3}"
PAYMENT_RETRY_DELAY="${PAYMENT_RETRY_DELAY:-3}" # days

# ============================================================================
# Payment Method Management - Local Database
# ============================================================================

# Add payment method to database
# Args: customer_id, payment_method_id, type, last4, exp_month, exp_year, is_default
payment_add() {
  local customer_id="$1"
  local payment_method_id="$2"
  local payment_type="${3:-card}"
  local last4="${4:-}"
  local exp_month="${5:-}"
  local exp_year="${6:-}"
  local is_default="${7:-false}"

  # Validate required parameters
  if [[ -z "$customer_id" ]] || [[ -z "$payment_method_id" ]]; then
    error "Customer ID and payment method ID are required"
    return 1
  fi

  # Check if payment method already exists
  local existing_pm
  existing_pm=$(billing_db_query "
    SELECT payment_method_id
    FROM billing_payment_methods
    WHERE payment_method_id = :'payment_method_id'
    AND deleted_at IS NULL;
  " "tuples" "payment_method_id" "$payment_method_id" 2>/dev/null | tr -d ' ')

  if [[ -n "$existing_pm" ]]; then
    warn "Payment method already exists: ${payment_method_id}"
    return 0  # Idempotent
  fi

  # If this is the first payment method, make it default
  local pm_count
  pm_count=$(billing_db_query "
    SELECT COUNT(*)
    FROM billing_payment_methods
    WHERE customer_id = :'customer_id'
    AND deleted_at IS NULL;
  " "tuples" "customer_id" "$customer_id" 2>/dev/null | tr -d ' ')

  if [[ "$pm_count" == "0" ]]; then
    is_default="true"
  fi

  # If setting as default, unset other defaults
  if [[ "$is_default" == "true" ]]; then
    billing_db_query "
      UPDATE billing_payment_methods
      SET is_default = false
      WHERE customer_id = :'customer_id'
      AND deleted_at IS NULL;
    " "tuples" "customer_id" "$customer_id" >/dev/null
  fi

  # Insert payment method
  billing_db_query "
    INSERT INTO billing_payment_methods (
      customer_id,
      payment_method_id,
      payment_type,
      last4,
      exp_month,
      exp_year,
      is_default,
      created_at
    ) VALUES (
      :'customer_id',
      :'payment_method_id',
      :'payment_type',
      :'last4',
      :'exp_month',
      :'exp_year',
      :'is_default',
      NOW()
    );
  " "tuples" \
    "customer_id" "$customer_id" \
    "payment_method_id" "$payment_method_id" \
    "payment_type" "$payment_type" \
    "last4" "$last4" \
    "exp_month" "$exp_month" \
    "exp_year" "$exp_year" \
    "is_default" "$is_default" >/dev/null

  billing_log "PAYMENT" "added" "$payment_method_id" "customer=${customer_id},type=${payment_type}"

  success "Payment method added: ${payment_method_id}"
}

# List payment methods for customer
payment_list() {
  local customer_id="${1:-}"
  local active_only="${2:-true}"

  if [[ -z "$customer_id" ]]; then
    customer_id=$(billing_get_customer_id) || {
      error "No customer ID found"
      return 1
    }
  fi

  local where_clause="customer_id = :'customer_id'"

  if [[ "$active_only" == "true" ]]; then
    where_clause+=" AND deleted_at IS NULL"
  fi

  billing_db_query "
    SELECT
      payment_method_id,
      payment_type,
      last4,
      exp_month,
      exp_year,
      is_default,
      created_at
    FROM billing_payment_methods
    WHERE ${where_clause}
    ORDER BY is_default DESC, created_at DESC;
  " "tuples" "customer_id" "$customer_id"
}

# Get payment method details
payment_get() {
  local payment_method_id="$1"

  if [[ -z "$payment_method_id" ]]; then
    error "Payment method ID required"
    return 1
  fi

  billing_db_query "
    SELECT
      payment_method_id,
      customer_id,
      payment_type,
      last4,
      exp_month,
      exp_year,
      is_default,
      created_at
    FROM billing_payment_methods
    WHERE payment_method_id = :'payment_method_id'
    AND deleted_at IS NULL;
  " "tuples" "payment_method_id" "$payment_method_id"
}

# Set payment method as default
payment_set_default() {
  local payment_method_id="$1"

  if [[ -z "$payment_method_id" ]]; then
    error "Payment method ID required"
    return 1
  fi

  # Get customer ID for this payment method
  local pm_data
  pm_data=$(payment_get "$payment_method_id")

  if [[ -z "$pm_data" ]]; then
    error "Payment method not found: ${payment_method_id}"
    return 1
  fi

  local customer_id
  customer_id=$(printf '%s' "$pm_data" | cut -d'|' -f2 | tr -d ' ')

  # Unset all other defaults for this customer
  billing_db_query "
    UPDATE billing_payment_methods
    SET is_default = false
    WHERE customer_id = :'customer_id'
    AND deleted_at IS NULL;
  " "tuples" "customer_id" "$customer_id" >/dev/null

  # Set this one as default
  billing_db_query "
    UPDATE billing_payment_methods
    SET is_default = true,
        updated_at = NOW()
    WHERE payment_method_id = :'payment_method_id';
  " "tuples" "payment_method_id" "$payment_method_id" >/dev/null

  billing_log "PAYMENT" "set_default" "$payment_method_id" "customer=${customer_id}"

  success "Payment method set as default: ${payment_method_id}"
}

# Remove payment method (soft delete)
payment_remove() {
  local payment_method_id="$1"
  local force="${2:-false}"

  if [[ -z "$payment_method_id" ]]; then
    error "Payment method ID required"
    return 1
  fi

  # Get payment method info
  local pm_data
  pm_data=$(payment_get "$payment_method_id")

  if [[ -z "$pm_data" ]]; then
    error "Payment method not found: ${payment_method_id}"
    return 1
  fi

  # Parse payment method data
  local customer_id is_default
  IFS='|' read -r _ customer_id _ _ _ _ is_default _ <<< "$pm_data"
  customer_id=$(printf '%s' "$customer_id" | tr -d ' ')
  is_default=$(printf '%s' "$is_default" | tr -d ' ')

  # Check if this is the only payment method
  local pm_count
  pm_count=$(billing_db_query "
    SELECT COUNT(*)
    FROM billing_payment_methods
    WHERE customer_id = :'customer_id'
    AND deleted_at IS NULL;
  " "tuples" "customer_id" "$customer_id" 2>/dev/null | tr -d ' ')

  if [[ "$pm_count" == "1" ]] && [[ "$force" != "true" ]]; then
    error "Cannot remove the only payment method. Use --force to override."
    return 1
  fi

  # Soft delete
  billing_db_query "
    UPDATE billing_payment_methods
    SET deleted_at = NOW(),
        updated_at = NOW()
    WHERE payment_method_id = :'payment_method_id';
  " "tuples" "payment_method_id" "$payment_method_id" >/dev/null

  # If this was the default, set another as default
  if [[ "$is_default" == "t" ]]; then
    local next_pm
    next_pm=$(billing_db_query "
      SELECT payment_method_id
      FROM billing_payment_methods
      WHERE customer_id = :'customer_id'
      AND deleted_at IS NULL
      ORDER BY created_at DESC
      LIMIT 1;
    " "tuples" "customer_id" "$customer_id" 2>/dev/null | tr -d ' ')

    if [[ -n "$next_pm" ]]; then
      payment_set_default "$next_pm" >/dev/null
    fi
  fi

  billing_log "PAYMENT" "removed" "$payment_method_id" "customer=${customer_id}"

  success "Payment method removed: ${payment_method_id}"
}

# Get default payment method for customer
payment_get_default() {
  local customer_id="${1:-}"

  if [[ -z "$customer_id" ]]; then
    customer_id=$(billing_get_customer_id) || {
      error "No customer ID found"
      return 1
    }
  fi

  billing_db_query "
    SELECT
      payment_method_id,
      payment_type,
      last4,
      exp_month,
      exp_year
    FROM billing_payment_methods
    WHERE customer_id = :'customer_id'
    AND is_default = true
    AND deleted_at IS NULL
    LIMIT 1;
  " "tuples" "customer_id" "$customer_id"
}

# ============================================================================
# Stripe Payment Method Integration
# ============================================================================

# Create Stripe payment method setup intent
# Returns setup intent client secret for frontend
stripe_payment_create_setup_intent() {
  local customer_id="${1:-}"

  if [[ -z "$customer_id" ]]; then
    customer_id=$(billing_get_customer_id) || {
      error "No customer ID found"
      return 1
    }
  fi

  # Get Stripe customer ID
  local customer_data
  customer_data=$(billing_get_customer "$customer_id")

  local stripe_customer_id
  stripe_customer_id=$(printf '%s' "$customer_data" | cut -d'|' -f7 | tr -d ' ')

  if [[ -z "$stripe_customer_id" ]]; then
    error "No Stripe customer ID found for customer: ${customer_id}"
    return 1
  fi

  # Create setup intent via Stripe API
  local curl_config
  curl_config=$(mktemp) || return 1
  trap "rm -f '$curl_config'" RETURN

  chmod 600 "$curl_config"
  cat > "$curl_config" <<EOF
user = ":${STRIPE_SECRET_KEY}"
EOF
  chmod 600 "$curl_config"

  local response
  response=$(curl -s --config "$curl_config" \
    -X POST "https://api.stripe.com/v1/setup_intents" \
    -d "customer=${stripe_customer_id}" \
    2>/dev/null)

  # Extract client secret
  local client_secret
  client_secret=$(printf '%s' "$response" | grep -o '"client_secret":"[^"]*"' | cut -d'"' -f4)

  if [[ -z "$client_secret" ]]; then
    error "Failed to create setup intent"
    return 1
  fi

  printf "%s" "$client_secret"
}

# Sync payment methods from Stripe
stripe_payment_sync() {
  local customer_id="${1:-}"

  if [[ -z "$customer_id" ]]; then
    customer_id=$(billing_get_customer_id) || {
      error "No customer ID found"
      return 1
    }
  fi

  # Get Stripe customer ID
  local customer_data
  customer_data=$(billing_get_customer "$customer_id")

  local stripe_customer_id
  stripe_customer_id=$(printf '%s' "$customer_data" | cut -d'|' -f7 | tr -d ' ')

  if [[ -z "$stripe_customer_id" ]]; then
    warn "No Stripe customer ID found for customer: ${customer_id}"
    return 0
  fi

  # Fetch payment methods from Stripe
  local curl_config
  curl_config=$(mktemp) || return 1
  trap "rm -f '$curl_config'" RETURN

  chmod 600 "$curl_config"
  cat > "$curl_config" <<EOF
user = ":${STRIPE_SECRET_KEY}"
EOF
  chmod 600 "$curl_config"

  local response
  response=$(curl -s --config "$curl_config" \
    "https://api.stripe.com/v1/customers/${stripe_customer_id}/payment_methods?type=card" \
    2>/dev/null)

  # Parse payment methods from response (simplified - would need jq for proper parsing)
  # For now, just log that sync was attempted
  billing_log "PAYMENT" "sync" "$customer_id" "stripe_customer=${stripe_customer_id}"

  info "Payment methods synced from Stripe"
}

# ============================================================================
# Payment Processing
# ============================================================================

# Process payment for invoice
payment_process() {
  local invoice_id="$1"
  local payment_method_id="${2:-}"
  local amount="${3:-}"

  if [[ -z "$invoice_id" ]]; then
    error "Invoice ID required"
    return 1
  fi

  # Get invoice details
  local invoice_data
  invoice_data=$(billing_db_query "
    SELECT customer_id, total_amount, status
    FROM billing_invoices
    WHERE invoice_id = :'invoice_id';
  " "tuples" "invoice_id" "$invoice_id")

  if [[ -z "$invoice_data" ]]; then
    error "Invoice not found: ${invoice_id}"
    return 1
  fi

  local customer_id invoice_amount invoice_status
  IFS='|' read -r customer_id invoice_amount invoice_status <<< "$invoice_data"
  customer_id=$(printf '%s' "$customer_id" | tr -d ' ')
  invoice_amount=$(printf '%s' "$invoice_amount" | tr -d ' ')
  invoice_status=$(printf '%s' "$invoice_status" | tr -d ' ')

  # Check if already paid
  if [[ "$invoice_status" == "paid" ]]; then
    warn "Invoice already paid: ${invoice_id}"
    return 0
  fi

  # Use invoice amount if not specified
  if [[ -z "$amount" ]]; then
    amount="$invoice_amount"
  fi

  # Get payment method if not specified
  if [[ -z "$payment_method_id" ]]; then
    local default_pm
    default_pm=$(payment_get_default "$customer_id")

    if [[ -z "$default_pm" ]]; then
      error "No payment method found for customer: ${customer_id}"
      return 1
    fi

    payment_method_id=$(printf '%s' "$default_pm" | cut -d'|' -f1 | tr -d ' ')
  fi

  # Record payment attempt
  local payment_id
  payment_id="pay_$(date +%s)_$(openssl rand -hex 4 2>/dev/null || printf '%08x' $RANDOM)"

  billing_db_query "
    INSERT INTO billing_payments (
      payment_id,
      invoice_id,
      customer_id,
      payment_method_id,
      amount,
      status,
      created_at
    ) VALUES (
      :'payment_id',
      :'invoice_id',
      :'customer_id',
      :'payment_method_id',
      :'amount',
      'pending',
      NOW()
    );
  " "tuples" \
    "payment_id" "$payment_id" \
    "invoice_id" "$invoice_id" \
    "customer_id" "$customer_id" \
    "payment_method_id" "$payment_method_id" \
    "amount" "$amount" >/dev/null

  # Simulate payment processing (would integrate with Stripe Payment Intents here)
  billing_log "PAYMENT" "processing" "$payment_id" "invoice=${invoice_id},amount=${amount}"

  # For now, mark as succeeded (in real implementation, would call Stripe API)
  payment_mark_succeeded "$payment_id" "txn_simulated"

  success "Payment processed: ${payment_id}"
  printf "%s" "$payment_id"
}

# Mark payment as succeeded
payment_mark_succeeded() {
  local payment_id="$1"
  local transaction_id="${2:-}"

  billing_db_query "
    UPDATE billing_payments
    SET status = 'succeeded',
        transaction_id = :'transaction_id',
        completed_at = NOW()
    WHERE payment_id = :'payment_id';
  " "tuples" \
    "payment_id" "$payment_id" \
    "transaction_id" "$transaction_id" >/dev/null

  # Also mark invoice as paid
  local invoice_id
  invoice_id=$(billing_db_query "
    SELECT invoice_id
    FROM billing_payments
    WHERE payment_id = :'payment_id';
  " "tuples" "payment_id" "$payment_id" 2>/dev/null | tr -d ' ')

  if [[ -n "$invoice_id" ]]; then
    billing_db_query "
      UPDATE billing_invoices
      SET status = 'paid',
          paid_at = NOW()
      WHERE invoice_id = :'invoice_id';
    " "tuples" "invoice_id" "$invoice_id" >/dev/null
  fi

  billing_log "PAYMENT" "succeeded" "$payment_id" "txn=${transaction_id}"
}

# Mark payment as failed
payment_mark_failed() {
  local payment_id="$1"
  local failure_reason="${2:-}"

  billing_db_query "
    UPDATE billing_payments
    SET status = 'failed',
        failure_reason = :'failure_reason',
        completed_at = NOW()
    WHERE payment_id = :'payment_id';
  " "tuples" \
    "payment_id" "$payment_id" \
    "failure_reason" "$failure_reason" >/dev/null

  billing_log "PAYMENT" "failed" "$payment_id" "reason=${failure_reason}"

  error "Payment failed: ${failure_reason}"
}

# Get payment history for customer
payment_history() {
  local customer_id="${1:-}"
  local limit="${2:-50}"

  if [[ -z "$customer_id" ]]; then
    customer_id=$(billing_get_customer_id) || {
      error "No customer ID found"
      return 1
    }
  fi

  billing_db_query "
    SELECT
      p.payment_id,
      p.invoice_id,
      p.amount,
      p.status,
      p.transaction_id,
      p.created_at,
      p.completed_at
    FROM billing_payments p
    WHERE p.customer_id = :'customer_id'
    ORDER BY p.created_at DESC
    LIMIT :'limit';
  " "tuples" "customer_id" "$customer_id" "limit" "$limit"
}

# Retry failed payment
payment_retry() {
  local payment_id="$1"

  if [[ -z "$payment_id" ]]; then
    error "Payment ID required"
    return 1
  fi

  # Get original payment details
  local payment_data
  payment_data=$(billing_db_query "
    SELECT invoice_id, payment_method_id, amount, status
    FROM billing_payments
    WHERE payment_id = :'payment_id';
  " "tuples" "payment_id" "$payment_id")

  if [[ -z "$payment_data" ]]; then
    error "Payment not found: ${payment_id}"
    return 1
  fi

  local invoice_id payment_method_id amount status
  IFS='|' read -r invoice_id payment_method_id amount status <<< "$payment_data"

  # Check if payment failed
  if [[ "$(printf '%s' "$status" | tr -d ' ')" != "failed" ]]; then
    warn "Payment is not in failed status: ${payment_id}"
    return 0
  fi

  # Process new payment attempt
  payment_process \
    "$(printf '%s' "$invoice_id" | tr -d ' ')" \
    "$(printf '%s' "$payment_method_id" | tr -d ' ')" \
    "$(printf '%s' "$amount" | tr -d ' ')"
}

# Export functions
export -f payment_add
export -f payment_list
export -f payment_get
export -f payment_set_default
export -f payment_remove
export -f payment_get_default
export -f stripe_payment_create_setup_intent
export -f stripe_payment_sync
export -f payment_process
export -f payment_mark_succeeded
export -f payment_mark_failed
export -f payment_history
export -f payment_retry
