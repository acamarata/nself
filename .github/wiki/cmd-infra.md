# nself infra

<!-- BEGIN PROSE:summary -->
> Provision nSelf infrastructure on cloud providers via Terraform.
<!-- END PROSE:summary -->

## Synopsis

```
nself infra <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself infra` provisions cloud servers and supporting resources for nSelf deployments using provider-specific Terraform modules. The modules live at `terraform/modules/<provider>/` relative to the CLI repository root.

Each subcommand wraps the corresponding Terraform operation (`init` is run automatically before `plan`, `apply`, or `destroy`). The `terraform` binary must be installed and available in `PATH`.

Provisioning creates a server sized for nSelf's required services (Postgres, Hasura, Auth, nginx) and outputs connection information including `NSELF_URL`, `ssh_host`, and `postgres_url`.

### nself infra plan
### nself infra apply
### nself infra destroy
## Terraform Backend

The Terraform modules use a local state backend by default. To use remote state, pass `--state-bucket` with an S3 (AWS) or GCS (GCP) bucket name. The bucket must already exist and your environment must have credentials with read/write access.

State bucket configuration is passed to Terraform as `-backend-config=bucket=<name>`. Providers that do not support S3/GCS backends (such as Hetzner) ignore this flag.

Module path resolution: `terraform/modules/<provider>/` relative to the repo root.
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
| `apply` | Provision ɳSelf infrastructure via Terraform |
| `destroy` | Destroy ɳSelf infrastructure |
| `plan` | Show the Terraform plan for the given provider |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
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
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[cmd-init]] — initialise a new nSelf project before provisioning
- [[cmd-start]] — start the nSelf stack on a provisioned server
- [[cmd-deploy]] — deploy to an already-running nSelf instance
- [[cmd-k8s]] — deploy nSelf on Kubernetes via Helm
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
