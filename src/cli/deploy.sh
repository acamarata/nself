#!/usr/bin/env bash
# deploy.sh - SSH deployment commands for VPS
# POSIX-compliant, no Bash 4+ features

# Determine root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"
LIB_DIR="$SCRIPT_DIR/../lib"

# Source required utilities
source "$LIB_DIR/utils/display.sh"
source "$LIB_DIR/utils/env.sh"
source "$LIB_DIR/utils/platform-compat.sh"
source "$LIB_DIR/utils/header.sh" 2>/dev/null || true

# Source new deployment modules (v0.4.3)
source "$LIB_DIR/deploy/ssh.sh" 2>/dev/null || true
source "$LIB_DIR/deploy/credentials.sh" 2>/dev/null || true
source "$LIB_DIR/deploy/health-check.sh" 2>/dev/null || true
source "$LIB_DIR/deploy/zero-downtime.sh" 2>/dev/null || true

# Source environment modules
source "$LIB_DIR/env/create.sh" 2>/dev/null || true
source "$LIB_DIR/env/switch.sh" 2>/dev/null || true

# Show help for deploy command
show_deploy_help() {
  cat <<EOF
nself deploy - SSH deployment to VPS

Usage: nself deploy [environment] [OPTIONS]
       nself deploy <subcommand> [OPTIONS]

Environment Deployment:
  nself deploy staging              Deploy to staging environment
  nself deploy prod                 Deploy to production environment
  nself deploy <env-name>           Deploy to custom environment

Subcommands:
  init          Initialize deployment configuration
  check         Pre-deployment validation checks
  ssh           Deploy to VPS via SSH (legacy)
  status        Show deployment status
  rollback      Rollback deployment
  logs          View deployment logs
  webhook       Setup GitHub webhook
  health        Check deployment health
  check-access  Verify SSH access to environments

Options:
  --dry-run           Preview deployment without executing
  --check-access      Verify SSH connectivity before deploy
  --force             Skip confirmation prompts
  --rolling           Use rolling deployment (zero-downtime)
  --skip-health       Skip health checks after deployment
  --include-frontends Include frontend apps (default for staging)
  --exclude-frontends Exclude frontend apps (default for production)
  --backend-only      Alias for --exclude-frontends

Service Architecture:
  • Core Services (4):     PostgreSQL, Hasura, Auth, Nginx (always deployed)
  • Optional Services (7): nself-admin, MinIO, Redis, Functions, etc.
  • Monitoring Bundle (10): Prometheus, Grafana, Loki, etc.
  • Remote Schemas:        Multi-app Hasura endpoints (same DB, different APIs)
  • Custom Services (CS_N): Independent backend apps (NestJS, Express, Python)
  • Frontend Apps:         React/Next/Vue apps (staging only by default)

Default Behavior:
  • Staging:    Deploy EVERYTHING including frontend apps
  • Production: Deploy backend only (frontends on Vercel/CDN/mobile)

Supported VPS Providers:
  - DigitalOcean, Linode, Vultr, Hetzner, OVH
  - Any Ubuntu/Debian VPS with SSH access

Examples:
  nself deploy staging              # Deploy to staging
  nself deploy prod --dry-run       # Preview production deploy
  nself deploy --check-access       # Check SSH access to all envs
  nself deploy init                 # Setup deployment config
  nself deploy status               # Check deployment
  nself deploy webhook              # Setup auto-deploy

Environment Configuration:
  Environments are configured in .environments/<name>/
  Each environment can have:
    - .env           Configuration variables
    - .env.secrets   Sensitive credentials (chmod 600)
    - server.json    SSH connection details

  Create environments with: nself env create <name> <template>
EOF
}

# Initialize deployment configuration
deploy_init() {
  show_command_header "nself deploy init" "Initialize deployment configuration"
  
  log_info "Setting up deployment configuration..."
  echo ""
  
  # Ask for deployment details
  echo -n "VPS hostname or IP address: "
  local host
  read host
  
  if [[ -z "$host" ]]; then
    log_error "Host is required"
    return 1
  fi
  
  echo -n "SSH user [root]: "
  local user
  read user
  user="${user:-root}"
  
  echo -n "SSH key path [~/.ssh/id_rsa]: "
  local key_path
  read key_path
  key_path="${key_path:-~/.ssh/id_rsa}"
  
  # Expand tilde
  key_path="${key_path/#\~/$HOME}"
  
  # Check if key exists
  if [[ ! -f "$key_path" ]]; then
    log_error "SSH key not found: $key_path"
    echo ""
    echo "Generate a key with: ssh-keygen -t rsa -b 4096"
    return 1
  fi
  
  echo -n "Target directory on server [/opt/nself]: "
  local target_dir
  read target_dir
  target_dir="${target_dir:-/opt/nself}"
  
  echo -n "Git repository URL (e.g., https://github.com/user/repo.git): "
  local repo_url
  read repo_url
  
  if [[ -z "$repo_url" ]]; then
    log_error "Repository URL is required"
    return 1
  fi
  
  echo -n "Git branch to deploy [main]: "
  local branch
  read branch
  branch="${branch:-main}"
  
  # Update .env.local
  log_info "Saving deployment configuration..."
  
  if grep -q "^DEPLOY_HOST=" .env.local 2>/dev/null; then
    safe_sed_inline ".env.local" "s/^DEPLOY_HOST=.*/DEPLOY_HOST=$host/"
  else
    echo "DEPLOY_HOST=$host" >> .env.local
  fi

  if grep -q "^DEPLOY_USER=" .env.local 2>/dev/null; then
    safe_sed_inline ".env.local" "s/^DEPLOY_USER=.*/DEPLOY_USER=$user/"
  else
    echo "DEPLOY_USER=$user" >> .env.local
  fi

  if grep -q "^DEPLOY_KEY_PATH=" .env.local 2>/dev/null; then
    safe_sed_inline ".env.local" "s|^DEPLOY_KEY_PATH=.*|DEPLOY_KEY_PATH=$key_path|"
  else
    echo "DEPLOY_KEY_PATH=$key_path" >> .env.local
  fi

  if grep -q "^DEPLOY_TARGET_DIR=" .env.local 2>/dev/null; then
    safe_sed_inline ".env.local" "s|^DEPLOY_TARGET_DIR=.*|DEPLOY_TARGET_DIR=$target_dir|"
  else
    echo "DEPLOY_TARGET_DIR=$target_dir" >> .env.local
  fi

  if grep -q "^DEPLOY_REPO_URL=" .env.local 2>/dev/null; then
    safe_sed_inline ".env.local" "s|^DEPLOY_REPO_URL=.*|DEPLOY_REPO_URL=$repo_url|"
  else
    echo "DEPLOY_REPO_URL=$repo_url" >> .env.local
  fi

  if grep -q "^DEPLOY_BRANCH=" .env.local 2>/dev/null; then
    safe_sed_inline ".env.local" "s/^DEPLOY_BRANCH=.*/DEPLOY_BRANCH=$branch/"
  else
    echo "DEPLOY_BRANCH=$branch" >> .env.local
  fi
  
  # Test SSH connection
  echo ""
  log_info "Testing SSH connection..."
  
  if ssh -i "$key_path" -o ConnectTimeout=5 -o StrictHostKeyChecking=no \
     "$user@$host" "echo 'SSH connection successful'" 2>/dev/null; then
    log_success "SSH connection successful"
  else
    log_error "SSH connection failed"
    echo ""
    echo "Troubleshooting:"
    echo "  1. Ensure SSH key is added to server: ssh-copy-id -i $key_path $user@$host"
    echo "  2. Check firewall allows SSH on port 22"
    echo "  3. Verify server is running and accessible"
    return 1
  fi
  
  echo ""
  log_success "Deployment configuration initialized"
  echo ""
  echo "Next steps:"
  echo "  1. Create .env.prod with production settings"
  echo "  2. Create .env.secrets with sensitive data (API keys, passwords)"
  echo "  3. Run 'nself deploy ssh' to deploy"
}

# Deploy via SSH
deploy_ssh() {
  show_command_header "nself deploy ssh" "Deploy to VPS via SSH"
  
  # Load deployment config
  load_env_with_priority
  
  local host="${DEPLOY_HOST:-}"
  local user="${DEPLOY_USER:-root}"
  local key_path="${DEPLOY_KEY_PATH:-~/.ssh/id_rsa}"
  local target_dir="${DEPLOY_TARGET_DIR:-/opt/nself}"
  local repo_url="${DEPLOY_REPO_URL:-}"
  local branch="${DEPLOY_BRANCH:-main}"
  
  # Expand tilde
  key_path="${key_path/#\~/$HOME}"
  
  if [[ -z "$host" ]] || [[ -z "$repo_url" ]]; then
    log_error "Deployment not configured"
    log_info "Run 'nself deploy init' first"
    return 1
  fi
  
  log_info "Deploying to $user@$host:$target_dir"
  echo ""
  
  # Create deployment script
  local deploy_script=$(cat <<'DEPLOY_SCRIPT'
#!/bin/bash
set -e

echo "=== nself Deployment Script ==="
echo ""

# Install Docker if not present
if ! command -v docker &> /dev/null; then
  echo "Installing Docker..."
  curl -fsSL https://get.docker.com | sh
  systemctl enable docker
  systemctl start docker
fi

# Install Docker Compose if not present
if ! command -v docker-compose &> /dev/null; then
  echo "Installing Docker Compose..."
  curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" \
    -o /usr/local/bin/docker-compose
  chmod +x /usr/local/bin/docker-compose
fi

# Install Git if not present
if ! command -v git &> /dev/null; then
  echo "Installing Git..."
  apt-get update && apt-get install -y git
fi

# Create target directory
mkdir -p TARGET_DIR
cd TARGET_DIR

# Clone or pull repository
if [ -d ".git" ]; then
  echo "Updating existing repository..."
  git fetch origin
  git checkout BRANCH
  git pull origin BRANCH
else
  echo "Cloning repository..."
  # For now, assume public repo or SSH key is configured
  git clone -b BRANCH REPO_URL .
fi

# Merge environment files
if [ -f ".env.prod" ] && [ -f ".env.secrets" ]; then
  echo "Merging production environment files..."
  cat .env.prod > .env
  echo "" >> .env
  cat .env.secrets >> .env
elif [ -f ".env.prod" ]; then
  cp .env.prod .env
fi

# Build and start services
echo "Building services..."
docker-compose build

echo "Starting services..."
docker-compose up -d

# Setup systemd service for auto-start
cat > /etc/systemd/system/nself.service <<EOF
[Unit]
Description=nself Application
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
WorkingDirectory=TARGET_DIR
ExecStart=/usr/local/bin/docker-compose up -d
ExecStop=/usr/local/bin/docker-compose down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable nself

echo ""
echo "=== Deployment Complete ==="
echo "Services are running at:"
docker-compose ps
DEPLOY_SCRIPT
)
  
  # Replace placeholders
  deploy_script="${deploy_script//TARGET_DIR/$target_dir}"
  deploy_script="${deploy_script//BRANCH/$branch}"
  deploy_script="${deploy_script//REPO_URL/$repo_url}"
  
  # Execute deployment script on server
  log_info "Connecting to server..."
  
  # Execute with error checking
  local ssh_output
  local ssh_exit_code
  
  ssh_output=$(ssh -i "$key_path" -o StrictHostKeyChecking=no -o ConnectTimeout=30 \
    "$user@$host" "$deploy_script" 2>&1)
  ssh_exit_code=$?
  
  # Display output
  echo "$ssh_output"
  
  # Check SSH exit code
  if [[ $ssh_exit_code -eq 0 ]]; then
    log_success "Deployment completed successfully"
    
    echo ""
    log_info "Your application is deployed to:"
    echo "  http://$host"
    
    if [[ -n "${BASE_DOMAIN:-}" ]]; then
      echo "  Configure DNS to point $BASE_DOMAIN to $host"
    fi
  elif [[ $ssh_exit_code -eq 255 ]]; then
    log_error "SSH connection failed"
    log_info "Check your SSH key and network connectivity"
    return 255
  else
    log_error "Deployment script failed with exit code: $ssh_exit_code"
    log_info "Review the output above for error details"
    return $ssh_exit_code
  fi
}

# Show deployment status
deploy_status() {
  show_command_header "nself deploy status" "Deployment status"
  
  # Load deployment config
  load_env_with_priority
  
  local host="${DEPLOY_HOST:-}"
  local user="${DEPLOY_USER:-root}"
  local key_path="${DEPLOY_KEY_PATH:-~/.ssh/id_rsa}"
  local target_dir="${DEPLOY_TARGET_DIR:-/opt/nself}"
  
  # Expand tilde
  key_path="${key_path/#\~/$HOME}"
  
  if [[ -z "$host" ]]; then
    log_error "Deployment not configured"
    log_info "Run 'nself deploy init' first"
    return 1
  fi
  
  echo "Deployment Configuration:"
  echo "  Host:       $host"
  echo "  User:       $user"
  echo "  Target:     $target_dir"
  echo "  Branch:     ${DEPLOY_BRANCH:-main}"
  echo "  Auto SSL:   ${DEPLOY_AUTO_SSL:-true}"
  echo ""
  
  log_info "Checking server status..."
  
  # Check if server is reachable
  if ! ssh -i "$key_path" -o ConnectTimeout=5 -o StrictHostKeyChecking=no \
     "$user@$host" "echo 'Connected'" 2>/dev/null; then
    log_error "Cannot connect to server"
    return 1
  fi
  
  # Check if application is deployed
  if ssh -i "$key_path" -o StrictHostKeyChecking=no \
     "$user@$host" "[ -d $target_dir/.git ]" 2>/dev/null; then
    log_success "Application is deployed"
    
    # Get git status
    echo ""
    echo "Git Status:"
    ssh -i "$key_path" -o StrictHostKeyChecking=no \
      "$user@$host" "cd $target_dir && git log -1 --oneline" 2>/dev/null
    
    # Check Docker status
    echo ""
    echo "Docker Status:"
    ssh -i "$key_path" -o StrictHostKeyChecking=no \
      "$user@$host" "cd $target_dir && docker-compose ps" 2>/dev/null
  else
    log_warning "Application not deployed"
    log_info "Run 'nself deploy ssh' to deploy"
  fi
}

# Rollback deployment
deploy_rollback() {
  show_command_header "nself deploy rollback" "Rollback deployment"
  
  # Load deployment config
  load_env_with_priority
  
  local host="${DEPLOY_HOST:-}"
  local user="${DEPLOY_USER:-root}"
  local key_path="${DEPLOY_KEY_PATH:-~/.ssh/id_rsa}"
  local target_dir="${DEPLOY_TARGET_DIR:-/opt/nself}"
  
  # Expand tilde
  key_path="${key_path/#\~/$HOME}"
  
  if [[ -z "$host" ]]; then
    log_error "Deployment not configured"
    return 1
  fi
  
  log_warning "This will rollback to the previous git commit"
  echo -n "Continue? (y/N): "
  local confirm
  read confirm
  
  if [[ "$confirm" != "y" ]] && [[ "$confirm" != "Y" ]]; then
    log_info "Rollback cancelled"
    return 0
  fi
  
  log_info "Rolling back deployment..."
  
  local rollback_script="
    cd $target_dir
    git reset --hard HEAD~1
    docker-compose down
    docker-compose up -d
  "
  
  if ssh -i "$key_path" -o StrictHostKeyChecking=no \
     "$user@$host" "$rollback_script" 2>/dev/null; then
    log_success "Rollback successful"
  else
    log_error "Rollback failed"
    return 1
  fi
}

# View deployment logs
deploy_logs() {
  show_command_header "nself deploy logs" "Deployment logs"
  
  # Load deployment config
  load_env_with_priority
  
  local host="${DEPLOY_HOST:-}"
  local user="${DEPLOY_USER:-root}"
  local key_path="${DEPLOY_KEY_PATH:-~/.ssh/id_rsa}"
  local target_dir="${DEPLOY_TARGET_DIR:-/opt/nself}"
  
  # Expand tilde
  key_path="${key_path/#\~/$HOME}"
  
  if [[ -z "$host" ]]; then
    log_error "Deployment not configured"
    return 1
  fi
  
  log_info "Fetching logs from $host..."
  echo ""
  
  # Get Docker Compose logs
  ssh -i "$key_path" -o StrictHostKeyChecking=no \
    "$user@$host" "cd $target_dir && docker-compose logs --tail=100"
}

# Setup GitHub webhook
deploy_webhook() {
  show_command_header "nself deploy webhook" "Setup GitHub webhook"
  
  # Load deployment config
  load_env_with_priority
  
  local host="${DEPLOY_HOST:-}"
  
  if [[ -z "$host" ]]; then
    log_error "Deployment not configured"
    log_info "Run 'nself deploy init' first"
    return 1
  fi
  
  # Generate webhook secret if not present
  if [[ -z "${DEPLOY_WEBHOOK_SECRET:-}" ]]; then
    local secret=$(openssl rand -hex 16)

    if grep -q "^DEPLOY_WEBHOOK_SECRET=" .env.local 2>/dev/null; then
      safe_sed_inline ".env.local" "s/^DEPLOY_WEBHOOK_SECRET=.*/DEPLOY_WEBHOOK_SECRET=$secret/"
    else
      echo "DEPLOY_WEBHOOK_SECRET=$secret" >> .env.local
    fi
    
    log_info "Generated webhook secret"
  else
    local secret="${DEPLOY_WEBHOOK_SECRET}"
  fi
  
  echo ""
  log_info "GitHub Webhook Configuration:"
  echo ""
  echo "1. Go to your GitHub repository settings"
  echo "2. Navigate to Webhooks > Add webhook"
  echo "3. Use these settings:"
  echo ""
  echo "   Payload URL:  http://$host:9000/hooks/deploy"
  echo "   Content type: application/json"
  echo "   Secret:       $secret"
  echo "   Events:       Push events"
  echo ""
  echo "4. Install webhook listener on server:"
  echo ""
  echo "   ssh $DEPLOY_USER@$host"
  echo "   docker run -d --name webhook \\"
  echo "     -p 9000:9000 \\"
  echo "     -v /opt/nself:/opt/nself \\"
  echo "     -e WEBHOOK_SECRET=$secret \\"
  echo "     adnanh/webhook"
  echo ""
  log_info "After setup, pushes to GitHub will auto-deploy"
}

# ============================================================
# NEW v0.4.3 FUNCTIONS: Environment-based deployment
# ============================================================

# Deploy to a specific environment
deploy_to_env() {
  local env_name="$1"
  shift
  local dry_run=false
  local check_access=false
  local force=false
  local rolling=false
  local skip_health=false
  local include_frontends=""
  local exclude_frontends=""

  # Parse options
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --dry-run)
        dry_run=true
        shift
        ;;
      --check-access)
        check_access=true
        shift
        ;;
      --force|-f)
        force=true
        shift
        ;;
      --rolling)
        rolling=true
        shift
        ;;
      --skip-health)
        skip_health=true
        shift
        ;;
      --include-frontends)
        include_frontends=true
        shift
        ;;
      --exclude-frontends|--backend-only)
        exclude_frontends=true
        shift
        ;;
      *)
        shift
        ;;
    esac
  done

  # ═══════════════════════════════════════════════════════════════
  # DEPLOYMENT SERVICE SCOPE
  # ═══════════════════════════════════════════════════════════════
  #
  # nself Services Architecture:
  #   - Core Services (4):     PostgreSQL, Hasura, Auth, Nginx
  #   - Optional Services (7): nself-admin, MinIO, Redis, Functions, MLflow, Mail, Search
  #   - Monitoring Bundle (10): Prometheus, Grafana, Loki, etc.
  #   - Custom Services (CS_N): User APIs, workers, remote schemas for Hasura
  #   - Frontend Apps:         External React/Next/Vue apps
  #
  # Default Deployment Behavior:
  #   - STAGING:    Deploy ALL including Frontend Apps (complete replica)
  #   - PRODUCTION: Deploy backend only (frontends are on Vercel/CDN/mobile)
  #
  # Frontend apps in staging are served by Nginx on subdomains.
  # In production, frontends are hosted externally (Vercel, Cloudflare, mobile).
  # ═══════════════════════════════════════════════════════════════

  # Determine frontend deployment based on environment type
  local deploy_frontends="false"
  local env_type=""

  # Check environment type from config
  local env_type_file=".environments/$env_name/.env"
  if [[ -f "$env_type_file" ]]; then
    env_type=$(grep "^ENV=" "$env_type_file" 2>/dev/null | cut -d'=' -f2)
  fi

  # Default: staging includes frontends, production excludes
  case "$env_type" in
    staging|stage|development|dev|local)
      deploy_frontends="true"
      ;;
    production|prod)
      deploy_frontends="false"
      ;;
    *)
      # For custom environments, check name
      case "$env_name" in
        staging|stage|dev|local)
          deploy_frontends="true"
          ;;
        prod|production)
          deploy_frontends="false"
          ;;
        *)
          deploy_frontends="true"  # Default to including frontends
          ;;
      esac
      ;;
  esac

  # Override with explicit flags
  if [[ "$include_frontends" == "true" ]]; then
    deploy_frontends="true"
  elif [[ "$exclude_frontends" == "true" ]]; then
    deploy_frontends="false"
  fi

  local env_dir=".environments/$env_name"

  # Check if environment exists
  if [[ ! -d "$env_dir" ]]; then
    log_error "Environment '$env_name' not found"
    log_info "Create it with: nself env create $env_name"
    return 1
  fi

  # Load environment configuration
  if [[ -f "$env_dir/server.json" ]]; then
    local host port user key_file deploy_path

    host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
    user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

    port="${port:-22}"
    user="${user:-root}"
    deploy_path="${deploy_path:-/opt/nself}"

    # Auto-detect SSH key if not specified
    if [[ -z "$key_file" ]]; then
      if command -v creds::find_key_for_host >/dev/null 2>&1; then
        key_file=$(creds::find_key_for_host "$host" "$env_name")
      else
        key_file="$HOME/.ssh/id_ed25519"
        [[ ! -f "$key_file" ]] && key_file="$HOME/.ssh/id_rsa"
      fi
    fi

    # Expand tilde
    key_file="${key_file/#\~/$HOME}"
  else
    log_error "No server.json found in $env_dir"
    log_info "Configure server connection details in: $env_dir/server.json"
    return 1
  fi

  # Validate we have minimum required info
  if [[ -z "$host" ]]; then
    log_error "No host configured for environment '$env_name'"
    log_info "Edit $env_dir/server.json and set 'host' field"
    return 1
  fi

  show_command_header "nself deploy $env_name" "Deploy to $env_name environment"

  printf "Environment: ${COLOR_CYAN}%s${COLOR_RESET}\n" "$env_name"
  printf "Server:      %s@%s:%s\n" "$user" "$host" "$port"
  printf "Deploy path: %s\n" "$deploy_path"
  printf "SSH key:     %s\n" "$key_file"
  printf "\n"

  # Show deployment scope
  printf "${COLOR_CYAN}Deployment Scope:${COLOR_RESET}\n"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} Core Services      (PostgreSQL, Hasura, Auth, Nginx)\n"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} Optional Services  (based on *_ENABLED vars)\n"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} Monitoring Bundle  (if MONITORING_ENABLED=true)\n"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} Remote Schemas     (multi-app Hasura GraphQL endpoints)\n"
  printf "  ${COLOR_GREEN}✓${COLOR_RESET} Custom Services    (CS_N - NestJS, Express, Python APIs)\n"
  if [[ "$deploy_frontends" == "true" ]]; then
    printf "  ${COLOR_GREEN}✓${COLOR_RESET} Frontend Apps      (FRONTEND_APP_N - served by Nginx)\n"
  else
    printf "  ${COLOR_YELLOW}○${COLOR_RESET} Frontend Apps      (excluded - deploy externally: Vercel, CDN)\n"
    if [[ "$env_type" == "production" || "$env_type" == "prod" ]]; then
      printf "                     ${COLOR_DIM}(use --include-frontends to override)${COLOR_RESET}\n"
    fi
  fi
  printf "\n"

  # Check SSH access first if requested
  if [[ "$check_access" == "true" ]]; then
    printf "Checking SSH access... "
    if command -v ssh::test_connection >/dev/null 2>&1; then
      if ssh::test_connection "$host" "$user" "$port" "$key_file"; then
        printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
      else
        printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
        log_error "Cannot connect to $host"
        return 1
      fi
    else
      # Fallback to basic SSH test
      if ssh -i "$key_file" -o ConnectTimeout=10 -o BatchMode=yes -p "$port" \
         "$user@$host" "echo ok" 2>/dev/null | grep -q "ok"; then
        printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
      else
        printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
        return 1
      fi
    fi
  fi

  # Dry run - show what would happen
  if [[ "$dry_run" == "true" ]]; then
    printf "\n${COLOR_YELLOW}=== DRY RUN ===${COLOR_RESET}\n\n"
    printf "Would deploy to:\n"
    printf "  • Server: %s@%s\n" "$user" "$host"
    printf "  • Path:   %s\n" "$deploy_path"
    printf "\n"
    printf "Files that would be synced:\n"
    printf "  • docker-compose.yml     (service definitions)\n"
    printf "  • nginx/                 (nginx configs, SSL certs)\n"
    printf "  • postgres/              (database init scripts)\n"
    [[ -d "services" ]] && printf "  • services/              (custom service code)\n"
    [[ -d "monitoring" ]] && printf "  • monitoring/            (prometheus, grafana configs)\n"
    [[ -d "ssl/certificates" ]] && printf "  • ssl/certificates/      (SSL certificates)\n"
    printf "  • .env                   (environment config)\n"
    [[ -f "$env_dir/.env.secrets" ]] && printf "  • .env.secrets           (sensitive credentials)\n"
    printf "\n"
    printf "Steps that would be executed:\n"
    printf "  1. Run 'nself build' locally (ensure configs up-to-date)\n"
    printf "  2. Create remote directory structure\n"
    printf "  3. Sync project files via rsync/scp\n"
    printf "  4. Sync environment configuration\n"
    printf "  5. Pull Docker images on server\n"
    if [[ "$rolling" == "true" ]]; then
      printf "  6. Rolling update (zero-downtime)\n"
    else
      printf "  6. Start/restart services with docker compose\n"
    fi
    if [[ "$skip_health" != "true" ]]; then
      printf "  7. Run health checks\n"
    fi
    printf "\n"
    printf "${COLOR_GREEN}No changes made (dry run)${COLOR_RESET}\n"
    return 0
  fi

  # Confirm deployment
  if [[ "$force" != "true" ]]; then
    printf "Deploy to %s? [y/N] " "$env_name"
    read -r confirm
    confirm=$(printf "%s" "$confirm" | tr '[:upper:]' '[:lower:]')
    if [[ "$confirm" != "y" && "$confirm" != "yes" ]]; then
      log_info "Deployment cancelled"
      return 0
    fi
  fi

  printf "\n"

  # Execute deployment
  if [[ "$rolling" == "true" ]] && command -v rolling::deploy >/dev/null 2>&1; then
    # Use rolling deployment
    log_info "Starting rolling deployment..."
    rolling::deploy "$host" "$deploy_path" "$user" "$port" "$key_file"
  else
    # Use standard deployment
    log_info "Starting deployment..."

    # Step 1: Ensure build is up to date
    printf "  Running local build... "
    if bash "$LIB_DIR/../cli/build.sh" --quiet 2>/dev/null; then
      printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
    else
      printf "${COLOR_YELLOW}SKIP${COLOR_RESET} (build manually with 'nself build')\n"
    fi

    # Step 2: Create remote directory structure
    printf "  Creating remote directories... "
    ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
      mkdir -p '$deploy_path'/{nginx/sites,nginx/conf.d,nginx/ssl,postgres/init,services,monitoring}
    " 2>/dev/null
    printf "${COLOR_GREEN}OK${COLOR_RESET}\n"

    # Step 3: Sync ALL project files (not just env)
    printf "  Syncing project files... "

    # Build list of files to sync
    local sync_items=(
      "docker-compose.yml"
      "nginx/"
      "postgres/"
      "monitoring/"
    )

    # Add services directory if custom services exist
    if [[ -d "services" ]]; then
      sync_items+=("services/")
    fi

    # Add ssl certificates
    if [[ -d "ssl/certificates" ]]; then
      sync_items+=("ssl/")
    fi

    # Use rsync for efficient sync (excludes .git, node_modules, etc.)
    local rsync_result
    rsync_result=$(rsync -avz --delete \
      --exclude '.git' \
      --exclude 'node_modules' \
      --exclude '.env.secrets' \
      --exclude '.env.local' \
      --exclude '*.log' \
      --exclude '.DS_Store' \
      -e "ssh -i '$key_file' -p '$port' -o BatchMode=yes" \
      docker-compose.yml nginx/ postgres/ monitoring/ \
      $([ -d "services" ] && echo "services/") \
      $([ -d "ssl/certificates" ] && echo "ssl/") \
      "$user@$host:$deploy_path/" 2>&1) || true

    if [[ $? -eq 0 ]]; then
      printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
    else
      # Fallback to individual scp if rsync fails
      printf "${COLOR_YELLOW}RETRY${COLOR_RESET}\n"
      printf "  Syncing via scp (fallback)... "
      scp -i "$key_file" -P "$port" -r docker-compose.yml "$user@$host:$deploy_path/" 2>/dev/null || true
      scp -i "$key_file" -P "$port" -r nginx "$user@$host:$deploy_path/" 2>/dev/null || true
      scp -i "$key_file" -P "$port" -r postgres "$user@$host:$deploy_path/" 2>/dev/null || true
      [[ -d "services" ]] && scp -i "$key_file" -P "$port" -r services "$user@$host:$deploy_path/" 2>/dev/null || true
      [[ -d "monitoring" ]] && scp -i "$key_file" -P "$port" -r monitoring "$user@$host:$deploy_path/" 2>/dev/null || true
      printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
    fi

    # Step 4: Sync environment files (with secrets handling)
    printf "  Syncing environment config... "
    if [[ -f "$env_dir/.env" ]]; then
      scp -i "$key_file" -P "$port" "$env_dir/.env" "$user@$host:$deploy_path/.env" 2>/dev/null || true
    fi
    if [[ -f "$env_dir/.env.secrets" ]]; then
      scp -i "$key_file" -P "$port" "$env_dir/.env.secrets" "$user@$host:$deploy_path/.env.secrets" 2>/dev/null || true
      # Set proper permissions on secrets file
      ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "chmod 600 '$deploy_path/.env.secrets'" 2>/dev/null || true
    fi
    if [[ -f "$env_dir/server.json" ]]; then
      scp -i "$key_file" -P "$port" "$env_dir/server.json" "$user@$host:$deploy_path/server.json" 2>/dev/null || true
    fi
    printf "${COLOR_GREEN}OK${COLOR_RESET}\n"

    # Step 5: Merge env files and start services on remote
    printf "  Starting services... "
    local deploy_result
    deploy_result=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
      cd '$deploy_path' || exit 1

      # Merge .env and .env.secrets if both exist
      if [ -f '.env' ] && [ -f '.env.secrets' ]; then
        cat .env > .env.combined
        echo '' >> .env.combined
        cat .env.secrets >> .env.combined
        mv .env.combined .env
        rm -f .env.secrets
        chmod 600 .env
      fi

      # Pull Docker images
      docker compose pull 2>/dev/null || true

      # Start services (recreate to pick up config changes)
      docker compose up -d --remove-orphans --force-recreate 2>/dev/null

      echo 'deploy_ok'
    " 2>/dev/null)

    if echo "$deploy_result" | grep -q "deploy_ok"; then
      printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
    else
      printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
      log_error "Deployment failed. Check server logs."
      return 1
    fi
  fi

  # Health checks
  if [[ "$skip_health" != "true" ]]; then
    printf "\n"
    log_info "Waiting for services to start..."

    # Give services time to start up
    sleep 5

    if command -v health::check_deployment >/dev/null 2>&1; then
      health::check_deployment "$host" "$deploy_path" "$user" "$port" "$key_file"
    else
      # Basic health check fallback
      printf "  Checking Docker services... "
      local container_count
      container_count=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
        cd '$deploy_path' && docker compose ps --format '{{.Status}}' 2>/dev/null | grep -c 'Up' || echo '0'
      " 2>/dev/null)

      if [[ "$container_count" -gt 0 ]]; then
        printf "${COLOR_GREEN}%s container(s) running${COLOR_RESET}\n" "$container_count"
      else
        printf "${COLOR_YELLOW}Containers starting...${COLOR_RESET}\n"
        log_info "Services may take a moment to fully start"
        log_info "Check status with: nself deploy status"
      fi

      # Check nginx specifically
      printf "  Checking nginx... "
      local nginx_status
      nginx_status=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
        docker compose ps nginx --format '{{.Status}}' 2>/dev/null | head -1
      " 2>/dev/null)

      if echo "$nginx_status" | grep -qi "up"; then
        printf "${COLOR_GREEN}Running${COLOR_RESET}\n"
      elif [[ -n "$nginx_status" ]]; then
        printf "${COLOR_YELLOW}%s${COLOR_RESET}\n" "$nginx_status"
      else
        printf "${COLOR_YELLOW}Starting...${COLOR_RESET}\n"
      fi
    fi
  fi

  printf "\n"
  log_success "Deployment to '$env_name' complete"

  # Show helpful next steps
  printf "\n${COLOR_DIM}Next steps:${COLOR_RESET}\n"
  printf "  • View service status:  ${COLOR_CYAN}nself deploy status${COLOR_RESET}\n"
  printf "  • View logs:            ${COLOR_CYAN}nself deploy logs${COLOR_RESET}\n"
  printf "  • Check health:         ${COLOR_CYAN}nself deploy health %s${COLOR_RESET}\n" "$env_name"
}

# Check SSH access to all configured environments
deploy_check_access() {
  show_command_header "nself deploy check-access" "Verify SSH connectivity"

  local env_dir=".environments"

  if [[ ! -d "$env_dir" ]]; then
    log_warning "No environments configured"
    log_info "Create one with: nself env create <name>"
    return 0
  fi

  printf "Checking SSH access to all environments:\n\n"

  local all_ok=true

  for dir in "$env_dir"/*/; do
    if [[ -d "$dir" ]]; then
      local env_name
      env_name=$(basename "$dir")

      if [[ -f "$dir/server.json" ]]; then
        local host port user key_file

        host=$(grep '"host"' "$dir/server.json" 2>/dev/null | cut -d'"' -f4)
        port=$(grep '"port"' "$dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
        user=$(grep '"user"' "$dir/server.json" 2>/dev/null | cut -d'"' -f4)
        key_file=$(grep '"key"' "$dir/server.json" 2>/dev/null | cut -d'"' -f4)

        port="${port:-22}"
        user="${user:-root}"

        if [[ -z "$host" ]]; then
          printf "  ${COLOR_YELLOW}%-15s${COLOR_RESET} No host configured\n" "$env_name"
          continue
        fi

        # Auto-detect key if not specified
        if [[ -z "$key_file" ]]; then
          if command -v creds::find_key_for_host >/dev/null 2>&1; then
            key_file=$(creds::find_key_for_host "$host" "$env_name")
          else
            key_file="$HOME/.ssh/id_ed25519"
            [[ ! -f "$key_file" ]] && key_file="$HOME/.ssh/id_rsa"
          fi
        fi
        key_file="${key_file/#\~/$HOME}"

        printf "  %-15s %s@%s:%s ... " "$env_name" "$user" "$host" "$port"

        if command -v ssh::test_connection >/dev/null 2>&1; then
          if ssh::test_connection "$host" "$user" "$port" "$key_file" 5; then
            printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
          else
            printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
            all_ok=false
          fi
        else
          # Fallback
          if ssh -i "$key_file" -o ConnectTimeout=5 -o BatchMode=yes -p "$port" \
             "$user@$host" "echo ok" 2>/dev/null | grep -q "ok"; then
            printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
          else
            printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
            all_ok=false
          fi
        fi
      else
        printf "  ${COLOR_YELLOW}%-15s${COLOR_RESET} No server.json (local only)\n" "$env_name"
      fi
    fi
  done

  printf "\n"
  if [[ "$all_ok" == "true" ]]; then
    log_success "All remote environments accessible"
  else
    log_warning "Some environments are not accessible"
  fi
}

# Health check for current deployment
deploy_health() {
  local env_name="${1:-}"

  if [[ -z "$env_name" ]]; then
    # Use current environment or default deploy config
    load_env_with_priority

    local host="${DEPLOY_HOST:-}"
    local user="${DEPLOY_USER:-root}"
    local key_path="${DEPLOY_KEY_PATH:-~/.ssh/id_rsa}"
    local target_dir="${DEPLOY_TARGET_DIR:-/opt/nself}"

    key_path="${key_path/#\~/$HOME}"

    if [[ -z "$host" ]]; then
      log_error "No deployment configured"
      log_info "Run 'nself deploy init' or specify environment: nself deploy health staging"
      return 1
    fi

    show_command_header "nself deploy health" "Deployment health check"

    if command -v health::full_report >/dev/null 2>&1; then
      health::full_report "$host" "$target_dir" "$user" "22" "$key_path"
    else
      log_error "Health check module not available"
      return 1
    fi
  else
    # Environment-specific health check
    local env_dir=".environments/$env_name"

    if [[ ! -d "$env_dir" ]]; then
      log_error "Environment '$env_name' not found"
      return 1
    fi

    local host port user key_file deploy_path

    host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
    user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

    port="${port:-22}"
    user="${user:-root}"
    deploy_path="${deploy_path:-/opt/nself}"
    key_file="${key_file/#\~/$HOME}"

    show_command_header "nself deploy health $env_name" "Deployment health check"

    if command -v health::full_report >/dev/null 2>&1; then
      health::full_report "$host" "$deploy_path" "$user" "$port" "$key_file"
    else
      log_error "Health check module not available"
      return 1
    fi
  fi
}

# ============================================================
# Pre-deployment validation checks (v0.4.6)
# ============================================================

deploy_check() {
  local target_env="${1:-}"
  local json_mode=false
  local fix_mode=false

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --json) json_mode=true; shift ;;
      --fix) fix_mode=true; shift ;;
      *) target_env="$1"; shift ;;
    esac
  done

  show_command_header "nself deploy check" "Pre-deployment Validation"
  echo ""

  local errors=0
  local warnings=0
  local checks_passed=0

  # Helper function for check results
  check_result() {
    local status="$1"
    local check="$2"
    local message="$3"

    if [[ "$status" == "pass" ]]; then
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} %s\n" "$check"
      checks_passed=$((checks_passed + 1))
    elif [[ "$status" == "warn" ]]; then
      printf "  ${COLOR_YELLOW}!${COLOR_RESET} %s: %s\n" "$check" "$message"
      warnings=$((warnings + 1))
    else
      printf "  ${COLOR_RED}✗${COLOR_RESET} %s: %s\n" "$check" "$message"
      errors=$((errors + 1))
    fi
  }

  # 1. Check if build has been run
  printf "${COLOR_CYAN}➞ Build Artifacts${COLOR_RESET}\n"
  if [[ -f "docker-compose.yml" ]]; then
    check_result "pass" "docker-compose.yml exists"
  else
    check_result "fail" "docker-compose.yml" "Run 'nself build' first"
  fi

  if [[ -d "nginx" ]] && [[ -f "nginx/nginx.conf" ]]; then
    check_result "pass" "Nginx configuration exists"
  else
    check_result "fail" "Nginx config" "Run 'nself build' first"
  fi
  echo ""

  # 2. Check environment configuration
  printf "${COLOR_CYAN}➞ Environment Configuration${COLOR_RESET}\n"
  if [[ -n "$target_env" ]]; then
    if [[ -d ".environments/$target_env" ]]; then
      check_result "pass" "Environment directory exists"

      if [[ -f ".environments/$target_env/server.json" ]]; then
        check_result "pass" "Server configuration exists"

        # Check required fields
        local host=$(grep '"host"' ".environments/$target_env/server.json" 2>/dev/null | head -1)
        if [[ -n "$host" ]]; then
          check_result "pass" "Host configured"
        else
          check_result "fail" "Host" "Missing in server.json"
        fi
      else
        check_result "fail" "server.json" "Missing deployment configuration"
      fi

      if [[ -f ".environments/$target_env/.env" ]]; then
        check_result "pass" "Environment variables configured"
      else
        check_result "warn" "Environment variables" "No .env file (using defaults)"
      fi

      if [[ -f ".environments/$target_env/.env.secrets" ]]; then
        # Check permissions
        local perms=$(stat -c "%a" ".environments/$target_env/.env.secrets" 2>/dev/null || stat -f "%OLp" ".environments/$target_env/.env.secrets" 2>/dev/null)
        if [[ "$perms" == "600" ]]; then
          check_result "pass" "Secrets file permissions (600)"
        else
          check_result "warn" "Secrets permissions" "Should be 600, got $perms"
          if [[ "$fix_mode" == "true" ]]; then
            chmod 600 ".environments/$target_env/.env.secrets"
            log_info "Fixed: Set permissions to 600"
          fi
        fi
      fi
    else
      check_result "fail" "Environment" "Not found: .environments/$target_env"
    fi
  else
    if [[ -f ".env" ]]; then
      check_result "pass" "Local .env exists"
    else
      check_result "fail" ".env" "Missing configuration"
    fi
  fi
  echo ""

  # 3. Check required secrets
  printf "${COLOR_CYAN}➞ Required Secrets${COLOR_RESET}\n"
  load_env_with_priority

  local required_secrets=(
    "POSTGRES_PASSWORD"
    "HASURA_GRAPHQL_ADMIN_SECRET"
    "AUTH_SECRET_KEY"
  )

  for secret in "${required_secrets[@]}"; do
    local value="${!secret:-}"
    if [[ -n "$value" ]] && [[ "$value" != "changeme" ]] && [[ ${#value} -ge 12 ]]; then
      check_result "pass" "$secret configured"
    elif [[ -n "$value" ]] && [[ ${#value} -lt 12 ]]; then
      check_result "warn" "$secret" "Password too short (< 12 chars)"
    elif [[ "$value" == "changeme" ]]; then
      check_result "fail" "$secret" "Using default value 'changeme'"
    else
      check_result "fail" "$secret" "Not configured"
    fi
  done
  echo ""

  # 4. Check SSH access (if target_env specified)
  if [[ -n "$target_env" ]] && [[ -f ".environments/$target_env/server.json" ]]; then
    printf "${COLOR_CYAN}➞ SSH Connectivity${COLOR_RESET}\n"

    local host=$(grep '"host"' ".environments/$target_env/server.json" 2>/dev/null | sed 's/.*: *"\([^"]*\)".*/\1/')
    local user=$(grep '"user"' ".environments/$target_env/server.json" 2>/dev/null | sed 's/.*: *"\([^"]*\)".*/\1/')
    user="${user:-root}"

    if [[ -n "$host" ]]; then
      if ssh -o ConnectTimeout=5 -o BatchMode=yes "${user}@${host}" "echo ok" >/dev/null 2>&1; then
        check_result "pass" "SSH connection to ${user}@${host}"
      else
        check_result "fail" "SSH connection" "Cannot connect to ${user}@${host}"
      fi
    fi
    echo ""
  fi

  # 5. Check Docker resources
  printf "${COLOR_CYAN}➞ Docker Resources${COLOR_RESET}\n"
  if docker info >/dev/null 2>&1; then
    check_result "pass" "Docker is running"

    # Check disk space (simplified)
    local disk_free=$(df -h . 2>/dev/null | tail -1 | awk '{print $4}')
    check_result "pass" "Disk space available: $disk_free"
  else
    check_result "fail" "Docker" "Not running or not accessible"
  fi
  echo ""

  # Summary
  printf "${COLOR_CYAN}➞ Summary${COLOR_RESET}\n"
  echo "  Passed:   $checks_passed"
  echo "  Warnings: $warnings"
  echo "  Errors:   $errors"
  echo ""

  if [[ $errors -eq 0 ]]; then
    log_success "All critical checks passed"
    [[ $warnings -gt 0 ]] && log_warning "Review warnings before deploying"
    return 0
  else
    log_error "Deployment blocked: $errors error(s) must be resolved"
    return 1
  fi
}

# ============================================================
# Main command function
# ============================================================

cmd_deploy() {
  local subcommand="${1:-}"
  shift || true

  # Check if subcommand is actually an environment name
  if [[ -d ".environments/$subcommand" ]]; then
    deploy_to_env "$subcommand" "$@"
    return $?
  fi

  # Also handle common environment shortcuts
  case "$subcommand" in
  staging|stage)
    if [[ -d ".environments/staging" ]]; then
      deploy_to_env "staging" "$@"
    else
      log_error "Staging environment not configured"
      log_info "Create it with: nself env create staging staging"
    fi
    return $?
    ;;
  prod|production)
    if [[ -d ".environments/prod" ]]; then
      deploy_to_env "prod" "$@"
    else
      log_error "Production environment not configured"
      log_info "Create it with: nself env create prod prod"
    fi
    return $?
    ;;
  init)
    deploy_init "$@"
    ;;
  check)
    deploy_check "$@"
    ;;
  ssh)
    deploy_ssh "$@"
    ;;
  status)
    deploy_status "$@"
    ;;
  rollback)
    deploy_rollback "$@"
    ;;
  logs)
    deploy_logs "$@"
    ;;
  webhook)
    deploy_webhook "$@"
    ;;
  health)
    deploy_health "$@"
    ;;
  check-access|access)
    deploy_check_access "$@"
    ;;
  -h | --help | help | "")
    show_deploy_help
    ;;
  --check-access)
    deploy_check_access
    ;;
  --dry-run)
    log_error "Specify an environment for dry-run"
    log_info "Example: nself deploy staging --dry-run"
    return 1
    ;;
  *)
    log_error "Unknown subcommand or environment: $subcommand"
    echo ""
    show_deploy_help
    return 1
    ;;
  esac
}

# Export for use as library
export -f cmd_deploy

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  cmd_deploy "$@"
fi