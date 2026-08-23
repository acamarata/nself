# nself urls

<!-- BEGIN PROSE:summary -->
> Display all service URLs with route conflict detection.
<!-- END PROSE:summary -->

## Synopsis

```
nself urls [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself urls` prints all URLs that the ɳSelf stack exposes, grouped by type: Required Services, Optional Services, Custom Services, and Frontend Apps. URLs are computed from `BASE_DOMAIN` and each service's route configuration, no services need to be running.

Internal-only services (PostgreSQL, Redis) are shown with their `127.0.0.1` binding and labeled as internal. Publicly routed services show their full HTTPS URL. Use `--all` to include internal routes alongside public ones.

Use `--check-conflicts` to detect if two services share the same route prefix, which would cause Nginx routing ambiguity. Use `--diff` to compare URLs between two environments (e.g., dev vs prod) to spot configuration drift.
<!-- END PROSE:description -->

## Flags

<!-- BEGIN GENERATED:flags -->
| Flag | Default | Description |
|------|---------|-------------|
| `--all`, `-a` | `false` | Show all routes including internal |
| `--check-conflicts` | `false` | Check for route conflicts |
| `--diff` | `""` | Compare URLs between environments |
| `--env` | `""` | Show URLs for specific environment |
| `--json` | `false` | JSON output |
| `--help`, `-h` | — | Show help |
<!-- END GENERATED:flags -->

## Examples

<!-- BEGIN PROSE:examples -->
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
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Commands]] — full command index
- [[Core-Services]] — what a stack is made of
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
