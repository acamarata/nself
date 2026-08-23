# nself k8s

<!-- BEGIN PROSE:summary -->
> Deploy and manage nSelf on any Kubernetes cluster using the official Helm chart.
<!-- END PROSE:summary -->

## Synopsis

```
nself k8s <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself k8s` deploys and manages the nSelf stack on Kubernetes via the official Helm chart,
published at `https://charts.nself.org`.

The chart provisions Postgres (StatefulSet), Hasura, Auth, and an Nginx Ingress in a single
`helm install` call. TLS is handled by cert-manager, which must be pre-installed in the cluster
before running `nself k8s install`.

Three subcommands cover the full lifecycle:

- `install` performs a first-time deployment and waits up to five minutes for all pods to become ready.
- `upgrade` applies the latest chart version with a rolling update, reusing existing values.
- `status` returns the Helm release status as JSON output from `helm status`.

The `--domain` flag on `install` sets the primary ingress hostname for the deployment. Plugins can
be installed at deploy time via `--plugins`; a valid license key must be available in the
`NSELF_PLUGIN_LICENSE_KEY` environment variable for pro plugins to activate.

`helm` must be installed and available in `PATH`. Install from [helm.sh](https://helm.sh).

### install
### upgrade
### status
## Helm Chart Specifics

The chart is published under the `nself` repository alias at `https://charts.nself.org`. The
chart name is `nself/nself`. The CLI adds and updates the repository automatically before
`install` or `upgrade` runs.

Key Helm values set by the CLI:

| Value | Set by flag | Description |
|-------|------------|-------------|
| `domain` | `--domain` | Primary ingress hostname |
| `license.key` | env `NSELF_PLUGIN_LICENSE_KEY` | Pro plugin license key |
| `plugins.install[N]` | `--plugins` | Plugin names to install at deploy time |

The upgrade path uses `--reuse-values`, so previously set values persist across upgrades
unless explicitly overridden with `--set` passed directly to `helm`. To override values
not exposed as CLI flags, run `helm upgrade` directly against the `nself/nself` chart using
the same release name.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Subcommands

<!-- BEGIN GENERATED:subcommands -->
| Name | Description |
|------|-------------|
| `install` | Install ɳSelf on a Kubernetes cluster |
| `status` | Show the Helm release status |
| `upgrade` | Upgrade the ɳSelf Helm release |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
Install nSelf on a cluster with the default release name, pointing at a domain:

```bash
nself k8s install --domain myapp.example.com
```

Install with a custom release name and a specific kubeconfig:

```bash
nself k8s install --domain myapp.example.com --cluster ~/.kube/staging.yaml --release myapp-nself
```

Install and activate pro plugins at deploy time (requires `NSELF_PLUGIN_LICENSE_KEY`):

```bash
export NSELF_PLUGIN_LICENSE_KEY=nself_pro_...
nself k8s install --domain myapp.example.com --plugins ai,claw,mux
```

Check the Helm release status for the default release:

```bash
nself k8s status
```

Check status for a named release using a specific kubeconfig:

```bash
nself k8s status --release myapp-nself --cluster ~/.kube/prod.yaml
```

Upgrade the default release to the latest chart version:

```bash
nself k8s upgrade
```

Upgrade a specific release using a non-default kubeconfig:

```bash
nself k8s upgrade --release myapp-nself --cluster ~/.kube/prod.yaml
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-start.md](cmd-start.md) — start the nSelf stack on a local or VPS deployment
- [cmd-init.md](cmd-init.md) — initialise a new nSelf project
- [cmd-plugin.md](cmd-plugin.md) — manage plugins after the stack is running
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
