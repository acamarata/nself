# nself urls

> Display all service URLs with route conflict detection.

## Synopsis

```
nself urls [flags]
```

## Description

`nself urls` prints all URLs that the ɳSelf stack exposes, grouped by type: Required Services, Optional Services, Custom Services, and Frontend Apps. URLs are computed from `BASE_DOMAIN` and each service's route configuration, no services need to be running.

Internal-only services (PostgreSQL, Redis) are shown with their `127.0.0.1` binding and labeled as internal. Publicly routed services show their full HTTPS URL. Use `--all` to include internal routes alongside public ones.

Use `--check-conflicts` to detect if two services share the same route prefix, which would cause Nginx routing ambiguity. Use `--diff` to compare URLs between two environments (e.g., dev vs prod) to spot configuration drift.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--all`, `-a` | false | Show all routes including internal |
| `--json` | false | JSON output |
| `--env` | `""` | Show URLs for a specific environment |
| `--diff` | `""` | Compare URLs between environments (e.g. `--env dev --diff prod`) |
| `--check-conflicts` | false | Check for route conflicts |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Show all public URLs
nself urls

# Include internal routes
nself urls --all

# JSON output
nself urls --json

# Check for route conflicts
nself urls --check-conflicts

# URLs for a specific environment
nself urls --env staging

# Compare dev and prod URL configurations
nself urls --env dev --diff prod
```

**Sample output:**

```
Required Services:
  PostgreSQL       127.0.0.1:5432          (internal only)
  Hasura GraphQL   https://api.localhost
  Auth             https://auth.localhost
  Nginx            https://localhost

Optional Services:
  Redis            127.0.0.1:6379          (internal only)
  Mailpit UI       https://mail.localhost
  MinIO Console    https://storage-console.localhost

Custom Services:
  ping-api         https://ping.localhost

12 routes on localhost
```

← [[Commands]] | [[Home]] →
