# nself infra

> Provision nSelf infrastructure on cloud providers via Terraform.

## Synopsis

```bash
nself infra <subcommand> [flags]
```

## Description

`nself infra` provisions cloud servers and supporting resources for nSelf deployments using provider-specific Terraform modules. The modules live at `terraform/modules/<provider>/` relative to the CLI repository root.

Each subcommand wraps the corresponding Terraform operation (`init` is run automatically before `plan`, `apply`, or `destroy`). The `terraform` binary must be installed and available in `PATH`.

Provisioning creates a server sized for nSelf's required services (Postgres, Hasura, Auth, nginx) and outputs connection information including `NSELF_URL`, `ssh_host`, and `postgres_url`.

## Subcommands

| Subcommand | Description |
|-----------|-------------|
| `plan` | Show the Terraform plan for the given provider |
| `apply` | Provision nSelf infrastructure |
| `destroy` | Destroy all provisioned infrastructure |

## Flags

### nself infra plan

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--provider` | | string | | Cloud provider: `aws`, `gcp`, `azure`, `hetzner`, `do`, `linode` (required) |
| `--domain` | | string | | Domain for the nSelf deployment (required) |
| `--state-bucket` | | string | | S3 or GCS bucket name for Terraform remote state |

### nself infra apply

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--provider` | | string | | Cloud provider: `aws`, `gcp`, `azure`, `hetzner`, `do`, `linode` (required) |
| `--domain` | | string | | Domain for the nSelf deployment (required) |
| `--state-bucket` | | string | | S3 or GCS bucket name for Terraform remote state |
| `--auto-approve` | | bool | `false` | Skip interactive approval |

### nself infra destroy

| Flag | Short | Type | Default | Description |
|------|-------|------|---------|-------------|
| `--provider` | | string | | Cloud provider: `aws`, `gcp`, `azure`, `hetzner`, `do`, `linode` (required) |
| `--domain` | | string | | Domain for the nSelf deployment |
| `--auto-approve` | | bool | `false` | Confirm destruction without prompt (required to proceed) |

## Terraform Backend

The Terraform modules use a local state backend by default. To use remote state, pass `--state-bucket` with an S3 (AWS) or GCS (GCP) bucket name. The bucket must already exist and your environment must have credentials with read/write access.

State bucket configuration is passed to Terraform as `-backend-config=bucket=<name>`. Providers that do not support S3/GCS backends (such as Hetzner) ignore this flag.

Module path resolution: `terraform/modules/<provider>/` relative to the repo root.

## Examples

```bash
# Preview Hetzner provisioning
nself infra plan --provider hetzner --domain myapp.com
```

```bash
# Provision on Hetzner
nself infra apply --provider hetzner --domain myapp.com
```

```bash
# Provision on AWS with remote state
nself infra apply \
  --provider aws \
  --domain myapp.com \
  --state-bucket my-nself-terraform-state \
  --auto-approve
```

```bash
# Preview GCP provisioning
nself infra plan --provider gcp --domain myapp.com
```

```bash
# Preview DigitalOcean provisioning
nself infra plan --provider do --domain myapp.com
```

```bash
# Destroy Hetzner infrastructure
nself infra destroy --provider hetzner --auto-approve
```

```bash
# Preview Azure provisioning
nself infra plan --provider azure --domain myapp.com
```

## See Also

- [[cmd-init]] — initialise a new nSelf project before provisioning
- [[cmd-start]] — start the nSelf stack on a provisioned server
- [[cmd-deploy]] — deploy to an already-running nSelf instance
- [[cmd-k8s]] — deploy nSelf on Kubernetes via Helm

← [[Commands]] | [[Home]] →
