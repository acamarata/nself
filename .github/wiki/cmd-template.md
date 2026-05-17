# nself template

> Browse, install, and publish full-stack app templates from the nSelf registry.

## Synopsis

```
nself template <subcommand> [flags]
```

## Description

`nself template` provides access to the nSelf template registry: a catalog of community and official full-stack starter templates. Each template bundles Postgres schema, Hasura metadata, seed data, and a Flutter starter in a single archive.

Templates can be listed, inspected, and applied during project init:

```bash
nself init --template saas-starter ./my-app
```

This command also covers publishing new templates and applying incremental schema migrations from an installed template's `migrations/` directory.

Two template sources are available:

- **Bundled:** embedded in the CLI binary, always available offline
- **Registry:** community templates at `nself.org/templates`, fetched on demand (graceful fallback when unavailable)

## Subcommands

| Subcommand | Description |
|------------|-------------|
| `list` | List available templates from bundled store and the online registry |
| `info <slug>` | Show full detail for a single template |
| `publish` | Validate and submit a template to the nSelf registry |
| `update` | Apply incremental SQL migrations from the installed template |

## Flags

### `nself template list`

| Flag | Default | Description |
|------|---------|-------------|
| `--category` | `""` | Filter by category: `saas`, `marketplace`, `social`, `productivity`, `media`, `ecommerce` |
| `--free` | false | Show free templates only |
| `--sort` | `installs` | Sort by: `installs`, `rating`, `newest`, `price` |
| `--json` | false | Output raw JSON (includes both bundled and registry entries) |

### `nself template info`

| Flag | Default | Description |
|------|---------|-------------|
| `--json` | false | Output raw JSON |

### `nself template publish`

| Flag | Default | Description |
|------|---------|-------------|
| `--tarball` | `""` | Path to the compiled template tarball (`.tar.gz`). Required. |
| `--manifest` | `template.yml` | Path to the template manifest file |

### `nself template update`

| Flag | Default | Description |
|------|---------|-------------|
| `--force` | false | Allow destructive migrations containing `DROP` or `TRUNCATE` |
| `--dry-run` | false | Show pending migrations without applying them |

## Environment variables

| Variable | Description |
|----------|-------------|
| `NSELF_TEMPLATE_REGISTRY_URL` | Override the registry API base URL (default: `https://nself.org/api/templates`) |

## Examples

```bash
# List all templates (bundled + registry)
nself template list

# Filter by category
nself template list --category saas

# Free templates only, sorted by newest
nself template list --free --sort newest

# Detail for the saas-starter template
nself template info saas-starter

# Apply a template during init
nself init --template saas-starter ./my-app

# Preview pending schema migrations (dry run)
nself template update --dry-run

# Apply pending migrations
nself template update

# Apply a migration that includes DROP statements
nself template update --force

# Compute tarball SHA256 and print submission instructions
nself template publish --tarball ./dist/my-template.tar.gz

# Use a non-default manifest path
nself template publish --tarball ./dist/my-template.tar.gz --manifest my-template.yml
```

## Publishing a template

Templates are submitted via the web form at `nself.org/developers/templates`. `nself template publish` validates your manifest locally and prints the tarball SHA256 needed for the submission form. The publishing workflow is:

1. Build your template archive: `tar -czf dist/my-template.tar.gz schema/ metadata/ seed/ flutter/`
2. Run `nself template publish --tarball ./dist/my-template.tar.gz` to validate and get the SHA256.
3. Upload the tarball to a public URL (e.g. a GitHub release).
4. Fill in the submission form at `https://nself.org/developers/templates`.

Author review and approval typically takes 1-3 business days.

## Manifest required fields

A valid `template.yml` must include these fields:

```yaml
slug: my-template
display_name: My Template
version: 1.0.0
category: saas
```

## See Also

- [[cmd-init]] — scaffold a project from a template
- [[cmd-db]] — run schema migrations on a live project
- [[Plugin-Dev-Guide]] — build and publish plugins

← [[Commands]] | [[Home]] →
