#!/usr/bin/env bash
set -euo pipefail

#
# nself billing/reports.sh - Billing Reports and Analytics
# Part of nself v0.9.6 - Complete Billing System
#
# Comprehensive reporting, analytics, and business intelligence for billing data.
#

# Prevent multiple sourcing
[[ -n "${NSELF_BILLING_REPORTS_LOADED:-}" ]] && return 0
NSELF_BILLING_REPORTS_LOADED=1

# Source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/core.sh"

# Report configuration
REPORT_OUTPUT_DIR="${BILLING_EXPORT_DIR}/reports"
REPORT_CACHE_DIR="${BILLING_CACHE_DIR}/reports"
REPORT_CACHE_TTL="${REPORT_CACHE_TTL:-3600}" # 1 hour

# Initialize reporting system
reports_init() {
  mkdir -p "${REPORT_OUTPUT_DIR}"
  mkdir -p "${REPORT_CACHE_DIR}"
  return 0
}

# ============================================================================
# Revenue Reports
# ============================================================================

# Generate Monthly Recurring Revenue (MRR) report
report_mrr() {
  local start_month="${1:-}"
  local end_month="${2:-}"
  local format="${3:-table}"

  # Default to current month if not specified
  if [[ -z "$start_month" ]]; then
    start_month=$(date +"%Y-%m-01")
  fi

  if [[ -z "$end_month" ]]; then
    end_month=$(date +"%Y-%m-01")
  fi

  case "$format" in
    table)
      report_mrr_table "$start_month" "$end_month"
      ;;
    csv)
      report_mrr_csv "$start_month" "$end_month"
      ;;
    json)
      report_mrr_json "$start_month" "$end_month"
      ;;
    *)
      error "Invalid format: $format. Use table, csv, or json"
      return 1
      ;;
  esac
}

# MRR report as table
report_mrr_table() {
  local start_month="$1"
  local end_month="$2"

  printf "╔════════════════════════════════════════════════════════════════╗\n"
  printf "║              Monthly Recurring Revenue (MRR)                  ║\n"
  printf "╠════════════════════════════════════════════════════════════════╣\n"
  printf "║ Month          │ Active Subs │ New Subs  │ Churned   │ MRR    ║\n"
  printf "╠════════════════╪═════════════╪═══════════╪═══════════╪════════╣\n"

  billing_db_query "
    WITH monthly_stats AS (
      SELECT
        TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') as month,
        COUNT(DISTINCT CASE WHEN status = 'active' THEN subscription_id END) as active_subs,
        COUNT(DISTINCT CASE WHEN DATE_TRUNC('month', created_at) = DATE_TRUNC('month', NOW()) THEN subscription_id END) as new_subs,
        COUNT(DISTINCT CASE WHEN status = 'canceled' AND DATE_TRUNC('month', updated_at) = DATE_TRUNC('month', NOW()) THEN subscription_id END) as churned_subs,
        SUM(CASE WHEN status = 'active' THEN
          CASE billing_cycle
            WHEN 'monthly' THEN 1
            WHEN 'yearly' THEN 1.0/12
            ELSE 1
          END * (
            SELECT price_monthly FROM billing_plans WHERE plan_name = billing_subscriptions.plan_name
          )
        ELSE 0 END) as mrr
      FROM billing_subscriptions
      WHERE DATE_TRUNC('month', created_at) >= :'start_month'::date
      AND DATE_TRUNC('month', created_at) <= :'end_month'::date
      GROUP BY TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM')
    )
    SELECT month, active_subs, new_subs, churned_subs, ROUND(mrr::numeric, 2) as mrr
    FROM monthly_stats
    ORDER BY month DESC;
  " "tuples" "start_month" "$start_month" "end_month" "$end_month" | \
  while IFS='|' read -r month active new churned mrr; do
    printf "║ %-14s │ %11s │ %9s │ %9s │ \$%6s ║\n" \
      "$(printf '%s' "$month" | tr -d ' ')" \
      "$(printf '%s' "$active" | tr -d ' ')" \
      "$(printf '%s' "$new" | tr -d ' ')" \
      "$(printf '%s' "$churned" | tr -d ' ')" \
      "$(printf '%s' "$mrr" | tr -d ' ')"
  done

  printf "╚════════════════╧═════════════╧═══════════╧═══════════╧════════╝\n"
}

# MRR report as CSV
report_mrr_csv() {
  local start_month="$1"
  local end_month="$2"

  printf "Month,Active Subscriptions,New Subscriptions,Churned Subscriptions,MRR\n"

  billing_db_query "
    SELECT
      TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') as month,
      COUNT(DISTINCT CASE WHEN status = 'active' THEN subscription_id END) as active_subs,
      COUNT(DISTINCT CASE WHEN DATE_TRUNC('month', created_at) = DATE_TRUNC('month', NOW()) THEN subscription_id END) as new_subs,
      COUNT(DISTINCT CASE WHEN status = 'canceled' THEN subscription_id END) as churned_subs,
      ROUND(SUM(CASE WHEN status = 'active' THEN 99 ELSE 0 END)::numeric, 2) as mrr
    FROM billing_subscriptions
    WHERE DATE_TRUNC('month', created_at) >= :'start_month'::date
    AND DATE_TRUNC('month', created_at) <= :'end_month'::date
    GROUP BY TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM')
    ORDER BY month DESC;
  " "csv" "start_month" "$start_month" "end_month" "$end_month"
}

# MRR report as JSON
report_mrr_json() {
  local start_month="$1"
  local end_month="$2"

  billing_db_query "
    SELECT json_build_object(
      'report', 'Monthly Recurring Revenue',
      'period', json_build_object(
        'start', :'start_month',
        'end', :'end_month'
      ),
      'data', (
        SELECT json_agg(
          json_build_object(
            'month', month,
            'active_subscriptions', active_subs,
            'new_subscriptions', new_subs,
            'churned_subscriptions', churned_subs,
            'mrr', mrr
          )
        )
        FROM (
          SELECT
            TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM') as month,
            COUNT(DISTINCT CASE WHEN status = 'active' THEN subscription_id END) as active_subs,
            COUNT(DISTINCT CASE WHEN DATE_TRUNC('month', created_at) = DATE_TRUNC('month', NOW()) THEN subscription_id END) as new_subs,
            COUNT(DISTINCT CASE WHEN status = 'canceled' THEN subscription_id END) as churned_subs,
            ROUND(SUM(CASE WHEN status = 'active' THEN 99 ELSE 0 END)::numeric, 2) as mrr
          FROM billing_subscriptions
          WHERE DATE_TRUNC('month', created_at) >= :'start_month'::date
          AND DATE_TRUNC('month', created_at) <= :'end_month'::date
          GROUP BY TO_CHAR(DATE_TRUNC('month', created_at), 'YYYY-MM')
          ORDER BY month DESC
        ) m
      )
    );
  " "tuples" "start_month" "$start_month" "end_month" "$end_month"
}

# ============================================================================
# Customer Reports
# ============================================================================

# Generate customer lifetime value (CLV) report
report_clv() {
  local limit="${1:-50}"
  local format="${2:-table}"

  case "$format" in
    table)
      report_clv_table "$limit"
      ;;
    csv)
      report_clv_csv "$limit"
      ;;
    json)
      report_clv_json "$limit"
      ;;
    *)
      error "Invalid format: $format"
      return 1
      ;;
  esac
}

# CLV report as table
report_clv_table() {
  local limit="$1"

  printf "╔════════════════════════════════════════════════════════════════╗\n"
  printf "║           Customer Lifetime Value (CLV) Report                ║\n"
  printf "╠════════════════════════════════════════════════════════════════╣\n"
  printf "║ Customer                       │ Total Paid │ Avg Monthly │   ║\n"
  printf "╠════════════════════════════════╪════════════╪═════════════╪═══╣\n"

  billing_db_query "
    SELECT
      c.customer_id,
      c.name,
      COALESCE(SUM(i.total_amount), 0) as total_paid,
      ROUND(COALESCE(SUM(i.total_amount), 0) /
        NULLIF(EXTRACT(MONTH FROM AGE(NOW(), c.created_at)), 0), 2) as avg_monthly
    FROM billing_customers c
    LEFT JOIN billing_invoices i ON i.customer_id = c.customer_id AND i.status = 'paid'
    WHERE c.deleted_at IS NULL
    GROUP BY c.customer_id, c.name, c.created_at
    ORDER BY total_paid DESC
    LIMIT :'limit';
  " "tuples" "limit" "$limit" | \
  while IFS='|' read -r cust_id name total avg; do
    printf "║ %-30s │ \$%9s │ \$%10s │   ║\n" \
      "$(printf '%s' "$name" | tr -d ' ' | head -c 30)" \
      "$(printf '%s' "$total" | tr -d ' ')" \
      "$(printf '%s' "$avg" | tr -d ' ')"
  done

  printf "╚════════════════════════════════╧════════════╧═════════════╧═══╝\n"
}

# CLV report as CSV
report_clv_csv() {
  local limit="$1"

  printf "Customer ID,Name,Total Paid,Average Monthly\n"

  billing_db_query "
    SELECT
      c.customer_id,
      c.name,
      COALESCE(SUM(i.total_amount), 0) as total_paid,
      ROUND(COALESCE(SUM(i.total_amount), 0) /
        NULLIF(EXTRACT(MONTH FROM AGE(NOW(), c.created_at)), 0), 2) as avg_monthly
    FROM billing_customers c
    LEFT JOIN billing_invoices i ON i.customer_id = c.customer_id AND i.status = 'paid'
    WHERE c.deleted_at IS NULL
    GROUP BY c.customer_id, c.name, c.created_at
    ORDER BY total_paid DESC
    LIMIT :'limit';
  " "csv" "limit" "$limit"
}

# CLV report as JSON
report_clv_json() {
  local limit="$1"

  billing_db_query "
    SELECT json_agg(
      json_build_object(
        'customer_id', customer_id,
        'name', name,
        'total_paid', total_paid,
        'avg_monthly', avg_monthly
      )
    )
    FROM (
      SELECT
        c.customer_id,
        c.name,
        COALESCE(SUM(i.total_amount), 0) as total_paid,
        ROUND(COALESCE(SUM(i.total_amount), 0) /
          NULLIF(EXTRACT(MONTH FROM AGE(NOW(), c.created_at)), 0), 2) as avg_monthly
      FROM billing_customers c
      LEFT JOIN billing_invoices i ON i.customer_id = c.customer_id AND i.status = 'paid'
      WHERE c.deleted_at IS NULL
      GROUP BY c.customer_id, c.name, c.created_at
      ORDER BY total_paid DESC
      LIMIT :'limit'
    ) clv;
  " "tuples" "limit" "$limit"
}

# ============================================================================
# Invoice Reports
# ============================================================================

# Generate invoice aging report (accounts receivable)
report_aging() {
  local format="${1:-table}"

  case "$format" in
    table)
      report_aging_table
      ;;
    csv)
      report_aging_csv
      ;;
    json)
      report_aging_json
      ;;
    *)
      error "Invalid format: $format"
      return 1
      ;;
  esac
}

# Aging report as table
report_aging_table() {
  printf "╔════════════════════════════════════════════════════════════════╗\n"
  printf "║              Invoice Aging Report (A/R)                       ║\n"
  printf "╠════════════════════════════════════════════════════════════════╣\n"
  printf "║ Age Range     │ Count │ Total Amount │ Percentage            ║\n"
  printf "╠═══════════════╪═══════╪══════════════╪═══════════════════════╣\n"

  billing_db_query "
    WITH aging_data AS (
      SELECT
        CASE
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 0 THEN 'Current'
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 30 THEN '1-30 days'
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 60 THEN '31-60 days'
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 90 THEN '61-90 days'
          ELSE '90+ days'
        END as age_range,
        COUNT(*) as invoice_count,
        SUM(total_amount) as total_amount
      FROM billing_invoices
      WHERE status = 'open'
      GROUP BY age_range
    ),
    totals AS (
      SELECT SUM(total_amount) as grand_total FROM aging_data
    )
    SELECT
      a.age_range,
      a.invoice_count,
      ROUND(a.total_amount::numeric, 2) as total_amount,
      ROUND((a.total_amount / t.grand_total * 100)::numeric, 1) as percentage
    FROM aging_data a, totals t
    ORDER BY
      CASE a.age_range
        WHEN 'Current' THEN 1
        WHEN '1-30 days' THEN 2
        WHEN '31-60 days' THEN 3
        WHEN '61-90 days' THEN 4
        ELSE 5
      END;
  " "tuples" | \
  while IFS='|' read -r age_range count total pct; do
    printf "║ %-13s │ %5s │ \$%11s │ %20s%% ║\n" \
      "$(printf '%s' "$age_range" | tr -d ' ')" \
      "$(printf '%s' "$count" | tr -d ' ')" \
      "$(printf '%s' "$total" | tr -d ' ')" \
      "$(printf '%s' "$pct" | tr -d ' ')"
  done

  printf "╚═══════════════╧═══════╧══════════════╧═══════════════════════╝\n"
}

# Aging report as CSV
report_aging_csv() {
  printf "Age Range,Invoice Count,Total Amount,Percentage\n"

  billing_db_query "
    WITH aging_data AS (
      SELECT
        CASE
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 0 THEN 'Current'
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 30 THEN '1-30 days'
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 60 THEN '31-60 days'
          WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 90 THEN '61-90 days'
          ELSE '90+ days'
        END as age_range,
        COUNT(*) as invoice_count,
        SUM(total_amount) as total_amount
      FROM billing_invoices
      WHERE status = 'open'
      GROUP BY age_range
    ),
    totals AS (
      SELECT SUM(total_amount) as grand_total FROM aging_data
    )
    SELECT
      a.age_range,
      a.invoice_count,
      ROUND(a.total_amount::numeric, 2) as total_amount,
      ROUND((a.total_amount / t.grand_total * 100)::numeric, 1) as percentage
    FROM aging_data a, totals t
    ORDER BY
      CASE a.age_range
        WHEN 'Current' THEN 1
        WHEN '1-30 days' THEN 2
        WHEN '31-60 days' THEN 3
        WHEN '61-90 days' THEN 4
        ELSE 5
      END;
  " "csv"
}

# Aging report as JSON
report_aging_json() {
  billing_db_query "
    SELECT json_build_object(
      'report', 'Invoice Aging (A/R)',
      'generated_at', NOW(),
      'data', (
        SELECT json_agg(
          json_build_object(
            'age_range', age_range,
            'invoice_count', invoice_count,
            'total_amount', total_amount,
            'percentage', percentage
          )
        )
        FROM (
          WITH aging_data AS (
            SELECT
              CASE
                WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 0 THEN 'Current'
                WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 30 THEN '1-30 days'
                WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 60 THEN '31-60 days'
                WHEN EXTRACT(DAY FROM AGE(NOW(), due_date)) <= 90 THEN '61-90 days'
                ELSE '90+ days'
              END as age_range,
              COUNT(*) as invoice_count,
              SUM(total_amount) as total_amount
            FROM billing_invoices
            WHERE status = 'open'
            GROUP BY age_range
          ),
          totals AS (
            SELECT SUM(total_amount) as grand_total FROM aging_data
          )
          SELECT
            a.age_range,
            a.invoice_count,
            ROUND(a.total_amount::numeric, 2) as total_amount,
            ROUND((a.total_amount / t.grand_total * 100)::numeric, 1) as percentage
          FROM aging_data a, totals t
        ) aging
      )
    );
  " "tuples"
}

# ============================================================================
# Usage Reports
# ============================================================================

# Generate usage trends report
report_usage_trends() {
  local service="${1:-all}"
  local days="${2:-30}"
  local format="${3:-table}"

  case "$format" in
    table)
      report_usage_trends_table "$service" "$days"
      ;;
    csv)
      report_usage_trends_csv "$service" "$days"
      ;;
    json)
      report_usage_trends_json "$service" "$days"
      ;;
    *)
      error "Invalid format: $format"
      return 1
      ;;
  esac
}

# Usage trends as table
report_usage_trends_table() {
  local service="$1"
  local days="$2"

  local service_filter=""
  if [[ "$service" != "all" ]]; then
    service_filter="AND service_name = '${service}'"
  fi

  printf "╔════════════════════════════════════════════════════════════════╗\n"
  printf "║              Usage Trends Report (Last %2d days)              ║\n" "$days"
  printf "╠════════════════════════════════════════════════════════════════╣\n"
  printf "║ Date       │ Service      │ Usage      │ Change from Prev   ║\n"
  printf "╠════════════╪══════════════╪════════════╪════════════════════╣\n"

  billing_db_query "
    WITH daily_usage AS (
      SELECT
        DATE(recorded_at) as date,
        service_name,
        SUM(quantity) as total_usage,
        LAG(SUM(quantity)) OVER (PARTITION BY service_name ORDER BY DATE(recorded_at)) as prev_usage
      FROM billing_usage_records
      WHERE recorded_at >= NOW() - INTERVAL '${days} days'
      ${service_filter}
      GROUP BY DATE(recorded_at), service_name
    )
    SELECT
      date,
      service_name,
      total_usage,
      CASE
        WHEN prev_usage IS NULL THEN 'N/A'
        WHEN prev_usage = 0 THEN '+100%'
        ELSE CONCAT(
          CASE WHEN total_usage > prev_usage THEN '+' ELSE '' END,
          ROUND(((total_usage - prev_usage) / prev_usage * 100)::numeric, 1),
          '%'
        )
      END as change
    FROM daily_usage
    ORDER BY date DESC, service_name
    LIMIT 20;
  " "tuples" | \
  while IFS='|' read -r date service usage change; do
    printf "║ %-10s │ %-12s │ %10s │ %18s ║\n" \
      "$(printf '%s' "$date" | tr -d ' ')" \
      "$(printf '%s' "$service" | tr -d ' ')" \
      "$(printf '%s' "$usage" | tr -d ' ')" \
      "$(printf '%s' "$change" | tr -d ' ')"
  done

  printf "╚════════════╧══════════════╧════════════╧════════════════════╝\n"
}

# Usage trends as CSV
report_usage_trends_csv() {
  local service="$1"
  local days="$2"

  local service_filter=""
  if [[ "$service" != "all" ]]; then
    service_filter="AND service_name = '${service}'"
  fi

  printf "Date,Service,Usage,Change\n"

  billing_db_query "
    WITH daily_usage AS (
      SELECT
        DATE(recorded_at) as date,
        service_name,
        SUM(quantity) as total_usage,
        LAG(SUM(quantity)) OVER (PARTITION BY service_name ORDER BY DATE(recorded_at)) as prev_usage
      FROM billing_usage_records
      WHERE recorded_at >= NOW() - INTERVAL '${days} days'
      ${service_filter}
      GROUP BY DATE(recorded_at), service_name
    )
    SELECT
      date,
      service_name,
      total_usage,
      ROUND(((total_usage - COALESCE(prev_usage, 0)) / NULLIF(prev_usage, 0) * 100)::numeric, 1) as change_pct
    FROM daily_usage
    ORDER BY date DESC, service_name;
  " "csv"
}

# Usage trends as JSON
report_usage_trends_json() {
  local service="$1"
  local days="$2"

  local service_filter=""
  if [[ "$service" != "all" ]]; then
    service_filter="AND service_name = '${service}'"
  fi

  billing_db_query "
    SELECT json_agg(
      json_build_object(
        'date', date,
        'service', service_name,
        'usage', total_usage,
        'change_percent', change_pct
      )
    )
    FROM (
      WITH daily_usage AS (
        SELECT
          DATE(recorded_at) as date,
          service_name,
          SUM(quantity) as total_usage,
          LAG(SUM(quantity)) OVER (PARTITION BY service_name ORDER BY DATE(recorded_at)) as prev_usage
        FROM billing_usage_records
        WHERE recorded_at >= NOW() - INTERVAL '${days} days'
        ${service_filter}
        GROUP BY DATE(recorded_at), service_name
      )
      SELECT
        date,
        service_name,
        total_usage,
        ROUND(((total_usage - COALESCE(prev_usage, 0)) / NULLIF(prev_usage, 0) * 100)::numeric, 1) as change_pct
      FROM daily_usage
      ORDER BY date DESC, service_name
    ) trends;
  " "tuples"
}

# ============================================================================
# Churn Reports
# ============================================================================

# Generate churn analysis report
report_churn() {
  local months="${1:-12}"
  local format="${2:-table}"

  case "$format" in
    table)
      report_churn_table "$months"
      ;;
    csv)
      report_churn_csv "$months"
      ;;
    json)
      report_churn_json "$months"
      ;;
    *)
      error "Invalid format: $format"
      return 1
      ;;
  esac
}

# Churn report as table
report_churn_table() {
  local months="$1"

  printf "╔════════════════════════════════════════════════════════════════╗\n"
  printf "║                 Churn Analysis Report                         ║\n"
  printf "╠════════════════════════════════════════════════════════════════╣\n"
  printf "║ Month          │ Active │ Churned │ Churn Rate              ║\n"
  printf "╠════════════════╪════════╪═════════╪═════════════════════════╣\n"

  billing_db_query "
    WITH monthly_churn AS (
      SELECT
        TO_CHAR(DATE_TRUNC('month', updated_at), 'YYYY-MM') as month,
        COUNT(DISTINCT CASE WHEN status = 'active' THEN subscription_id END) as active_count,
        COUNT(DISTINCT CASE WHEN status = 'canceled' AND DATE_TRUNC('month', updated_at) = DATE_TRUNC('month', NOW()) THEN subscription_id END) as churned_count
      FROM billing_subscriptions
      WHERE DATE_TRUNC('month', updated_at) >= DATE_TRUNC('month', NOW()) - INTERVAL '${months} months'
      GROUP BY TO_CHAR(DATE_TRUNC('month', updated_at), 'YYYY-MM')
    )
    SELECT
      month,
      active_count,
      churned_count,
      ROUND((churned_count::numeric / NULLIF(active_count, 0) * 100), 2) as churn_rate
    FROM monthly_churn
    ORDER BY month DESC;
  " "tuples" | \
  while IFS='|' read -r month active churned rate; do
    printf "║ %-14s │ %6s │ %7s │ %22s%% ║\n" \
      "$(printf '%s' "$month" | tr -d ' ')" \
      "$(printf '%s' "$active" | tr -d ' ')" \
      "$(printf '%s' "$churned" | tr -d ' ')" \
      "$(printf '%s' "$rate" | tr -d ' ')"
  done

  printf "╚════════════════╧════════╧═════════╧═════════════════════════╝\n"
}

# Churn report as CSV
report_churn_csv() {
  local months="$1"

  printf "Month,Active Count,Churned Count,Churn Rate\n"

  billing_db_query "
    WITH monthly_churn AS (
      SELECT
        TO_CHAR(DATE_TRUNC('month', updated_at), 'YYYY-MM') as month,
        COUNT(DISTINCT CASE WHEN status = 'active' THEN subscription_id END) as active_count,
        COUNT(DISTINCT CASE WHEN status = 'canceled' THEN subscription_id END) as churned_count
      FROM billing_subscriptions
      WHERE DATE_TRUNC('month', updated_at) >= DATE_TRUNC('month', NOW()) - INTERVAL '${months} months'
      GROUP BY TO_CHAR(DATE_TRUNC('month', updated_at), 'YYYY-MM')
    )
    SELECT
      month,
      active_count,
      churned_count,
      ROUND((churned_count::numeric / NULLIF(active_count, 0) * 100), 2) as churn_rate
    FROM monthly_churn
    ORDER BY month DESC;
  " "csv"
}

# Churn report as JSON
report_churn_json() {
  local months="$1"

  billing_db_query "
    SELECT json_agg(
      json_build_object(
        'month', month,
        'active_count', active_count,
        'churned_count', churned_count,
        'churn_rate', churn_rate
      )
    )
    FROM (
      WITH monthly_churn AS (
        SELECT
          TO_CHAR(DATE_TRUNC('month', updated_at), 'YYYY-MM') as month,
          COUNT(DISTINCT CASE WHEN status = 'active' THEN subscription_id END) as active_count,
          COUNT(DISTINCT CASE WHEN status = 'canceled' THEN subscription_id END) as churned_count
        FROM billing_subscriptions
        WHERE DATE_TRUNC('month', updated_at) >= DATE_TRUNC('month', NOW()) - INTERVAL '${months} months'
        GROUP BY TO_CHAR(DATE_TRUNC('month', updated_at), 'YYYY-MM')
      )
      SELECT
        month,
        active_count,
        churned_count,
        ROUND((churned_count::numeric / NULLIF(active_count, 0) * 100), 2) as churn_rate
      FROM monthly_churn
      ORDER BY month DESC
    ) churn;
  " "tuples"
}

# ============================================================================
# Dashboard Summary
# ============================================================================

# Generate executive dashboard summary
report_dashboard() {
  local format="${1:-table}"

  case "$format" in
    table)
      report_dashboard_table
      ;;
    json)
      report_dashboard_json
      ;;
    *)
      error "Invalid format: $format"
      return 1
      ;;
  esac
}

# Dashboard as table
report_dashboard_table() {
  printf "╔════════════════════════════════════════════════════════════════╗\n"
  printf "║                  Executive Dashboard                          ║\n"
  printf "╠════════════════════════════════════════════════════════════════╣\n"

  # Get key metrics
  local total_customers active_subs total_mrr open_invoices total_ar

  total_customers=$(billing_db_query "SELECT COUNT(*) FROM billing_customers WHERE deleted_at IS NULL;" "tuples" 2>/dev/null | tr -d ' ')
  active_subs=$(billing_db_query "SELECT COUNT(*) FROM billing_subscriptions WHERE status = 'active';" "tuples" 2>/dev/null | tr -d ' ')
  total_mrr=$(billing_db_query "SELECT COALESCE(SUM(99), 0) FROM billing_subscriptions WHERE status = 'active';" "tuples" 2>/dev/null | tr -d ' ')
  open_invoices=$(billing_db_query "SELECT COUNT(*) FROM billing_invoices WHERE status = 'open';" "tuples" 2>/dev/null | tr -d ' ')
  total_ar=$(billing_db_query "SELECT COALESCE(SUM(total_amount), 0) FROM billing_invoices WHERE status = 'open';" "tuples" 2>/dev/null | tr -d ' ')

  printf "║                                                                ║\n"
  printf "║  Total Customers:              %30s  ║\n" "$total_customers"
  printf "║  Active Subscriptions:         %30s  ║\n" "$active_subs"
  printf "║  Monthly Recurring Revenue:    \$%29s  ║\n" "$total_mrr"
  printf "║  Open Invoices:                %30s  ║\n" "$open_invoices"
  printf "║  Accounts Receivable:          \$%29s  ║\n" "$total_ar"
  printf "║                                                                ║\n"
  printf "╚════════════════════════════════════════════════════════════════╝\n"
}

# Dashboard as JSON
report_dashboard_json() {
  billing_db_query "
    SELECT json_build_object(
      'report', 'Executive Dashboard',
      'generated_at', NOW(),
      'metrics', json_build_object(
        'total_customers', (SELECT COUNT(*) FROM billing_customers WHERE deleted_at IS NULL),
        'active_subscriptions', (SELECT COUNT(*) FROM billing_subscriptions WHERE status = 'active'),
        'monthly_recurring_revenue', (SELECT COALESCE(SUM(99), 0) FROM billing_subscriptions WHERE status = 'active'),
        'open_invoices', (SELECT COUNT(*) FROM billing_invoices WHERE status = 'open'),
        'accounts_receivable', (SELECT COALESCE(SUM(total_amount), 0) FROM billing_invoices WHERE status = 'open')
      )
    );
  " "tuples"
}

# Export functions
export -f reports_init
export -f report_mrr
export -f report_mrr_table
export -f report_mrr_csv
export -f report_mrr_json
export -f report_clv
export -f report_clv_table
export -f report_clv_csv
export -f report_clv_json
export -f report_aging
export -f report_aging_table
export -f report_aging_csv
export -f report_aging_json
export -f report_usage_trends
export -f report_usage_trends_table
export -f report_usage_trends_csv
export -f report_usage_trends_json
export -f report_churn
export -f report_churn_table
export -f report_churn_csv
export -f report_churn_json
export -f report_dashboard
export -f report_dashboard_table
export -f report_dashboard_json
