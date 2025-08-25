#!/usr/bin/env bash
# deploy.sh - SSH deployment commands for VPS

# Determine root directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(dirname "$(dirname "$SCRIPT_DIR")")"

# Source required utilities
source "$SCRIPT_DIR/../lib/utils/display.sh"
source "$SCRIPT_DIR/../lib/utils/env.sh"

# Show help for deploy command
show_deploy_help() {
  echo "nself deploy - SSH deployment to VPS"
  echo ""
  echo "Usage: nself deploy <subcommand> [OPTIONS]"
  echo ""
  echo "Subcommands:"
  echo "  init       Initialize deployment configuration"
  echo "  ssh        Deploy to VPS via SSH"
  echo "  status     Show deployment status"
  echo "  rollback   Rollback deployment"
  echo "  logs       View deployment logs"
  echo "  webhook    Setup GitHub webhook"
  echo ""
  echo "Description:"
  echo "  Deploys your nself application to a VPS server via SSH."
  echo "  Supports automated setup, SSL configuration, and continuous"
  echo "  deployment through GitHub webhooks."
  echo ""
  echo "Supported VPS Providers:"
  echo "  - DigitalOcean, Linode, Vultr, Hetzner, OVH"
  echo "  - Any Ubuntu/Debian VPS with SSH access"
  echo ""
  echo "Examples:"
  echo "  nself deploy init              # Setup deployment config"
  echo "  nself deploy ssh               # Deploy to server"
  echo "  nself deploy status            # Check deployment"
  echo "  nself deploy webhook           # Setup auto-deploy"
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
    sed -i.bak "s/^DEPLOY_HOST=.*/DEPLOY_HOST=$host/" .env.local
  else
    echo "DEPLOY_HOST=$host" >> .env.local
  fi
  
  if grep -q "^DEPLOY_USER=" .env.local 2>/dev/null; then
    sed -i.bak "s/^DEPLOY_USER=.*/DEPLOY_USER=$user/" .env.local
  else
    echo "DEPLOY_USER=$user" >> .env.local
  fi
  
  if grep -q "^DEPLOY_KEY_PATH=" .env.local 2>/dev/null; then
    sed -i.bak "s|^DEPLOY_KEY_PATH=.*|DEPLOY_KEY_PATH=$key_path|" .env.local
  else
    echo "DEPLOY_KEY_PATH=$key_path" >> .env.local
  fi
  
  if grep -q "^DEPLOY_TARGET_DIR=" .env.local 2>/dev/null; then
    sed -i.bak "s|^DEPLOY_TARGET_DIR=.*|DEPLOY_TARGET_DIR=$target_dir|" .env.local
  else
    echo "DEPLOY_TARGET_DIR=$target_dir" >> .env.local
  fi
  
  if grep -q "^DEPLOY_REPO_URL=" .env.local 2>/dev/null; then
    sed -i.bak "s|^DEPLOY_REPO_URL=.*|DEPLOY_REPO_URL=$repo_url|" .env.local
  else
    echo "DEPLOY_REPO_URL=$repo_url" >> .env.local
  fi
  
  if grep -q "^DEPLOY_BRANCH=" .env.local 2>/dev/null; then
    sed -i.bak "s/^DEPLOY_BRANCH=.*/DEPLOY_BRANCH=$branch/" .env.local
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
      sed -i.bak "s/^DEPLOY_WEBHOOK_SECRET=.*/DEPLOY_WEBHOOK_SECRET=$secret/" .env.local
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

# Main command function
cmd_deploy() {
  local subcommand="${1:-}"
  shift || true
  
  case "$subcommand" in
  init)
    deploy_init "$@"
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
  -h | --help | help | "")
    show_deploy_help
    ;;
  *)
    log_error "Unknown subcommand: $subcommand"
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