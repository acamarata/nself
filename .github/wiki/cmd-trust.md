# nself trust

> Set up local dev trust (DNS, SSL, port forwarding).

## Synopsis

```
nself trust [flags]
```

## Description

`nself trust` configures everything needed for `*.local` projects to work locally with HTTPS: dnsmasq for wildcard DNS resolution, mkcert for trusted SSL certificates, and pfctl/iptables for port 80→8080 and 443→8443 forwarding. One run covers every ɳSelf project on the machine, not per project.

The command auto-detects the OS (macOS or Linux) and chooses the right toolchain. On macOS it uses Homebrew dnsmasq, `/etc/resolver/local`, and pfctl; on Linux it uses systemd-resolved or raw dnsmasq plus iptables. Steps that need elevation prompt for sudo (or trigger the macOS native admin dialog).

Use `--status` to see which components are already set up without making changes. Use `--undo` to print the exact commands to remove every change `nself trust` made. Use `--skip-dns`, `--skip-ssl`, or `--skip-ports` to opt out of any layer.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--skip-dns` | false | Skip dnsmasq and `/etc/resolver` setup |
| `--skip-ssl` | false | Skip mkcert CA and certificate generation |
| `--skip-ports` | false | Skip port forwarding setup |
| `--status` | false | Show current trust status and exit |
| `--undo` | false | Print instructions to undo all trust changes |
| `--project`, `-p` | `""` | Path to nself project directory (allows running from any cwd) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Full setup (run once per machine)
nself trust

# Check what's already configured
nself trust --status

# Set up only mkcert SSL, skip DNS and port forwarding
nself trust --skip-dns --skip-ports

# Print undo instructions
nself trust --undo

# Run from outside the project root
nself trust --project /Users/me/projects/myapp
```

## See Also

- [[cmd-dns-setup]], `/etc/hosts` entries (alternative to dnsmasq)
- [[cmd-ssl]], manage SSL certificates
- [[cmd-doctor]], diagnose connectivity issues
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
