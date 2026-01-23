# nself v0.4.7 - Complete Development Plan

**Codename**: "Infrastructure Everywhere"
**Target Release**: Q2 2026
**Last Updated**: January 23, 2026

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Version Goals](#version-goals)
3. [Provider Expansion](#part-1-provider-expansion)
4. [Kubernetes & Helm Support](#part-2-kubernetes--helm-support)
5. [Command Organization Audit](#part-3-command-organization-audit)
6. [CLI Improvements](#part-4-cli-improvements)
7. [Sync & Deploy Enhancements](#part-5-sync--deploy-enhancements)
8. [Testing Requirements](#part-6-testing-requirements)
9. [Documentation Requirements](#part-7-documentation-requirements)
10. [Implementation Order](#part-8-implementation-order)
11. [File Changes Summary](#part-9-file-changes-summary)

---

## Executive Summary

v0.4.7 is a major infrastructure release that:

1. **Expands provider support from 10 to 26 providers**
2. **Adds complete Kubernetes and Helm support**
3. **Reorganizes commands for better discoverability**
4. **Improves sync and deploy workflows**
5. **Adds comprehensive provider-agnostic operations**

**New Commands**: 4 (`k8s`, `helm`, `infra`, `cloud`)
**Enhanced Commands**: 15+
**New Providers**: 16
**Lines of Code (Estimated)**: 15,000-20,000

---

## Version Goals

### Primary Goals
- [ ] Support all 26 cloud/VPS providers
- [ ] Complete Kubernetes manifest generation from docker-compose.yml
- [ ] Full Helm chart management
- [ ] Improved command organization (fewer top-level commands)
- [ ] Better sync/deploy workflows across all providers

### Secondary Goals
- [ ] Multi-cluster Kubernetes management
- [ ] GitOps workflow support
- [ ] Infrastructure as Code export (Terraform, Pulumi)
- [ ] Cost tracking across providers

### Non-Goals (Deferred to v0.4.8+)
- Plugin system (v0.4.8)
- nself-admin deep integration (v0.4.9)
- Interactive TUI mode (v0.4.9)

---

## Part 1: Provider Expansion

### 1.1 Current Provider Status (v0.4.5/v0.4.6)

| Provider | Status | File |
|----------|--------|------|
| AWS | ✅ Supported | `src/lib/providers/aws.sh` |
| GCP | ✅ Supported | `src/lib/providers/gcp.sh` |
| Azure | ✅ Supported | `src/lib/providers/azure.sh` |
| DigitalOcean | ✅ Supported | `src/lib/providers/digitalocean.sh` |
| Hetzner | ✅ Supported | `src/lib/providers/hetzner.sh` |
| Linode | ✅ Supported | `src/lib/providers/linode.sh` |
| Vultr | ✅ Supported | `src/lib/providers/vultr.sh` |
| IONOS | ✅ Supported | `src/lib/providers/ionos.sh` |
| OVH | ✅ Supported | `src/lib/providers/ovh.sh` |
| Scaleway | ✅ Supported | `src/lib/providers/scaleway.sh` |

### 1.2 New Providers to Add (16)

#### Priority 1: Major Cloud (2)
- [ ] **Oracle Cloud** - Best free tier in industry
- [ ] **IBM Cloud** - Enterprise/compliance focused

#### Priority 2: Developer Cloud (1)
- [ ] **UpCloud** - MaxIOPS storage performance

#### Priority 3: Budget EU (2)
- [ ] **Contabo** - Most specs per dollar
- [ ] **Netcup** - German quality budget

#### Priority 4: Budget Global (4)
- [ ] **Hostinger** - Beginner-friendly
- [ ] **Hostwinds** - Managed VPS option
- [ ] **Kamatera** - Highly configurable
- [ ] **SSD Nodes** - Nested virtualization

#### Priority 5: Regional/Specialty (4)
- [ ] **Exoscale** - Swiss privacy/GDPR
- [ ] **Alibaba Cloud** - Asia-Pacific
- [ ] **Tencent Cloud** - China market
- [ ] **Yandex Cloud** - Russian market

#### Priority 6: Extreme Budget (3)
- [ ] **RackNerd** - Cheapest option
- [ ] **BuyVM** - Unmetered bandwidth
- [ ] **Time4VPS** - Budget EU

### 1.3 Provider Module Structure

Each provider module must implement:

```bash
# src/lib/providers/<provider>.sh

# Required Functions
provider_<name>_init()           # Initialize/configure provider
provider_<name>_validate()       # Validate credentials
provider_<name>_list_regions()   # List available regions
provider_<name>_list_sizes()     # List instance sizes
provider_<name>_provision()      # Create server
provider_<name>_destroy()        # Destroy server
provider_<name>_status()         # Get server status
provider_<name>_ssh()            # SSH into server
provider_<name>_list()           # List servers

# Optional Functions
provider_<name>_estimate_cost()  # Cost estimation
provider_<name>_create_firewall() # Firewall setup
provider_<name>_create_volume()  # Volume management
provider_<name>_snapshot()       # Snapshot management
provider_<name>_k8s_create()     # Managed K8s (if supported)
```

### 1.4 Provider Implementation Tasks

#### Oracle Cloud
```
File: src/lib/providers/oracle.sh
Lines: ~400

Tasks:
[ ] OCI CLI integration
[ ] Compartment management
[ ] Always Free tier detection
[ ] ARM instance support (A1.Flex)
[ ] AMD instance support
[ ] OKE (Kubernetes) integration
[ ] Object Storage setup
[ ] VCN (Virtual Cloud Network) setup

API: OCI CLI (oci)
Auth: API key or instance principal
Regions: 40+
```

#### IBM Cloud
```
File: src/lib/providers/ibm.sh
Lines: ~350

Tasks:
[ ] IBM Cloud CLI integration
[ ] Classic vs VPC infrastructure
[ ] Bare metal support
[ ] IKS (Kubernetes) integration
[ ] Cloud Object Storage
[ ] Resource groups

API: IBM Cloud CLI (ibmcloud)
Auth: API key
Regions: 19
```

#### UpCloud
```
File: src/lib/providers/upcloud.sh
Lines: ~300

Tasks:
[ ] API integration
[ ] MaxIOPS storage tier
[ ] Simple backup setup
[ ] Firewall rules
[ ] Private networking

API: REST API
Auth: API credentials
Regions: 13
```

#### Contabo
```
File: src/lib/providers/contabo.sh
Lines: ~280

Tasks:
[ ] API integration (relatively new API)
[ ] Long provisioning time handling
[ ] Snapshot management
[ ] VNC console access

API: REST API
Auth: API credentials
Regions: 9
```

#### Netcup
```
File: src/lib/providers/netcup.sh
Lines: ~250

Tasks:
[ ] SCP (Server Control Panel) API
[ ] vServer management
[ ] Root server support
[ ] Snapshot handling

API: SCP API
Auth: API key + customer ID
Regions: 3
```

#### Hostinger
```
File: src/lib/providers/hostinger.sh
Lines: ~250

Tasks:
[ ] API integration
[ ] VPS management
[ ] Firewall configuration
[ ] Backup scheduling

API: REST API
Auth: API token
Regions: 8
```

#### Hostwinds
```
File: src/lib/providers/hostwinds.sh
Lines: ~250

Tasks:
[ ] API integration
[ ] Managed vs unmanaged VPS
[ ] Snapshot management
[ ] Load balancer support

API: REST API
Auth: API credentials
Regions: 3
```

#### Kamatera
```
File: src/lib/providers/kamatera.sh
Lines: ~300

Tasks:
[ ] API integration
[ ] Custom configuration builder
[ ] Hourly billing tracking
[ ] Firewall management
[ ] Load balancer setup

API: REST API
Auth: API credentials
Regions: 22
```

#### SSD Nodes
```
File: src/lib/providers/ssdnodes.sh
Lines: ~220

Tasks:
[ ] API integration
[ ] Nested virtualization flag
[ ] Price lock handling

API: REST API
Auth: API token
Regions: 4
```

#### Exoscale
```
File: src/lib/providers/exoscale.sh
Lines: ~300

Tasks:
[ ] exo CLI integration
[ ] SKS (Kubernetes) support
[ ] Security Groups
[ ] S3-compatible storage
[ ] Private networks

API: exo CLI
Auth: API key + secret
Regions: 4
```

#### Alibaba Cloud
```
File: src/lib/providers/alibaba.sh
Lines: ~400

Tasks:
[ ] aliyun CLI integration
[ ] Region/zone handling
[ ] ACK (Kubernetes) support
[ ] OSS (Object Storage)
[ ] Security groups
[ ] VPC setup

API: aliyun CLI
Auth: AccessKey ID + Secret
Regions: 28
```

#### Tencent Cloud
```
File: src/lib/providers/tencent.sh
Lines: ~350

Tasks:
[ ] tccli integration
[ ] CVM instance management
[ ] TKE (Kubernetes) support
[ ] COS (Object Storage)
[ ] Security groups

API: tccli
Auth: SecretId + SecretKey
Regions: 25+
```

#### Yandex Cloud
```
File: src/lib/providers/yandex.sh
Lines: ~300

Tasks:
[ ] yc CLI integration
[ ] Compute instance management
[ ] Managed Kubernetes
[ ] Object Storage
[ ] VPC setup

API: yc CLI
Auth: OAuth or service account
Regions: 3
```

#### RackNerd
```
File: src/lib/providers/racknerd.sh
Lines: ~180

Tasks:
[ ] SolusVM API integration
[ ] Basic VPS management
[ ] Reboot/rebuild support

API: SolusVM API
Auth: API key + hash
Regions: 7
```

#### BuyVM
```
File: src/lib/providers/buyvm.sh
Lines: ~180

Tasks:
[ ] Stallion control panel API
[ ] Slice management
[ ] Block storage slabs

API: Stallion API
Auth: API key
Regions: 3
```

#### Time4VPS
```
File: src/lib/providers/time4vps.sh
Lines: ~180

Tasks:
[ ] API integration
[ ] Container VPS support
[ ] Basic management

API: REST API
Auth: API token
Regions: 1
```

### 1.5 Provider Abstraction Layer

Create unified provider interface:

```
File: src/lib/providers/provider-interface.sh
Lines: ~500

Functions:
- provider_load()           # Load provider module
- provider_detect()         # Detect provider from server
- provider_normalize_size() # Convert small/medium/large
- provider_normalize_region() # Standardize region names
- provider_get_pricing()    # Get estimated pricing
- provider_supports_k8s()   # Check managed K8s support
- provider_supports_arm()   # Check ARM support
```

### 1.6 Provider Size Mapping

Standardize sizes across all providers:

```yaml
# Standard size definitions
sizes:
  tiny:
    vcpu: 1
    ram: 512MB-1GB
    disk: 10-20GB

  small:
    vcpu: 1-2
    ram: 1-2GB
    disk: 20-40GB

  medium:
    vcpu: 2-4
    ram: 4-8GB
    disk: 40-80GB

  large:
    vcpu: 4-8
    ram: 8-16GB
    disk: 80-160GB

  xlarge:
    vcpu: 8-16
    ram: 16-32GB
    disk: 160-320GB
```

---

## Part 2: Kubernetes & Helm Support

### 2.1 New Commands

#### `nself k8s` Command

```
File: src/cli/k8s.sh
Lines: ~1200

Subcommands:
  generate     Generate K8s manifests from docker-compose.yml
  apply        Apply manifests to cluster
  status       Show deployment status
  pods         List pods
  logs         View pod logs
  exec         Execute command in pod
  shell        Interactive shell in pod
  rollout      Manage rollouts
  rollback     Rollback deployment
  scale        Scale deployment
  delete       Delete resources
  context      Manage kubectl contexts
  port-forward Port forwarding
  events       Show cluster events
  describe     Describe resources
```

##### k8s generate
```bash
nself k8s generate                    # Generate to ./k8s/
nself k8s generate --output ./manifests
nself k8s generate --namespace myapp
nself k8s generate --env staging
nself k8s generate --include-secrets  # Include secrets (careful!)
nself k8s generate --kustomize        # Generate Kustomize structure
nself k8s generate --split            # Split into multiple files
```

Generated structure:
```
k8s/
├── base/
│   ├── kustomization.yaml
│   ├── namespace.yaml
│   ├── configmap.yaml
│   ├── secrets.yaml (template)
│   ├── postgres/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   ├── pvc.yaml
│   │   └── configmap.yaml
│   ├── hasura/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml
│   ├── auth/
│   │   ├── deployment.yaml
│   │   └── service.yaml
│   ├── nginx/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml
│   └── ... (other services)
├── overlays/
│   ├── local/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   ├── staging/
│   │   ├── kustomization.yaml
│   │   └── patches/
│   └── production/
│       ├── kustomization.yaml
│       └── patches/
└── scripts/
    ├── apply.sh
    └── delete.sh
```

##### k8s apply
```bash
nself k8s apply                       # Apply to current context
nself k8s apply --env staging         # Apply staging overlay
nself k8s apply --env production      # Apply production overlay
nself k8s apply --dry-run             # Preview only
nself k8s apply --context my-cluster  # Specific cluster
nself k8s apply --wait                # Wait for rollout
nself k8s apply --timeout 300         # Custom timeout
```

##### k8s status
```bash
nself k8s status                      # Overall status
nself k8s status postgres             # Specific service
nself k8s status --watch              # Continuous monitoring
nself k8s status --json               # JSON output
```

##### k8s pods
```bash
nself k8s pods                        # List all pods
nself k8s pods postgres               # Pods for service
nself k8s pods --all-namespaces       # All namespaces
nself k8s pods --wide                 # Extended info
```

##### k8s logs
```bash
nself k8s logs postgres               # Pod logs
nself k8s logs postgres -f            # Follow logs
nself k8s logs postgres --tail 100    # Last 100 lines
nself k8s logs postgres --previous    # Previous container
nself k8s logs postgres --all         # All containers
nself k8s logs postgres --since 1h    # Last hour
```

##### k8s exec
```bash
nself k8s exec postgres -- psql       # Run command
nself k8s exec postgres -it -- bash   # Interactive
```

##### k8s shell
```bash
nself k8s shell postgres              # Interactive shell
nself k8s shell postgres --container db  # Specific container
```

##### k8s rollout
```bash
nself k8s rollout status              # All rollout status
nself k8s rollout status hasura       # Specific service
nself k8s rollout restart hasura      # Restart rollout
nself k8s rollout pause hasura        # Pause rollout
nself k8s rollout resume hasura       # Resume rollout
nself k8s rollout history hasura      # Rollout history
nself k8s rollout undo hasura         # Undo last rollout
```

##### k8s rollback
```bash
nself k8s rollback hasura             # Rollback to previous
nself k8s rollback hasura --to 3      # Rollback to revision 3
nself k8s rollback --all              # Rollback all services
```

##### k8s scale
```bash
nself k8s scale hasura 3              # Scale to 3 replicas
nself k8s scale hasura --replicas 3   # Same as above
nself k8s scale hasura 0              # Scale to zero
```

##### k8s context
```bash
nself k8s context                     # Show current context
nself k8s context list                # List all contexts
nself k8s context use prod-cluster    # Switch context
nself k8s context add                 # Add new context (interactive)
nself k8s context delete old-context  # Delete context
```

##### k8s port-forward
```bash
nself k8s port-forward postgres 5432  # Forward port
nself k8s port-forward postgres 5432:5432
nself k8s port-forward --all          # Forward all services
```

##### k8s events
```bash
nself k8s events                      # Recent events
nself k8s events --watch              # Watch events
nself k8s events postgres             # Events for service
```

##### k8s describe
```bash
nself k8s describe pod postgres-xxx   # Describe pod
nself k8s describe service hasura     # Describe service
nself k8s describe deployment auth    # Describe deployment
```

#### `nself helm` Command

```
File: src/cli/helm.sh
Lines: ~800

Subcommands:
  init         Initialize Helm chart
  package      Package chart
  lint         Lint chart
  template     Render templates locally
  install      Install chart to cluster
  upgrade      Upgrade release
  rollback     Rollback release
  uninstall    Uninstall release
  list         List releases
  status       Release status
  history      Release history
  values       Show/generate values
  repo         Repository management
  push         Push chart to registry
```

##### helm init
```bash
nself helm init                       # Create chart from compose
nself helm init --from-compose        # Explicit compose conversion
nself helm init --from-k8s            # From existing K8s manifests
nself helm init --name myapp          # Custom chart name
nself helm init --output ./charts     # Custom output directory
```

Generated structure:
```
charts/
└── myapp/
    ├── Chart.yaml
    ├── values.yaml
    ├── values-staging.yaml
    ├── values-production.yaml
    ├── .helmignore
    ├── templates/
    │   ├── _helpers.tpl
    │   ├── NOTES.txt
    │   ├── namespace.yaml
    │   ├── configmap.yaml
    │   ├── secrets.yaml
    │   ├── postgres/
    │   │   ├── deployment.yaml
    │   │   ├── service.yaml
    │   │   ├── pvc.yaml
    │   │   └── configmap.yaml
    │   ├── hasura/
    │   │   ├── deployment.yaml
    │   │   ├── service.yaml
    │   │   └── ingress.yaml
    │   └── ... (other services)
    └── charts/               # Subcharts (dependencies)
```

##### helm package
```bash
nself helm package                    # Package chart
nself helm package --version 1.0.0    # Set version
nself helm package --app-version 0.4.7
nself helm package --destination ./dist
```

##### helm lint
```bash
nself helm lint                       # Lint chart
nself helm lint --strict              # Strict mode
nself helm lint --values values-prod.yaml
```

##### helm template
```bash
nself helm template                   # Render templates
nself helm template --env staging     # With staging values
nself helm template --set image.tag=v2.0
nself helm template --output-dir ./rendered
```

##### helm install
```bash
nself helm install                    # Install to cluster
nself helm install --name myrelease   # Custom release name
nself helm install --namespace myapp  # Custom namespace
nself helm install --env staging      # Use staging values
nself helm install --dry-run          # Preview only
nself helm install --wait             # Wait for ready
nself helm install --timeout 600      # Custom timeout
nself helm install --atomic           # Rollback on failure
```

##### helm upgrade
```bash
nself helm upgrade                    # Upgrade release
nself helm upgrade --env production   # With production values
nself helm upgrade --set image.tag=v2.1
nself helm upgrade --reuse-values     # Keep existing values
nself helm upgrade --reset-values     # Reset to chart defaults
nself helm upgrade --install          # Install if not exists
nself helm upgrade --atomic           # Rollback on failure
```

##### helm rollback
```bash
nself helm rollback                   # Rollback to previous
nself helm rollback 3                 # Rollback to revision 3
nself helm rollback --dry-run         # Preview
```

##### helm uninstall
```bash
nself helm uninstall                  # Uninstall release
nself helm uninstall --keep-history   # Keep history
nself helm uninstall --dry-run        # Preview
```

##### helm list
```bash
nself helm list                       # List releases
nself helm list --all                 # Include failed
nself helm list --all-namespaces      # All namespaces
nself helm list --json                # JSON output
```

##### helm status
```bash
nself helm status                     # Release status
nself helm status --json              # JSON output
```

##### helm history
```bash
nself helm history                    # Release history
nself helm history --max 10           # Limit results
```

##### helm values
```bash
nself helm values                     # Show default values
nself helm values --env staging       # Show staging values
nself helm values --all               # All computed values
nself helm values generate            # Generate values from .env
```

##### helm repo
```bash
nself helm repo add bitnami https://charts.bitnami.com/bitnami
nself helm repo list                  # List repos
nself helm repo update                # Update repos
nself helm repo remove bitnami        # Remove repo
```

##### helm push
```bash
nself helm push                       # Push to configured registry
nself helm push --registry oci://ghcr.io/myorg
nself helm push --registry https://charts.example.com
```

### 2.2 Kubernetes Manifest Generation

#### Docker Compose to Kubernetes Conversion

```
File: src/lib/k8s/compose-to-k8s.sh
Lines: ~800

Functions:
- convert_service_to_deployment()
- convert_service_to_statefulset()
- convert_ports_to_service()
- convert_volumes_to_pvc()
- convert_environment_to_configmap()
- convert_secrets_to_secret()
- convert_networks_to_networkpolicy()
- convert_healthcheck_to_probes()
- convert_deploy_to_resources()
- generate_ingress()
- generate_hpa()
```

Conversion mapping:

| Docker Compose | Kubernetes |
|----------------|------------|
| `services.<name>` | `Deployment` or `StatefulSet` |
| `services.<name>.image` | `spec.containers[].image` |
| `services.<name>.ports` | `Service` |
| `services.<name>.environment` | `ConfigMap` + `env` |
| `services.<name>.env_file` | `ConfigMap` from file |
| `services.<name>.volumes` | `PersistentVolumeClaim` |
| `services.<name>.healthcheck` | `livenessProbe` + `readinessProbe` |
| `services.<name>.deploy.replicas` | `spec.replicas` |
| `services.<name>.deploy.resources` | `spec.resources` |
| `services.<name>.depends_on` | `initContainers` or annotations |
| `networks` | `NetworkPolicy` |
| `secrets` | `Secret` |
| `configs` | `ConfigMap` |

#### StatefulSet Detection

Services that should use StatefulSet instead of Deployment:
- PostgreSQL (postgres)
- Redis (redis)
- Any service with persistent volumes
- Services requiring stable network identities

```bash
# Automatic detection based on:
# 1. Service name patterns (postgres, redis, mysql, etc.)
# 2. Presence of volumes
# 3. Explicit configuration in .nself.yml
```

#### Resource Defaults

```yaml
# Default resource requests/limits
resources:
  small:
    requests:
      cpu: "100m"
      memory: "128Mi"
    limits:
      cpu: "500m"
      memory: "512Mi"

  medium:
    requests:
      cpu: "250m"
      memory: "256Mi"
    limits:
      cpu: "1000m"
      memory: "1Gi"

  large:
    requests:
      cpu: "500m"
      memory: "512Mi"
    limits:
      cpu: "2000m"
      memory: "2Gi"
```

### 2.3 Managed Kubernetes Integration

Support for managed Kubernetes across providers:

```
File: src/lib/k8s/managed-k8s.sh
Lines: ~600

Functions:
- k8s_create_cluster()
- k8s_delete_cluster()
- k8s_list_clusters()
- k8s_get_kubeconfig()
- k8s_upgrade_cluster()
- k8s_scale_cluster()
```

Provider-specific implementations:

| Provider | Service | File |
|----------|---------|------|
| AWS | EKS | `src/lib/k8s/providers/eks.sh` |
| GCP | GKE | `src/lib/k8s/providers/gke.sh` |
| Azure | AKS | `src/lib/k8s/providers/aks.sh` |
| DigitalOcean | DOKS | `src/lib/k8s/providers/doks.sh` |
| Linode | LKE | `src/lib/k8s/providers/lke.sh` |
| Vultr | VKE | `src/lib/k8s/providers/vke.sh` |
| Scaleway | Kapsule | `src/lib/k8s/providers/kapsule.sh` |
| OVH | OVHcloud K8s | `src/lib/k8s/providers/ovhk8s.sh` |
| Oracle | OKE | `src/lib/k8s/providers/oke.sh` |
| IONOS | IONOS K8s | `src/lib/k8s/providers/ionosk8s.sh` |
| Exoscale | SKS | `src/lib/k8s/providers/sks.sh` |
| Alibaba | ACK | `src/lib/k8s/providers/ack.sh` |

### 2.4 k3s Support (Self-Hosted Kubernetes)

For providers without managed Kubernetes (Hetzner, Contabo, etc.):

```
File: src/lib/k8s/k3s.sh
Lines: ~500

Functions:
- k3s_install_server()
- k3s_install_agent()
- k3s_join_cluster()
- k3s_get_kubeconfig()
- k3s_upgrade()
- k3s_uninstall()
```

Commands:
```bash
nself k8s k3s install                 # Install k3s on current server
nself k8s k3s install --server        # Install as server
nself k8s k3s install --agent         # Install as agent
nself k8s k3s join <server-ip>        # Join existing cluster
nself k8s k3s kubeconfig              # Get kubeconfig
nself k8s k3s upgrade                 # Upgrade k3s
nself k8s k3s uninstall               # Remove k3s
```

### 2.5 Ingress Controller Support

```
File: src/lib/k8s/ingress.sh
Lines: ~300

Supported ingress controllers:
- nginx-ingress (default)
- traefik
- kong
- contour
- istio-gateway
```

Commands:
```bash
nself k8s ingress install nginx       # Install nginx ingress
nself k8s ingress install traefik     # Install traefik
nself k8s ingress status              # Ingress status
nself k8s ingress list                # List ingress rules
```

### 2.6 Service Mesh (Optional)

```
File: src/lib/k8s/service-mesh.sh
Lines: ~400

Supported:
- Istio
- Linkerd
```

Commands:
```bash
nself k8s mesh install istio          # Install Istio
nself k8s mesh install linkerd        # Install Linkerd
nself k8s mesh status                 # Mesh status
nself k8s mesh dashboard              # Open dashboard
nself k8s mesh inject <service>       # Inject sidecar
```

---

## Part 3: Command Organization Audit

### 3.1 Current Command Structure Analysis

Current top-level commands (v0.4.6): **43 commands**

```
Core (8): init, build, start, stop, restart, reset, clean, version
Status (6): status, logs, exec, urls, doctor, help
Management (4): update, ssl, trust, admin
Services (6): email, search, functions, mlflow, metrics, monitor
Deployment (6): env, deploy, prod, staging, sync, ci
Database (1): db (with 10+ subcommands)
Provider (2): providers, provision
Performance (4): perf, bench, scale, migrate
Operations (5): health, frontend, history, config, servers
New in 0.4.7 (2): k8s, helm
```

### 3.2 Proposed Command Reorganization

**Goal**: Reduce cognitive load by grouping related commands

#### Option A: Consolidate into Parent Commands (Recommended)

**New Structure**:

```
nself
├── Core Lifecycle (unchanged - frequently used)
│   ├── init
│   ├── build
│   ├── start
│   ├── stop
│   ├── restart
│   ├── reset
│   └── clean
│
├── Status & Debugging (unchanged - frequently used)
│   ├── status
│   ├── logs
│   ├── exec
│   ├── urls
│   └── doctor
│
├── db <subcommand>              # Already consolidated
│   ├── migrate up|down|create|status|fresh|repair
│   ├── seed [name]
│   ├── mock [--auto|--seed]
│   ├── backup [list]
│   ├── restore <file>
│   ├── shell [--readonly]
│   ├── query <sql>
│   ├── types [go|python]
│   ├── schema scaffold|import|diagram|apply
│   ├── inspect [size|slow|indexes]
│   └── data export|import|anonymize
│
├── deploy <subcommand>          # CONSOLIDATE deployment commands
│   ├── staging                  # Was: nself staging
│   ├── prod                     # Was: nself prod
│   ├── status
│   ├── rollback
│   ├── logs
│   ├── check
│   └── webhook
│
├── env <subcommand>             # Already consolidated
│   ├── list
│   ├── create <name>
│   ├── switch <name>
│   └── diff <env1> <env2>
│
├── sync <subcommand>            # Already consolidated
│   ├── pull <env>
│   ├── push <env>
│   ├── db <src> <target>
│   ├── files <src> <target>
│   └── config <src> <target>
│
├── cloud <subcommand>           # NEW: Consolidate provider operations
│   ├── providers [init|status|list|remove]  # Was: nself providers
│   ├── provision <provider>     # Was: nself provision
│   ├── servers [add|remove|status|ssh|logs|reboot]  # Was: nself servers
│   └── costs [compare|estimate]
│
├── k8s <subcommand>             # NEW
│   ├── generate
│   ├── apply
│   ├── status
│   ├── pods
│   ├── logs
│   ├── exec
│   ├── shell
│   ├── rollout
│   ├── rollback
│   ├── scale
│   ├── context
│   ├── events
│   ├── describe
│   ├── port-forward
│   └── k3s [install|join|kubeconfig]
│
├── helm <subcommand>            # NEW
│   ├── init
│   ├── package
│   ├── lint
│   ├── template
│   ├── install
│   ├── upgrade
│   ├── rollback
│   ├── uninstall
│   ├── list
│   ├── status
│   ├── history
│   ├── values
│   ├── repo
│   └── push
│
├── service <subcommand>         # NEW: Consolidate service commands
│   ├── email [status|enable|test|providers]  # Was: nself email
│   ├── search [status|enable|index|query]    # Was: nself search
│   ├── functions [status|init|create|test]   # Was: nself functions
│   ├── mlflow [status|enable|runs|models]    # Was: nself mlflow
│   └── admin [status|enable|open|password]   # Was: nself admin
│
├── monitor <subcommand>         # Consolidate monitoring
│   ├── status                   # Was: nself metrics status
│   ├── enable                   # Was: nself metrics enable
│   ├── profile [minimal|standard|full]  # Was: nself metrics profile
│   ├── open [grafana|prometheus|alertmanager]  # Was: nself monitor
│   └── alerts [list|add|remove|test]
│
├── perf <subcommand>            # Already has subcommands
│   ├── profile [service]
│   ├── analyze
│   ├── slow-queries
│   ├── report
│   ├── dashboard
│   └── suggest
│
├── bench <subcommand>           # Already has subcommands
│   ├── run [target]
│   ├── baseline
│   ├── compare
│   ├── stress
│   └── report
│
├── config <subcommand>          # Already has subcommands
│   ├── show
│   ├── get <key>
│   ├── set <key> <value>
│   ├── list
│   ├── edit
│   ├── validate
│   ├── diff <env>
│   ├── export
│   ├── import
│   └── reset
│
├── ci <subcommand>              # Already has subcommands
│   ├── init <platform>
│   ├── validate
│   └── status
│
├── Utility (unchanged)
│   ├── update
│   ├── ssl
│   ├── trust
│   ├── completion
│   ├── version
│   └── help
│
└── Legacy (aliases, show deprecation warning)
    ├── staging    → nself deploy staging
    ├── prod       → nself deploy prod
    ├── providers  → nself cloud providers
    ├── provision  → nself cloud provision
    ├── servers    → nself cloud servers
    ├── email      → nself service email
    ├── search     → nself service search
    ├── functions  → nself service functions
    ├── mlflow     → nself service mlflow
    ├── admin      → nself service admin
    ├── metrics    → nself monitor
    └── monitor    → nself monitor open
```

### 3.3 Command Count Comparison

| Version | Top-Level Commands | With Subcommands | Total Operations |
|---------|-------------------|------------------|------------------|
| v0.4.6 | 43 | 43 | ~120 |
| v0.4.7 (proposed) | 28 | 28 | ~180 |

**Reduction**: 43 → 28 top-level commands (35% reduction)
**Capability increase**: 120 → 180 operations (50% increase)

### 3.4 Implementation Tasks for Reorganization

#### New Parent Commands

1. **`nself cloud`** - Consolidate cloud infrastructure
   ```
   File: src/cli/cloud.sh
   Lines: ~600

   Incorporates:
   - providers.sh functionality
   - provision.sh functionality
   - servers.sh functionality
   - New: costs subcommand
   ```

2. **`nself service`** - Consolidate optional services
   ```
   File: src/cli/service.sh
   Lines: ~300 (dispatches to existing modules)

   Incorporates:
   - email.sh as subcommand
   - search.sh as subcommand
   - functions.sh as subcommand
   - mlflow.sh as subcommand
   - admin.sh as subcommand
   ```

3. **`nself monitor`** - Consolidate monitoring
   ```
   File: src/cli/monitor-new.sh (rename from monitor.sh)
   Lines: ~400

   Incorporates:
   - metrics.sh functionality
   - monitor.sh functionality
   - alerts management
   ```

#### Legacy Aliases

```
File: src/lib/utils/legacy-aliases.sh
Lines: ~100

Function: handle_legacy_command()

Shows deprecation warning and forwards to new command:
"Warning: 'nself staging' is deprecated. Use 'nself deploy staging' instead."
"This alias will be removed in v0.5.0"
```

### 3.5 Help System Updates

Update help.sh to show new command structure:

```bash
nself help                    # Show all commands (grouped)
nself help cloud              # Show cloud subcommands
nself help cloud provision    # Show provision usage
nself cloud --help            # Same as above
nself cloud provision --help  # Detailed help
```

---

## Part 4: CLI Improvements

### 4.1 Output Improvements

#### JSON Output Everywhere

Every command with meaningful output should support `--json`:

```
Files to update:
- src/cli/status.sh    ✅ Already has --json
- src/cli/urls.sh      ✅ Already has --json
- src/cli/version.sh   ✅ Already has --json
- src/cli/doctor.sh    [ ] Add --json
- src/cli/logs.sh      [ ] Add --json (structured logs)
- src/cli/perf.sh      [ ] Add --json
- src/cli/bench.sh     [ ] Add --json
- src/cli/health.sh    [ ] Add --json
- src/cli/history.sh   [ ] Add --json
- src/cli/config.sh    [ ] Add --json
- src/cli/servers.sh   [ ] Add --json
- src/cli/frontend.sh  [ ] Add --json
- src/cli/k8s.sh       [ ] Add --json (new)
- src/cli/helm.sh      [ ] Add --json (new)
```

#### Watch Mode

Commands that benefit from continuous monitoring:

```bash
nself status --watch              ✅ Exists
nself logs -f                     ✅ Exists
nself perf dashboard              ✅ Exists
nself health watch                ✅ Exists
nself k8s status --watch          [ ] Add
nself k8s pods --watch            [ ] Add
nself k8s events --watch          [ ] Add
nself bench stress --live         [ ] Add
```

#### Progress Indicators

Improve progress feedback for long operations:

```
File: src/lib/utils/progress.sh
Lines: ~150

Functions:
- show_spinner()              # For indeterminate progress
- show_progress_bar()         # For determinate progress
- show_step_progress()        # For multi-step operations
```

### 4.2 Error Handling Improvements

#### Actionable Error Messages

Every error should suggest a fix:

```bash
# Before
Error: Docker is not running

# After
Error: Docker is not running

  To fix this:
  1. Start Docker Desktop (macOS/Windows)
     Or: sudo systemctl start docker (Linux)

  2. Run 'nself doctor' to verify setup
```

```
File: src/lib/utils/errors.sh
Lines: ~300

Functions:
- error_with_suggestion()
- suggest_fix_for_error()
- show_common_fixes()
```

#### Error Codes

Standardize exit codes:

```
0   - Success
1   - General error
2   - Invalid arguments
3   - Configuration error
4   - Docker error
5   - Database error
6   - Network error
7   - Permission error
8   - Provider API error
9   - Kubernetes error
10  - Timeout error
126 - Permission denied (standard)
127 - Command not found (standard)
```

### 4.3 Configuration Improvements

#### nself.yml Configuration File

New configuration file for project-level settings:

```yaml
# .nself.yml (or nself.yml)
version: "1"

project:
  name: myapp

environments:
  default: local

kubernetes:
  generate:
    namespace: myapp
    ingress: nginx
    replicas:
      default: 1
      hasura: 2
    resources:
      postgres: large
      hasura: medium

helm:
  chart:
    name: myapp
    version: "1.0.0"

providers:
  default: hetzner
  staging: digitalocean
  production: aws

sync:
  exclude:
    - "*.log"
    - "node_modules"
    - ".git"
```

```
File: src/lib/config/nself-yml.sh
Lines: ~200

Functions:
- load_nself_config()
- get_nself_setting()
- validate_nself_config()
```

### 4.4 Completion Improvements

Enhance shell completions:

```
Files:
- src/completions/nself.bash (~500 lines)
- src/completions/_nself (zsh, ~500 lines)
- src/completions/nself.fish (~400 lines)

New completions for:
- k8s subcommands and flags
- helm subcommands and flags
- All new providers
- Dynamic completion for server names
- Dynamic completion for environment names
- Dynamic completion for service names
```

### 4.5 Verbosity Levels

Standardize verbosity across all commands:

```bash
nself <command>                   # Normal output
nself <command> --quiet           # Minimal output (errors only)
nself <command> --verbose         # Detailed output
nself <command> --debug           # Debug output (very detailed)

# Environment variable
NSELF_LOG_LEVEL=debug nself <command>
```

```
File: src/lib/utils/verbosity.sh
Lines: ~100

Functions:
- set_verbosity_level()
- log_debug()
- log_verbose()
- log_normal()
- is_quiet()
- is_verbose()
- is_debug()
```

---

## Part 5: Sync & Deploy Enhancements

### 5.1 Enhanced Sync Command

```
File: src/cli/sync.sh (update)
Lines: +400

New subcommands:
- sync auto           # Automated sync based on config
- sync watch          # Watch for changes and sync
- sync status         # Show sync status
- sync history        # Show sync history
- sync rollback       # Rollback last sync
```

#### Auto Sync

```bash
nself sync auto                   # Sync based on .nself.yml config
nself sync auto --dry-run         # Preview sync operations
nself sync auto --schedule "0 * * * *"  # Schedule hourly sync
```

Configuration in .nself.yml:

```yaml
sync:
  auto:
    db:
      source: staging
      target: local
      schedule: "0 2 * * *"  # Daily at 2 AM
      anonymize: true
    config:
      source: staging
      target: local
      schedule: on-change
    files:
      source: staging
      target: local
      paths:
        - uploads/
        - media/
      exclude:
        - "*.tmp"
```

#### Watch Sync

```bash
nself sync watch local staging    # Watch local, push to staging
nself sync watch --files          # Only watch files
nself sync watch --db             # Only watch DB changes
```

### 5.2 Enhanced Deploy Command

```
File: src/cli/deploy.sh (update)
Lines: +300

Enhanced subcommands:
- deploy preview      # Preview deployment changes
- deploy promote      # Promote from staging to prod
- deploy canary       # Canary deployment
- deploy blue-green   # Blue-green deployment
```

#### Preview

```bash
nself deploy preview staging      # Show what will change
nself deploy preview prod --diff  # Show config diff
```

#### Promote

```bash
nself deploy promote              # Promote staging → prod
nself deploy promote --skip-tests # Skip test verification
```

#### Canary Deployment

```bash
nself deploy canary prod          # Deploy to 10% of traffic
nself deploy canary prod --percent 25
nself deploy canary promote       # Promote canary to 100%
nself deploy canary rollback      # Rollback canary
```

#### Blue-Green Deployment

```bash
nself deploy blue-green prod      # Deploy to inactive environment
nself deploy blue-green switch    # Switch traffic
nself deploy blue-green rollback  # Switch back
```

### 5.3 Multi-Target Deployment

Deploy to multiple targets simultaneously:

```bash
nself deploy --targets staging,prod  # Deploy to both
nself deploy --all-envs              # Deploy to all configured envs
```

### 5.4 Deployment Hooks

```yaml
# .nself.yml
deploy:
  hooks:
    pre-deploy:
      - nself db backup
      - npm run test
    post-deploy:
      - nself health check
      - curl -X POST https://slack.webhook.url
    on-rollback:
      - nself db restore latest
```

---

## Part 6: Testing Requirements

### 6.1 Unit Tests

Required tests for new functionality:

```
tests/
├── unit/
│   ├── providers/
│   │   ├── test-oracle.sh
│   │   ├── test-ibm.sh
│   │   ├── test-upcloud.sh
│   │   ├── test-contabo.sh
│   │   ├── test-netcup.sh
│   │   ├── test-hostinger.sh
│   │   ├── test-hostwinds.sh
│   │   ├── test-kamatera.sh
│   │   ├── test-ssdnodes.sh
│   │   ├── test-exoscale.sh
│   │   ├── test-alibaba.sh
│   │   ├── test-tencent.sh
│   │   ├── test-yandex.sh
│   │   ├── test-racknerd.sh
│   │   ├── test-buyvm.sh
│   │   └── test-time4vps.sh
│   ├── k8s/
│   │   ├── test-k8s-generate.sh
│   │   ├── test-k8s-apply.sh
│   │   ├── test-k8s-conversion.sh
│   │   └── test-k8s-helpers.sh
│   ├── helm/
│   │   ├── test-helm-init.sh
│   │   ├── test-helm-package.sh
│   │   └── test-helm-template.sh
│   └── commands/
│       ├── test-cloud-command.sh
│       ├── test-service-command.sh
│       └── test-monitor-command.sh
```

### 6.2 Integration Tests

```
tests/
├── integration/
│   ├── test-k8s-full-workflow.sh     # Generate → Apply → Verify
│   ├── test-helm-full-workflow.sh    # Init → Package → Install
│   ├── test-provider-provision.sh    # Provision → Deploy → Destroy
│   ├── test-sync-full-workflow.sh    # Full sync testing
│   └── test-deploy-strategies.sh     # Canary, blue-green
```

### 6.3 E2E Tests

```
tests/
├── e2e/
│   ├── test-fresh-install.sh         # Complete fresh install
│   ├── test-k8s-deployment.sh        # Full K8s deployment
│   └── test-multi-provider.sh        # Deploy across providers
```

### 6.4 Test Coverage Targets

| Category | Current | Target |
|----------|---------|--------|
| Unit Tests | ~70% | 85% |
| Integration | ~50% | 75% |
| E2E | ~30% | 60% |

---

## Part 7: Documentation Requirements

### 7.1 New Documentation Files

```
docs/
├── commands/
│   ├── K8S.md                    # ~500 lines
│   ├── HELM.md                   # ~400 lines
│   ├── CLOUD.md                  # ~300 lines
│   └── SERVICE.md                # ~200 lines
├── providers/
│   ├── PROVIDERS-COMPLETE.md     ✅ Created
│   ├── ORACLE.md                 # ~150 lines
│   ├── IBM.md                    # ~150 lines
│   ├── CONTABO.md                # ~100 lines
│   ├── NETCUP.md                 # ~100 lines
│   └── ... (12 more)
├── guides/
│   ├── KUBERNETES-GUIDE.md       # ~800 lines
│   ├── HELM-GUIDE.md             # ~500 lines
│   ├── MULTI-CLOUD-GUIDE.md      # ~400 lines
│   └── MIGRATION-K8S.md          # ~300 lines
└── releases/
    └── v0.4.7.md                 # Release notes
```

### 7.2 Documentation Updates

```
Updates required:
- docs/commands/COMMANDS.md       # Complete rewrite with new structure
- docs/Home.md                    # Update for v0.4.7
- docs/README.md                  # Update for v0.4.7
- docs/_Sidebar.md                # Add new sections
- docs/guides/Quick-Start.md      # Add K8s quick start
- docs/releases/ROADMAP.md        # Update roadmap
```

---

## Part 8: Implementation Order

### Phase 1: Foundation (Week 1-2)

```
Priority 1 - Core Infrastructure

[ ] 1.1 Create provider abstraction layer
    File: src/lib/providers/provider-interface.sh

[ ] 1.2 Create k8s conversion library
    File: src/lib/k8s/compose-to-k8s.sh

[ ] 1.3 Create command reorganization framework
    File: src/lib/utils/legacy-aliases.sh

[ ] 1.4 Update error handling
    File: src/lib/utils/errors.sh

[ ] 1.5 Add nself.yml support
    File: src/lib/config/nself-yml.sh
```

### Phase 2: Provider Expansion (Week 2-4)

```
Priority 2 - Major Cloud Providers

[ ] 2.1 Oracle Cloud provider
    File: src/lib/providers/oracle.sh
    Test: tests/unit/providers/test-oracle.sh

[ ] 2.2 IBM Cloud provider
    File: src/lib/providers/ibm.sh
    Test: tests/unit/providers/test-ibm.sh

Priority 3 - Budget Providers (High Value)

[ ] 2.3 Contabo provider
    File: src/lib/providers/contabo.sh

[ ] 2.4 Netcup provider
    File: src/lib/providers/netcup.sh

[ ] 2.5 UpCloud provider
    File: src/lib/providers/upcloud.sh

Priority 4 - Additional Budget Providers

[ ] 2.6 Hostinger provider
[ ] 2.7 Hostwinds provider
[ ] 2.8 Kamatera provider
[ ] 2.9 SSD Nodes provider

Priority 5 - Regional Providers

[ ] 2.10 Exoscale provider
[ ] 2.11 Alibaba Cloud provider
[ ] 2.12 Tencent Cloud provider
[ ] 2.13 Yandex Cloud provider

Priority 6 - Extreme Budget

[ ] 2.14 RackNerd provider
[ ] 2.15 BuyVM provider
[ ] 2.16 Time4VPS provider
```

### Phase 3: Kubernetes Support (Week 4-6)

```
Priority 7 - Core K8s

[ ] 3.1 k8s command main file
    File: src/cli/k8s.sh

[ ] 3.2 k8s generate subcommand
    File: src/lib/k8s/generate.sh

[ ] 3.3 k8s apply subcommand
    File: src/lib/k8s/apply.sh

[ ] 3.4 k8s status/pods/logs subcommands
    File: src/lib/k8s/status.sh

[ ] 3.5 k8s exec/shell subcommands
    File: src/lib/k8s/exec.sh

[ ] 3.6 k8s rollout/rollback subcommands
    File: src/lib/k8s/rollout.sh

[ ] 3.7 k8s context management
    File: src/lib/k8s/context.sh

[ ] 3.8 k8s scale subcommand
    File: src/lib/k8s/scale.sh

Priority 8 - Managed K8s

[ ] 3.9 EKS integration
[ ] 3.10 GKE integration
[ ] 3.11 AKS integration
[ ] 3.12 DOKS integration
[ ] 3.13 LKE integration
[ ] 3.14 VKE integration
[ ] 3.15 Other managed K8s providers

Priority 9 - k3s Support

[ ] 3.16 k3s installation
[ ] 3.17 k3s cluster management
```

### Phase 4: Helm Support (Week 6-7)

```
Priority 10 - Helm Commands

[ ] 4.1 helm command main file
    File: src/cli/helm.sh

[ ] 4.2 helm init subcommand
    File: src/lib/helm/init.sh

[ ] 4.3 helm package subcommand
    File: src/lib/helm/package.sh

[ ] 4.4 helm template subcommand
    File: src/lib/helm/template.sh

[ ] 4.5 helm install/upgrade subcommands
    File: src/lib/helm/install.sh

[ ] 4.6 helm rollback subcommand
    File: src/lib/helm/rollback.sh

[ ] 4.7 helm repo management
    File: src/lib/helm/repo.sh
```

### Phase 5: Command Reorganization (Week 7-8)

```
Priority 11 - New Parent Commands

[ ] 5.1 Create cloud command
    File: src/cli/cloud.sh

[ ] 5.2 Create service command
    File: src/cli/service.sh

[ ] 5.3 Update monitor command
    File: src/cli/monitor-new.sh

[ ] 5.4 Legacy alias handling
    Updates to: src/cli/nself.sh

Priority 12 - Help Updates

[ ] 5.5 Update help.sh for new structure
[ ] 5.6 Update completions
```

### Phase 6: Sync & Deploy Enhancements (Week 8-9)

```
Priority 13 - Sync Improvements

[ ] 6.1 Add sync auto
[ ] 6.2 Add sync watch
[ ] 6.3 Add sync status/history

Priority 14 - Deploy Improvements

[ ] 6.4 Add deploy preview
[ ] 6.5 Add deploy promote
[ ] 6.6 Add canary deployment
[ ] 6.7 Add blue-green deployment
```

### Phase 7: Testing & Documentation (Week 9-10)

```
Priority 15 - Testing

[ ] 7.1 Write all unit tests
[ ] 7.2 Write integration tests
[ ] 7.3 Write E2E tests
[ ] 7.4 Run full test suite

Priority 16 - Documentation

[ ] 7.5 Write K8S.md
[ ] 7.6 Write HELM.md
[ ] 7.7 Write CLOUD.md
[ ] 7.8 Write provider docs
[ ] 7.9 Write guides
[ ] 7.10 Update existing docs
[ ] 7.11 Write release notes
```

### Phase 8: QA & Release (Week 10)

```
Priority 17 - Final QA

[ ] 8.1 Cross-platform testing
[ ] 8.2 Bash 3.2 compatibility check
[ ] 8.3 Performance testing
[ ] 8.4 Security review
[ ] 8.5 Documentation review

Priority 18 - Release

[ ] 8.6 Version bump
[ ] 8.7 Create tag
[ ] 8.8 GitHub release
[ ] 8.9 Update package managers
[ ] 8.10 Announce release
```

---

## Part 9: File Changes Summary

### New Files (Estimated: 50+ files, 15,000+ lines)

```
src/cli/
├── k8s.sh                        # ~1200 lines
├── helm.sh                       # ~800 lines
├── cloud.sh                      # ~600 lines
└── service.sh                    # ~300 lines

src/lib/providers/
├── provider-interface.sh         # ~500 lines
├── oracle.sh                     # ~400 lines
├── ibm.sh                        # ~350 lines
├── upcloud.sh                    # ~300 lines
├── contabo.sh                    # ~280 lines
├── netcup.sh                     # ~250 lines
├── hostinger.sh                  # ~250 lines
├── hostwinds.sh                  # ~250 lines
├── kamatera.sh                   # ~300 lines
├── ssdnodes.sh                   # ~220 lines
├── exoscale.sh                   # ~300 lines
├── alibaba.sh                    # ~400 lines
├── tencent.sh                    # ~350 lines
├── yandex.sh                     # ~300 lines
├── racknerd.sh                   # ~180 lines
├── buyvm.sh                      # ~180 lines
└── time4vps.sh                   # ~180 lines

src/lib/k8s/
├── compose-to-k8s.sh             # ~800 lines
├── generate.sh                   # ~400 lines
├── apply.sh                      # ~300 lines
├── status.sh                     # ~250 lines
├── exec.sh                       # ~200 lines
├── rollout.sh                    # ~300 lines
├── context.sh                    # ~200 lines
├── scale.sh                      # ~150 lines
├── managed-k8s.sh                # ~600 lines
├── k3s.sh                        # ~500 lines
├── ingress.sh                    # ~300 lines
├── service-mesh.sh               # ~400 lines
└── providers/
    ├── eks.sh                    # ~250 lines
    ├── gke.sh                    # ~250 lines
    ├── aks.sh                    # ~250 lines
    ├── doks.sh                   # ~200 lines
    ├── lke.sh                    # ~200 lines
    ├── vke.sh                    # ~200 lines
    ├── kapsule.sh                # ~200 lines
    ├── ovhk8s.sh                 # ~200 lines
    ├── oke.sh                    # ~200 lines
    ├── ionosk8s.sh               # ~200 lines
    ├── sks.sh                    # ~200 lines
    └── ack.sh                    # ~200 lines

src/lib/helm/
├── init.sh                       # ~400 lines
├── package.sh                    # ~200 lines
├── template.sh                   # ~200 lines
├── install.sh                    # ~300 lines
├── rollback.sh                   # ~150 lines
└── repo.sh                       # ~200 lines

src/lib/utils/
├── errors.sh                     # ~300 lines
├── progress.sh                   # ~150 lines
├── verbosity.sh                  # ~100 lines
└── legacy-aliases.sh             # ~100 lines

src/lib/config/
└── nself-yml.sh                  # ~200 lines

src/completions/
├── nself.bash                    # Update (+200 lines)
├── _nself                        # Update (+200 lines)
└── nself.fish                    # Update (+150 lines)

docs/
├── commands/K8S.md               # ~500 lines
├── commands/HELM.md              # ~400 lines
├── commands/CLOUD.md             # ~300 lines
├── commands/SERVICE.md           # ~200 lines
├── providers/PROVIDERS-COMPLETE.md  ✅ Created
├── providers/ORACLE.md           # ~150 lines
├── providers/IBM.md              # ~150 lines
├── providers/CONTABO.md          # ~100 lines
├── providers/NETCUP.md           # ~100 lines
├── providers/UPCLOUD.md          # ~100 lines
├── providers/HOSTINGER.md        # ~100 lines
├── providers/HOSTWINDS.md        # ~100 lines
├── providers/KAMATERA.md         # ~100 lines
├── providers/SSDNODES.md         # ~100 lines
├── providers/EXOSCALE.md         # ~100 lines
├── providers/ALIBABA.md          # ~150 lines
├── providers/TENCENT.md          # ~150 lines
├── providers/YANDEX.md           # ~100 lines
├── providers/RACKNERD.md         # ~80 lines
├── providers/BUYVM.md            # ~80 lines
├── providers/TIME4VPS.md         # ~80 lines
├── guides/KUBERNETES-GUIDE.md    # ~800 lines
├── guides/HELM-GUIDE.md          # ~500 lines
├── guides/MULTI-CLOUD-GUIDE.md   # ~400 lines
├── guides/MIGRATION-K8S.md       # ~300 lines
└── releases/v0.4.7.md            # ~500 lines

tests/
├── unit/providers/               # ~16 files, ~2000 lines
├── unit/k8s/                     # ~4 files, ~800 lines
├── unit/helm/                    # ~3 files, ~500 lines
├── unit/commands/                # ~3 files, ~400 lines
├── integration/                  # ~5 files, ~1000 lines
└── e2e/                          # ~3 files, ~600 lines
```

### Modified Files

```
src/cli/
├── nself.sh                      # Add legacy alias handling
├── help.sh                       # New command structure
├── sync.sh                       # Add auto/watch/status
├── deploy.sh                     # Add preview/promote/canary
├── doctor.sh                     # Add --json
├── perf.sh                       # Add --json
├── bench.sh                      # Add --json
├── health.sh                     # Add --json
├── history.sh                    # Add --json
├── config.sh                     # Add --json
├── servers.sh                    # Add --json
└── frontend.sh                   # Add --json

src/lib/providers/
├── aws.sh                        # Standardize interface
├── gcp.sh                        # Standardize interface
├── azure.sh                      # Standardize interface
├── digitalocean.sh               # Standardize interface
├── hetzner.sh                    # Standardize interface
├── linode.sh                     # Standardize interface
├── vultr.sh                      # Standardize interface
├── ionos.sh                      # Standardize interface
├── ovh.sh                        # Standardize interface
└── scaleway.sh                   # Standardize interface

docs/
├── commands/COMMANDS.md          # Complete rewrite
├── Home.md                       # Version update
├── README.md                     # Version update
├── _Sidebar.md                   # New sections
├── guides/Quick-Start.md         # K8s quick start
└── releases/ROADMAP.md           # Update timeline
```

---

## Summary Statistics

| Metric | Value |
|--------|-------|
| **New Files** | ~75 |
| **Modified Files** | ~25 |
| **New Lines of Code** | ~15,000-20,000 |
| **New Providers** | 16 |
| **New Commands** | 4 |
| **New Subcommands** | 50+ |
| **Documentation Pages** | 25+ |
| **Test Files** | 30+ |
| **Estimated Development Time** | 8-10 weeks |

---

## Completion Checklist

### Phase 1: Foundation
- [ ] Provider abstraction layer
- [ ] K8s conversion library
- [ ] Legacy alias framework
- [ ] Error handling improvements
- [ ] nself.yml support

### Phase 2: Providers (16)
- [ ] Oracle Cloud
- [ ] IBM Cloud
- [ ] UpCloud
- [ ] Contabo
- [ ] Netcup
- [ ] Hostinger
- [ ] Hostwinds
- [ ] Kamatera
- [ ] SSD Nodes
- [ ] Exoscale
- [ ] Alibaba Cloud
- [ ] Tencent Cloud
- [ ] Yandex Cloud
- [ ] RackNerd
- [ ] BuyVM
- [ ] Time4VPS

### Phase 3: Kubernetes
- [ ] k8s command (all subcommands)
- [ ] Managed K8s integrations (12 providers)
- [ ] k3s support

### Phase 4: Helm
- [ ] helm command (all subcommands)
- [ ] Chart generation
- [ ] Repository management

### Phase 5: Command Reorganization
- [ ] cloud command
- [ ] service command
- [ ] monitor command update
- [ ] Legacy aliases
- [ ] Help updates
- [ ] Completion updates

### Phase 6: Sync & Deploy
- [ ] sync auto/watch/status
- [ ] deploy preview/promote
- [ ] Canary deployment
- [ ] Blue-green deployment

### Phase 7: Testing
- [ ] Unit tests (85% coverage)
- [ ] Integration tests (75% coverage)
- [ ] E2E tests (60% coverage)

### Phase 8: Documentation
- [ ] All command docs
- [ ] All provider docs
- [ ] All guides
- [ ] Release notes
- [ ] ROADMAP update

### Phase 9: Release
- [ ] Version bump to 0.4.7
- [ ] Full QA pass
- [ ] Cross-platform testing
- [ ] Security review
- [ ] GitHub release
- [ ] Package manager updates

---

*This plan is the complete reference for v0.4.7 development.*
*Last Updated: January 23, 2026*
