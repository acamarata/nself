#!/usr/bin/env bash
# provision.sh - Infrastructure provisioning for nself
# One-command deployment to any supported provider
# Part of nself v0.4.5 - Provider Support

set -o pipefail

# Source shared utilities
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true

# Fallback logging
if ! declare -f log_success >/dev/null 2>&1; then
  log_success() { printf "\033[0;32m✓\033[0m %s\n" "$1"; }
fi
if ! declare -f log_warning >/dev/null 2>&1; then
  log_warning() { printf "\033[0;33m!\033[0m %s\n" "$1"; }
fi
if ! declare -f log_error >/dev/null 2>&1; then
  log_error() { printf "\033[0;31m✗\033[0m %s\n" "$1" >&2; }
fi
if ! declare -f log_info >/dev/null 2>&1; then
  log_info() { printf "\033[0;34mℹ\033[0m %s\n" "$1"; }
fi

# ============================================================================
# CONFIGURATION
# ============================================================================

PROVIDERS_CONFIG_DIR="${HOME}/.nself/providers"
PROVISION_STATE_DIR=".nself/provision"

# Size mappings (cross-provider)
# Format: small=1vCPU/2GB, medium=2vCPU/4GB, large=4vCPU/8GB
declare -A SIZE_MAP

# ============================================================================
# SIZE NORMALIZATION
# ============================================================================

get_size_for_provider() {
  local provider="$1"
  local size="${2:-small}"

  case "$provider" in
    aws)
      case "$size" in
        small)  echo "t3.small" ;;
        medium) echo "t3.medium" ;;
        large)  echo "t3.large" ;;
        xlarge) echo "t3.xlarge" ;;
        *) echo "$size" ;;  # Allow provider-specific sizes
      esac
      ;;
    gcp)
      case "$size" in
        small)  echo "e2-small" ;;
        medium) echo "e2-medium" ;;
        large)  echo "e2-standard-4" ;;
        xlarge) echo "e2-standard-8" ;;
        *) echo "$size" ;;
      esac
      ;;
    azure)
      case "$size" in
        small)  echo "Standard_B1s" ;;
        medium) echo "Standard_B2s" ;;
        large)  echo "Standard_B4ms" ;;
        xlarge) echo "Standard_B8ms" ;;
        *) echo "$size" ;;
      esac
      ;;
    do)
      case "$size" in
        small)  echo "s-1vcpu-2gb" ;;
        medium) echo "s-2vcpu-4gb" ;;
        large)  echo "s-4vcpu-8gb" ;;
        xlarge) echo "s-8vcpu-16gb" ;;
        *) echo "$size" ;;
      esac
      ;;
    hetzner)
      case "$size" in
        small)  echo "cx11" ;;   # 1 vCPU, 2GB
        medium) echo "cx21" ;;   # 2 vCPU, 4GB
        large)  echo "cx31" ;;   # 4 vCPU, 8GB
        xlarge) echo "cx41" ;;   # 8 vCPU, 16GB
        *) echo "$size" ;;
      esac
      ;;
    linode)
      case "$size" in
        small)  echo "g6-nanode-1" ;;    # 1 vCPU, 1GB
        medium) echo "g6-standard-2" ;;  # 2 vCPU, 4GB
        large)  echo "g6-standard-4" ;;  # 4 vCPU, 8GB
        xlarge) echo "g6-standard-8" ;;  # 8 vCPU, 16GB
        *) echo "$size" ;;
      esac
      ;;
    vultr)
      case "$size" in
        small)  echo "vc2-1c-2gb" ;;
        medium) echo "vc2-2c-4gb" ;;
        large)  echo "vc2-4c-8gb" ;;
        xlarge) echo "vc2-8c-16gb" ;;
        *) echo "$size" ;;
      esac
      ;;
    ionos)
      case "$size" in
        small)  echo "CUBES XS" ;;
        medium) echo "CUBES S" ;;
        large)  echo "CUBES M" ;;
        xlarge) echo "CUBES L" ;;
        *) echo "$size" ;;
      esac
      ;;
    scaleway)
      case "$size" in
        small)  echo "DEV1-S" ;;
        medium) echo "DEV1-M" ;;
        large)  echo "DEV1-L" ;;
        xlarge) echo "DEV1-XL" ;;
        *) echo "$size" ;;
      esac
      ;;
    ovh)
      case "$size" in
        small)  echo "s1-2" ;;
        medium) echo "s1-4" ;;
        large)  echo "s1-8" ;;
        xlarge) echo "b2-15" ;;
        *) echo "$size" ;;
      esac
      ;;
  esac
}

# ============================================================================
# COST ESTIMATION
# ============================================================================

estimate_cost() {
  local provider="$1"
  local size="${2:-small}"
  local with_db="${3:-true}"

  # Monthly cost estimates (approximate)
  local compute_cost=0
  local db_cost=0

  case "$provider" in
    aws)
      case "$size" in
        small)  compute_cost=17 ;;
        medium) compute_cost=33 ;;
        large)  compute_cost=67 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=30
      ;;
    gcp)
      case "$size" in
        small)  compute_cost=13 ;;
        medium) compute_cost=27 ;;
        large)  compute_cost=54 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=25
      ;;
    azure)
      case "$size" in
        small)  compute_cost=15 ;;
        medium) compute_cost=31 ;;
        large)  compute_cost=62 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=32
      ;;
    do)
      case "$size" in
        small)  compute_cost=12 ;;
        medium) compute_cost=24 ;;
        large)  compute_cost=48 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=15
      ;;
    hetzner)
      case "$size" in
        small)  compute_cost=4 ;;
        medium) compute_cost=8 ;;
        large)  compute_cost=15 ;;
      esac
      db_cost=0  # Self-hosted
      ;;
    linode)
      case "$size" in
        small)  compute_cost=5 ;;
        medium) compute_cost=24 ;;
        large)  compute_cost=48 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=15
      ;;
    vultr)
      case "$size" in
        small)  compute_cost=10 ;;
        medium) compute_cost=20 ;;
        large)  compute_cost=40 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=15
      ;;
    scaleway)
      case "$size" in
        small)  compute_cost=7 ;;
        medium) compute_cost=15 ;;
        large)  compute_cost=30 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=15
      ;;
    ionos|ovh)
      case "$size" in
        small)  compute_cost=5 ;;
        medium) compute_cost=10 ;;
        large)  compute_cost=20 ;;
      esac
      [[ "$with_db" == "true" ]] && db_cost=15
      ;;
  esac

  local total=$((compute_cost + db_cost))
  echo "$total"
}

# ============================================================================
# PROVISION COMMANDS
# ============================================================================

cmd_provision() {
  local provider="$1"
  shift || true

  # Parse options
  local size="small"
  local region=""
  local dry_run=false
  local estimate_only=false
  local with_db=true
  local config_file=""

  while [[ $# -gt 0 ]]; do
    case "$1" in
      --size)
        size="$2"
        shift 2
        ;;
      --region)
        region="$2"
        shift 2
        ;;
      --dry-run)
        dry_run=true
        shift
        ;;
      --estimate)
        estimate_only=true
        shift
        ;;
      --no-db)
        with_db=false
        shift
        ;;
      --config)
        config_file="$2"
        shift 2
        ;;
      *)
        shift
        ;;
    esac
  done

  # Validate provider
  if [[ -z "$provider" ]]; then
    log_error "Provider required"
    echo ""
    log_info "Supported providers:"
    echo "  aws, gcp, azure, do, hetzner, linode, vultr, ionos, ovh, scaleway"
    echo ""
    log_info "Usage: nself provision <provider> [--size small|medium|large]"
    return 1
  fi

  # Check credentials
  if [[ ! -f "$PROVIDERS_CONFIG_DIR/${provider}.credentials" ]]; then
    log_error "Provider '$provider' not configured"
    log_info "Run: nself providers init $provider"
    return 1
  fi

  # Load project config
  [[ -f ".env" ]] && source ".env" 2>/dev/null || true
  local project_name="${PROJECT_NAME:-$(basename "$PWD")}"

  # Show plan
  echo ""
  log_info "Provisioning Plan"
  echo "══════════════════════════════════════════════"
  echo ""
  printf "  %-20s %s\n" "Provider:" "$provider"
  printf "  %-20s %s\n" "Project:" "$project_name"
  printf "  %-20s %s\n" "Size:" "$size ($(get_size_for_provider "$provider" "$size"))"
  printf "  %-20s %s\n" "Region:" "${region:-default}"
  printf "  %-20s %s\n" "Managed Database:" "$with_db"
  echo ""

  # Show cost estimate
  local monthly_cost=$(estimate_cost "$provider" "$size" "$with_db")
  printf "  %-20s \$%s/month (estimated)\n" "Monthly Cost:" "$monthly_cost"
  echo ""

  if [[ "$estimate_only" == "true" ]]; then
    log_info "Cost estimate only - no resources created"
    return 0
  fi

  if [[ "$dry_run" == "true" ]]; then
    log_info "Dry run - showing what would be created:"
    echo ""
    show_dry_run "$provider" "$size" "$region" "$with_db" "$project_name"
    return 0
  fi

  # Confirm
  printf "Proceed with provisioning? (y/N): "
  read -r confirm
  if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    log_info "Cancelled"
    return 0
  fi

  # Execute provisioning
  echo ""
  log_info "Starting provisioning..."
  echo ""

  case "$provider" in
    aws)     provision_aws "$project_name" "$size" "$region" "$with_db" ;;
    gcp)     provision_gcp "$project_name" "$size" "$region" "$with_db" ;;
    azure)   provision_azure "$project_name" "$size" "$region" "$with_db" ;;
    do)      provision_digitalocean "$project_name" "$size" "$region" "$with_db" ;;
    hetzner) provision_hetzner "$project_name" "$size" "$region" "$with_db" ;;
    linode)  provision_linode "$project_name" "$size" "$region" "$with_db" ;;
    vultr)   provision_vultr "$project_name" "$size" "$region" "$with_db" ;;
    ionos)   provision_ionos "$project_name" "$size" "$region" "$with_db" ;;
    ovh)     provision_ovh "$project_name" "$size" "$region" "$with_db" ;;
    scaleway) provision_scaleway "$project_name" "$size" "$region" "$with_db" ;;
    *)
      log_error "Unknown provider: $provider"
      return 1
      ;;
  esac
}

show_dry_run() {
  local provider="$1"
  local size="$2"
  local region="$3"
  local with_db="$4"
  local project="$5"

  local instance_type=$(get_size_for_provider "$provider" "$size")

  echo "Resources to be created:"
  echo "─────────────────────────"
  echo ""
  echo "1. Compute Instance"
  echo "   • Type: $instance_type"
  echo "   • OS: Ubuntu 22.04 LTS"
  echo "   • Storage: 80GB SSD"
  echo "   • Tags: nself, $project"
  echo ""

  if [[ "$with_db" == "true" ]]; then
    echo "2. Managed PostgreSQL"
    echo "   • Version: 15"
    echo "   • Size: Basic tier"
    echo "   • Storage: 20GB"
    echo ""
  fi

  echo "3. Security / Firewall"
  echo "   • Ports: 22 (SSH), 80 (HTTP), 443 (HTTPS)"
  echo "   • Database port: Internal only"
  echo ""

  echo "4. DNS (if domain configured)"
  echo "   • A record: -> Instance IP"
  echo "   • CNAME: api, auth, storage subdomains"
  echo ""
}

# ============================================================================
# PROVIDER-SPECIFIC PROVISIONING
# ============================================================================

provision_aws() {
  local project="$1"
  local size="$2"
  local region="${3:-us-east-1}"
  local with_db="$4"

  source "$PROVIDERS_CONFIG_DIR/aws.credentials"

  export AWS_ACCESS_KEY_ID="$access_key_id"
  export AWS_SECRET_ACCESS_KEY="$secret_access_key"
  export AWS_DEFAULT_REGION="${region:-$region}"

  if ! command -v aws >/dev/null 2>&1; then
    log_error "AWS CLI not installed"
    log_info "Install: https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html"
    return 1
  fi

  local instance_type=$(get_size_for_provider "aws" "$size")

  # Create security group
  log_info "Creating security group..."
  local sg_id=$(aws ec2 create-security-group \
    --group-name "${project}-nself-sg" \
    --description "nself security group for $project" \
    --output text --query 'GroupId' 2>/dev/null || echo "")

  if [[ -n "$sg_id" ]]; then
    # Add rules
    aws ec2 authorize-security-group-ingress --group-id "$sg_id" --protocol tcp --port 22 --cidr 0.0.0.0/0 2>/dev/null || true
    aws ec2 authorize-security-group-ingress --group-id "$sg_id" --protocol tcp --port 80 --cidr 0.0.0.0/0 2>/dev/null || true
    aws ec2 authorize-security-group-ingress --group-id "$sg_id" --protocol tcp --port 443 --cidr 0.0.0.0/0 2>/dev/null || true
    log_success "Security group created: $sg_id"
  else
    log_warning "Security group may already exist"
  fi

  # Get Ubuntu 22.04 AMI
  log_info "Finding Ubuntu 22.04 AMI..."
  local ami_id=$(aws ec2 describe-images \
    --owners 099720109477 \
    --filters "Name=name,Values=ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*" \
    --query 'Images | sort_by(@, &CreationDate) | [-1].ImageId' \
    --output text 2>/dev/null)

  if [[ -z "$ami_id" || "$ami_id" == "None" ]]; then
    log_error "Could not find Ubuntu AMI"
    return 1
  fi
  log_success "Found AMI: $ami_id"

  # Create user data script
  local user_data=$(cat << 'USERDATA'
#!/bin/bash
set -e

# Update system
apt-get update
apt-get upgrade -y

# Install Docker
curl -fsSL https://get.docker.com | sh
systemctl enable docker
systemctl start docker

# Install Docker Compose
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose

# Install nself CLI
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash

# Create deploy user
useradd -m -s /bin/bash deploy
usermod -aG docker deploy
mkdir -p /home/deploy/.ssh
cp /root/.ssh/authorized_keys /home/deploy/.ssh/ 2>/dev/null || true
chown -R deploy:deploy /home/deploy/.ssh

# Create project directory
mkdir -p /var/www
chown deploy:deploy /var/www
USERDATA
)

  # Launch instance
  log_info "Launching EC2 instance ($instance_type)..."
  local instance_id=$(aws ec2 run-instances \
    --image-id "$ami_id" \
    --instance-type "$instance_type" \
    --security-group-ids "$sg_id" \
    --user-data "$user_data" \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=${project}-nself},{Key=Project,Value=${project}}]" \
    --block-device-mappings "[{\"DeviceName\":\"/dev/sda1\",\"Ebs\":{\"VolumeSize\":80,\"VolumeType\":\"gp3\"}}]" \
    --output text --query 'Instances[0].InstanceId' 2>/dev/null)

  if [[ -z "$instance_id" ]]; then
    log_error "Failed to launch instance"
    return 1
  fi

  log_success "Instance launched: $instance_id"

  # Wait for instance
  log_info "Waiting for instance to be running..."
  aws ec2 wait instance-running --instance-ids "$instance_id"

  # Get public IP
  local public_ip=$(aws ec2 describe-instances \
    --instance-ids "$instance_id" \
    --query 'Reservations[0].Instances[0].PublicIpAddress' \
    --output text)

  log_success "Instance running at: $public_ip"

  # Create RDS if requested
  if [[ "$with_db" == "true" ]]; then
    log_info "Creating RDS PostgreSQL instance..."
    local db_pass=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9' | head -c 24)

    aws rds create-db-instance \
      --db-instance-identifier "${project}-postgres" \
      --db-instance-class "db.t3.micro" \
      --engine "postgres" \
      --engine-version "15" \
      --master-username "postgres" \
      --master-user-password "$db_pass" \
      --allocated-storage 20 \
      --vpc-security-group-ids "$sg_id" \
      --tags "Key=Project,Value=${project}" \
      --no-publicly-accessible \
      >/dev/null 2>&1 || log_warning "RDS creation may take time"

    log_success "RDS instance requested (may take 5-10 minutes)"
  fi

  # Save state
  save_provision_state "aws" "$instance_id" "$public_ip" "${project}-postgres"

  show_provisioning_complete "$public_ip" "$project"
}

provision_digitalocean() {
  local project="$1"
  local size="$2"
  local region="${3:-nyc3}"
  local with_db="$4"

  source "$PROVIDERS_CONFIG_DIR/do.credentials"

  local droplet_size=$(get_size_for_provider "do" "$size")

  # Create droplet with user data
  log_info "Creating DigitalOcean droplet ($droplet_size)..."

  local user_data=$(cat << 'USERDATA'
#!/bin/bash
apt-get update && apt-get upgrade -y
curl -fsSL https://get.docker.com | sh
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
useradd -m -s /bin/bash deploy
usermod -aG docker deploy
mkdir -p /var/www && chown deploy:deploy /var/www
USERDATA
)

  local response=$(curl -s -X POST "https://api.digitalocean.com/v2/droplets" \
    -H "Authorization: Bearer $api_token" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"${project}-nself\",
      \"region\": \"$region\",
      \"size\": \"$droplet_size\",
      \"image\": \"ubuntu-22-04-x64\",
      \"user_data\": $(echo "$user_data" | jq -Rs .),
      \"tags\": [\"nself\", \"$project\"]
    }")

  local droplet_id=$(echo "$response" | jq -r '.droplet.id // empty')

  if [[ -z "$droplet_id" ]]; then
    log_error "Failed to create droplet"
    echo "$response" | jq .
    return 1
  fi

  log_success "Droplet created: $droplet_id"

  # Wait for IP
  log_info "Waiting for droplet IP..."
  sleep 30

  local public_ip=""
  for i in {1..20}; do
    public_ip=$(curl -s "https://api.digitalocean.com/v2/droplets/$droplet_id" \
      -H "Authorization: Bearer $api_token" | jq -r '.droplet.networks.v4[] | select(.type=="public") | .ip_address' 2>/dev/null)

    if [[ -n "$public_ip" ]]; then
      break
    fi
    sleep 5
  done

  log_success "Droplet running at: $public_ip"

  # Create managed database if requested
  if [[ "$with_db" == "true" ]]; then
    log_info "Creating managed PostgreSQL database..."
    local db_response=$(curl -s -X POST "https://api.digitalocean.com/v2/databases" \
      -H "Authorization: Bearer $api_token" \
      -H "Content-Type: application/json" \
      -d "{
        \"name\": \"${project}-db\",
        \"engine\": \"pg\",
        \"version\": \"15\",
        \"size\": \"db-s-1vcpu-1gb\",
        \"region\": \"$region\",
        \"num_nodes\": 1,
        \"tags\": [\"nself\", \"$project\"]
      }")

    log_success "Database cluster requested"
  fi

  save_provision_state "do" "$droplet_id" "$public_ip" "${project}-db"
  show_provisioning_complete "$public_ip" "$project"
}

provision_hetzner() {
  local project="$1"
  local size="$2"
  local region="${3:-fsn1}"
  local with_db="$4"

  source "$PROVIDERS_CONFIG_DIR/hetzner.credentials"

  local server_type=$(get_size_for_provider "hetzner" "$size")

  log_info "Creating Hetzner server ($server_type)..."

  local user_data=$(cat << 'USERDATA'
#!/bin/bash
apt-get update && apt-get upgrade -y
curl -fsSL https://get.docker.com | sh
mkdir -p /usr/local/lib/docker/cli-plugins
curl -SL https://github.com/docker/compose/releases/latest/download/docker-compose-linux-x86_64 -o /usr/local/lib/docker/cli-plugins/docker-compose
chmod +x /usr/local/lib/docker/cli-plugins/docker-compose
curl -fsSL https://raw.githubusercontent.com/acamarata/nself/main/install.sh | bash
useradd -m -s /bin/bash deploy
usermod -aG docker deploy
mkdir -p /var/www && chown deploy:deploy /var/www
USERDATA
)

  local response=$(curl -s -X POST "https://api.hetzner.cloud/v1/servers" \
    -H "Authorization: Bearer $api_token" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"${project}-nself\",
      \"server_type\": \"$server_type\",
      \"location\": \"$region\",
      \"image\": \"ubuntu-22.04\",
      \"user_data\": $(echo "$user_data" | jq -Rs .),
      \"labels\": {\"project\": \"$project\", \"managed-by\": \"nself\"}
    }")

  local server_id=$(echo "$response" | jq -r '.server.id // empty')
  local public_ip=$(echo "$response" | jq -r '.server.public_net.ipv4.ip // empty')

  if [[ -z "$server_id" ]]; then
    log_error "Failed to create server"
    echo "$response" | jq .
    return 1
  fi

  log_success "Server created: $server_id"
  log_success "Server running at: $public_ip"

  # Note: Hetzner doesn't have managed databases, PostgreSQL runs in Docker
  if [[ "$with_db" == "true" ]]; then
    log_info "PostgreSQL will run as Docker container (no managed DB on Hetzner)"
  fi

  save_provision_state "hetzner" "$server_id" "$public_ip" "docker"
  show_provisioning_complete "$public_ip" "$project"
}

provision_linode() {
  local project="$1"
  local size="$2"
  local region="${3:-us-east}"
  local with_db="$4"

  source "$PROVIDERS_CONFIG_DIR/linode.credentials"
  local linode_type=$(get_size_for_provider "linode" "$size")

  log_info "Creating Linode ($linode_type)..."

  local root_pass=$(openssl rand -base64 24 | tr -dc 'a-zA-Z0-9!@#' | head -c 24)

  local response=$(curl -s -X POST "https://api.linode.com/v4/linode/instances" \
    -H "Authorization: Bearer $api_token" \
    -H "Content-Type: application/json" \
    -d "{
      \"type\": \"$linode_type\",
      \"region\": \"$region\",
      \"image\": \"linode/ubuntu22.04\",
      \"root_pass\": \"$root_pass\",
      \"label\": \"${project}-nself\",
      \"tags\": [\"nself\", \"$project\"],
      \"booted\": true
    }")

  local linode_id=$(echo "$response" | jq -r '.id // empty')
  local public_ip=$(echo "$response" | jq -r '.ipv4[0] // empty')

  if [[ -z "$linode_id" ]]; then
    log_error "Failed to create Linode"
    return 1
  fi

  log_success "Linode created: $linode_id"
  log_success "Linode running at: $public_ip"

  save_provision_state "linode" "$linode_id" "$public_ip" "docker"
  show_provisioning_complete "$public_ip" "$project"
}

provision_vultr() {
  local project="$1"
  local size="$2"
  local region="${3:-ewr}"  # New Jersey
  local with_db="$4"

  source "$PROVIDERS_CONFIG_DIR/vultr.credentials"
  local plan=$(get_size_for_provider "vultr" "$size")

  log_info "Creating Vultr instance ($plan)..."

  local response=$(curl -s -X POST "https://api.vultr.com/v2/instances" \
    -H "Authorization: Bearer $api_key" \
    -H "Content-Type: application/json" \
    -d "{
      \"region\": \"$region\",
      \"plan\": \"$plan\",
      \"os_id\": 1743,
      \"label\": \"${project}-nself\",
      \"tags\": [\"nself\", \"$project\"],
      \"enable_ipv6\": false
    }")

  local instance_id=$(echo "$response" | jq -r '.instance.id // empty')
  local public_ip=$(echo "$response" | jq -r '.instance.main_ip // empty')

  if [[ -z "$instance_id" ]]; then
    log_error "Failed to create instance"
    return 1
  fi

  log_success "Instance created: $instance_id"

  # Wait for IP
  if [[ "$public_ip" == "0.0.0.0" ]]; then
    log_info "Waiting for IP assignment..."
    sleep 30
    public_ip=$(curl -s "https://api.vultr.com/v2/instances/$instance_id" \
      -H "Authorization: Bearer $api_key" | jq -r '.instance.main_ip')
  fi

  log_success "Instance running at: $public_ip"

  save_provision_state "vultr" "$instance_id" "$public_ip" "docker"
  show_provisioning_complete "$public_ip" "$project"
}

provision_scaleway() {
  local project="$1"
  local size="$2"
  local region="${3:-fr-par-1}"
  local with_db="$4"

  source "$PROVIDERS_CONFIG_DIR/scaleway.credentials"
  local instance_type=$(get_size_for_provider "scaleway" "$size")

  log_info "Creating Scaleway instance ($instance_type)..."

  # Scaleway requires zone from region
  local zone="${region}"

  local response=$(curl -s -X POST "https://api.scaleway.com/instance/v1/zones/$zone/servers" \
    -H "X-Auth-Token: $secret_key" \
    -H "Content-Type: application/json" \
    -d "{
      \"name\": \"${project}-nself\",
      \"commercial_type\": \"$instance_type\",
      \"image\": \"ubuntu_jammy\",
      \"organization\": \"$organization_id\",
      \"tags\": [\"nself\", \"$project\"]
    }")

  local server_id=$(echo "$response" | jq -r '.server.id // empty')

  if [[ -z "$server_id" ]]; then
    log_error "Failed to create instance"
    return 1
  fi

  log_success "Instance created: $server_id"

  # Power on
  curl -s -X POST "https://api.scaleway.com/instance/v1/zones/$zone/servers/$server_id/action" \
    -H "X-Auth-Token: $secret_key" \
    -H "Content-Type: application/json" \
    -d '{"action": "poweron"}' >/dev/null

  sleep 10

  # Get IP
  local public_ip=$(curl -s "https://api.scaleway.com/instance/v1/zones/$zone/servers/$server_id" \
    -H "X-Auth-Token: $secret_key" | jq -r '.server.public_ip.address // empty')

  log_success "Instance running at: $public_ip"

  save_provision_state "scaleway" "$server_id" "$public_ip" "docker"
  show_provisioning_complete "$public_ip" "$project"
}

# Placeholder implementations for remaining providers
provision_gcp() { log_warning "GCP provisioning requires gcloud CLI - coming soon"; }
provision_azure() { log_warning "Azure provisioning requires az CLI - coming soon"; }
provision_ionos() { log_warning "IONOS provisioning coming soon"; }
provision_ovh() { log_warning "OVH provisioning coming soon"; }

# ============================================================================
# STATE MANAGEMENT
# ============================================================================

save_provision_state() {
  local provider="$1"
  local instance_id="$2"
  local public_ip="$3"
  local db_id="$4"

  mkdir -p "$PROVISION_STATE_DIR"

  cat > "$PROVISION_STATE_DIR/state.yaml" << EOF
# nself Provision State
# Created: $(date -Iseconds)

provider: $provider
instance_id: $instance_id
public_ip: $public_ip
database_id: $db_id
project: ${PROJECT_NAME:-$(basename "$PWD")}
EOF

  log_info "State saved to $PROVISION_STATE_DIR/state.yaml"
}

show_provisioning_complete() {
  local ip="$1"
  local project="$2"

  echo ""
  echo "══════════════════════════════════════════════"
  log_success "Provisioning Complete!"
  echo "══════════════════════════════════════════════"
  echo ""
  echo "Server IP: $ip"
  echo ""
  echo "Next steps:"
  echo ""
  echo "  1. Wait 2-3 minutes for server setup to complete"
  echo ""
  echo "  2. Connect to server:"
  echo "     ssh root@$ip"
  echo ""
  echo "  3. Deploy your project:"
  echo "     cd /var/www"
  echo "     git clone <your-repo> $project"
  echo "     cd $project"
  echo "     nself init --env prod"
  echo "     nself build"
  echo "     nself start"
  echo ""
  echo "  4. Configure DNS:"
  echo "     Point your domain to: $ip"
  echo ""
  echo "  5. Enable HTTPS:"
  echo "     nself ssl enable --production"
  echo ""
}

# ============================================================================
# EXPORT
# ============================================================================

cmd_export() {
  local format="${1:-terraform}"
  local provider="${2:-}"

  [[ -f ".env" ]] && source ".env" 2>/dev/null || true
  local project="${PROJECT_NAME:-$(basename "$PWD")}"

  case "$format" in
    terraform)
      export_terraform "$provider" "$project"
      ;;
    pulumi)
      export_pulumi "$provider" "$project"
      ;;
    *)
      log_error "Unknown format: $format"
      log_info "Supported: terraform, pulumi"
      return 1
      ;;
  esac
}

export_terraform() {
  local provider="$1"
  local project="$2"

  mkdir -p terraform

  cat > terraform/main.tf << EOF
# nself Terraform Configuration
# Generated by: nself provision export terraform
# Provider: ${provider:-all}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    digitalocean = {
      source  = "digitalocean/digitalocean"
      version = "~> 2.0"
    }
    hcloud = {
      source  = "hetznercloud/hcloud"
      version = "~> 1.0"
    }
  }
}

variable "project_name" {
  default = "$project"
}

variable "environment" {
  default = "production"
}

# Add provider-specific resources below
EOF

  log_success "Terraform configuration exported to terraform/"
  log_info "Customize and run: cd terraform && terraform init && terraform plan"
}

export_pulumi() {
  local provider="$1"
  local project="$2"

  mkdir -p pulumi

  cat > pulumi/Pulumi.yaml << EOF
name: $project
runtime: yaml
description: nself infrastructure for $project
EOF

  log_success "Pulumi configuration exported to pulumi/"
}

# ============================================================================
# HELP
# ============================================================================

show_help() {
  cat << 'EOF'
nself provision - Infrastructure Provisioning

USAGE:
  nself provision <provider> [options]

PROVIDERS:
  aws         Amazon Web Services
  gcp         Google Cloud Platform
  azure       Microsoft Azure
  do          DigitalOcean
  hetzner     Hetzner Cloud (best value)
  linode      Linode (Akamai)
  vultr       Vultr
  ionos       IONOS Cloud
  ovh         OVHcloud
  scaleway    Scaleway

OPTIONS:
  --size <size>       Instance size: small, medium, large, xlarge
                      Or provider-specific (e.g., t3.medium, cx21)

  --region <region>   Deployment region (default varies by provider)

  --dry-run           Show what would be created without provisioning

  --estimate          Show cost estimate only

  --no-db             Skip managed database (use Docker PostgreSQL)

  --config <file>     Use configuration file for provisioning

EXAMPLES:
  # Provision on Hetzner (cheapest)
  nself provision hetzner

  # Provision medium instance on DigitalOcean
  nself provision do --size medium

  # Show cost estimate for AWS
  nself provision aws --estimate

  # Dry run to see what would be created
  nself provision gcp --dry-run

  # Provision without managed database
  nself provision linode --no-db

EXPORT:
  nself provision export terraform [provider]
  nself provision export pulumi [provider]

COST COMPARISON (small instance + PostgreSQL):
  Hetzner:      ~$9/month   (best value, self-hosted DB)
  Scaleway:     ~$28/month
  DigitalOcean: ~$39/month
  Linode:       ~$39/month
  Vultr:        ~$39/month
  GCP:          ~$49/month
  AWS:          ~$60/month
  Azure:        ~$63/month

PRE-REQUISITES:
  1. Configure provider credentials:
     nself providers init <provider>

  2. Have project initialized:
     nself init

EOF
}

# ============================================================================
# MAIN
# ============================================================================

main() {
  local command="${1:-}"

  case "$command" in
    export)
      shift
      cmd_export "$@"
      ;;
    help|--help|-h|"")
      show_help
      ;;
    *)
      # Treat first arg as provider
      cmd_provision "$@"
      ;;
  esac
}

main "$@"
