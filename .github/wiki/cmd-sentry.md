# nself sentry

**This command moved to a plugin.**

`nself sentry` is no longer part of the CLI core. ɳSentry is a separate product
surface, not part of the self-hosted backend lifecycle the core covers.

## Install

```bash
nself install sentry-cli
```

One plugin provides both `nself sentry` and `nself sentry-server`. Once
installed, each works exactly as before — the CLI proxies each command to its
own binary.

## Related

`nself mcp` keeps its ɳSentry MCP tools whether or not this plugin is
installed, because `mcp` is a core command and plugins cannot contribute MCP
tools yet.

---

← [[Commands]] · [[Plugin-Overview]] · [[Home]]
