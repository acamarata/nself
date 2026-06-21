# nself clean

> Remove generated build artifacts: `docker-compose.yml`, nginx configs, and build cache.

## Synopsis

```
nself clean [flags]
```

## Description

`nself clean` removes the files that `nself build` generates: `docker-compose.yml`, `nginx/sites/` config files, `.nself/cache/`, and the Docker builder layer cache (`docker builder prune --filter type=exec.cachemount`). It is non-destructive: no `.env` files, Docker volumes, container data, or user files are touched.

Run `nself build` after `nself clean` to regenerate all artifacts.

To stop running containers and remove Docker volumes (which deletes your data), use `nself reset` instead.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Remove generated artifacts and build cache
nself clean

# Rebuild after cleaning
nself clean && nself build
```

**What is removed:**

| Item | Removed |
|------|---------|
| `docker-compose.yml` | Yes |
| `nginx/sites/*.conf` | Yes |
| `.nself/cache/` | Yes |
| Docker builder layer cache | Yes (non-fatal if Docker not running) |

**What is preserved:**

| Item | Preserved |
|------|-----------|
| `.env` and all `.env.*` variants | Yes |
| Docker volumes and container data | Yes |
| User-managed files and source code | Yes |

## See Also

- [[cmd-build]], regenerate the artifacts that clean removes
- [[cmd-reset]], stop containers and remove generated files
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
