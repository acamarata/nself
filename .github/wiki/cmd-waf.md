# nself waf

> Manage the Web Application Firewall (Coraza + OWASP CRS).

## Synopsis

```
nself waf <subcommand> [flags]
```

## Description

`nself waf` manages a Coraza WAF integrated with nginx, using the OWASP Core Rule Set (CRS). It supports two modes: `detection` (log suspicious requests but allow them through) and `blocking` (reject requests matching CRS rules). Detection is the safer first step when introducing a WAF.

`waf enable` provisions `nginx/waf/coraza.conf` and an empty `nginx/waf/custom.conf` for project-specific rules. `waf mode <detection|blocking>` flips the `SecRuleEngine` directive in `coraza.conf`. `waf report` reads the WAF audit log inside the running nginx container and prints recent events; `--since` filters by time window.

After enabling or switching mode, run `nself build && nself restart nginx` so nginx picks up the change. WAF events are also emitted with the `waf_event` label so Loki and Grafana dashboards can chart them.

## Subcommands

| Name | Description |
|------|-------------|
| `enable` | Enable the WAF in detection mode |
| `mode <detection\|blocking>` | Set WAF enforcement mode |
| `report` | Show recent WAF events |

## Flags

### `waf report`

| Flag | Default | Description |
|------|---------|-------------|
| `--since` | `24h` | Time window for report (e.g., `1h`, `24h`, `7d`) |

## Examples

```bash
# First-time enable in safe detection mode
nself waf enable
nself build && nself restart nginx

# Switch to blocking mode after a soak period
nself waf mode blocking
nself restart nginx

# Switch back to detection if blocking causes false positives
nself waf mode detection
nself restart nginx

# Inspect events from the last hour
nself waf report --since 1h
```

## See Also

- [[cmd-security]], security audit and setup
- [[cmd-alerts]], alert rules
- [[cmd-monitor]], Grafana dashboards
- [[Commands]], full command index

← [[Commands]] | [[Home]] →
