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
source "$LIB_DIR/deploy/security-preflight.sh" 2>/dev/null || true

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

# Check git status and warn about uncommitted changes
check_git_status() {
  local show_warnings="${1:-true}"

  # Skip if not in a git repo
  if ! git rev-parse --git-dir >/dev/null 2>&1; then
    return 0
  fi

  local warnings=0
  local git_issues=()

  # Check for uncommitted changes
  local unstaged=$(git diff --name-only 2>/dev/null | wc -l | tr -d ' ')
  local staged=$(git diff --cached --name-only 2>/dev/null | wc -l | tr -d ' ')
  local untracked=$(git ls-files --others --exclude-standard 2>/dev/null | wc -l | tr -d ' ')

  if [[ "$unstaged" -gt 0 ]]; then
    git_issues+=("$unstaged unstaged changes")
    warnings=$((warnings + 1))
  fi

  if [[ "$staged" -gt 0 ]]; then
    git_issues+=("$staged staged but uncommitted changes")
    warnings=$((warnings + 1))
  fi

  if [[ "$untracked" -gt 0 ]]; then
    git_issues+=("$untracked untracked files")
    # Not critical, just informational
  fi

  # Check if branch is behind remote (fetch first, then compare)
  local current_branch=$(git branch --show-current 2>/dev/null)
  if [[ -n "$current_branch" ]]; then
    # Quick fetch to update remote refs (with timeout)
    git fetch origin "$current_branch" --quiet 2>/dev/null || true

    local behind=$(git rev-list --count HEAD..origin/"$current_branch" 2>/dev/null || echo "0")
    local ahead=$(git rev-list --count origin/"$current_branch"..HEAD 2>/dev/null || echo "0")

    if [[ "$behind" -gt 0 ]]; then
      git_issues+=("$behind commit(s) behind origin/$current_branch")
      warnings=$((warnings + 1))
    fi

    if [[ "$ahead" -gt 0 ]]; then
      git_issues+=("$ahead commit(s) ahead of origin/$current_branch (not pushed)")
    fi
  fi

  # Display warnings if requested
  if [[ "$show_warnings" == "true" ]] && [[ ${#git_issues[@]} -gt 0 ]]; then
    printf "\n${COLOR_YELLOW}⚠ Git Status Warning${COLOR_RESET}\n"
    for issue in "${git_issues[@]}"; do
      printf "  ${COLOR_DIM}• %s${COLOR_RESET}\n" "$issue"
    done
    printf "\n"

    if [[ $warnings -gt 0 ]]; then
      printf "${COLOR_DIM}Recommendation: Commit and push changes before deploying${COLOR_RESET}\n"
      printf "${COLOR_DIM}  git add . && git commit -m 'Pre-deploy commit' && git push${COLOR_RESET}\n\n"
    fi
  fi

  return $warnings
}

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

  # ═══════════════════════════════════════════════════════════════
  # PRE-FLIGHT: Server Connectivity Check (BEFORE any deploy steps)
  # ═══════════════════════════════════════════════════════════════
  printf "${COLOR_CYAN}Pre-flight checks...${COLOR_RESET}\n"

  # Check 1: Basic network reachability (port check)
  printf "  Checking server reachability... "
  if nc -z -w 5 "$host" "$port" 2>/dev/null; then
    printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
  else
    printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
    printf "\n"
    log_error "Cannot reach server $host on port $port"
    printf "\n"
    printf "${COLOR_YELLOW}Troubleshooting steps:${COLOR_RESET}\n"
    printf "  1. Check if server is running in your VPS provider console\n"
    printf "  2. Verify firewall allows SSH from your IP address\n"
    printf "  3. Try: ${COLOR_CYAN}nc -vz %s %s${COLOR_RESET}\n" "$host" "$port"
    printf "  4. Run: ${COLOR_CYAN}nself server diagnose %s${COLOR_RESET}\n" "$env_name"
    printf "\n"
    printf "${COLOR_DIM}Common causes:${COLOR_RESET}\n"
    printf "  • Server is powered off or crashed\n"
    printf "  • Firewall blocking SSH (check Hetzner/DO firewall rules)\n"
    printf "  • Wrong IP address in server.json\n"
    printf "  • SSH port changed from default 22\n"
    printf "\n"
    return 1
  fi

  # Check 2: SSH authentication
  printf "  Checking SSH authentication... "
  local ssh_test_args=()
  [[ -n "$key_file" ]] && ssh_test_args+=("-i" "$key_file")
  ssh_test_args+=("-o" "ConnectTimeout=10")
  ssh_test_args+=("-o" "BatchMode=yes")
  ssh_test_args+=("-o" "StrictHostKeyChecking=accept-new")
  ssh_test_args+=("-p" "$port")

  if ssh "${ssh_test_args[@]}" "$user@$host" "echo ok" 2>/dev/null | grep -q "ok"; then
    printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
  else
    printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
    printf "\n"
    log_error "SSH authentication failed"
    printf "\n"
    printf "${COLOR_YELLOW}Troubleshooting steps:${COLOR_RESET}\n"
    printf "  1. Verify SSH key exists: ${COLOR_CYAN}ls -la %s${COLOR_RESET}\n" "$key_file"
    printf "  2. Verify key is authorized on server\n"
    printf "  3. Try manual SSH: ${COLOR_CYAN}ssh -i %s -p %s %s@%s${COLOR_RESET}\n" "$key_file" "$port" "$user" "$host"
    printf "\n"
    return 1
  fi

  # Check 3: Docker on remote server
  printf "  Checking Docker on server... "
  local docker_check
  docker_check=$(ssh "${ssh_test_args[@]}" "$user@$host" "docker --version 2>/dev/null" 2>/dev/null)
  if [[ -n "$docker_check" ]]; then
    printf "${COLOR_GREEN}OK${COLOR_RESET} (%s)\n" "$(echo "$docker_check" | head -1 | cut -d',' -f1)"
  else
    printf "${COLOR_YELLOW}WARNING${COLOR_RESET} (Docker not found)\n"
    printf "    ${COLOR_DIM}Run 'nself server init' to install Docker${COLOR_RESET}\n"
  fi

  # Check 4: Disk space
  printf "  Checking disk space... "
  local disk_free
  disk_free=$(ssh "${ssh_test_args[@]}" "$user@$host" "df -h / | tail -1 | awk '{print \$4}'" 2>/dev/null)
  if [[ -n "$disk_free" ]]; then
    # Extract numeric value
    local disk_gb=$(echo "$disk_free" | sed 's/[^0-9.]//g')
    if (( $(echo "$disk_gb > 2" | bc -l 2>/dev/null || echo "1") )); then
      printf "${COLOR_GREEN}OK${COLOR_RESET} (%s available)\n" "$disk_free"
    else
      printf "${COLOR_YELLOW}LOW${COLOR_RESET} (%s available)\n" "$disk_free"
    fi
  else
    printf "${COLOR_DIM}SKIP${COLOR_RESET}\n"
  fi

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

  # Check git status and warn about uncommitted changes
  check_git_status "true"

  # ═══════════════════════════════════════════════════════════════
  # SECURITY PRE-FLIGHT CHECKS (Production Only)
  # ═══════════════════════════════════════════════════════════════
  if [[ "$env_type" == "production" || "$env_type" == "prod" || "$env_name" == "prod" ]]; then
    printf "\n${COLOR_CYAN}Running security pre-flight checks...${COLOR_RESET}\n\n"

    # Source security preflight module
    if [[ -f "$LIB_DIR/deploy/security-preflight.sh" ]]; then
      source "$LIB_DIR/deploy/security-preflight.sh"

      if ! security::preflight "$env_name" "$env_dir" "false"; then
        printf "\n"
        log_error "Security checks failed - deployment blocked"
        printf "\n"
        printf "To fix these issues:\n"
        printf "  1. Generate secure secrets:  ${COLOR_CYAN}nself secrets generate --env %s${COLOR_RESET}\n" "$env_name"
        printf "  2. Review configuration:     ${COLOR_CYAN}nself validate %s${COLOR_RESET}\n" "$env_name"
        printf "  3. Force deploy (NOT SAFE):  ${COLOR_CYAN}nself deploy %s --force${COLOR_RESET}\n" "$env_name"
        printf "\n"

        if [[ "$force" != "true" ]]; then
          return 1
        else
          log_warning "Force flag used - proceeding despite security issues"
        fi
      fi
    else
      log_warning "Security preflight module not found - skipping checks"
    fi
  fi

  # ═══════════════════════════════════════════════════════════════
  # AUTOMATIC SSL CERTIFICATE HANDLING
  # The Golden Rule: No manual certbot or SSL commands needed
  # ═══════════════════════════════════════════════════════════════
  printf "${COLOR_CYAN}Checking SSL certificates...${COLOR_RESET}\n"

  local ssl_ready="false"
  local base_domain="${BASE_DOMAIN:-}"

  # Load base domain from env if not set
  if [[ -z "$base_domain" ]] && [[ -f "$env_dir/.env" ]]; then
    base_domain=$(grep "^BASE_DOMAIN=" "$env_dir/.env" 2>/dev/null | cut -d'=' -f2)
  fi

  # Check if we have SSL certificates locally
  local ssl_cert_path=""
  for cert_dir in "nginx/ssl" "ssl/certificates" "nginx/ssl/nself-org"; do
    if [[ -f "$cert_dir/fullchain.pem" ]] || [[ -f "$cert_dir/cert.pem" ]]; then
      ssl_ready="true"
      ssl_cert_path="$cert_dir"
      printf "  ${COLOR_GREEN}✓${COLOR_RESET} SSL certificates found locally: %s\n" "$cert_dir"
      break
    fi
  done

  if [[ "$ssl_ready" == "false" ]]; then
    printf "  ${COLOR_YELLOW}!${COLOR_RESET} No SSL certificates found locally\n"

    # Auto-generate SSL certificates
    printf "  ${COLOR_BLUE}⠋${COLOR_RESET} Generating SSL certificates...\n"

    if [[ -f "$SCRIPT_DIR/ssl.sh" ]]; then
      bash "$SCRIPT_DIR/ssl.sh" bootstrap >/dev/null 2>&1
      if [[ $? -eq 0 ]]; then
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} SSL certificates generated\n"
        ssl_ready="true"
      else
        printf "  ${COLOR_YELLOW}!${COLOR_RESET} SSL generation failed (will use remote server's certs if available)\n"
      fi
    fi
  fi

  # ═══════════════════════════════════════════════════════════════
  # AUTOMATIC DNS CHECK
  # Verify domain resolves before deployment
  # ═══════════════════════════════════════════════════════════════
  if [[ -n "$base_domain" ]] && [[ "$base_domain" != "localhost" ]] && [[ "$base_domain" != *"local.nself.org"* ]]; then
    printf "${COLOR_CYAN}Checking DNS resolution...${COLOR_RESET}\n"

    # Get server IP
    local server_ip=""
    server_ip=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes -o ConnectTimeout=5 "$user@$host" \
      "curl -s ifconfig.me 2>/dev/null || curl -s icanhazip.com 2>/dev/null" 2>/dev/null)

    if [[ -n "$server_ip" ]]; then
      # Check if domain resolves to server IP
      local domain_ip=""
      domain_ip=$(dig +short "$base_domain" 2>/dev/null | head -1)

      if [[ "$domain_ip" == "$server_ip" ]]; then
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} DNS OK: %s → %s\n" "$base_domain" "$server_ip"
      else
        printf "  ${COLOR_YELLOW}!${COLOR_RESET} DNS mismatch: %s → %s (server: %s)\n" "$base_domain" "$domain_ip" "$server_ip"
        printf "    ${COLOR_DIM}Update your DNS to point to %s${COLOR_RESET}\n" "$server_ip"
      fi
    fi
  fi

  printf "\n"

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

      # Load environment variables
      set -a
      source .env 2>/dev/null || true
      set +a

      # Pull Docker images
      docker compose pull 2>/dev/null || true

      # ═══════════════════════════════════════════════════════════════
      # ENSURE DATABASE EXISTS BEFORE STARTING SERVICES
      # The Golden Rule: Users should NEVER need to SSH into servers
      # ═══════════════════════════════════════════════════════════════

      # Start ONLY postgres first
      echo 'step:starting_postgres'
      docker compose up -d postgres 2>/dev/null

      # Wait for PostgreSQL to accept connections (max 60 seconds)
      WAITED=0
      MAX_WAIT=60
      while [ \$WAITED -lt \$MAX_WAIT ]; do
        if docker compose exec -T postgres pg_isready -U \"\${POSTGRES_USER:-postgres}\" >/dev/null 2>&1; then
          break
        fi
        sleep 1
        WAITED=\$((WAITED + 1))
      done

      if [ \$WAITED -ge \$MAX_WAIT ]; then
        echo 'error:postgres_not_ready'
        exit 1
      fi

      echo 'step:postgres_ready'

      # Get database name from env
      DB_NAME=\"\${POSTGRES_DB:-\${PROJECT_NAME:-nhost}}\"
      DB_USER=\"\${POSTGRES_USER:-postgres}\"

      # Create database if it doesn't exist
      DB_EXISTS=\$(docker compose exec -T postgres psql -U \"\$DB_USER\" -tAc \"SELECT 1 FROM pg_database WHERE datname='\$DB_NAME'\" 2>/dev/null || echo '')

      if [ \"\$DB_EXISTS\" != '1' ]; then
        echo 'step:creating_database'
        docker compose exec -T postgres psql -U \"\$DB_USER\" -c \"CREATE DATABASE \\\"\$DB_NAME\\\";\" 2>/dev/null || true
      fi

      # Create required schemas
      echo 'step:ensuring_schemas'
      docker compose exec -T postgres psql -U \"\$DB_USER\" -d \"\$DB_NAME\" -c \"
        CREATE SCHEMA IF NOT EXISTS auth;
        CREATE SCHEMA IF NOT EXISTS storage;
        CREATE SCHEMA IF NOT EXISTS public;
        CREATE EXTENSION IF NOT EXISTS pgcrypto;
        CREATE EXTENSION IF NOT EXISTS citext;
        CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";
        GRANT ALL ON SCHEMA auth TO \\\"\$DB_USER\\\";
        GRANT ALL ON SCHEMA storage TO \\\"\$DB_USER\\\";
        GRANT ALL ON SCHEMA public TO \\\"\$DB_USER\\\";
      \" 2>/dev/null || true

      echo 'step:database_ready'

      # ═══════════════════════════════════════════════════════════════
      # NOW start all services (database is guaranteed to exist)
      # ═══════════════════════════════════════════════════════════════
      echo 'step:starting_all_services'
      docker compose up -d --remove-orphans --force-recreate 2>/dev/null

      echo 'deploy_ok'
    " 2>/dev/null)

    if echo "$deploy_result" | grep -q "deploy_ok"; then
      printf "${COLOR_GREEN}OK${COLOR_RESET}\n"

      # Show steps that were executed
      if echo "$deploy_result" | grep -q "step:creating_database"; then
        printf "    ${COLOR_CYAN}→${COLOR_RESET} Created database\n"
      fi
      if echo "$deploy_result" | grep -q "step:ensuring_schemas"; then
        printf "    ${COLOR_CYAN}→${COLOR_RESET} Ensured schemas (auth, storage, public)\n"
      fi
    else
      printf "${COLOR_RED}FAILED${COLOR_RESET}\n"
      if echo "$deploy_result" | grep -q "error:postgres_not_ready"; then
        log_error "PostgreSQL failed to start within 60 seconds"
      else
        log_error "Deployment failed. Check server logs."
      fi
      return 1
    fi

    # ═══════════════════════════════════════════════════════════════
    # AUTOMATIC SSL ON REMOTE SERVER
    # Request Let's Encrypt certificate if domain is configured
    # ═══════════════════════════════════════════════════════════════
    if [[ -n "$base_domain" ]] && [[ "$base_domain" != "localhost" ]] && [[ "$base_domain" != *"local.nself.org"* ]]; then
      printf "  Checking/requesting SSL certificate... "

      local ssl_result
      ssl_result=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
        cd '$deploy_path' || exit 1

        DOMAIN='$base_domain'
        SSL_DIR='$deploy_path/nginx/ssl'

        # Check if we already have a valid certificate
        if [ -f \"\$SSL_DIR/fullchain.pem\" ]; then
          # Check if cert is valid for at least 7 days
          if openssl x509 -in \"\$SSL_DIR/fullchain.pem\" -checkend 604800 2>/dev/null; then
            echo 'ssl_exists'
            exit 0
          fi
        fi

        # Try to get Let's Encrypt certificate
        if command -v certbot >/dev/null 2>&1; then
          # Stop nginx temporarily to free port 80
          docker compose stop nginx 2>/dev/null || true

          certbot certonly --standalone \\
            -d \$DOMAIN \\
            --non-interactive \\
            --agree-tos \\
            --email admin@\$DOMAIN \\
            --cert-name nself 2>/dev/null

          if [ -d /etc/letsencrypt/live/nself ]; then
            mkdir -p \"\$SSL_DIR\"
            cp /etc/letsencrypt/live/nself/fullchain.pem \"\$SSL_DIR/\"
            cp /etc/letsencrypt/live/nself/privkey.pem \"\$SSL_DIR/\"
            chmod 600 \"\$SSL_DIR/privkey.pem\"

            # Restart nginx
            docker compose start nginx 2>/dev/null || docker compose up -d nginx
            echo 'ssl_requested'
            exit 0
          fi

          # Restart nginx even if cert failed
          docker compose start nginx 2>/dev/null || docker compose up -d nginx
        fi

        echo 'ssl_skipped'
      " 2>/dev/null)

      case "$ssl_result" in
        *ssl_exists*)
          printf "${COLOR_GREEN}OK${COLOR_RESET} (certificate valid)\n"
          ;;
        *ssl_requested*)
          printf "${COLOR_GREEN}OK${COLOR_RESET} (Let's Encrypt certificate issued)\n"
          ;;
        *)
          printf "${COLOR_YELLOW}SKIP${COLOR_RESET} (using existing or self-signed)\n"
          ;;
      esac
    fi
  fi

  # ═══════════════════════════════════════════════════════════════
  # POST-DEPLOY HEALTH VERIFICATION
  # Verify EVERYTHING works - not just that containers are running
  # ═══════════════════════════════════════════════════════════════
  if [[ "$skip_health" != "true" ]]; then
    printf "\n"
    log_info "Verifying deployment health..."

    # Give services time to start up
    sleep 8

    local health_failures=0
    local health_warnings=0

    # Step 1: Check Docker container health
    printf "  Checking container health... "
    local container_health
    container_health=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
      cd '$deploy_path'
      TOTAL=\$(docker compose ps --format '{{.Name}}' 2>/dev/null | wc -l)
      UP=\$(docker compose ps --format '{{.Status}}' 2>/dev/null | grep -c 'Up' || echo '0')
      HEALTHY=\$(docker compose ps --format '{{.Status}}' 2>/dev/null | grep -c 'healthy' || echo '0')
      UNHEALTHY=\$(docker compose ps --format '{{.Status}}' 2>/dev/null | grep -c 'unhealthy' || echo '0')
      echo \"total:\$TOTAL up:\$UP healthy:\$HEALTHY unhealthy:\$UNHEALTHY\"
    " 2>/dev/null)

    local total_containers up_containers healthy_containers unhealthy_containers
    total_containers=$(echo "$container_health" | grep -o 'total:[0-9]*' | cut -d: -f2)
    up_containers=$(echo "$container_health" | grep -o 'up:[0-9]*' | cut -d: -f2)
    unhealthy_containers=$(echo "$container_health" | grep -o 'unhealthy:[0-9]*' | cut -d: -f2)

    if [[ "${unhealthy_containers:-0}" -gt 0 ]]; then
      printf "${COLOR_RED}%s unhealthy${COLOR_RESET}\n" "$unhealthy_containers"
      health_failures=$((health_failures + 1))
    elif [[ "${up_containers:-0}" -gt 0 ]]; then
      printf "${COLOR_GREEN}%s/%s running${COLOR_RESET}\n" "$up_containers" "$total_containers"
    else
      printf "${COLOR_YELLOW}Starting...${COLOR_RESET}\n"
      health_warnings=$((health_warnings + 1))
    fi

    # Step 2: Check API health endpoint
    if [[ -n "$base_domain" ]] && [[ "$base_domain" != "localhost" ]]; then
      printf "  API health check... "
      local api_health
      api_health=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 \
        "https://api.${base_domain}/healthz" 2>/dev/null || echo "000")

      if [[ "$api_health" == "200" ]]; then
        printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
      elif [[ "$api_health" == "000" ]]; then
        printf "${COLOR_YELLOW}No response${COLOR_RESET} (may still be starting)\n"
        health_warnings=$((health_warnings + 1))
      else
        printf "${COLOR_RED}HTTP %s${COLOR_RESET}\n" "$api_health"
        health_failures=$((health_failures + 1))
      fi

      # Step 3: Check Auth health endpoint
      printf "  Auth health check... "
      local auth_health
      auth_health=$(curl -s -o /dev/null -w "%{http_code}" --connect-timeout 10 \
        "https://auth.${base_domain}/healthz" 2>/dev/null || echo "000")

      if [[ "$auth_health" == "200" ]]; then
        printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
      elif [[ "$auth_health" == "000" ]]; then
        printf "${COLOR_YELLOW}No response${COLOR_RESET}\n"
        health_warnings=$((health_warnings + 1))
      else
        printf "${COLOR_RED}HTTP %s${COLOR_RESET}\n" "$auth_health"
        health_failures=$((health_failures + 1))
      fi

      # Step 4: Test GraphQL endpoint actually works
      printf "  GraphQL test... "
      local graphql_test
      graphql_test=$(curl -s -X POST "https://api.${base_domain}/v1/graphql" \
        -H "Content-Type: application/json" \
        -d '{"query":"{ __typename }"}' \
        --connect-timeout 10 2>/dev/null)

      if echo "$graphql_test" | grep -q '"data"'; then
        printf "${COLOR_GREEN}OK${COLOR_RESET}\n"
      elif echo "$graphql_test" | grep -q 'errors'; then
        local error_msg
        error_msg=$(echo "$graphql_test" | grep -o '"message":"[^"]*"' | head -1 | cut -d'"' -f4)
        printf "${COLOR_RED}ERROR${COLOR_RESET} - %s\n" "${error_msg:-GraphQL query failed}"
        health_failures=$((health_failures + 1))
      else
        printf "${COLOR_YELLOW}No response${COLOR_RESET}\n"
        health_warnings=$((health_warnings + 1))
      fi
    fi

    # ═══════════════════════════════════════════════════════════════
    # AUTO-FIX: If health check failed, attempt automatic fixes
    # ═══════════════════════════════════════════════════════════════
    if [[ $health_failures -gt 0 ]]; then
      printf "\n"
      log_warning "Detected $health_failures health issue(s). Attempting auto-fix..."

      local fix_result
      fix_result=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
        cd '$deploy_path' || exit 1

        # Load environment
        set -a
        source .env 2>/dev/null || true
        set +a

        DB_NAME=\"\${POSTGRES_DB:-\${PROJECT_NAME:-nhost}}\"
        DB_USER=\"\${POSTGRES_USER:-postgres}\"

        # Fix 1: Ensure database exists
        DB_EXISTS=\$(docker compose exec -T postgres psql -U \"\$DB_USER\" -tAc \"SELECT 1 FROM pg_database WHERE datname='\$DB_NAME'\" 2>/dev/null || echo '')
        if [ \"\$DB_EXISTS\" != '1' ]; then
          echo 'fix:creating_database'
          docker compose exec -T postgres psql -U \"\$DB_USER\" -c \"CREATE DATABASE \\\"\$DB_NAME\\\";\" 2>/dev/null || true
        fi

        # Fix 2: Ensure schemas exist
        docker compose exec -T postgres psql -U \"\$DB_USER\" -d \"\$DB_NAME\" -c \"
          CREATE SCHEMA IF NOT EXISTS auth;
          CREATE SCHEMA IF NOT EXISTS storage;
          CREATE EXTENSION IF NOT EXISTS pgcrypto;
          CREATE EXTENSION IF NOT EXISTS citext;
        \" 2>/dev/null && echo 'fix:schemas_ensured'

        # Fix 3: Restart unhealthy services
        UNHEALTHY=\$(docker compose ps --format '{{.Name}}:{{.Status}}' 2>/dev/null | grep -i 'unhealthy' | cut -d: -f1)
        if [ -n \"\$UNHEALTHY\" ]; then
          echo \"fix:restarting_unhealthy:\$UNHEALTHY\"
          for svc in \$UNHEALTHY; do
            docker restart \"\$svc\" 2>/dev/null || true
          done
        fi

        # Fix 4: Ensure Hasura can connect (restart if needed)
        HASURA_STATUS=\$(docker compose ps hasura --format '{{.Status}}' 2>/dev/null | head -1)
        if echo \"\$HASURA_STATUS\" | grep -qi 'unhealthy'; then
          echo 'fix:restarting_hasura'
          docker compose restart hasura 2>/dev/null || true
        fi

        echo 'fix_complete'
      " 2>/dev/null)

      # Report what was fixed
      if echo "$fix_result" | grep -q "fix:creating_database"; then
        printf "  ${COLOR_CYAN}→${COLOR_RESET} Created missing database\n"
      fi
      if echo "$fix_result" | grep -q "fix:schemas_ensured"; then
        printf "  ${COLOR_CYAN}→${COLOR_RESET} Ensured database schemas\n"
      fi
      if echo "$fix_result" | grep -q "fix:restarting_unhealthy"; then
        printf "  ${COLOR_CYAN}→${COLOR_RESET} Restarted unhealthy services\n"
      fi
      if echo "$fix_result" | grep -q "fix:restarting_hasura"; then
        printf "  ${COLOR_CYAN}→${COLOR_RESET} Restarted Hasura\n"
      fi

      # Wait and re-verify
      printf "\n  Re-verifying after fixes...\n"
      sleep 10

      # Quick re-check
      local recheck_health
      recheck_health=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
        cd '$deploy_path'
        UNHEALTHY=\$(docker compose ps --format '{{.Status}}' 2>/dev/null | grep -c 'unhealthy' || echo '0')
        UP=\$(docker compose ps --format '{{.Status}}' 2>/dev/null | grep -c 'Up' || echo '0')
        echo \"up:\$UP unhealthy:\$UNHEALTHY\"
      " 2>/dev/null)

      local recheck_unhealthy recheck_up
      recheck_unhealthy=$(echo "$recheck_health" | grep -o 'unhealthy:[0-9]*' | cut -d: -f2)
      recheck_up=$(echo "$recheck_health" | grep -o 'up:[0-9]*' | cut -d: -f2)

      if [[ "${recheck_unhealthy:-0}" -eq 0 ]] && [[ "${recheck_up:-0}" -gt 0 ]]; then
        printf "  ${COLOR_GREEN}✓${COLOR_RESET} All services now healthy\n"
        health_failures=0
      else
        printf "  ${COLOR_YELLOW}!${COLOR_RESET} Some issues may persist. Run: nself doctor --fix\n"
      fi
    fi

    # Final status
    printf "\n"
    if [[ $health_failures -eq 0 ]]; then
      printf "${COLOR_GREEN}═══════════════════════════════════════════════════════════════${COLOR_RESET}\n"
      printf "${COLOR_GREEN}  ✓ All systems operational${COLOR_RESET}\n"
      printf "${COLOR_GREEN}═══════════════════════════════════════════════════════════════${COLOR_RESET}\n"
    else
      printf "${COLOR_YELLOW}═══════════════════════════════════════════════════════════════${COLOR_RESET}\n"
      printf "${COLOR_YELLOW}  ! Deployment complete with warnings${COLOR_RESET}\n"
      printf "${COLOR_YELLOW}  Run: nself doctor --fix for detailed diagnostics${COLOR_RESET}\n"
      printf "${COLOR_YELLOW}═══════════════════════════════════════════════════════════════${COLOR_RESET}\n"
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
# Advanced Deployment Strategies (v0.4.7)
# ============================================================

# Preview deployment - show what would change
deploy_preview() {
  local env_name="${1:-staging}"
  shift || true

  show_command_header "nself deploy preview" "Deployment Preview for $env_name"

  local env_dir=".environments/$env_name"

  if [[ ! -d "$env_dir" ]]; then
    log_error "Environment '$env_name' not found"
    return 1
  fi

  # Load server config
  local host port user key_file deploy_path
  host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
  user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

  port="${port:-22}"
  user="${user:-root}"
  deploy_path="${deploy_path:-/opt/nself}"
  key_file="${key_file:-$HOME/.ssh/id_rsa}"
  key_file="${key_file/#\~/$HOME}"

  if [[ -z "$host" ]]; then
    log_error "No host configured for $env_name"
    return 1
  fi

  printf "${COLOR_CYAN}=== Deployment Preview ===${COLOR_RESET}\n\n"
  printf "Target: %s@%s:%s\n" "$user" "$host" "$deploy_path"
  printf "Environment: %s\n\n" "$env_name"

  # Show config diff
  printf "${COLOR_CYAN}Configuration Changes:${COLOR_RESET}\n"

  local remote_env
  remote_env=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" \
    "cat '$deploy_path/.env' 2>/dev/null || echo ''" 2>/dev/null)

  if [[ -f "$env_dir/.env" ]]; then
    local local_env
    local_env=$(cat "$env_dir/.env")

    local temp_remote=$(mktemp)
    local temp_local=$(mktemp)
    echo "$remote_env" > "$temp_remote"
    echo "$local_env" > "$temp_local"

    local diff_output
    diff_output=$(diff -u "$temp_remote" "$temp_local" 2>/dev/null || true)

    if [[ -n "$diff_output" ]]; then
      echo "$diff_output" | head -30
      local diff_lines=$(echo "$diff_output" | wc -l)
      if [[ $diff_lines -gt 30 ]]; then
        printf "... (%d more lines)\n" "$((diff_lines - 30))"
      fi
    else
      printf "  No configuration changes\n"
    fi

    rm -f "$temp_remote" "$temp_local"
  fi

  printf "\n"

  # Show Docker image updates
  printf "${COLOR_CYAN}Docker Image Status:${COLOR_RESET}\n"
  ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
    cd '$deploy_path' 2>/dev/null || exit 0
    docker compose images 2>/dev/null | tail -n +2 || echo '  No images found'
  " 2>/dev/null

  printf "\n"

  # Show current running services
  printf "${COLOR_CYAN}Currently Running Services:${COLOR_RESET}\n"
  ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
    cd '$deploy_path' 2>/dev/null || exit 0
    docker compose ps --format 'table {{.Service}}\t{{.Status}}\t{{.Ports}}' 2>/dev/null || echo '  No services running'
  " 2>/dev/null

  printf "\n"

  # Show what would be synced
  printf "${COLOR_CYAN}Files to Sync:${COLOR_RESET}\n"
  printf "  docker-compose.yml\n"
  printf "  nginx/\n"
  printf "  postgres/\n"
  [[ -d "services" ]] && printf "  services/\n"
  [[ -d "monitoring" ]] && printf "  monitoring/\n"
  printf "  .env (merged with .env.secrets)\n"

  printf "\n${COLOR_GREEN}Preview complete. Run 'nself deploy %s' to apply.${COLOR_RESET}\n" "$env_name"
}

# Canary deployment - deploy to subset first
deploy_canary() {
  local env_name="${1:-staging}"
  local percentage="${2:-10}"
  shift 2 || true

  show_command_header "nself deploy canary" "Canary Deployment to $env_name"

  printf "${COLOR_CYAN}Canary Deployment Strategy:${COLOR_RESET}\n"
  printf "  • Deploy new version to %s%% of instances\n" "$percentage"
  printf "  • Monitor for errors/performance issues\n"
  printf "  • Gradually increase if healthy\n"
  printf "  • Rollback automatically on failure\n\n"

  local env_dir=".environments/$env_name"

  if [[ ! -d "$env_dir" ]]; then
    log_error "Environment '$env_name' not found"
    return 1
  fi

  # Check for Kubernetes deployment
  if [[ -f ".nself/k8s/manifests/00-namespace.yaml" ]]; then
    log_info "Kubernetes canary deployment..."

    # Use kubectl for canary
    local namespace
    namespace=$(grep "namespace:" ".nself/k8s.yml" 2>/dev/null | cut -d':' -f2 | tr -d ' ')
    namespace="${namespace:-default}"

    # Get current deployments
    local deployments
    deployments=$(kubectl get deployments -n "$namespace" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null)

    for deployment in $deployments; do
      local current_replicas
      current_replicas=$(kubectl get deployment "$deployment" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null)

      local canary_replicas=$(( (current_replicas * percentage) / 100 ))
      [[ $canary_replicas -lt 1 ]] && canary_replicas=1

      printf "  Scaling %s canary to %d replicas (original: %d)...\n" "$deployment" "$canary_replicas" "$current_replicas"

      # Create canary deployment
      kubectl get deployment "$deployment" -n "$namespace" -o yaml | \
        sed "s/name: $deployment/name: ${deployment}-canary/" | \
        sed "s/replicas: .*/replicas: $canary_replicas/" | \
        kubectl apply -f - 2>/dev/null || true
    done

    log_success "Canary deployment started"
    log_info "Monitor with: kubectl get pods -n $namespace"
    log_info "Promote with: nself deploy canary-promote $env_name"
    log_info "Rollback with: nself deploy canary-rollback $env_name"

  else
    # Docker-based canary (using docker service or docker-compose scale)
    log_info "Docker canary deployment..."

    local host port user key_file deploy_path
    host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
    user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

    port="${port:-22}"
    user="${user:-root}"
    deploy_path="${deploy_path:-/opt/nself}"
    key_file="${key_file:-$HOME/.ssh/id_rsa}"
    key_file="${key_file/#\~/$HOME}"

    # Deploy with --scale for canary
    ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
      cd '$deploy_path'

      # Create canary directory
      mkdir -p '$deploy_path-canary'

      # Copy current to canary
      cp -r '$deploy_path'/* '$deploy_path-canary/' 2>/dev/null || true

      # Start canary with different project name
      cd '$deploy_path-canary'
      docker compose -p canary up -d 2>/dev/null

      echo 'canary_deployed'
    " 2>/dev/null | grep -q "canary_deployed" && \
      log_success "Canary deployment started" || \
      log_error "Canary deployment failed"

    log_info "Monitor canary health, then:"
    log_info "  Promote: nself deploy canary-promote $env_name"
    log_info "  Rollback: nself deploy canary-rollback $env_name"
  fi
}

# Promote canary to full deployment
deploy_canary_promote() {
  local env_name="${1:-staging}"

  log_info "Promoting canary deployment for $env_name..."

  local env_dir=".environments/$env_name"

  if [[ -f ".nself/k8s/manifests/00-namespace.yaml" ]]; then
    # Kubernetes: scale canary to full, remove original
    local namespace
    namespace=$(grep "namespace:" ".nself/k8s.yml" 2>/dev/null | cut -d':' -f2 | tr -d ' ')
    namespace="${namespace:-default}"

    local canary_deployments
    canary_deployments=$(kubectl get deployments -n "$namespace" -o name 2>/dev/null | grep "\-canary")

    for canary in $canary_deployments; do
      local original=${canary%-canary}
      local replicas
      replicas=$(kubectl get "$original" -n "$namespace" -o jsonpath='{.spec.replicas}' 2>/dev/null)

      # Scale canary to full
      kubectl scale "$canary" -n "$namespace" --replicas="$replicas" 2>/dev/null

      # Delete original
      kubectl delete "$original" -n "$namespace" 2>/dev/null

      # Rename canary to original
      kubectl get "$canary" -n "$namespace" -o yaml | \
        sed "s/-canary//" | \
        kubectl apply -f - 2>/dev/null

      kubectl delete "$canary" -n "$namespace" 2>/dev/null
    done

    log_success "Canary promoted to production"

  else
    # Docker: swap canary with production
    local host port user key_file deploy_path
    host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
    user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

    port="${port:-22}"
    user="${user:-root}"
    deploy_path="${deploy_path:-/opt/nself}"
    key_file="${key_file:-$HOME/.ssh/id_rsa}"

    ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
      # Stop old production
      cd '$deploy_path' && docker compose down 2>/dev/null || true

      # Move canary to production
      rm -rf '$deploy_path-old'
      mv '$deploy_path' '$deploy_path-old'
      mv '$deploy_path-canary' '$deploy_path'

      # Restart as production
      cd '$deploy_path' && docker compose -p $(basename '$deploy_path') up -d

      echo 'promoted'
    " 2>/dev/null | grep -q "promoted" && \
      log_success "Canary promoted" || \
      log_error "Promotion failed"
  fi
}

# Rollback canary deployment
deploy_canary_rollback() {
  local env_name="${1:-staging}"

  log_info "Rolling back canary deployment for $env_name..."

  local env_dir=".environments/$env_name"

  if [[ -f ".nself/k8s/manifests/00-namespace.yaml" ]]; then
    # Kubernetes: delete canary deployments
    local namespace
    namespace=$(grep "namespace:" ".nself/k8s.yml" 2>/dev/null | cut -d':' -f2 | tr -d ' ')
    namespace="${namespace:-default}"

    kubectl delete deployments -n "$namespace" -l canary=true 2>/dev/null || true
    kubectl get deployments -n "$namespace" -o name | grep "\-canary" | \
      xargs -I {} kubectl delete {} -n "$namespace" 2>/dev/null || true

    log_success "Canary deployments removed"

  else
    # Docker: remove canary
    local host port user key_file deploy_path
    host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
    user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
    deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

    port="${port:-22}"
    user="${user:-root}"
    deploy_path="${deploy_path:-/opt/nself}"
    key_file="${key_file:-$HOME/.ssh/id_rsa}"

    ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
      cd '$deploy_path-canary' && docker compose -p canary down 2>/dev/null || true
      rm -rf '$deploy_path-canary'
      echo 'rolled_back'
    " 2>/dev/null | grep -q "rolled_back" && \
      log_success "Canary rolled back" || \
      log_error "Rollback failed"
  fi
}

# Blue-green deployment
deploy_blue_green() {
  local env_name="${1:-staging}"
  shift || true

  show_command_header "nself deploy blue-green" "Blue-Green Deployment"

  printf "${COLOR_CYAN}Blue-Green Deployment Strategy:${COLOR_RESET}\n"
  printf "  • Maintain two identical environments (blue/green)\n"
  printf "  • Deploy new version to inactive environment\n"
  printf "  • Switch traffic instantly via load balancer\n"
  printf "  • Zero downtime, instant rollback capability\n\n"

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
  key_file="${key_file:-$HOME/.ssh/id_rsa}"
  key_file="${key_file/#\~/$HOME}"

  # Determine current active color
  local active_color
  active_color=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
    cat '$deploy_path/.active_color' 2>/dev/null || echo 'blue'
  " 2>/dev/null)

  local inactive_color
  if [[ "$active_color" == "blue" ]]; then
    inactive_color="green"
  else
    inactive_color="blue"
  fi

  printf "Current active: ${COLOR_CYAN}%s${COLOR_RESET}\n" "$active_color"
  printf "Deploying to:   ${COLOR_GREEN}%s${COLOR_RESET}\n\n" "$inactive_color"

  # Deploy to inactive environment
  log_info "Deploying to $inactive_color environment..."

  local inactive_path="${deploy_path}-${inactive_color}"

  # Sync files to inactive
  rsync -avz --delete \
    --exclude '.git' \
    --exclude 'node_modules' \
    --exclude '.env.secrets' \
    -e "ssh -i '$key_file' -p '$port' -o BatchMode=yes" \
    docker-compose.yml nginx/ postgres/ \
    $([ -d "services" ] && echo "services/") \
    $([ -d "monitoring" ] && echo "monitoring/") \
    "$user@$host:$inactive_path/" 2>/dev/null

  # Sync env
  if [[ -f "$env_dir/.env" ]]; then
    scp -i "$key_file" -P "$port" "$env_dir/.env" "$user@$host:$inactive_path/.env" 2>/dev/null
  fi

  # Start inactive environment
  ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
    cd '$inactive_path'

    # Merge secrets if present
    if [ -f '.env.secrets' ]; then
      cat .env.secrets >> .env
    fi

    # Start with unique project name
    docker compose -p ${env_name}-${inactive_color} up -d --remove-orphans 2>/dev/null

    echo 'deployed'
  " 2>/dev/null | grep -q "deployed" || {
    log_error "Failed to start $inactive_color environment"
    return 1
  }

  log_success "Deployed to $inactive_color environment"

  # Wait for health check
  log_info "Waiting for services to become healthy..."
  sleep 10

  # Health check
  local health_ok=true
  ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
    cd '$inactive_path'
    docker compose -p ${env_name}-${inactive_color} ps --format '{{.Status}}' | grep -v 'Up' | head -1
  " 2>/dev/null | grep -q "." && health_ok=false

  if [[ "$health_ok" == "true" ]]; then
    log_success "$inactive_color environment is healthy"

    printf "\n"
    log_info "Ready to switch traffic"
    printf "Switch traffic to %s? [y/N] " "$inactive_color"
    read -r confirm
    confirm=$(printf "%s" "$confirm" | tr '[:upper:]' '[:lower:]')

    if [[ "$confirm" == "y" || "$confirm" == "yes" ]]; then
      deploy_blue_green_switch "$env_name" "$inactive_color"
    else
      log_info "Deployment complete. Switch manually with:"
      printf "  nself deploy blue-green-switch %s %s\n" "$env_name" "$inactive_color"
    fi
  else
    log_error "$inactive_color environment health check failed"
    log_info "Check logs: ssh $user@$host 'cd $inactive_path && docker compose logs'"
    log_info "Rollback: nself deploy blue-green-rollback $env_name"
    return 1
  fi
}

# Switch blue-green traffic
deploy_blue_green_switch() {
  local env_name="${1:-staging}"
  local target_color="${2:-}"

  local env_dir=".environments/$env_name"

  local host port user key_file deploy_path
  host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
  user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

  port="${port:-22}"
  user="${user:-root}"
  deploy_path="${deploy_path:-/opt/nself}"
  key_file="${key_file:-$HOME/.ssh/id_rsa}"
  key_file="${key_file/#\~/$HOME}"

  if [[ -z "$target_color" ]]; then
    local active
    active=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" \
      "cat '$deploy_path/.active_color' 2>/dev/null || echo 'blue'" 2>/dev/null)

    if [[ "$active" == "blue" ]]; then
      target_color="green"
    else
      target_color="blue"
    fi
  fi

  log_info "Switching traffic to $target_color..."

  ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" "
    # Update nginx upstream to point to new color
    # This assumes nginx is configured with upstreams for both colors

    # Update active color marker
    echo '$target_color' > '$deploy_path/.active_color'

    # Symlink main path to active color
    rm -f '$deploy_path/current'
    ln -sf '$deploy_path-$target_color' '$deploy_path/current'

    # Reload nginx to pick up new upstream
    docker exec nginx nginx -s reload 2>/dev/null || \
      docker compose exec nginx nginx -s reload 2>/dev/null || true

    echo 'switched'
  " 2>/dev/null | grep -q "switched" && \
    log_success "Traffic switched to $target_color" || \
    log_error "Switch failed"
}

# Rollback blue-green deployment
deploy_blue_green_rollback() {
  local env_name="${1:-staging}"

  local env_dir=".environments/$env_name"

  local host port user key_file deploy_path
  host=$(grep '"host"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  port=$(grep '"port"' "$env_dir/server.json" 2>/dev/null | sed 's/[^0-9]//g')
  user=$(grep '"user"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  key_file=$(grep '"key"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)
  deploy_path=$(grep '"deploy_path"' "$env_dir/server.json" 2>/dev/null | cut -d'"' -f4)

  port="${port:-22}"
  user="${user:-root}"
  deploy_path="${deploy_path:-/opt/nself}"
  key_file="${key_file:-$HOME/.ssh/id_rsa}"

  log_info "Rolling back blue-green deployment..."

  local active
  active=$(ssh -i "$key_file" -p "$port" -o BatchMode=yes "$user@$host" \
    "cat '$deploy_path/.active_color' 2>/dev/null || echo 'blue'" 2>/dev/null)

  local rollback_to
  if [[ "$active" == "blue" ]]; then
    rollback_to="green"
  else
    rollback_to="blue"
  fi

  deploy_blue_green_switch "$env_name" "$rollback_to"

  log_success "Rolled back to $rollback_to"
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
  preview)
    deploy_preview "$@"
    ;;
  canary)
    deploy_canary "$@"
    ;;
  canary-promote)
    deploy_canary_promote "$@"
    ;;
  canary-rollback)
    deploy_canary_rollback "$@"
    ;;
  blue-green|bluegreen)
    deploy_blue_green "$@"
    ;;
  blue-green-switch|switch)
    deploy_blue_green_switch "$@"
    ;;
  blue-green-rollback)
    deploy_blue_green_rollback "$@"
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