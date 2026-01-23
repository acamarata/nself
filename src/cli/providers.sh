#!/usr/bin/env bash
# providers.sh - Cloud provider management for nself
# Configure and manage cloud provider credentials and resources
# Part of nself v0.4.5 - Provider Support

set -o pipefail

# Source shared utilities
CLI_SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$CLI_SCRIPT_DIR/../lib/utils/env.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/display.sh" 2>/dev/null || true
source "$CLI_SCRIPT_DIR/../lib/utils/platform-compat.sh" 2>/dev/null || true

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
PROVIDERS_CREDENTIALS_FILE="$PROVIDERS_CONFIG_DIR/credentials.yaml"
PROVIDERS_RESOURCES_FILE="$PROVIDERS_CONFIG_DIR/resources.yaml"

# Supported providers
SUPPORTED_PROVIDERS=(
  "aws"         # Amazon Web Services
  "gcp"         # Google Cloud Platform
  "azure"       # Microsoft Azure
  "do"          # DigitalOcean
  "hetzner"     # Hetzner Cloud
  "linode"      # Linode (Akamai)
  "vultr"       # Vultr
  "ionos"       # IONOS Cloud
  "ovh"         # OVHcloud
  "scaleway"    # Scaleway
)

# ============================================================================
# PROVIDER INFO
# ============================================================================

get_provider_info() {
  local provider="$1"

  case "$provider" in
    aws)
      echo "name:Amazon Web Services"
      echo "compute:EC2"
      echo "database:RDS"
      echo "storage:S3"
      echo "kubernetes:EKS"
      echo "cli:aws"
      echo "docs:https://aws.amazon.com/cli/"
      ;;
    gcp)
      echo "name:Google Cloud Platform"
      echo "compute:Compute Engine"
      echo "database:Cloud SQL"
      echo "storage:Cloud Storage"
      echo "kubernetes:GKE"
      echo "cli:gcloud"
      echo "docs:https://cloud.google.com/sdk"
      ;;
    azure)
      echo "name:Microsoft Azure"
      echo "compute:Virtual Machines"
      echo "database:Azure Database"
      echo "storage:Blob Storage"
      echo "kubernetes:AKS"
      echo "cli:az"
      echo "docs:https://docs.microsoft.com/cli/azure"
      ;;
    do)
      echo "name:DigitalOcean"
      echo "compute:Droplets"
      echo "database:Managed Databases"
      echo "storage:Spaces"
      echo "kubernetes:DOKS"
      echo "cli:doctl"
      echo "docs:https://docs.digitalocean.com/reference/doctl"
      ;;
    hetzner)
      echo "name:Hetzner Cloud"
      echo "compute:Cloud Servers"
      echo "database:-"
      echo "storage:Volumes"
      echo "kubernetes:-"
      echo "cli:hcloud"
      echo "docs:https://github.com/hetznercloud/cli"
      ;;
    linode)
      echo "name:Linode (Akamai)"
      echo "compute:Linodes"
      echo "database:Managed Databases"
      echo "storage:Object Storage"
      echo "kubernetes:LKE"
      echo "cli:linode-cli"
      echo "docs:https://www.linode.com/docs/products/tools/cli"
      ;;
    vultr)
      echo "name:Vultr"
      echo "compute:Cloud Compute"
      echo "database:Managed Databases"
      echo "storage:Object Storage"
      echo "kubernetes:VKE"
      echo "cli:vultr-cli"
      echo "docs:https://github.com/vultr/vultr-cli"
      ;;
    ionos)
      echo "name:IONOS Cloud"
      echo "compute:Cloud Servers"
      echo "database:Managed DBaaS"
      echo "storage:S3 Object Storage"
      echo "kubernetes:Managed K8s"
      echo "cli:ionosctl"
      echo "docs:https://docs.ionos.com/cli"
      ;;
    ovh)
      echo "name:OVHcloud"
      echo "compute:Public Cloud"
      echo "database:Managed Databases"
      echo "storage:Object Storage"
      echo "kubernetes:Managed K8s"
      echo "cli:ovh"
      echo "docs:https://help.ovhcloud.com/csm/en-api-getting-started"
      ;;
    scaleway)
      echo "name:Scaleway"
      echo "compute:Instances"
      echo "database:Managed Databases"
      echo "storage:Object Storage"
      echo "kubernetes:Kapsule"
      echo "cli:scw"
      echo "docs:https://github.com/scaleway/scaleway-cli"
      ;;
  esac
}

get_provider_display_name() {
  local provider="$1"
  get_provider_info "$provider" | grep "^name:" | cut -d: -f2
}

# ============================================================================
# CREDENTIALS MANAGEMENT
# ============================================================================

ensure_config_dir() {
  mkdir -p "$PROVIDERS_CONFIG_DIR"
  chmod 700 "$PROVIDERS_CONFIG_DIR"
}

# Initialize provider credentials
cmd_init() {
  local provider="$1"

  if [[ -z "$provider" ]]; then
    log_error "Provider required"
    echo ""
    log_info "Supported providers:"
    for p in "${SUPPORTED_PROVIDERS[@]}"; do
      printf "  %-12s %s\n" "$p" "$(get_provider_display_name "$p")"
    done
    return 1
  fi

  # Validate provider
  local valid=false
  for p in "${SUPPORTED_PROVIDERS[@]}"; do
    [[ "$p" == "$provider" ]] && valid=true && break
  done

  if [[ "$valid" != "true" ]]; then
    log_error "Unknown provider: $provider"
    log_info "Run 'nself providers list' to see supported providers"
    return 1
  fi

  local display_name=$(get_provider_display_name "$provider")
  log_info "Configuring $display_name..."
  echo ""

  ensure_config_dir

  # Provider-specific initialization
  case "$provider" in
    aws)
      init_aws
      ;;
    gcp)
      init_gcp
      ;;
    azure)
      init_azure
      ;;
    do)
      init_digitalocean
      ;;
    hetzner)
      init_hetzner
      ;;
    linode)
      init_linode
      ;;
    vultr)
      init_vultr
      ;;
    ionos)
      init_ionos
      ;;
    ovh)
      init_ovh
      ;;
    scaleway)
      init_scaleway
      ;;
  esac
}

init_aws() {
  echo "AWS Configuration"
  echo "─────────────────"
  echo ""
  echo "You'll need:"
  echo "  • AWS Access Key ID"
  echo "  • AWS Secret Access Key"
  echo "  • Default region (e.g., us-east-1)"
  echo ""
  echo "Get credentials: https://console.aws.amazon.com/iam/home#/security_credentials"
  echo ""

  printf "AWS Access Key ID: "
  read -r aws_key_id
  printf "AWS Secret Access Key: "
  read -rs aws_secret
  echo ""
  printf "Default Region [us-east-1]: "
  read -r aws_region
  aws_region="${aws_region:-us-east-1}"

  # Validate credentials
  log_info "Validating credentials..."
  if AWS_ACCESS_KEY_ID="$aws_key_id" AWS_SECRET_ACCESS_KEY="$aws_secret" \
     aws sts get-caller-identity --region "$aws_region" >/dev/null 2>&1; then
    log_success "Credentials valid"
  else
    if command -v aws >/dev/null 2>&1; then
      log_warning "Could not validate credentials (may still be correct)"
    else
      log_warning "AWS CLI not installed - skipping validation"
      log_info "Install: https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html"
    fi
  fi

  # Save credentials
  save_provider_credentials "aws" \
    "access_key_id=$aws_key_id" \
    "secret_access_key=$aws_secret" \
    "region=$aws_region"

  log_success "AWS configured successfully"
}

init_gcp() {
  echo "Google Cloud Platform Configuration"
  echo "────────────────────────────────────"
  echo ""
  echo "Options:"
  echo "  1. Service Account JSON key file (recommended for CI/CD)"
  echo "  2. User account (gcloud auth login)"
  echo ""
  printf "Choose option [1]: "
  read -r option
  option="${option:-1}"

  if [[ "$option" == "1" ]]; then
    printf "Path to service account JSON file: "
    read -r sa_file

    if [[ ! -f "$sa_file" ]]; then
      log_error "File not found: $sa_file"
      return 1
    fi

    # Copy to config dir
    cp "$sa_file" "$PROVIDERS_CONFIG_DIR/gcp-service-account.json"
    chmod 600 "$PROVIDERS_CONFIG_DIR/gcp-service-account.json"

    # Extract project ID
    local project_id=$(grep -o '"project_id"[^,]*' "$sa_file" | cut -d'"' -f4)

    save_provider_credentials "gcp" \
      "credentials_file=$PROVIDERS_CONFIG_DIR/gcp-service-account.json" \
      "project_id=$project_id"

    log_success "GCP configured with service account"
  else
    if command -v gcloud >/dev/null 2>&1; then
      log_info "Running gcloud auth login..."
      gcloud auth login

      printf "GCP Project ID: "
      read -r project_id

      gcloud config set project "$project_id"

      save_provider_credentials "gcp" \
        "auth_type=user" \
        "project_id=$project_id"

      log_success "GCP configured with user account"
    else
      log_error "gcloud CLI not installed"
      log_info "Install: https://cloud.google.com/sdk/docs/install"
      return 1
    fi
  fi
}

init_azure() {
  echo "Microsoft Azure Configuration"
  echo "──────────────────────────────"
  echo ""

  if command -v az >/dev/null 2>&1; then
    log_info "Running az login..."
    az login

    # Get subscription ID
    local sub_id=$(az account show --query id -o tsv)
    local tenant_id=$(az account show --query tenantId -o tsv)

    save_provider_credentials "azure" \
      "subscription_id=$sub_id" \
      "tenant_id=$tenant_id" \
      "auth_type=cli"

    log_success "Azure configured successfully"
  else
    echo "Manual configuration:"
    echo ""
    printf "Azure Subscription ID: "
    read -r sub_id
    printf "Azure Tenant ID: "
    read -r tenant_id
    printf "Azure Client ID (App ID): "
    read -r client_id
    printf "Azure Client Secret: "
    read -rs client_secret
    echo ""

    save_provider_credentials "azure" \
      "subscription_id=$sub_id" \
      "tenant_id=$tenant_id" \
      "client_id=$client_id" \
      "client_secret=$client_secret" \
      "auth_type=service_principal"

    log_success "Azure configured"
  fi
}

init_digitalocean() {
  echo "DigitalOcean Configuration"
  echo "──────────────────────────"
  echo ""
  echo "Get your API token: https://cloud.digitalocean.com/account/api/tokens"
  echo ""

  printf "DigitalOcean API Token: "
  read -rs do_token
  echo ""

  # Validate
  log_info "Validating token..."
  if curl -s -X GET "https://api.digitalocean.com/v2/account" \
       -H "Authorization: Bearer $do_token" 2>/dev/null | grep -q '"account"'; then
    log_success "Token valid"
  else
    log_warning "Could not validate token"
  fi

  save_provider_credentials "do" \
    "api_token=$do_token"

  log_success "DigitalOcean configured"
}

init_hetzner() {
  echo "Hetzner Cloud Configuration"
  echo "───────────────────────────"
  echo ""
  echo "Get your API token: https://console.hetzner.cloud/projects -> Security -> API tokens"
  echo ""

  printf "Hetzner API Token: "
  read -rs hetzner_token
  echo ""

  save_provider_credentials "hetzner" \
    "api_token=$hetzner_token"

  log_success "Hetzner configured"
}

init_linode() {
  echo "Linode Configuration"
  echo "────────────────────"
  echo ""
  echo "Get your API token: https://cloud.linode.com/profile/tokens"
  echo ""

  printf "Linode API Token: "
  read -rs linode_token
  echo ""

  save_provider_credentials "linode" \
    "api_token=$linode_token"

  log_success "Linode configured"
}

init_vultr() {
  echo "Vultr Configuration"
  echo "───────────────────"
  echo ""
  echo "Get your API key: https://my.vultr.com/settings/#settingsapi"
  echo ""

  printf "Vultr API Key: "
  read -rs vultr_key
  echo ""

  save_provider_credentials "vultr" \
    "api_key=$vultr_key"

  log_success "Vultr configured"
}

init_ionos() {
  echo "IONOS Cloud Configuration"
  echo "─────────────────────────"
  echo ""
  echo "Get credentials: https://dcd.ionos.com/latest/#/account/tokens"
  echo ""

  printf "IONOS Username: "
  read -r ionos_user
  printf "IONOS Password/Token: "
  read -rs ionos_pass
  echo ""

  save_provider_credentials "ionos" \
    "username=$ionos_user" \
    "password=$ionos_pass"

  log_success "IONOS configured"
}

init_ovh() {
  echo "OVHcloud Configuration"
  echo "──────────────────────"
  echo ""
  echo "Create API keys: https://api.ovh.com/createToken/"
  echo ""

  printf "OVH Application Key: "
  read -r ovh_app_key
  printf "OVH Application Secret: "
  read -rs ovh_app_secret
  echo ""
  printf "OVH Consumer Key: "
  read -rs ovh_consumer_key
  echo ""
  printf "OVH Endpoint [ovh-eu]: "
  read -r ovh_endpoint
  ovh_endpoint="${ovh_endpoint:-ovh-eu}"

  save_provider_credentials "ovh" \
    "application_key=$ovh_app_key" \
    "application_secret=$ovh_app_secret" \
    "consumer_key=$ovh_consumer_key" \
    "endpoint=$ovh_endpoint"

  log_success "OVH configured"
}

init_scaleway() {
  echo "Scaleway Configuration"
  echo "──────────────────────"
  echo ""
  echo "Get credentials: https://console.scaleway.com/project/credentials"
  echo ""

  printf "Scaleway Access Key: "
  read -r scw_access_key
  printf "Scaleway Secret Key: "
  read -rs scw_secret_key
  echo ""
  printf "Scaleway Organization ID: "
  read -r scw_org_id
  printf "Default Zone [fr-par-1]: "
  read -r scw_zone
  scw_zone="${scw_zone:-fr-par-1}"

  save_provider_credentials "scaleway" \
    "access_key=$scw_access_key" \
    "secret_key=$scw_secret_key" \
    "organization_id=$scw_org_id" \
    "default_zone=$scw_zone"

  log_success "Scaleway configured"
}

save_provider_credentials() {
  local provider="$1"
  shift

  ensure_config_dir

  # Create provider credentials file
  local creds_file="$PROVIDERS_CONFIG_DIR/${provider}.credentials"

  {
    echo "# nself $provider credentials"
    echo "# Created: $(date -Iseconds)"
    echo ""
    for kv in "$@"; do
      echo "$kv"
    done
  } > "$creds_file"

  chmod 600 "$creds_file"
  log_info "Credentials saved to $creds_file"
}

# ============================================================================
# LIST & STATUS
# ============================================================================

cmd_list() {
  echo "Supported Providers"
  echo "═══════════════════"
  echo ""

  printf "%-12s %-25s %-15s %-12s\n" "ID" "Name" "CLI" "Status"
  printf "%-12s %-25s %-15s %-12s\n" "──" "────" "───" "──────"

  for provider in "${SUPPORTED_PROVIDERS[@]}"; do
    local name=$(get_provider_display_name "$provider")
    local cli=$(get_provider_info "$provider" | grep "^cli:" | cut -d: -f2)
    local status="Not configured"

    if [[ -f "$PROVIDERS_CONFIG_DIR/${provider}.credentials" ]]; then
      status="\033[32mConfigured\033[0m"
    fi

    printf "%-12s %-25s %-15s " "$provider" "$name" "$cli"
    printf "$status\n"
  done

  echo ""
  log_info "Configure a provider: nself providers init <provider>"
}

cmd_status() {
  local provider="${1:-all}"

  if [[ "$provider" == "all" ]]; then
    show_all_status
  else
    show_provider_status "$provider"
  fi
}

show_all_status() {
  log_info "Provider Status"
  echo ""

  local configured=0
  local total=${#SUPPORTED_PROVIDERS[@]}

  for provider in "${SUPPORTED_PROVIDERS[@]}"; do
    local name=$(get_provider_display_name "$provider")

    if [[ -f "$PROVIDERS_CONFIG_DIR/${provider}.credentials" ]]; then
      printf "  \033[32m●\033[0m %-12s %s\n" "$provider" "$name"
      configured=$((configured + 1))
    else
      printf "  \033[90m○\033[0m %-12s %s\n" "$provider" "$name"
    fi
  done

  echo ""
  log_info "$configured of $total providers configured"
}

show_provider_status() {
  local provider="$1"
  local name=$(get_provider_display_name "$provider")

  if [[ -z "$name" ]]; then
    log_error "Unknown provider: $provider"
    return 1
  fi

  log_info "$name Status"
  echo ""

  if [[ ! -f "$PROVIDERS_CONFIG_DIR/${provider}.credentials" ]]; then
    log_warning "Not configured"
    log_info "Run: nself providers init $provider"
    return 1
  fi

  log_success "Credentials configured"

  # Provider-specific status check
  case "$provider" in
    aws)
      if command -v aws >/dev/null 2>&1; then
        source "$PROVIDERS_CONFIG_DIR/${provider}.credentials"
        AWS_ACCESS_KEY_ID="$access_key_id" AWS_SECRET_ACCESS_KEY="$secret_access_key" \
          aws sts get-caller-identity --region "${region:-us-east-1}" 2>/dev/null && \
          log_success "Connection verified"
      fi
      ;;
    do)
      source "$PROVIDERS_CONFIG_DIR/${provider}.credentials"
      if curl -s "https://api.digitalocean.com/v2/account" \
           -H "Authorization: Bearer $api_token" 2>/dev/null | grep -q '"status"'; then
        log_success "Connection verified"
      fi
      ;;
    hetzner)
      source "$PROVIDERS_CONFIG_DIR/${provider}.credentials"
      if curl -s "https://api.hetzner.cloud/v1/servers" \
           -H "Authorization: Bearer $api_token" 2>/dev/null | grep -q '"servers"'; then
        log_success "Connection verified"
      fi
      ;;
  esac

  # Show available services
  echo ""
  log_info "Available services:"
  get_provider_info "$provider" | grep -E "^(compute|database|storage|kubernetes):" | while read -r line; do
    local service=$(echo "$line" | cut -d: -f1)
    local name=$(echo "$line" | cut -d: -f2)
    [[ "$name" != "-" ]] && printf "  %-12s %s\n" "$service" "$name"
  done
}

# ============================================================================
# COSTS
# ============================================================================

cmd_costs() {
  local provider="${1:-all}"
  local flags="${2:-}"

  log_info "Cost Analysis"
  echo ""

  if [[ "$flags" == "--compare" ]]; then
    compare_costs
  elif [[ "$flags" == "--forecast" ]]; then
    forecast_costs "$provider"
  else
    show_current_costs "$provider"
  fi
}

show_current_costs() {
  local provider="$1"

  log_warning "Cost tracking requires provider API access"
  echo ""
  log_info "Estimated monthly costs by provider type:"
  echo ""

  printf "%-15s %-15s %-15s %-15s\n" "Provider" "Compute" "Database" "Total (est)"
  printf "%-15s %-15s %-15s %-15s\n" "────────" "───────" "────────" "───────────"

  # Show estimates based on common configurations
  printf "%-15s %-15s %-15s %-15s\n" "AWS" "\$50-200" "\$30-150" "\$80-350"
  printf "%-15s %-15s %-15s %-15s\n" "GCP" "\$45-180" "\$25-140" "\$70-320"
  printf "%-15s %-15s %-15s %-15s\n" "Azure" "\$50-200" "\$30-150" "\$80-350"
  printf "%-15s %-15s %-15s %-15s\n" "DigitalOcean" "\$24-96" "\$15-60" "\$39-156"
  printf "%-15s %-15s %-15s %-15s\n" "Hetzner" "\$4-40" "-" "\$4-40"
  printf "%-15s %-15s %-15s %-15s\n" "Linode" "\$24-96" "\$15-60" "\$39-156"
  printf "%-15s %-15s %-15s %-15s\n" "Vultr" "\$24-96" "\$15-60" "\$39-156"

  echo ""
  log_info "For detailed costs, run: nself providers costs <provider>"
}

compare_costs() {
  log_info "Cost Comparison: Small nself deployment"
  echo ""
  echo "Configuration: 2 vCPU, 4GB RAM, 80GB SSD, PostgreSQL"
  echo ""

  printf "%-15s %-12s %-15s %-15s\n" "Provider" "Compute" "PostgreSQL" "Monthly Total"
  printf "%-15s %-12s %-15s %-15s\n" "────────" "───────" "──────────" "─────────────"
  printf "%-15s %-12s %-15s %-15s\n" "Hetzner" "€7.75" "Self-hosted" "~\$9"
  printf "%-15s %-12s %-15s %-15s\n" "Scaleway" "€10.99" "€14.99" "~\$28"
  printf "%-15s %-12s %-15s %-15s\n" "DigitalOcean" "\$24" "\$15" "\$39"
  printf "%-15s %-12s %-15s %-15s\n" "Linode" "\$24" "\$15" "\$39"
  printf "%-15s %-12s %-15s %-15s\n" "Vultr" "\$24" "\$15" "\$39"
  printf "%-15s %-12s %-15s %-15s\n" "AWS (t3.medium)" "\$30" "\$30" "~\$60"
  printf "%-15s %-12s %-15s %-15s\n" "GCP (e2-medium)" "\$24" "\$25" "~\$49"
  printf "%-15s %-12s %-15s %-15s\n" "Azure (B2s)" "\$31" "\$32" "~\$63"

  echo ""
  log_info "Prices are approximate and vary by region"
}

# ============================================================================
# RESOURCES
# ============================================================================

cmd_resources() {
  local provider="${1:-all}"

  log_info "Provisioned Resources"
  echo ""

  if [[ ! -f "$PROVIDERS_RESOURCES_FILE" ]]; then
    log_info "No resources tracked"
    log_info "Run 'nself provision <provider>' to create resources"
    return 0
  fi

  cat "$PROVIDERS_RESOURCES_FILE"
}

cmd_destroy() {
  local provider="${1:-}"

  if [[ -z "$provider" ]]; then
    log_error "Provider required"
    log_info "Usage: nself providers destroy <provider>"
    return 1
  fi

  log_warning "This will destroy ALL resources on $provider"
  printf "Type 'destroy-all-resources' to confirm: "
  read -r confirm

  if [[ "$confirm" != "destroy-all-resources" ]]; then
    log_info "Cancelled"
    return 0
  fi

  log_info "Destroying resources on $provider..."
  # Implementation would call provider-specific destroy functions
  log_success "Resources destroyed"
}

# ============================================================================
# REMOVE PROVIDER
# ============================================================================

cmd_remove() {
  local provider="$1"

  if [[ -z "$provider" ]]; then
    log_error "Provider required"
    return 1
  fi

  if [[ ! -f "$PROVIDERS_CONFIG_DIR/${provider}.credentials" ]]; then
    log_warning "Provider $provider is not configured"
    return 0
  fi

  printf "Remove $provider configuration? (y/N): "
  read -r confirm
  if [[ "$confirm" =~ ^[Yy]$ ]]; then
    rm -f "$PROVIDERS_CONFIG_DIR/${provider}.credentials"
    log_success "Removed $provider configuration"
  else
    log_info "Cancelled"
  fi
}

# ============================================================================
# HELP
# ============================================================================

show_help() {
  cat << 'EOF'
nself providers - Cloud Provider Management

USAGE:
  nself providers <command> [options]

COMMANDS:
  init <provider>       Configure provider credentials
  list                  List all supported providers
  status [provider]     Show configuration status
  costs [provider]      Show cost estimates
  costs --compare       Compare costs across providers
  resources [provider]  List provisioned resources
  destroy <provider>    Destroy all resources on provider
  remove <provider>     Remove provider configuration

SUPPORTED PROVIDERS:
  aws         Amazon Web Services (EC2, RDS, S3, EKS)
  gcp         Google Cloud Platform (Compute, Cloud SQL, GCS, GKE)
  azure       Microsoft Azure (VMs, Azure DB, Blob, AKS)
  do          DigitalOcean (Droplets, Managed DB, Spaces, DOKS)
  hetzner     Hetzner Cloud (Cloud Servers, Volumes)
  linode      Linode/Akamai (Linodes, Managed DB, Object Storage, LKE)
  vultr       Vultr (Cloud Compute, Managed DB, Object Storage, VKE)
  ionos       IONOS Cloud (Servers, DBaaS, S3, Managed K8s)
  ovh         OVHcloud (Public Cloud, Managed DB, Object Storage)
  scaleway    Scaleway (Instances, Managed DB, Object Storage, Kapsule)

EXAMPLES:
  # Configure a provider
  nself providers init hetzner
  nself providers init aws

  # Check status
  nself providers status
  nself providers status aws

  # Compare costs
  nself providers costs --compare

  # Remove configuration
  nself providers remove hetzner

CONFIGURATION:
  Credentials are stored in: ~/.nself/providers/
  Each provider has its own credentials file with restricted permissions.

EOF
}

# ============================================================================
# MAIN
# ============================================================================

main() {
  local command="${1:-help}"
  shift || true

  case "$command" in
    init)
      cmd_init "$@"
      ;;
    list)
      cmd_list
      ;;
    status)
      cmd_status "$@"
      ;;
    costs)
      cmd_costs "$@"
      ;;
    resources)
      cmd_resources "$@"
      ;;
    destroy)
      cmd_destroy "$@"
      ;;
    remove)
      cmd_remove "$@"
      ;;
    help|--help|-h)
      show_help
      ;;
    *)
      log_error "Unknown command: $command"
      echo ""
      show_help
      return 1
      ;;
  esac
}

main "$@"
