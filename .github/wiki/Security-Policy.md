# Security Policy

## Reporting a Vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

Email: **security@nself.org**

Include in your report:
- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Any suggested fix (optional)

We will acknowledge your report within **48 hours** and keep you informed throughout the process.

## Disclosure Timeline

| Severity | Patch Target |
|----------|-------------|
| Critical | 14 days |
| High | 30 days |
| Medium / Low | 90 days |

We follow coordinated disclosure. We ask that you do not publicly disclose the vulnerability until we have released a patch and notified users.

## Scope

**In scope:**
- ɳSelf CLI binary (`nself`)
- Core service configuration generation (Compose, Nginx)
- Plugin installation and validation system
- License validation

**Out of scope:**
- Third-party Docker images (Postgres, Hasura, Auth, MinIO, etc.)
- User application code deployed on ɳSelf
- nself.org website infrastructure
- Plugins not maintained by ɳSelf (community plugins)

Vulnerabilities in third-party images should be reported to their respective projects.

## Security Updates

Security patches are released as patch versions (e.g. `v1.0.3`). Update with:

```bash
nself update
```

### Built-in Security Auditing (v1.0.3+)

The `nself security` command provides automated security auditing and hardening for your deployment. Run `nself security audit` to check firewall status, SSH hardening, Docker port exposure, `.env` file permissions, and service binding. Run `nself security setup --apply` to apply recommended hardening steps automatically.

See [[cmd-security]] for full usage and examples.

Subscribe to [GitHub Release notifications](https://github.com/nself-org/cli/releases) to be notified of security updates.

## See Also

- [[Security-Architecture]], how ɳSelf is designed for security
- [[Security-Hardening]], production hardening checklist
- [[Guide-Security-Hardening]], step-by-step security guide

---
← [[Home]] | [[_Sidebar]]
