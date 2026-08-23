# nself ai-studio

<!-- BEGIN PROSE:summary -->
> Google AI Studio integration for local nSelf instances via a secure Cloudflare Tunnel.
<!-- END PROSE:summary -->

## Synopsis

```
nself ai-studio <subcommand> [flags]
```

## Description

<!-- BEGIN PROSE:description -->
`nself ai-studio` connects a local nSelf Postgres instance to Google AI Studio so Gemini models can query your schema without any cloud deployment of your data.

The only current subcommand is `bridge`, which starts a local HTTP proxy and opens an ephemeral Cloudflare Tunnel (trycloudflare.com, no account required). The tunnel URL is printed to stdout; you paste it into AI Studio as a custom connector. The proxy sits between AI Studio and your Hasura GraphQL endpoint and enforces schema read-only access: no mutations, DDL, or DML pass through.

Each session issues a short-lived auth token (30-minute idle TTL by default) scoped to one project. On every response the proxy injects an `X-Nself-Schema-Context` header that carries your current Postgres schema so Gemini has full type information without making a separate introspection request. Pass `--no-context` to omit this header.

The bridge auto-closes when the idle timeout expires or on Ctrl-C. The `cloudflared` binary is downloaded automatically on first use if absent. The tunnel region defaults to `auto` but can be overridden with `--region` for latency tuning. IP access can be restricted to specific CIDR ranges with `--ip-allowlist`.

Passing `--dry-run` prints what would happen without starting the proxy or opening a tunnel.

### `ai-studio bridge`
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
| `bridge` | Start a secure AI Studio bridge via Cloudflare Tunnel |
<!-- END GENERATED:subcommands -->

## Examples

<!-- BEGIN PROSE:examples -->
```bash
# Start the bridge with defaults
nself ai-studio bridge
```

```bash
# Start on a custom local port
nself ai-studio bridge --port 9000
```

```bash
# Skip schema context injection
nself ai-studio bridge --no-context
```

```bash
# Preview what would start without actually opening the tunnel
nself ai-studio bridge --dry-run
```

```bash
# Restrict access to your home network
nself ai-studio bridge --ip-allowlist 192.168.1.0/24
```

```bash
# Keep the tunnel open for 60 minutes of idle time
nself ai-studio bridge --idle-timeout 60
```

```bash
# Route the tunnel through a specific region
nself ai-studio bridge --region eu
```
<!-- END PROSE:examples -->

## See Also

<!-- BEGIN PROSE:see-also -->
- [cmd-start.md](cmd-start.md) — start the nSelf stack the bridge connects to
- [cmd-doctor.md](cmd-doctor.md) — verify your local nSelf environment
- [cmd-plugin.md](cmd-plugin.md) — install and manage plugins
- [cmd-flag.md](cmd-flag.md) — runtime feature-flag plugin management
<!-- END PROSE:see-also -->

← [[Commands]] | [[Home]] →
