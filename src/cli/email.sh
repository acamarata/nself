#!/usr/bin/env bash

# email-providers.sh - Email provider configuration and management

set -e

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source utilities
source "$SCRIPT_DIR/../lib/utils/env.sh"
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/hooks/pre-command.sh"
source "$SCRIPT_DIR/../lib/hooks/post-command.sh"
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
        *gmail*|*google*) echo "gmail" ;;
        *office365*|*outlook*) echo "outlook" ;;
        *brevo*|*sendinblue*) echo "brevo" ;;
        *resend*) echo "resend" ;;
        *sparkpost*) echo "sparkpost" ;;
        *mandrill*) echo "mandrill" ;;
        *elastic*) echo "elastic" ;;
        *smtp2go*) echo "smtp2go" ;;
        *mailersend*) echo "mailersend" ;;
        mailpit|mailhog) echo "development" ;;
        "") echo "not-configured" ;;
        *) echo "custom" ;;
    esac
}

# Function to list all providers
list_providers() {
    echo ""
    log_info "📧 Available Email Providers:"
    echo ""
    echo "  Production Providers (API-based):"
    echo "  ├── sendgrid      - SendGrid (Popular, reliable)"
    echo "  ├── aws-ses       - Amazon SES (Cost-effective, scalable)"
    echo "  ├── mailgun       - Mailgun (Developer-friendly)"
    echo "  ├── postmark      - Postmark (Transactional focus)"
    echo "  ├── resend        - Resend (Modern, developer-first)"
    echo "  ├── brevo         - Brevo/Sendinblue (All-in-one)"
    echo "  ├── sparkpost     - SparkPost (High deliverability)"
    echo "  ├── mailchimp     - Mailchimp Transactional (Mandrill)"
    echo "  ├── elastic       - Elastic Email (Budget-friendly)"
    echo "  ├── smtp2go       - SMTP2GO (Global infrastructure)"
    echo "  └── mailersend    - MailerSend (Email automation)"
    echo ""
    echo "  Self-hosted/SMTP:"
    echo "  ├── postfix       - Postfix (Self-hosted mail server)"
    echo "  ├── gmail         - Gmail/Google Workspace"
    echo "  ├── outlook       - Outlook/Office 365"
    echo "  └── custom        - Custom SMTP server"
    echo ""
    echo "  Development:"
    echo "  └── development   - MailPit (local testing)"
    echo ""
}

# Function to configure a provider
configure_provider() {
    local provider="$1"
    local template
    
    template=$(get_provider_template "$provider") || {
        log_error "Unknown provider: $provider"
        list_providers
        return 1
    }
    
    echo ""
    log_info "Configuring $provider email provider..."
    echo ""
    
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
        echo "" >> .env.local
        echo "# ============================================" >> .env.local
        echo "# Email Configuration - $provider" >> .env.local
        echo "# Added on $(date)" >> .env.local
        echo "# ============================================" >> .env.local
        echo "$template" >> .env.local
        
        log_success "Configuration added to .env.local"
        log_info "Previous configuration backed up to .env.local.backup"
        echo ""
        log_warning "⚠️  Remember to:"
        echo "  1. Replace placeholder values with your actual credentials"
        echo "  2. Update AUTH_SMTP_SENDER with your verified sender address"
        echo "  3. Run 'nself build' to apply changes"
        echo "  4. Run 'nself email test' to verify configuration"
    fi
}

# Function to validate email configuration
validate_config() {
    local errors=0
    
    echo ""
    log_info "Validating email configuration..."
    echo ""
    
    # Check required variables
    if [[ -z "$AUTH_SMTP_HOST" ]]; then
        log_error "❌ AUTH_SMTP_HOST is not set"
        errors=$((errors + 1))
    else
        log_success "✅ AUTH_SMTP_HOST: $AUTH_SMTP_HOST"
    fi
    
    if [[ -z "$AUTH_SMTP_PORT" ]]; then
        log_error "❌ AUTH_SMTP_PORT is not set"
        errors=$((errors + 1))
    else
        log_success "✅ AUTH_SMTP_PORT: $AUTH_SMTP_PORT"
    fi
    
    if [[ -z "$AUTH_SMTP_SENDER" ]]; then
        log_error "❌ AUTH_SMTP_SENDER is not set"
        errors=$((errors + 1))
    else
        log_success "✅ AUTH_SMTP_SENDER: $AUTH_SMTP_SENDER"
    fi
    
    # Check authentication (not required for development)
    if [[ "$AUTH_SMTP_HOST" != "mailpit" ]] && [[ "$AUTH_SMTP_HOST" != "mailhog" ]]; then
        if [[ -z "$AUTH_SMTP_USER" ]]; then
            log_warning "⚠️  AUTH_SMTP_USER is not set (may be required)"
        else
            log_success "✅ AUTH_SMTP_USER: $AUTH_SMTP_USER"
        fi
        
        if [[ -z "$AUTH_SMTP_PASS" ]]; then
            log_warning "⚠️  AUTH_SMTP_PASS is not set (may be required)"
        else
            log_success "✅ AUTH_SMTP_PASS: [HIDDEN]"
        fi
    fi
    
    # Detect provider
    local provider=$(detect_provider)
    echo ""
    log_info "Detected provider: $provider"
    
    if [[ $errors -gt 0 ]]; then
        echo ""
        log_error "❌ Configuration has $errors error(s)"
        log_info "Run 'nself email setup' to configure a provider"
        return 1
    else
        echo ""
        log_success "✅ Email configuration appears valid"
        log_info "Run 'nself email test' to send a test email"
        return 0
    fi
}

# Function to test email sending
test_email() {
    local recipient="${1:-}"
    
    if [[ -z "$recipient" ]]; then
        read -p "Enter recipient email address: " recipient
    fi
    
    if [[ ! "$recipient" =~ ^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$ ]]; then
        log_error "Invalid email address format"
        return 1
    fi
    
    echo ""
    log_info "Sending test email to: $recipient"
    echo ""
    
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
            log_success "✅ Test email sent!"
            log_info "View at: http://localhost:8025"
        } || {
            log_error "❌ Failed to send test email"
            log_info "Make sure MailPit is running: docker ps | grep mailpit"
        }
    else
        # For production providers, we need to use the auth service
        log_info "Test email will be sent via the Auth service"
        log_info "Make sure services are running: nself up"
        echo ""
        log_warning "Note: Actual email sending requires the Auth service to be configured and running"
        log_info "You can verify your configuration with: nself email validate"
    fi
}

# Function to show email setup wizard
setup_wizard() {
    echo ""
    echo "╔══════════════════════════════════════════════╗"
    echo "║     📧 nself Email Setup Wizard              ║"
    echo "╚══════════════════════════════════════════════╝"
    echo ""
    
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
    
    case "$provider" in
        sendgrid)
            echo ""
            log_info "📚 SendGrid Setup Guide:"
            echo ""
            echo "  1. Sign up at: https://signup.sendgrid.com/"
            echo "  2. Verify your sender domain or email address"
            echo "  3. Go to Settings → API Keys"
            echo "  4. Create a new API key with 'Mail Send' permission"
            echo "  5. Copy the API key to AUTH_SMTP_PASS"
            echo ""
            echo "  Pricing: Free tier includes 100 emails/day"
            echo "  Documentation: https://docs.sendgrid.com/for-developers/sending-email/smtp"
            ;;
            
        aws-ses)
            echo ""
            log_info "📚 AWS SES Setup Guide:"
            echo ""
            echo "  1. Sign in to AWS Console: https://console.aws.amazon.com/ses/"
            echo "  2. Verify your domain or email address"
            echo "  3. Move out of sandbox mode (for production)"
            echo "  4. Go to SMTP Settings → Create SMTP Credentials"
            echo "  5. Save the SMTP username and password"
            echo ""
            echo "  Pricing: $0.10 per 1000 emails"
            echo "  Documentation: https://docs.aws.amazon.com/ses/latest/dg/send-email-smtp.html"
            ;;
            
        mailgun)
            echo ""
            log_info "📚 Mailgun Setup Guide:"
            echo ""
            echo "  1. Sign up at: https://signup.mailgun.com/new/signup"
            echo "  2. Add and verify your domain"
            echo "  3. Go to Sending → Domain settings"
            echo "  4. Find SMTP credentials section"
            echo "  5. Use 'postmaster@mg.yourdomain.com' as username"
            echo ""
            echo "  Pricing: Pay-as-you-go, first 1000 emails free"
            echo "  Documentation: https://documentation.mailgun.com/en/latest/user_manual.html#sending-via-smtp"
            ;;
            
        postmark)
            echo ""
            log_info "📚 Postmark Setup Guide:"
            echo ""
            echo "  1. Sign up at: https://account.postmarkapp.com/sign_up"
            echo "  2. Create a server for your application"
            echo "  3. Verify your sender signature"
            echo "  4. Go to Servers → [Your Server] → API Tokens"
            echo "  5. Copy the Server API Token"
            echo "  6. Use the same token for both USER and PASS"
            echo ""
            echo "  Pricing: 100 test emails free, then $15/mo for 10k emails"
            echo "  Documentation: https://postmarkapp.com/developer/user-guide/sending-email/sending-with-smtp"
            ;;
            
        gmail)
            echo ""
            log_info "📚 Gmail Setup Guide:"
            echo ""
            echo "  1. Enable 2-Factor Authentication on your Google account"
            echo "  2. Go to: https://myaccount.google.com/apppasswords"
            echo "  3. Select 'Mail' and your device"
            echo "  4. Generate an app password"
            echo "  5. Use this app password for AUTH_SMTP_PASS"
            echo ""
            echo "  ⚠️  Note: Regular passwords won't work, you must use app passwords"
            echo "  Limits: 500 recipients per day"
            echo "  Documentation: https://support.google.com/mail/answer/185833"
            ;;
            
        *)
            echo ""
            log_info "📚 Email Provider Documentation"
            echo ""
            echo "  General SMTP configuration guide:"
            echo "  1. Obtain SMTP server details from your provider"
            echo "  2. Get authentication credentials (username/password or API key)"
            echo "  3. Verify your sender domain or email address"
            echo "  4. Configure the SMTP settings in .env.local"
            echo "  5. Test with 'nself email test'"
            ;;
    esac
}

# Main command handler
main() {
    local command="${1:-help}"
    shift || true
    
    # Load environment if available
    if [[ -f ".env.local" ]]; then
        load_env_safe ".env.local"
    fi
    
    case "$command" in
        list)
            list_providers
            ;;
        setup)
            setup_wizard
            ;;
        configure)
            configure_provider "${1:-}"
            ;;
        validate)
            validate_config
            ;;
        test)
            test_email "${1:-}"
            ;;
        docs)
            show_docs "${1:-}"
            ;;
        detect)
            echo "Detected provider: $(detect_provider)"
            ;;
        help|*)
            echo ""
            echo "📧 nself Email Configuration"
            echo ""
            log_info "DEVELOPMENT (Default - Zero Config):"
            echo "  MailPit is pre-configured and works out of the box!"
            echo "  • All emails captured locally"
            echo "  • View at: https://mail.<your-domain>"
            echo "  • No setup required"
            echo ""
            log_info "PRODUCTION (Quick Setup):"
            echo "  Run: nself email setup"
            echo "  • Interactive wizard guides you"
            echo "  • Choose from 16+ providers"
            echo "  • Auto-configures everything"
            echo ""
            log_info "Available Commands:"
            echo "  setup             Interactive setup wizard (recommended)"
            echo "  list              See all email providers"
            echo "  configure <name>  Configure specific provider"
            echo "  validate          Check your configuration"
            echo "  test [email]      Send a test email"
            echo "  docs [provider]   Get setup instructions"
            echo "  detect            Show current provider"
            echo ""
            log_info "Quick Examples:"
            echo "  nself email setup                    # Start here for production"
            echo "  nself email configure sendgrid       # Use SendGrid specifically"
            echo "  nself email test admin@example.com   # Test your setup"
            echo ""
            log_success "💡 Tip: Development email works immediately. Production setup takes < 2 minutes!"
            echo ""
            ;;
    esac
}

# Run main function if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi