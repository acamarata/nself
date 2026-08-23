# nself sentry-server

<!-- BEGIN PROSE:summary -->
> Provision and manage ɳSentry ops servers.
<!-- END PROSE:summary -->

## Synopsis

```
nself sentry-server <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Provision and manage ɳSentry observability/ops servers.

The sentry-server command orchestrates the full lifecycle of an ops server:
Terraform provisioning, secrets push, and ops-profile deployment.
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
| `provision` | Provision an ops/sentry server for a project |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself sentry-server provision myproject
  nself sentry-server provision myproject --dry-run
  nself sentry-server provision myproject --host nself@1.2.3.4 --key-path ~/.ssh/ops_key
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
