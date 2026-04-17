# nself dns-setup

> Add project domains to /etc/hosts (run with sudo).

## Synopsis

```
nself dns-setup [flags]
```

## Description

`nself dns-setup` reads the current project's `.env`, collects every domain that nSelf generates nginx config for, and appends missing entries to `/etc/hosts` so local `*.<custom-domain>` URLs resolve to `127.0.0.1`. Because writing `/etc/hosts` requires root, run the command once with `sudo`.

Wildcard subdomains (e.g. `*.pro.ummat.local`) cannot live in `/etc/hosts`. The command prints the wildcard entries separately so you can configure dnsmasq with lines like `address=/.pro.ummat.local/127.0.0.1`. For a fully wildcard-friendly local trust setup, see `nself trust`.

This command is only needed when you use a custom `BASE_DOMAIN` (for example `ummat.local`). Standard `nself.org` and `localhost` setups do not need it. On macOS, if run without sudo, the command attempts to relaunch itself via `osascript` to surface a native admin password prompt.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--dry-run`, `-n` | false | Print entries that would be added without writing |
| `--project`, `-p` | `""` | Path to nself project directory (useful when running with sudo) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Preview entries without touching /etc/hosts
nself dns-setup --dry-run

# Apply entries (writes /etc/hosts, requires sudo)
sudo nself dns-setup

# Run with sudo from a path other than the project root
sudo nself dns-setup --project /Users/me/projects/myapp

# After running, see whether all domains resolve
nself urls
```

## See Also

- [[cmd-trust]] — full local dev trust (DNS, SSL, port forwarding)
- [[cmd-urls]] — list all service URLs
- [[cmd-ssl]] — manage SSL certificates
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
