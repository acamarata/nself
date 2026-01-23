#!/usr/bin/env bash
# k8s.sh - Kubernetes management command
# Part of nself v0.4.7 - Infrastructure Everywhere
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source shared utilities (same pattern as other CLI commands)
source "${SCRIPT_DIR}/../lib/utils/display.sh" 2>/dev/null || true
source "${SCRIPT_DIR}/../lib/utils/env.sh" 2>/dev/null || true

# Fallback logging if display.sh failed to load
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

# Source K8s library modules
K8S_LIB_DIR="${SCRIPT_DIR}/../lib/k8s"
if [[ -d "$K8S_LIB_DIR" ]]; then
  for lib_file in "$K8S_LIB_DIR"/*.sh; do
    [[ -f "$lib_file" ]] && source "$lib_file"
  done
fi

# Source provider interface for managed K8s
source "${SCRIPT_DIR}/../lib/providers/provider-interface.sh" 2>/dev/null || true

show_k8s_help() {
  cat << 'EOF'
nself k8s - Kubernetes Management

USAGE:
  nself k8s <subcommand> [options]

SUBCOMMANDS:
  init              Initialize Kubernetes configuration for project
  convert           Convert Docker Compose to Kubernetes manifests
  apply             Apply manifests to cluster
  deploy            Full deployment pipeline (convert + apply)
  status            Show deployment status
  logs              Stream pod logs
  scale             Scale deployments
  rollback          Rollback to previous version
  delete            Delete deployment from cluster

  cluster           Managed Kubernetes cluster operations
    create          Create managed K8s cluster
    delete          Delete managed K8s cluster
    list            List clusters
    kubeconfig      Get kubeconfig for cluster

  namespace         Namespace management
    create          Create namespace
    delete          Delete namespace
    list            List namespaces

EXAMPLES:
  nself k8s init                    # Initialize K8s config
  nself k8s convert                 # Generate K8s manifests from compose
  nself k8s deploy                  # Deploy to Kubernetes
  nself k8s status                  # Check deployment status
  nself k8s logs api                # Stream logs from api pods
  nself k8s scale api --replicas 3  # Scale api to 3 replicas
  nself k8s rollback api            # Rollback api deployment

  nself k8s cluster create --provider digitalocean --name prod
  nself k8s cluster kubeconfig prod > ~/.kube/config

OPTIONS:
  -n, --namespace   Kubernetes namespace (default: from config or 'default')
  -c, --context     Kubectl context to use
  -f, --file        Specific manifest file to use
  --dry-run         Show what would be applied without applying
  -h, --help        Show this help message

PREREQUISITES:
  - kubectl installed and configured
  - Valid kubeconfig (for apply/deploy/status operations)
  - Docker Compose file (for convert operations)

See 'nself k8s <subcommand> --help' for subcommand-specific options.
EOF
}

# Check kubectl availability
check_kubectl() {
  if ! command -v kubectl &>/dev/null; then
    log_error "kubectl not found. Please install kubectl first."
    log_info "Installation: https://kubernetes.io/docs/tasks/tools/"
    return 1
  fi
  return 0
}

# Check if cluster is accessible
check_cluster_connection() {
  if ! kubectl cluster-info &>/dev/null; then
    log_error "Cannot connect to Kubernetes cluster"
    log_info "Ensure kubeconfig is set up correctly"
    return 1
  fi
  return 0
}

# Get project namespace
get_namespace() {
  local namespace="${NSELF_K8S_NAMESPACE:-}"

  if [[ -z "$namespace" ]]; then
    # Try to read from config
    local config_file=".nself/k8s.yml"
    if [[ -f "$config_file" ]]; then
      namespace=$(grep "namespace:" "$config_file" | cut -d':' -f2 | tr -d ' "' || echo "")
    fi
  fi

  if [[ -z "$namespace" ]]; then
    # Use project name or default
    namespace="${PROJECT_NAME:-nself}"
  fi

  echo "$namespace"
}

# === INIT SUBCOMMAND ===
cmd_k8s_init() {
  local project_name="${PROJECT_NAME:-$(basename "$(pwd)")}"
  local namespace="${1:-$project_name}"

  log_info "Initializing Kubernetes configuration..."

  # Create K8s config directory
  mkdir -p .nself/k8s

  # Create K8s config file
  cat > .nself/k8s.yml << EOF
# nself Kubernetes Configuration
# Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")

namespace: ${namespace}
project: ${project_name}

# Deployment settings
replicas:
  default: 2
  api: 3
  worker: 2

# Resource defaults
resources:
  default:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"
  api:
    requests:
      cpu: "200m"
      memory: "256Mi"
    limits:
      cpu: "1000m"
      memory: "1Gi"

# Ingress settings
ingress:
  enabled: true
  class: nginx
  annotations:
    cert-manager.io/cluster-issuer: "letsencrypt-prod"

# Secrets (reference only, actual values in .env)
secrets:
  - database-credentials
  - api-keys
  - tls-certificates

# Storage
storage:
  postgres:
    size: "10Gi"
    class: "standard"
  minio:
    size: "50Gi"
    class: "standard"
EOF

  log_success "Kubernetes configuration initialized"
  log_info "Config file: .nself/k8s.yml"
  log_info "Next: Run 'nself k8s convert' to generate manifests"
}

# === CONVERT SUBCOMMAND ===
cmd_k8s_convert() {
  local compose_file="${1:-docker-compose.yml}"
  local output_dir="${2:-.nself/k8s/manifests}"
  local dry_run="${DRY_RUN:-false}"

  if [[ ! -f "$compose_file" ]]; then
    log_error "Docker Compose file not found: $compose_file"
    log_info "Run 'nself build' first to generate docker-compose.yml"
    return 1
  fi

  log_info "Converting Docker Compose to Kubernetes manifests..."

  # Create output directory
  mkdir -p "$output_dir"

  # Use the conversion library
  if type k8s_convert_compose &>/dev/null; then
    k8s_convert_compose "$compose_file" "$output_dir"
  else
    # Fallback: Use kompose if available
    if command -v kompose &>/dev/null; then
      log_info "Using kompose for conversion..."
      kompose convert -f "$compose_file" -o "$output_dir" --with-kompose-annotation=false
    else
      # Manual conversion
      log_info "Generating manifests manually..."
      _generate_k8s_manifests "$compose_file" "$output_dir"
    fi
  fi

  log_success "Kubernetes manifests generated in: $output_dir"

  # List generated files
  printf "\nGenerated manifests:\n"
  find "$output_dir" -name "*.yaml" -o -name "*.yml" | sort | while read -r f; do
    printf "  %s\n" "$f"
  done
}

# Manual manifest generation
_generate_k8s_manifests() {
  local compose_file="$1"
  local output_dir="$2"
  local namespace
  namespace=$(get_namespace)

  # Generate namespace
  cat > "$output_dir/00-namespace.yaml" << EOF
apiVersion: v1
kind: Namespace
metadata:
  name: ${namespace}
  labels:
    app.kubernetes.io/managed-by: nself
EOF

  # Parse compose file and generate deployments
  # This is a simplified parser - the full library handles complex cases
  local services
  services=$(grep -E "^[[:space:]]{2}[a-zA-Z0-9_-]+:$" "$compose_file" | tr -d ' :' || echo "")

  local counter=1
  for service in $services; do
    [[ -z "$service" ]] && continue
    [[ "$service" == "version" ]] && continue
    [[ "$service" == "services" ]] && continue
    [[ "$service" == "volumes" ]] && continue
    [[ "$service" == "networks" ]] && continue

    local padded_counter
    padded_counter=$(printf "%02d" $counter)

    # Generate deployment
    cat > "$output_dir/${padded_counter}-${service}-deployment.yaml" << EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ${service}
  namespace: ${namespace}
  labels:
    app: ${service}
    app.kubernetes.io/name: ${service}
    app.kubernetes.io/managed-by: nself
spec:
  replicas: 2
  selector:
    matchLabels:
      app: ${service}
  template:
    metadata:
      labels:
        app: ${service}
    spec:
      containers:
      - name: ${service}
        image: ${service}:latest
        resources:
          requests:
            cpu: "100m"
            memory: "128Mi"
          limits:
            cpu: "500m"
            memory: "512Mi"
EOF

    # Generate service
    cat > "$output_dir/${padded_counter}-${service}-service.yaml" << EOF
apiVersion: v1
kind: Service
metadata:
  name: ${service}
  namespace: ${namespace}
  labels:
    app: ${service}
spec:
  selector:
    app: ${service}
  ports:
  - port: 80
    targetPort: 8080
    protocol: TCP
EOF

    ((counter++))
  done

  log_info "Generated $((counter-1)) service manifests"
}

# === APPLY SUBCOMMAND ===
cmd_k8s_apply() {
  local manifest_dir="${1:-.nself/k8s/manifests}"
  local namespace
  namespace=$(get_namespace)
  local dry_run="${DRY_RUN:-false}"

  check_kubectl || return 1
  check_cluster_connection || return 1

  if [[ ! -d "$manifest_dir" ]]; then
    log_error "Manifest directory not found: $manifest_dir"
    log_info "Run 'nself k8s convert' first"
    return 1
  fi

  log_info "Applying Kubernetes manifests..."

  local apply_args=()
  if [[ "$dry_run" == "true" ]]; then
    apply_args+=("--dry-run=client")
    log_info "Dry-run mode - no changes will be applied"
  fi

  # Apply all manifests in order
  find "$manifest_dir" -name "*.yaml" -o -name "*.yml" | sort | while read -r manifest; do
    printf "  Applying: %s\n" "$(basename "$manifest")"
    kubectl apply -f "$manifest" "${apply_args[@]}" 2>&1 | sed 's/^/    /'
  done

  if [[ "$dry_run" != "true" ]]; then
    log_success "Manifests applied successfully"
    log_info "Check status with: nself k8s status"
  fi
}

# === DEPLOY SUBCOMMAND ===
cmd_k8s_deploy() {
  local environment="${1:-production}"

  log_info "Starting Kubernetes deployment pipeline..."

  # Step 1: Check prerequisites
  check_kubectl || return 1
  check_cluster_connection || return 1

  # Step 2: Convert if needed
  if [[ ! -d ".nself/k8s/manifests" ]] || [[ "${FORCE_CONVERT:-false}" == "true" ]]; then
    log_info "Step 1/4: Converting Docker Compose to K8s manifests..."
    cmd_k8s_convert
  else
    log_info "Step 1/4: Using existing manifests (use --force-convert to regenerate)"
  fi

  # Step 3: Validate manifests
  log_info "Step 2/4: Validating manifests..."
  if ! kubectl apply --dry-run=client -f .nself/k8s/manifests/ &>/dev/null; then
    log_error "Manifest validation failed"
    return 1
  fi
  log_success "Manifests validated"

  # Step 4: Apply manifests
  log_info "Step 3/4: Applying manifests to cluster..."
  cmd_k8s_apply

  # Step 5: Wait for rollout
  log_info "Step 4/4: Waiting for rollout to complete..."
  local namespace
  namespace=$(get_namespace)

  local deployments
  deployments=$(kubectl get deployments -n "$namespace" -o jsonpath='{.items[*].metadata.name}' 2>/dev/null || echo "")

  for deployment in $deployments; do
    printf "  Waiting for %s..." "$deployment"
    if kubectl rollout status deployment/"$deployment" -n "$namespace" --timeout=300s &>/dev/null; then
      printf " ready\n"
    else
      printf " timeout\n"
      log_warning "Deployment $deployment did not become ready in time"
    fi
  done

  log_success "Deployment complete!"
  cmd_k8s_status
}

# === STATUS SUBCOMMAND ===
cmd_k8s_status() {
  local namespace
  namespace=$(get_namespace)

  check_kubectl || return 1

  printf "\n=== Kubernetes Deployment Status ===\n"
  printf "Namespace: %s\n\n" "$namespace"

  # Check if namespace exists
  if ! kubectl get namespace "$namespace" &>/dev/null; then
    log_warning "Namespace '$namespace' does not exist"
    return 1
  fi

  # Deployments
  printf "DEPLOYMENTS:\n"
  kubectl get deployments -n "$namespace" -o wide 2>/dev/null || echo "  No deployments found"

  # Pods
  printf "\nPODS:\n"
  kubectl get pods -n "$namespace" -o wide 2>/dev/null || echo "  No pods found"

  # Services
  printf "\nSERVICES:\n"
  kubectl get services -n "$namespace" 2>/dev/null || echo "  No services found"

  # Ingress
  printf "\nINGRESS:\n"
  kubectl get ingress -n "$namespace" 2>/dev/null || echo "  No ingress found"

  # PVCs
  printf "\nPERSISTENT VOLUME CLAIMS:\n"
  kubectl get pvc -n "$namespace" 2>/dev/null || echo "  No PVCs found"

  echo ""
}

# === LOGS SUBCOMMAND ===
cmd_k8s_logs() {
  local service="$1"
  local follow="${FOLLOW:-false}"
  local tail="${TAIL:-100}"
  local namespace
  namespace=$(get_namespace)

  check_kubectl || return 1

  if [[ -z "$service" ]]; then
    log_error "Service name required"
    log_info "Usage: nself k8s logs <service> [--follow] [--tail N]"
    return 1
  fi

  local log_args=()
  log_args+=("-n" "$namespace")
  log_args+=("-l" "app=$service")
  log_args+=("--tail=$tail")

  if [[ "$follow" == "true" ]]; then
    log_args+=("-f")
  fi

  log_info "Streaming logs for $service..."
  kubectl logs "${log_args[@]}"
}

# === SCALE SUBCOMMAND ===
cmd_k8s_scale() {
  local service="$1"
  local replicas="${2:-}"
  local namespace
  namespace=$(get_namespace)

  check_kubectl || return 1

  if [[ -z "$service" ]]; then
    log_error "Service name required"
    log_info "Usage: nself k8s scale <service> --replicas N"
    return 1
  fi

  if [[ -z "$replicas" ]]; then
    log_error "Replica count required"
    log_info "Usage: nself k8s scale <service> --replicas N"
    return 1
  fi

  log_info "Scaling $service to $replicas replicas..."

  if kubectl scale deployment "$service" -n "$namespace" --replicas="$replicas"; then
    log_success "Scaled $service to $replicas replicas"
    kubectl rollout status deployment/"$service" -n "$namespace" --timeout=120s
  else
    log_error "Failed to scale $service"
    return 1
  fi
}

# === ROLLBACK SUBCOMMAND ===
cmd_k8s_rollback() {
  local service="$1"
  local revision="${2:-}"
  local namespace
  namespace=$(get_namespace)

  check_kubectl || return 1

  if [[ -z "$service" ]]; then
    log_error "Service name required"
    log_info "Usage: nself k8s rollback <service> [revision]"
    return 1
  fi

  log_info "Rolling back $service..."

  local rollback_args=()
  rollback_args+=("deployment/$service")
  rollback_args+=("-n" "$namespace")

  if [[ -n "$revision" ]]; then
    rollback_args+=("--to-revision=$revision")
  fi

  if kubectl rollout undo "${rollback_args[@]}"; then
    log_success "Rollback initiated for $service"
    kubectl rollout status deployment/"$service" -n "$namespace" --timeout=300s
  else
    log_error "Failed to rollback $service"
    return 1
  fi
}

# === DELETE SUBCOMMAND ===
cmd_k8s_delete() {
  local service="${1:-}"
  local all="${DELETE_ALL:-false}"
  local namespace
  namespace=$(get_namespace)

  check_kubectl || return 1

  if [[ "$all" == "true" ]]; then
    log_warning "This will delete ALL resources in namespace: $namespace"
    printf "Are you sure? [y/N]: "
    read -r confirm
    confirm=$(echo "$confirm" | tr '[:upper:]' '[:lower:]')

    if [[ "$confirm" != "y" && "$confirm" != "yes" ]]; then
      log_info "Aborted"
      return 0
    fi

    log_info "Deleting all resources in $namespace..."
    kubectl delete all --all -n "$namespace"
    kubectl delete pvc --all -n "$namespace" 2>/dev/null || true
    kubectl delete configmap --all -n "$namespace" 2>/dev/null || true
    kubectl delete secret --all -n "$namespace" 2>/dev/null || true

    log_success "All resources deleted"
  elif [[ -n "$service" ]]; then
    log_info "Deleting $service from $namespace..."
    kubectl delete deployment "$service" -n "$namespace" 2>/dev/null || true
    kubectl delete service "$service" -n "$namespace" 2>/dev/null || true
    kubectl delete ingress "$service" -n "$namespace" 2>/dev/null || true

    log_success "Service $service deleted"
  else
    log_error "Specify a service name or use --all to delete everything"
    return 1
  fi
}

# === CLUSTER SUBCOMMANDS ===
cmd_k8s_cluster() {
  local action="${1:-list}"
  shift || true

  case "$action" in
    create)
      cmd_k8s_cluster_create "$@"
      ;;
    delete)
      cmd_k8s_cluster_delete "$@"
      ;;
    list)
      cmd_k8s_cluster_list "$@"
      ;;
    kubeconfig)
      cmd_k8s_cluster_kubeconfig "$@"
      ;;
    *)
      log_error "Unknown cluster action: $action"
      log_info "Available: create, delete, list, kubeconfig"
      return 1
      ;;
  esac
}

cmd_k8s_cluster_create() {
  local provider="${PROVIDER:-}"
  local name="${1:-nself-cluster}"
  local region="${REGION:-}"
  local nodes="${NODES:-3}"
  local size="${SIZE:-medium}"

  if [[ -z "$provider" ]]; then
    log_error "Provider required. Use --provider <name>"
    log_info "Providers with managed K8s: aws, gcp, azure, digitalocean, linode, vultr, scaleway, exoscale"
    return 1
  fi

  # Check if provider supports K8s
  if type "provider_${provider}_k8s_supported" &>/dev/null; then
    local k8s_support
    k8s_support=$(provider_${provider}_k8s_supported)
    if [[ "$k8s_support" != "true" ]]; then
      log_error "Provider $provider does not support managed Kubernetes"
      return 1
    fi
  fi

  log_info "Creating Kubernetes cluster with $provider..."
  log_info "  Name: $name"
  log_info "  Nodes: $nodes"
  log_info "  Size: $size"

  if type "provider_${provider}_k8s_create" &>/dev/null; then
    provider_${provider}_k8s_create "$name" "$region" "$nodes" "$size"
  else
    log_error "Provider $provider K8s functions not available"
    return 1
  fi
}

cmd_k8s_cluster_delete() {
  local cluster_id="$1"
  local provider="${PROVIDER:-}"

  if [[ -z "$cluster_id" ]]; then
    log_error "Cluster ID required"
    return 1
  fi

  if [[ -z "$provider" ]]; then
    log_error "Provider required. Use --provider <name>"
    return 1
  fi

  log_warning "This will delete cluster: $cluster_id"
  printf "Are you sure? [y/N]: "
  read -r confirm
  confirm=$(echo "$confirm" | tr '[:upper:]' '[:lower:]')

  if [[ "$confirm" != "y" && "$confirm" != "yes" ]]; then
    log_info "Aborted"
    return 0
  fi

  if type "provider_${provider}_k8s_delete" &>/dev/null; then
    provider_${provider}_k8s_delete "$cluster_id"
  else
    log_error "Provider $provider K8s delete not available"
    return 1
  fi
}

cmd_k8s_cluster_list() {
  local provider="${PROVIDER:-}"

  if [[ -n "$provider" ]]; then
    log_info "Listing clusters for provider: $provider"
    if type "provider_${provider}_k8s_list" &>/dev/null; then
      provider_${provider}_k8s_list
    else
      log_warning "No K8s listing available for $provider"
    fi
  else
    # List from kubeconfig
    log_info "Available Kubernetes contexts:"
    kubectl config get-contexts
  fi
}

cmd_k8s_cluster_kubeconfig() {
  local cluster_id="$1"
  local provider="${PROVIDER:-}"

  if [[ -z "$cluster_id" ]]; then
    log_error "Cluster ID required"
    return 1
  fi

  if [[ -z "$provider" ]]; then
    log_error "Provider required. Use --provider <name>"
    return 1
  fi

  if type "provider_${provider}_k8s_kubeconfig" &>/dev/null; then
    provider_${provider}_k8s_kubeconfig "$cluster_id"
  else
    log_error "Provider $provider kubeconfig export not available"
    return 1
  fi
}

# === NAMESPACE SUBCOMMANDS ===
cmd_k8s_namespace() {
  local action="${1:-list}"
  shift || true

  check_kubectl || return 1

  case "$action" in
    create)
      local ns_name="$1"
      if [[ -z "$ns_name" ]]; then
        log_error "Namespace name required"
        return 1
      fi
      kubectl create namespace "$ns_name"
      log_success "Namespace $ns_name created"
      ;;
    delete)
      local ns_name="$1"
      if [[ -z "$ns_name" ]]; then
        log_error "Namespace name required"
        return 1
      fi
      log_warning "This will delete namespace $ns_name and ALL its resources"
      printf "Are you sure? [y/N]: "
      read -r confirm
      confirm=$(echo "$confirm" | tr '[:upper:]' '[:lower:]')
      if [[ "$confirm" == "y" || "$confirm" == "yes" ]]; then
        kubectl delete namespace "$ns_name"
        log_success "Namespace $ns_name deleted"
      else
        log_info "Aborted"
      fi
      ;;
    list)
      kubectl get namespaces
      ;;
    *)
      log_error "Unknown namespace action: $action"
      log_info "Available: create, delete, list"
      return 1
      ;;
  esac
}

# === MAIN ENTRY POINT ===
main() {
  local subcommand="${1:-}"
  shift || true

  # Parse global options
  while [[ $# -gt 0 ]]; do
    case "$1" in
      -n|--namespace)
        export NSELF_K8S_NAMESPACE="$2"
        shift 2
        ;;
      -c|--context)
        kubectl config use-context "$2" &>/dev/null
        shift 2
        ;;
      -f|--file)
        export MANIFEST_FILE="$2"
        shift 2
        ;;
      --dry-run)
        export DRY_RUN="true"
        shift
        ;;
      --follow|-F)
        export FOLLOW="true"
        shift
        ;;
      --tail)
        export TAIL="$2"
        shift 2
        ;;
      --replicas)
        export REPLICAS="$2"
        shift 2
        ;;
      --provider)
        export PROVIDER="$2"
        shift 2
        ;;
      --region)
        export REGION="$2"
        shift 2
        ;;
      --nodes)
        export NODES="$2"
        shift 2
        ;;
      --size)
        export SIZE="$2"
        shift 2
        ;;
      --all)
        export DELETE_ALL="true"
        shift
        ;;
      --force-convert)
        export FORCE_CONVERT="true"
        shift
        ;;
      -h|--help)
        show_k8s_help
        return 0
        ;;
      *)
        break
        ;;
    esac
  done

  case "$subcommand" in
    ""|help|-h|--help)
      show_k8s_help
      ;;
    init)
      cmd_k8s_init "$@"
      ;;
    convert)
      cmd_k8s_convert "$@"
      ;;
    apply)
      cmd_k8s_apply "$@"
      ;;
    deploy)
      cmd_k8s_deploy "$@"
      ;;
    status)
      cmd_k8s_status "$@"
      ;;
    logs)
      cmd_k8s_logs "$@"
      ;;
    scale)
      cmd_k8s_scale "${1:-}" "${REPLICAS:-}"
      ;;
    rollback)
      cmd_k8s_rollback "$@"
      ;;
    delete)
      cmd_k8s_delete "$@"
      ;;
    cluster)
      cmd_k8s_cluster "$@"
      ;;
    namespace|ns)
      cmd_k8s_namespace "$@"
      ;;
    *)
      log_error "Unknown subcommand: $subcommand"
      show_k8s_help
      return 1
      ;;
  esac
}

# Run if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  main "$@"
fi
