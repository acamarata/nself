# nself ops

<!-- BEGIN PROSE:summary -->
> Ops-profile deployment and management.
<!-- END PROSE:summary -->

## Synopsis

```
nself ops <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Manage deployments using the nSelf ops service profile.

The ops profile enables the observability + CI stack (prometheus, grafana,
loki, forgejo, container-registry) while excluding app-specific services
(minio, mailpit, nSelf Admin UI, search).

Required environment variable:
  NSELF_DEPLOY_HOST_OPS   Remote host in user@host:/remote/path format
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
| `deploy` | Deploy with the ops profile to the ops server |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
nself ops deploy
  nself ops deploy --dry-run
  nself ops deploy --follow
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
