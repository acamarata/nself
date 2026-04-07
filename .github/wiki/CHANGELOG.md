# Changelog

> See [CHANGELOG.md](../CHANGELOG.md) in the repo root for the full version history.

## v1.0.3

Security hardening release with expanded plugin ecosystem.

- [docs]: Added 6 feature wiki pages (nClaw, nChat, nTV, nFamily, nCloud, Licensing Guide)
- [docs]: Added comprehensive API Reference covering all plugin REST endpoints
- [docs]: Updated Configuration with plugin environment variables (inter-plugin auth, AI routing, nClaw behavior, resource limits)
- [docs]: Expanded Security Architecture with JWT model, passkey auth, ACL system, plugin-to-plugin auth, browser/shell/email security

**Highlights:**
- **`nself security` command** — new top-level command with `audit`, `setup`, and `status` subcommands for automated security hardening. Checks UFW, fail2ban, SSH, Docker port exposure, `.env` permissions, and service binding. See [[cmd-security]]
- **87 plugins** — the ecosystem now ships 25 free and 62 Pro plugins (up from 77 total in v1.0.0). New additions include **claw-budget** for AI token budget tracking and cost controls
- **Security hardening** — container security improvements, stricter default `.env` permissions, and internal plugin authentication via `PLUGIN_INTERNAL_SECRET`
- **New environment variables** — `CLAW_WEB_SECRET` for claw-web plugin authentication, `PLUGIN_INTERNAL_SECRET` for inter-plugin communication
- **Bug fixes** — resolved edge cases in compose generation, improved error messages for license validation failures

**Upgrade:**
```bash
nself update
nself doctor    # verify stack health after upgrade
```

---

## v1.0.0

First stable release. Complete Go rewrite of the nSelf CLI (previously Bash-based v0.x).

**Highlights:**
- 24 top-level commands with 295+ subcommands
- Full Docker Compose v2 stack generation (Postgres, Hasura, Auth, Nginx)
- Plugin system with 25 free plugins and 52 Pro plugins
- `nself migrate` command for v1 → v2 migration with rollback
- Interactive `nself init` wizard with multi-env support
- `nself health watch` for continuous monitoring
- Shell completion for bash, zsh, and fish
- `nself doctor` comprehensive system diagnostics
- License management for Pro plugins
