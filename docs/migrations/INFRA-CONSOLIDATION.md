# Infrastructure Command Consolidation

**Migration Guide: v0.9.6 → v1.0.0**

## Overview

In nself v1.0.0, all infrastructure-related commands have been consolidated under a single `infra` command for better organization and discoverability.

## What Changed

### Old Structure (v0.9.5)
- `nself provider` - Cloud provider management
- `nself cloud` - Alias for provider (deprecated)
- `nself k8s` - Kubernetes management
- `nself helm` - Helm chart management

### New Structure (v1.0.0)
- `nself infra provider` - Cloud provider infrastructure
- `nself infra k8s` - Kubernetes infrastructure
- `nself infra helm` - Helm chart management

## Complete Migration Map

### Provider Commands

| Old Command | New Command | Status |
|-------------|-------------|--------|
| `nself provider list` | `nself infra provider list` | ✅ Working |
| `nself provider init <name>` | `nself infra provider init <name>` | ✅ Working |
| `nself provider validate` | `nself infra provider validate` | ✅ Working |
| `nself provider show <name>` | `nself infra provider show <name>` | ✅ Working |
| `nself provider server create` | `nself infra provider server create` | ✅ Working |
| `nself provider server list` | `nself infra provider server list` | ✅ Working |
| `nself provider server ssh <id>` | `nself infra provider server ssh <id>` | ✅ Working |
| `nself provider cost estimate` | `nself infra provider cost estimate` | ✅ Working |
| `nself provider cost compare` | `nself infra provider cost compare` | ✅ Working |

### Kubernetes Commands

| Old Command | New Command | Status |
|-------------|-------------|--------|
| `nself k8s init` | `nself infra k8s init` | ✅ Working |
| `nself k8s convert` | `nself infra k8s convert` | ✅ Working |
| `nself k8s apply` | `nself infra k8s apply` | ✅ Working |
| `nself k8s deploy` | `nself infra k8s deploy` | ✅ Working |
| `nself k8s status` | `nself infra k8s status` | ✅ Working |
| `nself k8s logs <pod>` | `nself infra k8s logs <pod>` | ✅ Working |
| `nself k8s scale <dep> <n>` | `nself infra k8s scale <dep> <n>` | ✅ Working |
| `nself k8s rollback` | `nself infra k8s rollback` | ✅ Working |
| `nself k8s delete` | `nself infra k8s delete` | ✅ Working |
| `nself k8s cluster <action>` | `nself infra k8s cluster <action>` | ✅ Working |
| `nself k8s namespace <action>` | `nself infra k8s namespace <action>` | ✅ Working |

### Helm Commands

| Old Command | New Command | Status |
|-------------|-------------|--------|
| `nself helm init` | `nself infra helm init` | ✅ Working |
| `nself helm generate` | `nself infra helm generate` | ✅ Working |
| `nself helm install <release>` | `nself infra helm install <release>` | ✅ Working |
| `nself helm upgrade <release>` | `nself infra helm upgrade <release>` | ✅ Working |
| `nself helm rollback <release>` | `nself infra helm rollback <release>` | ✅ Working |
| `nself helm uninstall <release>` | `nself infra helm uninstall <release>` | ✅ Working |
| `nself helm list` | `nself infra helm list` | ✅ Working |
| `nself helm status <release>` | `nself infra helm status <release>` | ✅ Working |
| `nself helm values <release>` | `nself infra helm values <release>` | ✅ Working |
| `nself helm template` | `nself infra helm template` | ✅ Working |
| `nself helm package` | `nself infra helm package` | ✅ Working |
| `nself helm repo <action>` | `nself infra helm repo <action>` | ✅ Working |

## Deprecation Warnings

Old commands still work but show deprecation warnings:

```bash
$ nself cloud list
⚠ DEPRECATION: 'nself cloud' is deprecated and will be removed in v1.0.0
             Please use 'nself infra provider' instead.
             https://docs.nself.org/migration/cloud-to-infra

# ... then shows provider list ...
```

## Backward Compatibility

All old commands continue to work through v0.9.x releases with deprecation warnings. They will be removed in v1.0.0.

### Compatibility Wrappers

- `nself cloud` → redirects to `nself infra provider`
- `nself provider` → still works directly (no redirect needed yet)
- `nself k8s` → still works directly (no redirect needed yet)
- `nself helm` → still works directly (no redirect needed yet)

## Benefits of Consolidation

1. **Better Organization**: All infrastructure commands in one place
2. **Clearer Hierarchy**: `infra` clearly indicates infrastructure management
3. **Easier Discovery**: `nself infra --help` shows all 38 infrastructure subcommands
4. **Consistent Patterns**: Matches other consolidated commands (auth, service, config)
5. **Reduced Top-Level Clutter**: Fewer commands at top level (79 → 31)

## Total Subcommands

The `nself infra` command consolidates **38 subcommands** across three categories:

- **Provider**: 14 subcommands
- **K8s**: 11 subcommands
- **Helm**: 12 subcommands

## Migration Timeline

- **v0.9.5**: Old commands work, no warnings
- **v0.9.6**: Old commands work, deprecation warnings shown
- **v1.0.0**: Old commands removed, must use new `infra` structure

## Quick Reference

### Provider Management
```bash
# List providers
nself infra provider list

# Initialize provider
nself infra provider init digitalocean

# Create server
nself infra provider server create --provider hetzner --size medium

# SSH to server
nself infra provider server ssh myserver

# Compare costs
nself infra provider cost compare
```

### Kubernetes
```bash
# Initialize K8s
nself infra k8s init

# Convert compose to manifests
nself infra k8s convert

# Deploy
nself infra k8s deploy

# Check status
nself infra k8s status

# Scale deployment
nself infra k8s scale api --replicas 3
```

### Helm
```bash
# Initialize chart
nself infra helm init

# Generate from compose
nself infra helm generate

# Install release
nself infra helm install myapp

# Upgrade release
nself infra helm upgrade myapp

# View releases
nself infra helm list
```

## Support

For questions or issues with migration:
- GitHub Issues: https://github.com/acamarata/nself/issues
- Documentation: https://docs.nself.org/commands/infra
- Discord: https://discord.gg/nself
