# Error Codes Reference

Every user-facing error in the ɳSelf CLI includes a stable code, a plain-language explanation, and a suggested fix. This page is the catalog for all registered error codes.

---

## How errors appear in the CLI

When the CLI encounters a problem, it prints a structured message in this format:

```
[E001] Docker not installed
  Why: The docker binary was not found in PATH.
  Fix: Install Docker: https://docs.docker.com/get-docker/
  Docs: https://nself.org/docs/reference/error-codes#e001
```

The code in brackets (e.g., `[E001]`) is stable across CLI versions. You can search this page by code to find the cause and fix.

---

## Looking up an error

- Copy the bracketed code from the terminal output (e.g., `E102`).
- Search this page for that code (Ctrl+F or Cmd+F).
- The fix column gives you the fastest path to resolution.

---

## Docker (E001-E049)

These errors occur when the CLI cannot find or communicate with Docker.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E001 | Docker not installed | The `docker` binary was not found in `PATH`. | Install Docker: [https://docs.docker.com/get-docker/](https://docs.docker.com/get-docker/) |
| E002 | Docker daemon not running | The Docker daemon is not responding to commands. | Start Docker Desktop or run: `sudo systemctl start docker` |
| E003 | Docker Compose not available | `docker compose` v2 plugin is not installed. | Update Docker Desktop or install the compose plugin manually. |
| E004 | docker-compose.yml not found | No `docker-compose.yml` exists in the project directory. | Run `nself build` to generate the compose file. |
| E005 | Port conflict | A required port is already in use by another process. | Run `nself doctor` to identify the conflict, then stop the conflicting process or change the port in `.env`. |

---

## Config (E050-E099)

These errors occur during configuration validation or when reading `.env` files.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E050 | No .env file found | No `.env` or `.env.dev` file exists in the project directory. | Run `nself init` to generate a configuration file. |
| E051 | Invalid config key | The configuration key contains invalid characters. | Config keys must contain only A-Z, 0-9, and underscores. |
| E052 | Config validation failed | One or more configuration values are invalid. | Run `nself config validate` to see all issues, then fix the reported values. |
| E053 | Weak password detected | A password field does not meet minimum length or contains insecure patterns. | Use a strong, randomly generated password of at least 16 characters. |
| E054 | Invalid project name | Project name contains characters not allowed in Docker and DNS contexts. | Use only lowercase letters, numbers, and hyphens. Must start with a letter. |
| E055 | Duplicate route detected | Two or more services are configured with the same nginx route. | Check `ROUTE` values in `.env` and ensure each service has a unique route. |
| E056 | Unknown config key | The key is not recognized by nself. | Check for typos. Run `nself config list` to see all valid keys. |

---

## Plugin and License (E100-E149)

These errors occur when installing plugins or validating license keys.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E100 | Plugin not found | The requested plugin does not exist in the registry. | Run `nself plugin list` to see available plugins. |
| E101 | Invalid license key | The license key format is invalid. | License keys start with `nself_pro_` followed by 32+ characters. Check your key. |
| E102 | License tier insufficient | Your license tier does not include this plugin. | Upgrade your plan at [https://nself.org/pricing](https://nself.org/pricing) |
| E103 | License expired | The license key has expired. | Renew your license at [https://nself.org/account](https://nself.org/account) |
| E104 | License validation failed (network) | Cannot reach the license server and no valid local cache exists. | Check your internet connection. Previously validated licenses work offline for 7 days. |
| E105 | Circular plugin dependency | Plugin dependency graph contains a cycle. | Report this as a bug at [https://github.com/nself-org/cli/issues](https://github.com/nself-org/cli/issues) |

---

## SSL and Network (E150-E199)

These errors occur during SSL certificate generation or network operations.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E150 | mkcert not installed | `mkcert` is not installed; falling back to OpenSSL self-signed certs. | Install mkcert: `brew install mkcert` (macOS) or see [https://github.com/FiloSottile/mkcert](https://github.com/FiloSottile/mkcert) |
| E151 | SSL certificate generation failed | Could not generate SSL certificates for the configured domain. | Check domain configuration and ensure `openssl` is available. |

---

## Database (E200-E249)

These errors occur when the CLI interacts with the PostgreSQL container.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E200 | Database not running | PostgreSQL container is not running or not accepting connections. | Run `nself start` to start all services, or `nself doctor` to diagnose. |
| E201 | Migration failed | A database migration could not be applied. | Check the migration SQL for errors. Run `nself db migrate --status` to see pending migrations. |
| E202 | Backup failed | Database backup operation failed. | Ensure sufficient disk space and that the database is running. |

---

## Health (E250-E299)

These errors occur during health checks run by `nself doctor`, `nself health`, and startup probes.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E250 | Service unhealthy | A service health check returned an unhealthy status. | Run `nself doctor --verbose` for detailed diagnostics. |
| E251 | Health check timeout | The health check did not complete within the timeout period. | The service may be starting slowly. Wait and retry, or check logs with `nself logs`. |

---

## Init and Project (E300-E349)

These errors occur during `nself init` or when the CLI validates the working directory.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E300 | Project already initialized | A `.env` file already exists in this directory. | Use `nself config set` to modify existing config, or delete `.env` to reinitialize. |
| E301 | Source directory detected | You are running nself inside the CLI source repository. | Change to your project directory first: `cd /path/to/your/project` |

---

## Domain (E350-E399)

These errors occur when the CLI validates domain names or port numbers from config.

| Code | Summary | Why | Fix |
|------|---------|-----|-----|
| E350 | Invalid domain name | The configured domain name is not valid. | Use a valid domain like `example.com` or `localhost`. |
| E351 | Invalid port number | Port number is outside the valid range (1-65535). | Use a port number between 1024 and 65535 for non-privileged ports. |

---

## CLI output format

The full structured format for a CLI error:

```
[EXXXX] What happened (brief)
  Why: Root cause explanation.
  Fix: Steps to resolve the issue.
  Docs: https://nself.org/docs/reference/error-codes#eXXXX
```

The `Why` and `Fix` lines provide context to act on. The `Docs` line links directly to the anchor for that code on this page (via `nself.org/docs`, which mirrors this wiki).

---

## Error codes in scripts

If you are scripting around the CLI, you can check the exit code (non-zero on error) and parse the bracketed code from stderr:

```bash
output=$(nself start 2>&1)
if [ $? -ne 0 ]; then
  code=$(printf '%s' "$output" | grep -oE '\[E[0-9]+\]' | head -1)
  printf 'nself failed with %s\n' "$code"
fi
```

---

## Related pages

- [[cmd-doctor]], run diagnostics to identify and resolve common errors
- [[cmd-config]], view and edit configuration values
- [[Plugin-Licensing]], license key format and tier details
- [[Guide-SSL-Setup]], SSL certificate setup
- [[Config-Env-Vars]], all environment variable reference

---

← [[Home]] | [[_Sidebar]]
