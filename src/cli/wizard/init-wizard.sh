#!/usr/bin/env bash
# init-wizard.sh - Interactive setup wizard for nself

# Determine directories
WIZARD_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CLI_DIR="$(dirname "$WIZARD_DIR")"
ROOT_DIR="$(dirname "$(dirname "$(dirname "$WIZARD_DIR")")")"

# Source required modules with existence checks
for module in "$WIZARD_DIR/prompts.sh" "$WIZARD_DIR/detection.sh" "$WIZARD_DIR/templates.sh"; do
  if [[ ! -f "$module" ]]; then
    echo "Error: Required wizard module not found: $module" >&2
    exit 1
  fi
  source "$module"
done

for lib in "$CLI_DIR/../lib/utils/display.sh" "$CLI_DIR/../lib/utils/env.sh" "$CLI_DIR/../lib/wizard/environment-manager.sh"; do
  if [[ ! -f "$lib" ]]; then
    echo "Error: Required library not found: $lib" >&2
    exit 1
  fi
  source "$lib"
done

# Wizard state file
WIZARD_STATE=".nself-wizard-state"

# Save wizard state
save_wizard_state() {
  local step="$1"
  local data="$2"
  
  echo "STEP=$step" > "$WIZARD_STATE"
  echo "DATA=$data" >> "$WIZARD_STATE"
  echo "TIMESTAMP=$(date +%s)" >> "$WIZARD_STATE"
}

# Load wizard state
load_wizard_state() {
  if [[ -f "$WIZARD_STATE" ]]; then
    source "$WIZARD_STATE"
    
    # Check if state is less than 1 hour old
    local now=$(date +%s)
    local age=$((now - TIMESTAMP))
    
    if [[ $age -lt 3600 ]]; then
      echo "Resume from step $STEP? (y/N): "
      local resume
      read resume
      
      if [[ "$resume" == "y" ]] || [[ "$resume" == "Y" ]]; then
        return 0
      fi
    fi
  fi
  
  # Start fresh
  rm -f "$WIZARD_STATE"
  return 1
}

# Clean up wizard state
cleanup_wizard_state() {
  rm -f "$WIZARD_STATE"
}

# Main wizard function
run_init_wizard() {
  clear
  show_wizard_header "nself Init Wizard" "Interactive Project Setup"
  
  echo "Welcome to the nself initialization wizard!"
  echo "This will help you configure your project with the right services."
  echo ""
  
  # Check for resume
  if load_wizard_state; then
    log_info "Resuming from previous session..."
    echo ""
  fi
  
  # Step 1: Project Basics
  show_wizard_step 1 7 "Project Basics"
  
  echo "Let's configure your nself project."
  echo "We'll walk through the essential settings from .env configuration."
  echo ""
  
  # Detect existing project if any
  log_info "Checking for existing configuration..."
  if [[ -f ".env.local" ]]; then
    log_success "Found existing .env.local"
    echo ""
    echo -n "Would you like to update the existing configuration? (y/N): "
    local update_existing
    read update_existing
    if [[ "$update_existing" != "y" ]] && [[ "$update_existing" != "Y" ]]; then
      log_info "Keeping existing configuration"
      cleanup_wizard_state
      return 0
    fi
  fi
  
  echo ""
  press_any_key
  
  # Step 2: Core Configuration
  clear
  show_wizard_step 2 7 "Core Configuration"
  
  echo "📋 Basic Project Settings"
  echo ""
  
  # Project Name
  echo "Project name (used for Docker containers, database names, etc.):"
  local project_name
  prompt_input "Project name" "myapp" project_name "^[a-z][a-z0-9-]*$"
  
  echo ""
  
  # Environment Mode
  echo "Environment mode:"
  local env_modes=("dev - Development (debug tools, hot reload)" "prod - Production (optimized, secure)")
  local selected_env
  select_option "Select environment:" env_modes selected_env
  local env_mode=$([[ $selected_env -eq 0 ]] && echo "dev" || echo "prod")
  
  echo ""
  
  # Base Domain
  echo "Base domain for services:"
  echo "(All services will be subdomains of this domain)"
  if [[ "$env_mode" == "dev" ]]; then
    echo "Recommended: local.nself.org (automatic SSL for development)"
  fi
  
  local base_domain
  prompt_input "Base domain" "localhost" base_domain
  
  save_wizard_state "project_config" "$project_name:$base_domain"
  
  # Step 4: Service Selection
  clear
  show_wizard_step 4 6 "Service Selection"
  
  # Get template services
  local required_services=()
  local optional_services=()
  get_template_services "$template" required_services optional_services
  
  echo "Required services for $template:"
  for service in "${required_services[@]}"; do
    echo "  ✓ $service"
  done
  
  echo ""
  echo "Optional services (select with space, confirm with enter):"
  
  local selected_services=()
  multi_select optional_services selected_services
  
  # Combine required and selected optional services
  local all_services=("${required_services[@]}" "${selected_services[@]}")
  
  save_wizard_state "services" "$(IFS=,; echo "${all_services[*]}")"
  
  # Step 5: Environment Configuration
  clear
  show_wizard_step 5 6 "Environment Setup"
  
  echo "Configure environments:"
  echo ""
  
  local environments=("Development only" "Development + Staging" "Development + Staging + Production")
  local selected_env
  select_option "Select environments:" environments selected_env
  
  local env_setup="dev"
  case $selected_env in
  1) env_setup="dev,staging" ;;
  2) env_setup="dev,staging,prod" ;;
  esac
  
  # Database configuration
  echo ""
  echo "Database configuration:"
  echo ""
  
  local db_name
  prompt_input "Database name" "${project_name}_db" db_name
  
  local generate_passwords="n"
  if [[ "$env_setup" == *"prod"* ]]; then
    echo ""
    echo -n "Generate secure passwords for production? (Y/n): "
    read generate_passwords
    generate_passwords="${generate_passwords:-y}"
  fi
  
  save_wizard_state "environment" "$env_setup:$generate_passwords"
  
  # Step 6: Review and Confirm
  clear
  show_wizard_step 6 6 "Review Configuration"
  
  echo "Configuration Summary:"
  echo "====================="
  echo ""
  echo "Project Type:    $(get_template_name "$template")"
  echo "Project Name:    $project_name"
  echo "Base Domain:     $base_domain"
  echo "Database:        $db_name"
  echo "Environments:    $env_setup"
  echo ""
  echo "Services:"
  for service in "${all_services[@]}"; do
    echo "  • $service"
  done
  
  echo ""
  echo -n "Generate this configuration? (Y/n): "
  local confirm
  read confirm
  confirm="${confirm:-y}"
  
  if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
    log_info "Setup cancelled"
    cleanup_wizard_state
    return 1
  fi
  
  # Generate configuration
  echo ""
  log_info "Generating configuration..."
  
  # Create .env.local
  generate_env_from_template "$template" "$project_name" "$base_domain" "$db_name" "${all_services[@]}"
  
  # Create environment files if needed
  if [[ "$env_setup" == *"staging"* ]]; then
    create_environment_file "staging" "$project_name" "$base_domain"
  fi
  
  if [[ "$env_setup" == *"prod"* ]]; then
    create_environment_file "production" "$project_name" "$base_domain"
    
    if [[ "$generate_passwords" == "y" ]]; then
      generate_production_secrets
    fi
  fi
  
  # Clean up
  cleanup_wizard_state
  
  echo ""
  log_success "Configuration complete!"
  echo ""
  echo "Next steps:"
  echo "  1. Review .env.local"
  echo "  2. Run 'nself build' to generate infrastructure"
  echo "  3. Run 'nself start' to launch services"
  echo ""
  
  if [[ "$env_setup" == *"prod"* ]]; then
    echo "Production files created:"
    echo "  • .env.prod - Production configuration"
    echo "  • .env.secrets - Sensitive data (add to .gitignore)"
    echo ""
  fi
}

# Export for use
export -f run_init_wizard

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  run_init_wizard "$@"
fi