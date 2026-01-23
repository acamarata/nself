#!/usr/bin/env bash

# email-providers.sh - Email provider configuration and management
# Supports both SMTP and API-based email delivery

set -e

# Source shared utilities
CLI_SCRIPT_DIR="$(dirname "${BASH_SOURCE[0]}")"
SCRIPT_DIR="$CLI_SCRIPT_DIR"

# Source utilities - check if already sourced (from nself.sh)
if [[ -z "${DISPLAY_UTILS_SOURCED:-}" ]]; then
  source "$CLI_SCRIPT_DIR/../lib/utils/env.sh"
  source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
  source "$CLI_SCRIPT_DIR/../lib/utils/header.sh"
fi

# Only source hooks if running standalone
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  source "$CLI_SCRIPT_DIR/../lib/hooks/pre-command.sh"
  source "$CLI_SCRIPT_DIR/../lib/hooks/post-command.sh"
fi

# ============================================================
# API Email Provider Support (Added in v0.4.7)
# ============================================================
# API-based email providers offer advantages over SMTP:
# - Better deliverability (dedicated IP pools)
# - Webhooks for delivery tracking
# - Built-in analytics and reporting
# - No port 25/587 blocking issues
# - Easier firewall configuration (HTTPS only)
# ============================================================

# Supported API providers (6 core providers)
API_PROVIDERS="elastic-email sendgrid aws-ses resend postmark mailgun"

# Function to get API provider template
get_api_provider_template() {
  local provider="$1"

  case "$provider" in
  elastic-email|elastic)
    cat <<'EOF'
# Elastic Email API Configuration
# Docs: https://elasticemail.com/developers/api-documentation
AUTH_EMAIL_PROVIDER=elastic-email
AUTH_EMAIL_API_KEY=YOUR_ELASTIC_EMAIL_API_KEY
AUTH_EMAIL_SENDER=noreply@yourdomain.com

# Optional settings
AUTH_EMAIL_SENDER_NAME=My App
AUTH_EMAIL_REPLY_TO=support@yourdomain.com

# Get your API key from:
# https://elasticemail.com/account#/settings/new/manage-api
#
# Pricing: Pay-as-you-go, $0.09 per 1000 emails
# Free tier: 100 emails/day
EOF
    ;;

  sendgrid)
    cat <<'EOF'
# SendGrid API Configuration
# Docs: https://docs.sendgrid.com/api-reference/mail-send
AUTH_EMAIL_PROVIDER=sendgrid
AUTH_EMAIL_API_KEY=SG.xxxxxxxxxxxxxxxxxxxx
AUTH_EMAIL_SENDER=noreply@yourdomain.com

# Optional settings
AUTH_EMAIL_SENDER_NAME=My App
AUTH_EMAIL_REPLY_TO=support@yourdomain.com

# Get your API key from:
# https://app.sendgrid.com/settings/api_keys
# Create key with "Mail Send" permission
#
# Pricing: Free tier includes 100 emails/day
# Paid: Starting at $19.95/mo for 50k emails
EOF
    ;;

  aws-ses)
    cat <<'EOF'
# AWS SES API Configuration
# Docs: https://docs.aws.amazon.com/ses/latest/APIReference/
AUTH_EMAIL_PROVIDER=aws-ses
AUTH_EMAIL_API_KEY=YOUR_AWS_ACCESS_KEY_ID
AUTH_EMAIL_API_SECRET=YOUR_AWS_SECRET_ACCESS_KEY
AUTH_EMAIL_REGION=us-east-1
AUTH_EMAIL_SENDER=noreply@yourdomain.com

# Optional settings
AUTH_EMAIL_SENDER_NAME=My App
AUTH_EMAIL_REPLY_TO=support@yourdomain.com

# Setup steps:
# 1. Verify your domain in SES console
# 2. Move out of sandbox mode for production
# 3. Create IAM user with ses:SendEmail permission
#
# Pricing: $0.10 per 1000 emails (very cost-effective)
# Free: 62,000 emails/mo if sent from EC2
EOF
    ;;

  resend)
    cat <<'EOF'
# Resend API Configuration
# Docs: https://resend.com/docs/api-reference/emails/send-email
AUTH_EMAIL_PROVIDER=resend
AUTH_EMAIL_API_KEY=re_xxxxxxxxxxxxxxxxxxxx
AUTH_EMAIL_SENDER=noreply@yourdomain.com

# Optional settings
AUTH_EMAIL_SENDER_NAME=My App
AUTH_EMAIL_REPLY_TO=support@yourdomain.com

# Get your API key from:
# https://resend.com/api-keys
#
# Pricing: Free tier includes 3,000 emails/mo
# Paid: Starting at $20/mo for 50k emails
# Note: Modern, developer-first email service
EOF
    ;;

  postmark)
    cat <<'EOF'
# Postmark API Configuration
# Docs: https://postmarkapp.com/developer/api/email-api
AUTH_EMAIL_PROVIDER=postmark
AUTH_EMAIL_API_KEY=YOUR_POSTMARK_SERVER_TOKEN
AUTH_EMAIL_SENDER=noreply@yourdomain.com

# Optional settings
AUTH_EMAIL_SENDER_NAME=My App
AUTH_EMAIL_REPLY_TO=support@yourdomain.com
AUTH_EMAIL_MESSAGE_STREAM=outbound

# Get your Server API Token from:
# https://account.postmarkapp.com/servers
#
# Pricing: $15/mo for 10k emails
# Focus: Transactional emails with high deliverability
EOF
    ;;

  mailgun)
    cat <<'EOF'
# Mailgun API Configuration
# Docs: https://documentation.mailgun.com/en/latest/api-sending-messages.html
AUTH_EMAIL_PROVIDER=mailgun
AUTH_EMAIL_API_KEY=YOUR_MAILGUN_API_KEY
AUTH_EMAIL_DOMAIN=mg.yourdomain.com
AUTH_EMAIL_SENDER=noreply@mg.yourdomain.com

# Optional settings
AUTH_EMAIL_SENDER_NAME=My App
AUTH_EMAIL_REPLY_TO=support@yourdomain.com
AUTH_EMAIL_REGION=us  # or 'eu' for EU region

# Get your API key from:
# https://app.mailgun.com/app/account/security/api_keys
#
# Pricing: Pay-as-you-go, first 1000 emails free
# Trial: 5,000 emails for 3 months
EOF
    ;;

  *)
    return 1
    ;;
  esac
}

# Detect if using API-based email
detect_api_provider() {
  local provider="${AUTH_EMAIL_PROVIDER:-}"

  if [[ -n "$provider" ]]; then
    echo "$provider"
  else
    echo "not-configured"
  fi
}

# API pre-flight connection check
api_preflight_check() {
  local provider="${AUTH_EMAIL_PROVIDER:-}"
  local api_key="${AUTH_EMAIL_API_KEY:-}"

  show_command_header "nself email check --api" "API Connection Pre-flight Check"
  echo

  if [[ -z "$provider" ]]; then
    log_error "API email provider not configured"
    log_info "Set AUTH_EMAIL_PROVIDER in your .env"
    log_info "Run: nself email configure --api <provider>"
    return 1
  fi

  if [[ -z "$api_key" ]]; then
    log_error "API key not configured"
    log_info "Set AUTH_EMAIL_API_KEY in your .env"
    return 1
  fi

  printf "Provider: %s\n" "$provider"
  echo
  log_info "Checking API connection..."
  echo

  local result=0

  case "$provider" in
  elastic-email|elastic)
    printf "  API endpoint... "
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
      "https://api.elasticemail.com/v2/account/load?apikey=$api_key" 2>/dev/null)
    if [[ "$response" == "200" ]]; then
      printf "\033[0;32m✓\033[0m (authenticated)\n"
    else
      printf "\033[0;31m✗\033[0m (HTTP $response)\n"
      result=1
    fi
    ;;

  sendgrid)
    printf "  API endpoint... "
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer $api_key" \
      "https://api.sendgrid.com/v3/user/profile" 2>/dev/null)
    if [[ "$response" == "200" ]]; then
      printf "\033[0;32m✓\033[0m (authenticated)\n"
    else
      printf "\033[0;31m✗\033[0m (HTTP $response)\n"
      result=1
    fi
    ;;

  aws-ses)
    printf "  API endpoint... "
    local region="${AUTH_EMAIL_REGION:-us-east-1}"
    # AWS SES requires signed requests, just check endpoint reachability
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
      "https://email.$region.amazonaws.com/" 2>/dev/null)
    if [[ "$response" == "403" ]] || [[ "$response" == "400" ]]; then
      # 403/400 means endpoint is reachable (auth required)
      printf "\033[0;32m✓\033[0m (endpoint reachable)\n"
      printf "  Credentials... "
      if [[ -n "${AUTH_EMAIL_API_SECRET:-}" ]]; then
        printf "\033[0;32m✓\033[0m (configured)\n"
      else
        printf "\033[0;31m✗\033[0m (AUTH_EMAIL_API_SECRET not set)\n"
        result=1
      fi
    else
      printf "\033[0;31m✗\033[0m (HTTP $response)\n"
      result=1
    fi
    ;;

  resend)
    printf "  API endpoint... "
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "Authorization: Bearer $api_key" \
      "https://api.resend.com/domains" 2>/dev/null)
    if [[ "$response" == "200" ]]; then
      printf "\033[0;32m✓\033[0m (authenticated)\n"
    else
      printf "\033[0;31m✗\033[0m (HTTP $response)\n"
      result=1
    fi
    ;;

  postmark)
    printf "  API endpoint... "
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
      -H "X-Postmark-Server-Token: $api_key" \
      "https://api.postmarkapp.com/server" 2>/dev/null)
    if [[ "$response" == "200" ]]; then
      printf "\033[0;32m✓\033[0m (authenticated)\n"
    else
      printf "\033[0;31m✗\033[0m (HTTP $response)\n"
      result=1
    fi
    ;;

  mailgun)
    printf "  API endpoint... "
    local domain="${AUTH_EMAIL_DOMAIN:-}"
    local region="${AUTH_EMAIL_REGION:-us}"
    local api_base="https://api.mailgun.net"
    [[ "$region" == "eu" ]] && api_base="https://api.eu.mailgun.net"

    if [[ -z "$domain" ]]; then
      printf "\033[0;31m✗\033[0m (AUTH_EMAIL_DOMAIN not set)\n"
      result=1
    else
      local response
      response=$(curl -s -o /dev/null -w "%{http_code}" \
        -u "api:$api_key" \
        "$api_base/v3/$domain" 2>/dev/null)
      if [[ "$response" == "200" ]]; then
        printf "\033[0;32m✓\033[0m (authenticated)\n"
      else
        printf "\033[0;31m✗\033[0m (HTTP $response)\n"
        result=1
      fi
    fi
    ;;

  *)
    log_error "Unknown API provider: $provider"
    result=1
    ;;
  esac

  echo
  if [[ $result -eq 0 ]]; then
    log_success "API connection check passed"
    log_info "Test email sending: nself email test --api <recipient>"
  else
    log_error "API connection check failed"
    log_info "Verify your API key and configuration"
  fi
  echo

  return $result
}

# Send email via API
send_api_email() {
  local recipient="$1"
  local subject="${2:-nself Email Test}"
  local body="${3:-This is a test email from nself to verify API configuration.}"

  local provider="${AUTH_EMAIL_PROVIDER:-}"
  local api_key="${AUTH_EMAIL_API_KEY:-}"
  local sender="${AUTH_EMAIL_SENDER:-noreply@${BASE_DOMAIN:-localhost}}"
  local sender_name="${AUTH_EMAIL_SENDER_NAME:-nself}"

  if [[ -z "$provider" ]] || [[ -z "$api_key" ]]; then
    log_error "API email not configured"
    return 1
  fi

  log_info "Sending via $provider API..."

  local result=0
  local response

  case "$provider" in
  elastic-email|elastic)
    response=$(curl -s -X POST "https://api.elasticemail.com/v2/email/send" \
      -d "apikey=$api_key" \
      -d "from=$sender" \
      -d "fromName=$sender_name" \
      -d "to=$recipient" \
      -d "subject=$subject" \
      -d "bodyText=$body" 2>/dev/null)
    if echo "$response" | grep -q '"success":true'; then
      log_success "Email sent successfully via Elastic Email"
    else
      log_error "Failed to send: $response"
      result=1
    fi
    ;;

  sendgrid)
    response=$(curl -s -X POST "https://api.sendgrid.com/v3/mail/send" \
      -H "Authorization: Bearer $api_key" \
      -H "Content-Type: application/json" \
      -d "{
        \"personalizations\": [{\"to\": [{\"email\": \"$recipient\"}]}],
        \"from\": {\"email\": \"$sender\", \"name\": \"$sender_name\"},
        \"subject\": \"$subject\",
        \"content\": [{\"type\": \"text/plain\", \"value\": \"$body\"}]
      }" -w "\n%{http_code}" 2>/dev/null)
    local http_code
    http_code=$(echo "$response" | tail -1)
    if [[ "$http_code" == "202" ]]; then
      log_success "Email sent successfully via SendGrid"
    else
      log_error "Failed to send (HTTP $http_code)"
      result=1
    fi
    ;;

  aws-ses)
    local region="${AUTH_EMAIL_REGION:-us-east-1}"
    local secret="${AUTH_EMAIL_API_SECRET:-}"
    if [[ -z "$secret" ]]; then
      log_error "AUTH_EMAIL_API_SECRET required for AWS SES"
      return 1
    fi
    # AWS SES requires v4 signature - use aws cli if available
    if command -v aws >/dev/null 2>&1; then
      AWS_ACCESS_KEY_ID="$api_key" \
      AWS_SECRET_ACCESS_KEY="$secret" \
      aws ses send-email \
        --region "$region" \
        --from "$sender" \
        --to "$recipient" \
        --subject "$subject" \
        --text "$body" 2>/dev/null && {
        log_success "Email sent successfully via AWS SES"
      } || {
        log_error "Failed to send via AWS SES"
        result=1
      }
    else
      log_error "AWS CLI required for SES API. Install: brew install awscli"
      log_info "Alternative: Use SMTP mode with SES SMTP credentials"
      result=1
    fi
    ;;

  resend)
    response=$(curl -s -X POST "https://api.resend.com/emails" \
      -H "Authorization: Bearer $api_key" \
      -H "Content-Type: application/json" \
      -d "{
        \"from\": \"$sender_name <$sender>\",
        \"to\": [\"$recipient\"],
        \"subject\": \"$subject\",
        \"text\": \"$body\"
      }" 2>/dev/null)
    if echo "$response" | grep -q '"id":'; then
      log_success "Email sent successfully via Resend"
    else
      log_error "Failed to send: $response"
      result=1
    fi
    ;;

  postmark)
    local stream="${AUTH_EMAIL_MESSAGE_STREAM:-outbound}"
    response=$(curl -s -X POST "https://api.postmarkapp.com/email" \
      -H "X-Postmark-Server-Token: $api_key" \
      -H "Content-Type: application/json" \
      -d "{
        \"From\": \"$sender\",
        \"To\": \"$recipient\",
        \"Subject\": \"$subject\",
        \"TextBody\": \"$body\",
        \"MessageStream\": \"$stream\"
      }" 2>/dev/null)
    if echo "$response" | grep -q '"MessageID":'; then
      log_success "Email sent successfully via Postmark"
    else
      log_error "Failed to send: $response"
      result=1
    fi
    ;;

  mailgun)
    local domain="${AUTH_EMAIL_DOMAIN:-}"
    local region="${AUTH_EMAIL_REGION:-us}"
    local api_base="https://api.mailgun.net"
    [[ "$region" == "eu" ]] && api_base="https://api.eu.mailgun.net"

    if [[ -z "$domain" ]]; then
      log_error "AUTH_EMAIL_DOMAIN required for Mailgun"
      return 1
    fi

    response=$(curl -s -X POST "$api_base/v3/$domain/messages" \
      -u "api:$api_key" \
      -F "from=$sender_name <$sender>" \
      -F "to=$recipient" \
      -F "subject=$subject" \
      -F "text=$body" 2>/dev/null)
    if echo "$response" | grep -q '"id":'; then
      log_success "Email sent successfully via Mailgun"
    else
      log_error "Failed to send: $response"
      result=1
    fi
    ;;

  *)
    log_error "Unknown API provider: $provider"
    result=1
    ;;
  esac

  return $result
}

# Configure API provider
configure_api_provider() {
  local provider="$1"
  local template

  if [[ -z "$provider" ]]; then
    show_command_header "nself email configure --api" "Configure API email provider"
    echo
    log_error "No provider specified"
    echo
    echo "Usage: nself email configure --api <provider>"
    echo
    printf "${COLOR_CYAN}➞ Supported API Providers${COLOR_RESET}\n"
    echo "  • elastic-email  - Elastic Email (budget-friendly)"
    echo "  • sendgrid       - SendGrid (popular, reliable)"
    echo "  • aws-ses        - AWS SES (cost-effective at scale)"
    echo "  • resend         - Resend (modern, developer-first)"
    echo "  • postmark       - Postmark (transactional focus)"
    echo "  • mailgun        - Mailgun (developer-friendly)"
    echo
    printf "${COLOR_CYAN}➞ Why API over SMTP?${COLOR_RESET}\n"
    echo "  • Better deliverability (dedicated IP pools)"
    echo "  • Webhooks for delivery tracking"
    echo "  • Built-in analytics and reporting"
    echo "  • No port blocking issues (HTTPS only)"
    echo "  • Easier firewall configuration"
    echo
    return 1
  fi

  template=$(get_api_provider_template "$provider") || {
    show_command_header "nself email configure --api" "Configure API email provider"
    echo
    log_error "Unknown API provider: $provider"
    echo
    echo "Supported providers: $API_PROVIDERS"
    return 1
  }

  show_command_header "nself email configure --api" "Configure $provider API"
  echo

  # Show the template
  echo "Add these settings to your .env.local file:"
  echo ""
  echo "$template"
  echo ""

  # Ask if user wants to append to .env.local
  printf "Would you like to append these settings to .env.local? [y/N] "
  read -r REPLY
  REPLY=$(echo "$REPLY" | tr '[:upper:]' '[:lower:]')
  if [[ "$REPLY" == "y" ]] || [[ "$REPLY" == "yes" ]]; then
    # Backup current .env.local if exists
    if [[ -f ".env.local" ]]; then
      cp .env.local .env.local.backup
    fi

    # Append configuration
    {
      echo ""
      echo "# ============================================"
      echo "# Email API Configuration - $provider"
      echo "# Added on $(date)"
      echo "# ============================================"
      echo "$template"
    } >> .env.local

    log_success "Configuration added to .env.local"
    [[ -f ".env.local.backup" ]] && log_info "Backup saved: .env.local.backup"
    echo
    echo "Next steps:"
    echo "  1. Update placeholder values with actual credentials"
    echo "  2. Run: nself build"
    echo "  3. Run: nself email check --api"
    echo "  4. Run: nself email test --api your@email.com"
  fi
}

# Helper functions

# Function to get provider template
get_provider_template() {
  local provider="$1"

  case "$provider" in
  sendgrid)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.sendgrid.net
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=apikey
AUTH_SMTP_PASS=YOUR_SENDGRID_API_KEY
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Get your API key from: https://app.sendgrid.com/settings/api_keys
EOF
    ;;

  aws-ses)
    cat <<'EOF'
AUTH_SMTP_HOST=email-smtp.us-east-1.amazonaws.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=YOUR_AWS_SMTP_USERNAME
AUTH_SMTP_PASS=YOUR_AWS_SMTP_PASSWORD
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Note: SMTP credentials are different from AWS access keys
# Generate at: https://console.aws.amazon.com/ses/home#smtp-settings:
EOF
    ;;

  mailgun)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.mailgun.org
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=postmaster@mg.yourdomain.com
AUTH_SMTP_PASS=YOUR_MAILGUN_PASSWORD
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@mg.yourdomain.com
# EU Region? Use: smtp.eu.mailgun.org
# Get credentials from: https://app.mailgun.com/app/sending/domains
EOF
    ;;

  postmark)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.postmarkapp.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=YOUR_POSTMARK_SERVER_TOKEN
AUTH_SMTP_PASS=YOUR_POSTMARK_SERVER_TOKEN
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Note: Use the same Server Token for both user and password
# Get token from: https://account.postmarkapp.com/servers
EOF
    ;;

  gmail)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.gmail.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=your-email@gmail.com
AUTH_SMTP_PASS=YOUR_APP_PASSWORD
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=your-email@gmail.com
# IMPORTANT: Use App Password, not your regular password
# Enable 2FA and create app password at: https://myaccount.google.com/apppasswords
EOF
    ;;

  outlook)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.office365.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=your-email@outlook.com
AUTH_SMTP_PASS=YOUR_PASSWORD
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=your-email@outlook.com
# For custom domains, use your full email as username
EOF
    ;;

  brevo)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp-relay.brevo.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=YOUR_LOGIN_EMAIL
AUTH_SMTP_PASS=YOUR_SMTP_KEY
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Get SMTP key from: https://app.brevo.com/settings/keys/smtp
EOF
    ;;

  resend)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.resend.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=resend
AUTH_SMTP_PASS=YOUR_RESEND_API_KEY
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Get API key from: https://resend.com/api-keys
EOF
    ;;

  sparkpost)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.sparkpostmail.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=SMTP_Injection
AUTH_SMTP_PASS=YOUR_SPARKPOST_API_KEY
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# EU Region? Use: smtp.eu.sparkpostmail.com
# Get API key from: https://app.sparkpost.com/account/api-keys
EOF
    ;;

  mandrill)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.mandrillapp.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=YOUR_MANDRILL_USERNAME
AUTH_SMTP_PASS=YOUR_MANDRILL_API_KEY
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Get API key from: https://mandrillapp.com/settings/index
EOF
    ;;

  elastic)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.elasticemail.com
AUTH_SMTP_PORT=2525
AUTH_SMTP_USER=YOUR_ELASTIC_USERNAME
AUTH_SMTP_PASS=YOUR_ELASTIC_PASSWORD
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Alternative ports: 25, 587
# Get credentials from: https://elasticemail.com/account#/settings/new/manage-smtp
EOF
    ;;

  smtp2go)
    cat <<'EOF'
AUTH_SMTP_HOST=mail.smtp2go.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=YOUR_SMTP2GO_USERNAME
AUTH_SMTP_PASS=YOUR_SMTP2GO_PASSWORD
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Alternative ports: 25, 2525, 8025
# Get credentials from: https://app.smtp2go.com/settings/users
EOF
    ;;

  mailersend)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.mailersend.net
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=YOUR_MAILERSEND_USERNAME
AUTH_SMTP_PASS=YOUR_MAILERSEND_PASSWORD
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Get credentials from: https://app.mailersend.com/domains
EOF
    ;;

  postfix)
    cat <<'EOF'
AUTH_SMTP_HOST=localhost
AUTH_SMTP_PORT=25
AUTH_SMTP_USER=""
AUTH_SMTP_PASS=""
AUTH_SMTP_SECURE=false
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Self-hosted Postfix mail server
# Ensure Postfix is configured with proper SPF, DKIM, and DMARC records
# Reference: https://www.postfix.org/BASIC_CONFIGURATION_README.html
EOF
    ;;

  mailchimp)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.mandrillapp.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=YOUR_MAILCHIMP_USERNAME
AUTH_SMTP_PASS=YOUR_MANDRILL_API_KEY
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Mailchimp Transactional (formerly Mandrill)
# Get API key from: https://mandrillapp.com/settings
EOF
    ;;

  custom)
    cat <<'EOF'
AUTH_SMTP_HOST=smtp.yourdomain.com
AUTH_SMTP_PORT=587
AUTH_SMTP_USER=smtp-user
AUTH_SMTP_PASS=smtp-password
AUTH_SMTP_SECURE=true
AUTH_SMTP_SENDER=noreply@yourdomain.com
# Configure with your SMTP server details
EOF
    ;;

  development)
    cat <<'EOF'
AUTH_SMTP_HOST=mailpit
AUTH_SMTP_PORT=1025
AUTH_SMTP_USER=""
AUTH_SMTP_PASS=""
AUTH_SMTP_SECURE=false
AUTH_SMTP_SENDER=noreply@${BASE_DOMAIN}
# MailPit captures all emails locally
# View at: https://mail.${BASE_DOMAIN}
EOF
    ;;

  *)
    return 1
    ;;
  esac
}

# Function to detect email provider from current configuration
detect_provider() {
  local smtp_host="${AUTH_SMTP_HOST:-}"

  case "$smtp_host" in
  *sendgrid*) echo "sendgrid" ;;
  *amazonaws*) echo "aws-ses" ;;
  *mailgun*) echo "mailgun" ;;
  *postmark*) echo "postmark" ;;
  *gmail* | *google*) echo "gmail" ;;
  *office365* | *outlook*) echo "outlook" ;;
  *brevo* | *sendinblue*) echo "brevo" ;;
  *resend*) echo "resend" ;;
  *sparkpost*) echo "sparkpost" ;;
  *mandrill*) echo "mandrill" ;;
  *elastic*) echo "elastic" ;;
  *smtp2go*) echo "smtp2go" ;;
  *mailersend*) echo "mailersend" ;;
  mailpit | mailhog) echo "development" ;;
  "") echo "not-configured" ;;
  *) echo "custom" ;;
  esac
}

# Function to list all providers
list_providers() {
  show_command_header "nself email list" "Available email providers"
  echo
  printf "${COLOR_CYAN}➞ API Providers${COLOR_RESET} ${COLOR_DIM}(Recommended - use --api flag)${COLOR_RESET}\n"
  echo "  ├── sendgrid      - SendGrid (Popular, reliable)        [API+SMTP]"
  echo "  ├── aws-ses       - Amazon SES (Cost-effective)         [API+SMTP]"
  echo "  ├── mailgun       - Mailgun (Developer-friendly)        [API+SMTP]"
  echo "  ├── postmark      - Postmark (Transactional focus)      [API+SMTP]"
  echo "  ├── resend        - Resend (Modern, developer-first)    [API+SMTP]"
  echo "  └── elastic-email - Elastic Email (Budget-friendly)     [API+SMTP]"
  echo
  printf "  ${COLOR_DIM}Configure with: nself email configure --api <provider>${COLOR_RESET}\n"
  echo
  printf "${COLOR_CYAN}➞ SMTP-Only Providers${COLOR_RESET}\n"
  echo "  ├── brevo         - Brevo/Sendinblue (All-in-one)"
  echo "  ├── sparkpost     - SparkPost (High deliverability)"
  echo "  ├── mailchimp     - Mailchimp Transactional (Mandrill)"
  echo "  ├── smtp2go       - SMTP2GO (Global infrastructure)"
  echo "  └── mailersend    - MailerSend (Email automation)"
  echo
  printf "${COLOR_CYAN}➞ Self-hosted/Personal SMTP${COLOR_RESET}\n"
  echo "  ├── postfix       - Postfix (Self-hosted mail server)"
  echo "  ├── gmail         - Gmail/Google Workspace"
  echo "  ├── outlook       - Outlook/Office 365"
  echo "  └── custom        - Custom SMTP server"
  echo
  printf "${COLOR_CYAN}➞ Development${COLOR_RESET}\n"
  echo "  └── development   - MailPit (local testing, zero config)"
  echo
  printf "${COLOR_CYAN}➞ API vs SMTP${COLOR_RESET}\n"
  echo "  API mode offers: better deliverability, webhooks, analytics,"
  echo "  no port blocking, and easier firewall configuration."
  echo
}

# Function to configure a provider
configure_provider() {
  local provider="$1"
  local template

  if [[ -z "$provider" ]]; then
    show_command_header "nself email configure" "Configure email provider"
    echo
    log_error "No provider specified"
    echo
    echo "Usage: nself email configure <provider>"
    echo
    echo "Examples:"
    echo "  ${COLOR_BLUE}nself email configure sendgrid${COLOR_RESET}"
    echo "  ${COLOR_BLUE}nself email configure gmail${COLOR_RESET}"
    echo
    echo "Run '${COLOR_BLUE}nself email list${COLOR_RESET}' to see all available providers"
    return 1
  fi

  template=$(get_provider_template "$provider") || {
    show_command_header "nself email configure" "Configure email provider"
    echo
    log_error "Unknown provider: $provider"
    echo
    list_providers
    return 1
  }

  show_command_header "nself email configure" "Configure $provider"
  echo

  # Show the template
  echo "Add these settings to your .env.local file:"
  echo ""
  echo "# ============================================"
  echo "# Email Configuration - $provider"
  echo "# ============================================"
  echo "$template"
  echo ""

  # Ask if user wants to append to .env.local
  read -p "Would you like to append these settings to .env.local? [y/N] " -r
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    # Backup current .env.local
    cp .env.local .env.local.backup

    # Append configuration
    echo "" >>.env.local
    echo "# ============================================" >>.env.local
    echo "# Email Configuration - $provider" >>.env.local
    echo "# Added on $(date)" >>.env.local
    echo "# ============================================" >>.env.local
    echo "$template" >>.env.local

    log_success "Configuration added to .env.local"
    log_info "Backup saved: .env.local.backup"
    echo
    echo "Next steps:"
    echo "  1. Update placeholder values with actual credentials"
    echo "  2. Run: nself build"
    echo "  3. Run: nself email test"
  fi
}

# Function to validate email configuration
validate_config() {
  local errors=0

  show_command_header "nself email validate" "Check email configuration"
  echo

  # Detect provider first
  local provider=$(detect_provider)

  if [[ "$provider" == "not-configured" ]]; then
    log_error "Email not configured"
    log_info "Run: nself email setup"
    echo
    return 1
  fi

  printf "${COLOR_CYAN}Provider:${COLOR_RESET} %s\n" "$provider"
  echo

  # Check required variables
  [[ -z "$AUTH_SMTP_HOST" ]] && {
    log_error "Missing: AUTH_SMTP_HOST"
    errors=$((errors + 1))
  } || echo "✓ Host: $AUTH_SMTP_HOST"
  [[ -z "$AUTH_SMTP_PORT" ]] && {
    log_error "Missing: AUTH_SMTP_PORT"
    errors=$((errors + 1))
  } || echo "✓ Port: $AUTH_SMTP_PORT"
  [[ -z "$AUTH_SMTP_SENDER" ]] && {
    log_error "Missing: AUTH_SMTP_SENDER"
    errors=$((errors + 1))
  } || echo "✓ Sender: $AUTH_SMTP_SENDER"

  # Check authentication (not required for development)
  if [[ "$AUTH_SMTP_HOST" != "mailpit" ]] && [[ "$AUTH_SMTP_HOST" != "mailhog" ]]; then
    [[ -z "$AUTH_SMTP_USER" ]] && echo "⚠ User: not set" || echo "✓ User: $AUTH_SMTP_USER"
    [[ -z "$AUTH_SMTP_PASS" ]] && echo "⚠ Pass: not set" || echo "✓ Pass: [SET]"
  fi

  echo
  if [[ $errors -gt 0 ]]; then
    log_error "Configuration incomplete ($errors errors)"
    log_info "Run: nself email setup"
  else
    log_success "Configuration valid"
    log_info "Test with: nself email test"
  fi
  echo

  return $errors
}

# SMTP pre-flight connection check
smtp_preflight_check() {
  local host="${AUTH_SMTP_HOST:-}"
  local port="${AUTH_SMTP_PORT:-587}"
  local timeout="${SMTP_TIMEOUT:-10}"

  show_command_header "nself email check" "SMTP Connection Pre-flight Check"
  echo

  if [[ -z "$host" ]]; then
    log_error "SMTP host not configured"
    log_info "Run: nself email setup"
    return 1
  fi

  log_info "Checking SMTP connection to $host:$port..."
  echo

  # Step 1: DNS resolution
  printf "  DNS resolution... "
  if command -v nslookup >/dev/null 2>&1; then
    if nslookup "$host" >/dev/null 2>&1; then
      printf "\033[0;32m✓\033[0m\n"
    else
      printf "\033[0;31m✗\033[0m (DNS lookup failed)\n"
      log_error "Cannot resolve hostname: $host"
      return 1
    fi
  elif command -v host >/dev/null 2>&1; then
    if host "$host" >/dev/null 2>&1; then
      printf "\033[0;32m✓\033[0m\n"
    else
      printf "\033[0;31m✗\033[0m (DNS lookup failed)\n"
      log_error "Cannot resolve hostname: $host"
      return 1
    fi
  else
    printf "\033[0;33m?\033[0m (skipped - no DNS tools)\n"
  fi

  # Step 2: TCP connection
  printf "  TCP connection... "
  if command -v nc >/dev/null 2>&1; then
    if nc -z -w "$timeout" "$host" "$port" 2>/dev/null; then
      printf "\033[0;32m✓\033[0m\n"
    else
      printf "\033[0;31m✗\033[0m (connection failed)\n"
      log_error "Cannot connect to $host:$port"
      log_info "Check firewall settings and port availability"
      return 1
    fi
  elif command -v timeout >/dev/null 2>&1; then
    if timeout "$timeout" bash -c "echo >/dev/tcp/$host/$port" 2>/dev/null; then
      printf "\033[0;32m✓\033[0m\n"
    else
      printf "\033[0;31m✗\033[0m (connection failed)\n"
      log_error "Cannot connect to $host:$port"
      return 1
    fi
  else
    printf "\033[0;33m?\033[0m (skipped - no connection tools)\n"
  fi

  # Step 3: SMTP banner check (if openssl available)
  printf "  SMTP banner... "
  if command -v openssl >/dev/null 2>&1; then
    local banner
    if [[ "$port" == "465" ]]; then
      # SSL/TLS port - connect with SSL directly
      banner=$(echo "QUIT" | timeout "$timeout" openssl s_client -connect "$host:$port" -quiet 2>/dev/null | head -1)
    else
      # STARTTLS port - plain connection first
      banner=$(echo "QUIT" | timeout "$timeout" openssl s_client -connect "$host:$port" -starttls smtp -quiet 2>/dev/null | head -1)
      if [[ -z "$banner" ]]; then
        # Try plain connection without STARTTLS
        banner=$(echo "QUIT" | timeout "$timeout" nc "$host" "$port" 2>/dev/null | head -1)
      fi
    fi

    if [[ -n "$banner" ]] && echo "$banner" | grep -qE "^220"; then
      printf "\033[0;32m✓\033[0m (%s)\n" "$(echo "$banner" | cut -c1-50)"
    elif [[ -n "$banner" ]]; then
      printf "\033[0;33m?\033[0m (unexpected: %s)\n" "$(echo "$banner" | cut -c1-40)"
    else
      printf "\033[0;33m?\033[0m (no banner received)\n"
    fi
  else
    printf "\033[0;33m?\033[0m (skipped - openssl not available)\n"
  fi

  # Step 4: TLS support check
  printf "  TLS support... "
  if command -v openssl >/dev/null 2>&1; then
    local tls_result
    if [[ "$port" == "465" ]]; then
      tls_result=$(echo "QUIT" | timeout "$timeout" openssl s_client -connect "$host:$port" 2>&1 | grep -c "Verify return code: 0")
    else
      tls_result=$(echo "QUIT" | timeout "$timeout" openssl s_client -connect "$host:$port" -starttls smtp 2>&1 | grep -c "Verify return code: 0")
    fi

    if [[ "$tls_result" -gt 0 ]]; then
      printf "\033[0;32m✓\033[0m (certificate valid)\n"
    else
      printf "\033[0;33m?\033[0m (certificate may have issues)\n"
    fi
  else
    printf "\033[0;33m?\033[0m (skipped)\n"
  fi

  echo
  log_success "SMTP pre-flight check passed"
  log_info "Ready to send test email: nself email test"
  echo

  return 0
}

# Function to test email sending
test_email() {
  local recipient="${1:-}"

  show_command_header "nself email test" "Send test email"

  if [[ -z "$recipient" ]]; then
    echo
    printf "Enter recipient email address: "
    read recipient
  fi

  if [[ ! "$recipient" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
    echo
    log_error "Invalid email address format"
    echo
    return 1
  fi

  # Check if services are running
  local auth_running=$(docker ps --format "{{.Names}}" 2>/dev/null | grep -c "auth" | tr -d '\n' || echo 0)

  if [[ $auth_running -eq 0 ]]; then
    echo
    log_error "Unable to send test email, auth service not running!"
    log_warning "Note: Email requires services running, use: nself start"
    echo
    return 1
  fi

  # Create a test email using curl
  local subject="nself Email Test - $(date)"
  local body="This is a test email from your nself installation.\n\nConfiguration:\n- Provider: $(detect_provider)\n- Host: $AUTH_SMTP_HOST\n- Port: $AUTH_SMTP_PORT\n- Sender: $AUTH_SMTP_SENDER\n\nIf you received this email, your configuration is working correctly!"

  # For development environment (MailPit)
  if [[ "$AUTH_SMTP_HOST" == "mailpit" ]]; then
    # Send via MailPit API
    curl -X POST "http://localhost:1025/api/v1/send" \
      -H "Content-Type: application/json" \
      -d "{
                \"from\": \"$AUTH_SMTP_SENDER\",
                \"to\": [\"$recipient\"],
                \"subject\": \"$subject\",
                \"text\": \"$body\"
            }" 2>/dev/null && {
      echo
      log_success "Test email sent to $recipient"
      log_info "View at: http://localhost:8025"
    } || {
      echo
      log_error "Failed to send test email"
      log_warning "Check: nself status mailpit"
    }
  else
    # For production providers
    echo
    log_info "Sending via ${AUTH_SMTP_HOST}..."
    
    # Test SMTP connection and send test email
    local smtp_host="${AUTH_SMTP_HOST:-}"
    local smtp_port="${AUTH_SMTP_PORT:-587}"
    local smtp_user="${AUTH_SMTP_USER:-}"
    local smtp_pass="${AUTH_SMTP_PASS:-}"
    local smtp_sender="${AUTH_SMTP_SENDER:-noreply@${BASE_DOMAIN}}"
    local test_recipient="${1:-$smtp_sender}"
    
    if [[ -z "$smtp_host" ]]; then
      log_error "SMTP host not configured"
      return 1
    fi
    
    # Use swaks in Docker for SMTP testing
    log_info "Testing SMTP connection to $smtp_host:$smtp_port..."
    
    local auth_opts=""
    if [[ -n "$smtp_user" ]] && [[ -n "$smtp_pass" ]]; then
      auth_opts="--auth-user '$smtp_user' --auth-password '$smtp_pass'"
    fi
    
    # Run swaks in Docker container for SMTP test
    docker run --rm \
      --network host \
      boky/swaks \
      --to "$test_recipient" \
      --from "$smtp_sender" \
      --server "$smtp_host:$smtp_port" \
      --tls \
      $auth_opts \
      --header "Subject: nself Email Test" \
      --body "This is a test email from nself to verify SMTP configuration." \
      --timeout 10 2>&1 | {
        if grep -q "250 .*Message accepted"; then
          log_success "✅ Test email sent successfully to $test_recipient"
          log_info "Email configuration is working correctly"
        else
          log_warning "Could not verify email delivery"
          log_info "Check your SMTP settings and credentials"
        fi
      }
  fi
  echo
}

# Function to show email setup wizard
setup_wizard() {
  show_command_header "nself email setup" "Interactive email configuration wizard"
  echo

  # Show current configuration
  local current_provider=$(detect_provider)
  if [[ "$current_provider" == "development" ]]; then
    log_success "✅ Development email is already configured (MailPit)"
    echo ""
    log_info "Setting up production email?"
    echo ""
  elif [[ "$current_provider" != "not-configured" ]]; then
    log_info "Current provider: $current_provider"
    echo ""
    log_info "Want to change your email provider?"
    echo ""
  else
    log_info "Let's configure email for your project!"
    echo ""
  fi

  # Quick recommendations
  log_info "🎯 Quick Recommendations:"
  echo ""
  echo "  For Most Users:"
  echo "  • sendgrid    - Easy setup, reliable, 100 emails/day free"
  echo "  • mailgun     - Developer-friendly, good documentation"
  echo "  • postmark    - Best for transactional emails"
  echo ""
  echo "  For AWS Users:"
  echo "  • aws-ses     - Very cheap at scale ($0.10/1000 emails)"
  echo ""
  echo "  For Self-Hosting:"
  echo "  • postfix     - Full control, requires server setup"
  echo ""
  echo "  For Personal/Small Projects:"
  echo "  • gmail       - Use your Gmail account (requires app password)"
  echo ""

  # Ask user to select
  log_info "Enter provider name or type 'list' to see all options:"
  read -p "> " provider

  if [[ "$provider" == "list" ]]; then
    list_providers
    echo ""
    log_info "Enter provider name:"
    read -p "> " provider
  fi

  if [[ -z "$provider" ]] || [[ "$provider" == "cancel" ]]; then
    log_info "Setup cancelled. Development email (MailPit) remains active."
    return 0
  fi

  configure_provider "$provider"
}

# Function to show provider-specific documentation
show_docs() {
  local provider="${1:-$(detect_provider)}"

  show_command_header "nself email docs" "Setup guide for $provider"
  echo

  case "$provider" in
  sendgrid)
    printf "${COLOR_CYAN}➞ Setup Steps${COLOR_RESET}\n"
    echo "  1. Sign up at: https://signup.sendgrid.com/"
    echo "  2. Verify your sender domain or email address"
    echo "  3. Go to Settings → API Keys"
    echo "  4. Create a new API key with 'Mail Send' permission"
    echo "  5. Copy the API key to AUTH_SMTP_PASS"
    echo
    printf "${COLOR_CYAN}➞ Details${COLOR_RESET}\n"
    echo "  • Pricing: Free tier includes 100 emails/day"
    echo "  • Documentation: https://docs.sendgrid.com/for-developers/sending-email/smtp"
    ;;

  aws-ses)
    printf "${COLOR_CYAN}➞ Setup Steps${COLOR_RESET}\n"
    echo "  1. Sign in to AWS Console: https://console.aws.amazon.com/ses/"
    echo "  2. Verify your domain or email address"
    echo "  3. Move out of sandbox mode (for production)"
    echo "  4. Go to SMTP Settings → Create SMTP Credentials"
    echo "  5. Save the SMTP username and password"
    echo
    printf "${COLOR_CYAN}➞ Details${COLOR_RESET}\n"
    echo "  • Pricing: $0.10 per 1000 emails"
    echo "  • Documentation: https://docs.aws.amazon.com/ses/latest/dg/send-email-smtp.html"
    ;;

  mailgun)
    printf "${COLOR_CYAN}➞ Setup Steps${COLOR_RESET}\n"
    echo "  1. Sign up at: https://signup.mailgun.com/new/signup"
    echo "  2. Add and verify your domain"
    echo "  3. Go to Sending → Domain settings"
    echo "  4. Find SMTP credentials section"
    echo "  5. Use 'postmaster@mg.yourdomain.com' as username"
    echo
    printf "${COLOR_CYAN}➞ Details${COLOR_RESET}\n"
    echo "  • Pricing: Pay-as-you-go, first 1000 emails free"
    echo "  • Documentation: https://documentation.mailgun.com/en/latest/user_manual.html#sending-via-smtp"
    ;;

  postmark)
    printf "${COLOR_CYAN}➞ Setup Steps${COLOR_RESET}\n"
    echo "  1. Sign up at: https://account.postmarkapp.com/sign_up"
    echo "  2. Create a server for your application"
    echo "  3. Verify your sender signature"
    echo "  4. Go to Servers → [Your Server] → API Tokens"
    echo "  5. Copy the Server API Token"
    echo "  6. Use the same token for both USER and PASS"
    echo
    printf "${COLOR_CYAN}➞ Details${COLOR_RESET}\n"
    echo "  • Pricing: 100 test emails free, then \$15/mo for 10k emails"
    echo "  • Documentation: https://postmarkapp.com/developer/user-guide/sending-email/sending-with-smtp"
    ;;

  gmail)
    printf "${COLOR_CYAN}➞ Setup Steps${COLOR_RESET}\n"
    echo "  1. Enable 2-Factor Authentication on your Google account"
    echo "  2. Go to: https://myaccount.google.com/apppasswords"
    echo "  3. Select 'Mail' and your device"
    echo "  4. Generate an app password"
    echo "  5. Use this app password for AUTH_SMTP_PASS"
    echo
    printf "${COLOR_CYAN}➞ Details${COLOR_RESET}\n"
    echo "  ${COLOR_YELLOW}⚠${COLOR_RESET}  Regular passwords won't work, you must use app passwords"
    echo "  • Limits: 500 recipients per day"
    echo "  • Documentation: https://support.google.com/mail/answer/185833"
    ;;

  *)
    printf "${COLOR_CYAN}➞ General SMTP Configuration${COLOR_RESET}\n"
    echo "  1. Obtain SMTP server details from your provider"
    echo "  2. Get authentication credentials (username/password or API key)"
    echo "  3. Verify your sender domain or email address"
    echo "  4. Configure the SMTP settings in .env.local"
    echo "  5. Test with 'nself email test'"
    ;;
  esac
}

# Main command handler
email_main() {
  local command="${1:-help}"
  shift || true

  # Load environment if available
  if [[ -f ".env.local" ]]; then
    load_env_with_priority || true
  fi

  # Check for --api flag
  local use_api=false
  local args=()
  for arg in "$@"; do
    if [[ "$arg" == "--api" ]]; then
      use_api=true
    else
      args+=("$arg")
    fi
  done

  case "$command" in
  list)
    list_providers
    ;;
  setup)
    setup_wizard
    ;;
  configure)
    if [[ "$use_api" == "true" ]]; then
      configure_api_provider "${args[0]:-}"
    else
      configure_provider "${args[0]:-}"
    fi
    ;;
  validate)
    validate_config
    ;;
  check|preflight)
    if [[ "$use_api" == "true" ]]; then
      api_preflight_check
    else
      smtp_preflight_check
    fi
    ;;
  test)
    if [[ "$use_api" == "true" ]]; then
      local recipient="${args[0]:-}"
      if [[ -z "$recipient" ]]; then
        show_command_header "nself email test --api" "Send test email via API"
        echo
        printf "Enter recipient email address: "
        read recipient
      fi
      if [[ ! "$recipient" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        echo
        log_error "Invalid email address format"
        echo
        return 1
      fi
      send_api_email "$recipient" "nself Email Test - $(date)" \
        "This is a test email from nself to verify your API email configuration."
    else
      test_email "${args[0]:-}"
    fi
    ;;
  docs)
    show_docs "${1:-}"
    ;;
  detect)
    show_command_header "nself email detect" "Detect current email provider"
    echo
    local provider=$(detect_provider)
    printf "${COLOR_CYAN}➞ Current Configuration${COLOR_RESET}\n"
    echo "  Provider: ${COLOR_BLUE}$provider${COLOR_RESET}"

    if [[ "$provider" != "not-configured" ]]; then
      echo "  Host: $AUTH_SMTP_HOST"
      echo "  Port: $AUTH_SMTP_PORT"
      echo "  Sender: ${AUTH_SMTP_SENDER:-not set}"
    else
      echo
      log_warning "Email is not configured"
      echo
      echo "Run '${COLOR_BLUE}nself email setup${COLOR_RESET}' to configure email"
    fi
    echo
    ;;
  help | *)
    show_command_header "nself email" "Email configuration and management"
    echo

    printf "${COLOR_CYAN}➞ Development Mode${COLOR_RESET} ${COLOR_DIM}(Default - Zero Config)${COLOR_RESET}\n"
    echo "  MailPit is pre-configured and works out of the box!"
    echo "  • All emails captured locally"
    echo "  • View at: https://mail.<your-domain>"
    echo "  • No setup required"
    echo

    printf "${COLOR_CYAN}➞ Production Setup${COLOR_RESET}\n"
    echo "  Two options: SMTP or API-based delivery"
    echo
    echo "  ${COLOR_BLUE}SMTP Mode${COLOR_RESET} - Traditional email protocol"
    echo "  ${COLOR_BLUE}nself email setup${COLOR_RESET}"
    echo
    echo "  ${COLOR_BLUE}API Mode${COLOR_RESET} - Modern HTTP API (recommended)"
    echo "  ${COLOR_BLUE}nself email configure --api sendgrid${COLOR_RESET}"
    echo

    printf "${COLOR_CYAN}➞ Why Use API Mode?${COLOR_RESET}\n"
    echo "  • Better deliverability (dedicated IP pools)"
    echo "  • Webhooks for delivery tracking"
    echo "  • Built-in analytics and reporting"
    echo "  • No port 25/587 blocking issues"
    echo "  • Easier firewall configuration (HTTPS only)"
    echo

    printf "${COLOR_CYAN}➞ Available Commands${COLOR_RESET}\n"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "setup" "Interactive setup wizard (SMTP)"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "list" "See all email providers"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "configure <provider>" "Configure SMTP provider"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "configure --api <name>" "Configure API provider"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "validate" "Check your configuration"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "check" "SMTP connection pre-flight"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "check --api" "API connection pre-flight"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "test [email]" "Send test email (SMTP)"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "test --api [email]" "Send test email (API)"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "docs [provider]" "Get setup instructions"
    printf "  ${COLOR_BLUE}%-25s${COLOR_RESET} %s\n" "detect" "Show current provider"
    echo

    printf "${COLOR_CYAN}➞ API Providers (6 supported)${COLOR_RESET}\n"
    echo "  elastic-email, sendgrid, aws-ses, resend, postmark, mailgun"
    echo

    printf "${COLOR_CYAN}➞ Quick Examples${COLOR_RESET}\n"
    echo "  ${COLOR_DIM}# API mode - recommended for production${COLOR_RESET}"
    echo "  ${COLOR_BLUE}nself email configure --api sendgrid${COLOR_RESET}"
    echo "  ${COLOR_BLUE}nself email check --api${COLOR_RESET}"
    echo "  ${COLOR_BLUE}nself email test --api admin@example.com${COLOR_RESET}"
    echo
    echo "  ${COLOR_DIM}# SMTP mode - traditional approach${COLOR_RESET}"
    echo "  ${COLOR_BLUE}nself email setup${COLOR_RESET}"
    echo "  ${COLOR_BLUE}nself email check${COLOR_RESET}"
    echo "  ${COLOR_BLUE}nself email test admin@example.com${COLOR_RESET}"
    echo
    ;;
  esac
}

# Export command for nself integration
cmd_email() {
  email_main "$@"
}

# Export for use as library
export -f cmd_email
export -f email_main

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  pre_command "email" || exit $?
  email_main "$@"
  exit_code=$?
  post_command "email" $exit_code
  exit $exit_code
fi
