#!/usr/bin/env bash
# nself White-Label CLI
# Manages white-label branding, customization, and multi-tenant configurations
# Part of Sprint 14: White-Label & Customization (60pts) for v0.9.0
#
# Usage: nself whitelabel <command> [options]
# PREFERRED: nself tenant <branding|domains|email|themes> [options]
#
# This command can be called directly or via the tenant subcommands.
# Both forms work identically.

set -euo pipefail

# Get script directory and source dependencies
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/../.." && pwd)"

# Source required libraries
source "${PROJECT_ROOT}/src/lib/config/constants.sh"
source "${PROJECT_ROOT}/src/lib/utils/platform-compat.sh"
source "${PROJECT_ROOT}/src/lib/whitelabel/branding.sh"
source "${PROJECT_ROOT}/src/lib/whitelabel/email-templates.sh"
source "${PROJECT_ROOT}/src/lib/whitelabel/domains.sh"
source "${PROJECT_ROOT}/src/lib/whitelabel/themes.sh"

# CLI Version
readonly CLI_VERSION="0.9.0"

# ============================================================================
# Display Functions
# ============================================================================

show_help() {
  cat <<EOF
nself whitelabel - White-Label & Customization Management

USAGE:
  nself whitelabel <command> [options]
  nself tenant branding|domains|email|themes <options>  (Preferred)

NOTE: White-label features are now organized under 'nself tenant':
      - nself tenant branding  (logo, colors, fonts, CSS)
      - nself tenant domains   (custom domains, SSL)
      - nself tenant email     (email templates)
      - nself tenant themes    (UI themes)

COMMANDS:
  branding    Manage brand customization (logo, colors, fonts)
  domain      Configure custom domains and SSL
  email       Customize email templates
  theme       Create and manage themes
  logo        Upload and manage logos
  settings    View and update whitelabel settings
  init        Initialize whitelabel system
  list        List all brands/themes/domains
  export      Export branding configuration
  import      Import branding configuration

BRANDING COMMANDS:
  nself whitelabel branding create <brand-name>
  nself whitelabel branding set-colors --primary #hex --secondary #hex
  nself whitelabel branding set-fonts --primary "Font Name" --secondary "Font Name"
  nself whitelabel branding upload-logo <path-to-logo>
  nself whitelabel branding set-css <path-to-custom.css>
  nself whitelabel branding preview

DOMAIN COMMANDS:
  nself whitelabel domain add <domain>
  nself whitelabel domain verify <domain>
  nself whitelabel domain ssl <domain> [--auto-renew]
  nself whitelabel domain health <domain>
  nself whitelabel domain remove <domain>

EMAIL COMMANDS:
  nself whitelabel email list
  nself whitelabel email edit <template-name>
  nself whitelabel email preview <template-name>
  nself whitelabel email test <template-name> <email>
  nself whitelabel email set-language <lang-code>

THEME COMMANDS:
  nself whitelabel theme create <theme-name>
  nself whitelabel theme edit <theme-name>
  nself whitelabel theme activate <theme-name>
  nself whitelabel theme preview <theme-name>
  nself whitelabel theme export <theme-name>
  nself whitelabel theme import <path-to-theme.json>

LOGO COMMANDS:
  nself whitelabel logo upload <path> [--type main|icon|email]
  nself whitelabel logo list
  nself whitelabel logo remove <logo-id>

OPTIONS:
  --brand <name>        Specify brand name
  --tenant <id>         Specify tenant ID for multi-tenant mode
  --format <json|yaml>  Output format (default: json)
  --help, -h            Show this help message
  --version, -v         Show version information

EXAMPLES:
  # Create a new brand
  nself whitelabel branding create "My Company"

  # Set brand colors
  nself whitelabel branding set-colors --primary #0066cc --secondary #ff6600

  # Upload logo
  nself whitelabel logo upload ./logo.png --type main

  # Add custom domain
  nself whitelabel domain add app.mycompany.com

  # Customize email template
  nself whitelabel email edit welcome

  # Create custom theme
  nself whitelabel theme create dark-mode

  # Export entire branding config
  nself whitelabel export --format json > branding.json

VERSION:
  ${CLI_VERSION}

For more information, visit: https://docs.nself.org/whitelabel
EOF
}

show_version() {
  printf "nself whitelabel v%s\n" "${CLI_VERSION}"
}

# ============================================================================
# Main Command Router
# ============================================================================

main() {
  # Check for help or version flags
  if [[ $# -eq 0 ]] || [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    show_help
    exit 0
  fi

  if [[ "${1:-}" == "--version" ]] || [[ "${1:-}" == "-v" ]]; then
    show_version
    exit 0
  fi

  # Get command
  local command="$1"
  shift

  # Route to appropriate handler
  case "$command" in
    branding)
      handle_branding_command "$@"
      ;;
    domain)
      handle_domain_command "$@"
      ;;
    email)
      handle_email_command "$@"
      ;;
    theme)
      handle_theme_command "$@"
      ;;
    logo)
      handle_logo_command "$@"
      ;;
    settings)
      handle_settings_command "$@"
      ;;
    init)
      handle_init_command "$@"
      ;;
    list)
      handle_list_command "$@"
      ;;
    export)
      handle_export_command "$@"
      ;;
    import)
      handle_import_command "$@"
      ;;
    *)
      printf "${RED}Error: Unknown command '%s'${NC}\n" "$command" >&2
      printf "Run 'nself whitelabel --help' for usage information.\n" >&2
      exit 1
      ;;
  esac
}

# ============================================================================
# Command Handlers
# ============================================================================

handle_branding_command() {
  if [[ $# -eq 0 ]]; then
    printf "${RED}Error: Missing branding subcommand${NC}\n" >&2
    printf "Run 'nself whitelabel --help' for usage information.\n" >&2
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    create)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Brand name required${NC}\n" >&2
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
        printf "${RED}Error: Logo path required${NC}\n" >&2
        exit 1
      fi
      upload_brand_logo "$1"
      ;;
    set-css)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: CSS file path required${NC}\n" >&2
        exit 1
      fi
      set_custom_css "$1"
      ;;
    preview)
      preview_branding "$@"
      ;;
    *)
      printf "${RED}Error: Unknown branding subcommand '%s'${NC}\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

handle_domain_command() {
  if [[ $# -eq 0 ]]; then
    printf "${RED}Error: Missing domain subcommand${NC}\n" >&2
    printf "Run 'nself whitelabel --help' for usage information.\n" >&2
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    add)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Domain name required${NC}\n" >&2
        exit 1
      fi
      add_custom_domain "$1"
      ;;
    verify)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Domain name required${NC}\n" >&2
        exit 1
      fi
      verify_domain "$1"
      ;;
    ssl)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Domain name required${NC}\n" >&2
        exit 1
      fi
      provision_ssl "$@"
      ;;
    health)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Domain name required${NC}\n" >&2
        exit 1
      fi
      check_domain_health "$1"
      ;;
    remove)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Domain name required${NC}\n" >&2
        exit 1
      fi
      remove_custom_domain "$1"
      ;;
    *)
      printf "${RED}Error: Unknown domain subcommand '%s'${NC}\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

handle_email_command() {
  if [[ $# -eq 0 ]]; then
    printf "${RED}Error: Missing email subcommand${NC}\n" >&2
    printf "Run 'nself whitelabel --help' for usage information.\n" >&2
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
        printf "${RED}Error: Template name required${NC}\n" >&2
        exit 1
      fi
      edit_email_template "$1"
      ;;
    preview)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Template name required${NC}\n" >&2
        exit 1
      fi
      preview_email_template "$1"
      ;;
    test)
      if [[ $# -lt 2 ]]; then
        printf "${RED}Error: Template name and email address required${NC}\n" >&2
        exit 1
      fi
      test_email_template "$1" "$2"
      ;;
    set-language)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Language code required${NC}\n" >&2
        exit 1
      fi
      set_email_language "$1"
      ;;
    *)
      printf "${RED}Error: Unknown email subcommand '%s'${NC}\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

handle_theme_command() {
  if [[ $# -eq 0 ]]; then
    printf "${RED}Error: Missing theme subcommand${NC}\n" >&2
    printf "Run 'nself whitelabel --help' for usage information.\n" >&2
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    create)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Theme name required${NC}\n" >&2
        exit 1
      fi
      create_theme "$1"
      ;;
    edit)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Theme name required${NC}\n" >&2
        exit 1
      fi
      edit_theme "$1"
      ;;
    activate)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Theme name required${NC}\n" >&2
        exit 1
      fi
      activate_theme "$1"
      ;;
    preview)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Theme name required${NC}\n" >&2
        exit 1
      fi
      preview_theme "$1"
      ;;
    export)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Theme name required${NC}\n" >&2
        exit 1
      fi
      export_theme "$1"
      ;;
    import)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Theme file path required${NC}\n" >&2
        exit 1
      fi
      import_theme "$1"
      ;;
    *)
      printf "${RED}Error: Unknown theme subcommand '%s'${NC}\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

handle_logo_command() {
  if [[ $# -eq 0 ]]; then
    printf "${RED}Error: Missing logo subcommand${NC}\n" >&2
    printf "Run 'nself whitelabel --help' for usage information.\n" >&2
    exit 1
  fi

  local subcommand="$1"
  shift

  case "$subcommand" in
    upload)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Logo path required${NC}\n" >&2
        exit 1
      fi
      upload_logo "$@"
      ;;
    list)
      list_logos "$@"
      ;;
    remove)
      if [[ $# -eq 0 ]]; then
        printf "${RED}Error: Logo ID required${NC}\n" >&2
        exit 1
      fi
      remove_logo "$1"
      ;;
    *)
      printf "${RED}Error: Unknown logo subcommand '%s'${NC}\n" "$subcommand" >&2
      exit 1
      ;;
  esac
}

handle_settings_command() {
  view_whitelabel_settings "$@"
}

handle_init_command() {
  initialize_whitelabel_system "$@"
}

handle_list_command() {
  list_whitelabel_resources "$@"
}

handle_export_command() {
  export_whitelabel_config "$@"
}

handle_import_command() {
  if [[ $# -eq 0 ]]; then
    printf "${RED}Error: Config file path required${NC}\n" >&2
    exit 1
  fi
  import_whitelabel_config "$1"
}

# ============================================================================
# Execute Main
# ============================================================================

main "$@"
