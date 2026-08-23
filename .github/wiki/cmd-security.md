# nself security

<!-- BEGIN PROSE:summary -->
> Server security: audit, setup, and status.
<!-- END PROSE:summary -->

## Synopsis

```
nself security <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
Security audit, setup, and status commands for hardening your ɳSelf deployment.

## Usage

```
nself security <subcommand> [flags]
```

### nself security audit
Run security checks on your running stack. Inspects: UFW firewall status, fail2ban configuration, SSH hardening (key-only, root disabled), Docker port exposure, `.env` file permissions, and service binding.
```bash
nself security audit
```
Example output:
```
Security Audit Results
======================
[PASS] UFW firewall is active
[PASS] fail2ban is running
[WARN] SSH root login is enabled — disable with `PermitRootLogin no`
[PASS] Docker ports bound to 127.0.0.1
[FAIL] .env.secrets is world-readable — run `chmod 600 .env.secrets`
[PASS] All services bind to 127.0.0.1

Score: 4/6 checks passed
```
Exit codes: `0` all checks pass, `1` one or more checks failed.
### nself security setup
Apply security hardening steps. Runs in dry-run mode by default, showing what would change without modifying anything.
```bash
# Preview changes (dry-run)
nself security setup

# Apply changes (requires root)
sudo nself security setup --apply
```
Hardening steps applied:
- Enable and configure UFW (allow SSH, HTTP, HTTPS only)
- Install and configure fail2ban for SSH and Nginx
- Harden SSH config (disable root login, disable password auth)
- Set correct permissions on `.env` files (`600`)
- Verify Docker daemon binds to localhost only
### nself security status
Show a one-line security posture summary for the current project.
```bash
nself security status
```
Example output:
```
Security: 5/6 checks passing | Last audit: 2 hours ago
```
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
| `audit` | Run a read-only security audit |
| `setup` | Apply baseline server hardening |
| `status` | Show current security posture |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
<!-- TODO(docs): needs human prose -->

```bash
nself security
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [[Security-Policy]], security disclosure and patching policy
- [[Guide-Production-Deployment]], production hardening guide

---
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
