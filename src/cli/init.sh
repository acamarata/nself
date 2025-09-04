#!/usr/bin/env bash
set -euo pipefail

# init.sh - Initialize a new nself project with environment files

# Get script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source minimal display utilities for colors and formatting
for file in \
  "$SCRIPT_DIR/../lib/utils/display.sh" \
  "$SCRIPT_DIR/../lib/utils/output-formatter.sh"; do
  if [[ -f "$file" ]]; then
    source "$file" 2>/dev/null || true
  fi
done

# Fallback colors and functions if display utilities not loaded
if ! type -t log_error >/dev/null 2>&1; then
  log_error() { echo "${RED}✗${RESET} $*" >&2; }
  log_info() { echo "${BLUE}ℹ${RESET} $*" >&2; }
  log_success() { echo "${GREEN}✓${RESET} $*" >&2; }
  log_secondary() { echo "${BLUE}✓${RESET} $*" >&2; }
fi

# Define colors - use if available, otherwise empty
GREEN="${COLOR_GREEN:-}"
RED="${COLOR_RED:-}"
YELLOW="${COLOR_YELLOW:-}"
BLUE="${COLOR_BLUE:-}"
CYAN="${COLOR_CYAN:-}"
RESET="${COLOR_RESET:-}"
BOLD="${COLOR_BOLD:-}"
DIM="${COLOR_DIM:-}"

# Show help for init command
show_init_help() {
  echo "nself init - Initialize a new full-stack application"
  echo ""
  echo "Usage: nself init [OPTIONS]"
  echo ""
  echo "Description:"
  echo "  Creates a new nself project with .env configuration file"
  echo "  and .env.example reference documentation. Sets up the foundation"
  echo "  for a full-stack application with smart defaults."
  echo ""
  echo "Options:"
  echo "  --full              Create all environment files and starter schema"
  echo "  --wizard            Launch interactive setup wizard"
  echo "  --admin             Setup minimal admin UI only"
  echo "  -h, --help          Show this help message"
  echo ""
  echo "Examples:"
  echo "  mkdir myproject && cd myproject"
  echo "  nself init                     # Basic setup (.env + .env.example)"
  echo "  nself init --full              # Complete setup with all env files"
  echo "  nself init --wizard            # Interactive setup wizard"
  echo ""
  echo "Files Created (Basic):"
  echo "  • .env                         # Your personal dev configuration"
  echo "  • .env.example                 # Complete reference docs"
  echo "  • .gitignore                   # Security rules"
  echo ""
  echo "Files Created (--full):"
  echo "  • All basic files plus:"
  echo "  • .env.dev                     # Team-shared dev defaults"
  echo "  • .env.staging                 # Staging environment config"
  echo "  • .env.prod                    # Production config (public)"
  echo "  • .env.secrets                 # Sensitive data (git-ignored)"
  echo "  • schema.dbml                  # Example database schema"
  echo ""
  echo "Next Steps:"
  echo "  1. Edit .env (optional - defaults work!)"
  echo "  2. nself build                 # Generate infrastructure"
  echo "  3. nself start                 # Start services"
  echo ""
  echo "Notes:"
  echo "  • Safe to run multiple times"
  echo "  • Won't overwrite existing configuration"
  echo "  • Works with smart defaults out of the box"
}

# Ensure .gitignore has required entries
ensure_gitignore() {
  local required_entries=(
    ".env"
    ".env.secrets"
    "_backup*"
    ".volumes/"
    "logs/"
    "*.log"
    "node_modules/"
    ".DS_Store"
  )
  
  # Create .gitignore if it doesn't exist
  if [[ ! -f ".gitignore" ]]; then
    cat > .gitignore <<EOF
# Environment files (sensitive)
.env
.env
.env.secrets

# Backup folders from nself reset
_backup*

# Docker volumes
.volumes/

# Logs
logs/
*.log

# Dependencies
node_modules/

# System files
.DS_Store
.idea/
.vscode/
*.swp
*.swo
*~

# Build artifacts
dist/
build/
*.pid
EOF
    echo "${YELLOW}⚡${RESET} Created .gitignore with security rules"
  else
    # Check and add missing entries
    local added=false
    for entry in "${required_entries[@]}"; do
      # Check if entry already exists (exact match or as part of a pattern)
      if ! grep -q "^${entry}\$\|^${entry}\s*#" .gitignore 2>/dev/null; then
        # Special handling for first required entries
        if [[ "$entry" == ".env" ]] && ! $added; then
          echo -e "\n# Environment files (sensitive)" >> .gitignore
          added=true
        fi
        echo "$entry" >> .gitignore
      fi
    done
    
    if $added; then
      echo "${YELLOW}⚡${RESET} Updated .gitignore with required security rules"
    fi
  fi
}

# Command function
cmd_init() {
  local full_setup=false
  
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
    --full)
      full_setup=true
      shift
      ;;
    --wizard)
      # Source and run the practical configuration wizard
      if [[ -f "$SCRIPT_DIR/wizard/init-wizard-v2.sh" ]]; then
        source "$SCRIPT_DIR/wizard/init-wizard-v2.sh"
        run_config_wizard
      else
        # Fallback to old wizard if v2 not found
        source "$SCRIPT_DIR/wizard/init-wizard.sh"
        run_init_wizard
      fi
      return $?
      ;;
    --admin)
      # Setup minimal admin environment
      source "$SCRIPT_DIR/admin.sh"
      admin_minimal_setup
      return $?
      ;;
    -h | --help)
      show_init_help
      return 0
      ;;
    *)
      log_error "Unknown option: $1"
      echo "Use 'nself init --help' for usage information"
      return 1
      ;;
    esac
  done

  # Show header
  show_command_header "nself init" "Initialize a new full-stack application"

  # Check if environment files already exist
  if [[ -f ".env" ]]; then
    log_warning "Project already initialized (.env exists)"
    echo "Use 'nself reset' to start fresh or edit existing files"
    return 0
  fi

  # Get templates directory - check multiple possible locations
  local TEMPLATES_DIR=""

  # Check various possible template locations
  # 1. When installed via install.sh (src/templates is copied alongside src/cli)
  if [[ -d "$SCRIPT_DIR/../templates" ]]; then
    TEMPLATES_DIR="$SCRIPT_DIR/../templates"
  # 2. Development/source location
  elif [[ -d "$SCRIPT_DIR/../../src/templates" ]]; then
    TEMPLATES_DIR="$SCRIPT_DIR/../../src/templates"
  # 3. System installation
  elif [[ -d "/usr/share/nself/src/templates" ]]; then
    TEMPLATES_DIR="/usr/share/nself/src/templates"
  # 4. Local user installation
  elif [[ -d "$HOME/.nself/src/templates" ]]; then
    TEMPLATES_DIR="$HOME/.nself/src/templates"
  # 5. Custom installation path
  elif [[ -d "$HOME/.local/nself/src/templates" ]]; then
    TEMPLATES_DIR="$HOME/.local/nself/src/templates"
  else
    log_error "Cannot find templates directory"
    echo "Searched in:"
    echo "  - $SCRIPT_DIR/../templates"
    echo "  - $SCRIPT_DIR/../../src/templates"
    echo "  - /usr/share/nself/src/templates"
    echo "  - $HOME/.nself/src/templates"
    echo "  - $HOME/.local/nself/src/templates"
    echo "Please ensure nself is properly installed"
    return 1
  fi

  # Verify template files exist
  if [[ ! -f "$TEMPLATES_DIR/.env.example" ]] || [[ ! -f "$TEMPLATES_DIR/.env" ]]; then
    log_error "Template files not found in $TEMPLATES_DIR"
    echo "Please ensure nself is properly installed"
    return 1
  fi

  # Basic setup - always copy these
  cp "$TEMPLATES_DIR/.env.example" .env.example
  echo "${GREEN}✓${RESET} Created .env.example (exhaustive reference documentation)"
  
  cp "$TEMPLATES_DIR/.env" .env
  echo "${GREEN}✓${RESET} Created .env (your personal development config)"
  
  # Full setup - copy additional files
  if [[ "$full_setup" == true ]]; then
    cp "$TEMPLATES_DIR/.env.dev" .env.dev
    echo "${GREEN}✓${RESET} Created .env.dev (team-shared development defaults)"
    
    cp "$TEMPLATES_DIR/.env.staging" .env.staging
    echo "${GREEN}✓${RESET} Created .env.staging (staging environment config)"
    
    cp "$TEMPLATES_DIR/.env.prod" .env.prod
    echo "${GREEN}✓${RESET} Created .env.prod (production config - non-secrets)"
    
    cp "$TEMPLATES_DIR/.env.secrets" .env.secrets
    echo "${GREEN}✓${RESET} Created .env.secrets (sensitive data template)"
    
    # Copy schema.dbml if it exists
    if [[ -f "$TEMPLATES_DIR/schema.dbml" ]]; then
      cp "$TEMPLATES_DIR/schema.dbml" schema.dbml
      echo "${GREEN}✓${RESET} Created schema.dbml (example database schema)"
    fi
  fi
  
  # Always ensure .gitignore is properly configured
  ensure_gitignore
  
  echo ""

  # Quick Tips section
  echo "${CYAN}➞ Quick Tips${RESET}"
  echo ""
  echo "${YELLOW}⚡${RESET} Run ${BLUE}nself build${RESET} immediately - it works with defaults!"
  echo "${YELLOW}⚡${RESET} Use ${BLUE}nself prod${RESET} to generate secure production passwords"
  echo "${YELLOW}⚡${RESET} Check ${DIM}.env.example${RESET} for all available options"
  echo "${YELLOW}⚡${RESET} Run ${BLUE}nself help${RESET} for complete command documentation"
  if [[ "$full_setup" == true ]]; then
    echo "${YELLOW}⚡${RESET} Edit ${DIM}schema.dbml${RESET} and run ${BLUE}nself db run${RESET} to generate migrations"
  else
    echo "${YELLOW}⚡${RESET} Run ${BLUE}nself init --full${RESET} for all environment files and starter schema"
  fi
  echo ""

  # Next Steps section
  echo "${CYAN}➞ Next Steps${RESET}"
  echo ""
  echo "${BLUE}1.${RESET} Edit .env to customize (optional)"
  echo "   ${DIM}Only add what you want to change - defaults handle the rest${RESET}"
  echo ""
  echo "${BLUE}2.${RESET} nself build - Generate all project files"
  echo "   ${DIM}Creates Docker configs, service templates, and more${RESET}"
  echo ""
  echo "${BLUE}3.${RESET} nself start - Start your backend"
  echo "   ${DIM}Launches PostgreSQL, Hasura, and all configured services${RESET}"
  echo ""

  return 0
}

# Export for use as library
export -f cmd_init

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_init "$@"
fi