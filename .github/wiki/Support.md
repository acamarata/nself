# Support

Ways to get help when the ɳSelf CLI does not behave as expected.

---

## Self-service (fastest)

1. **Run `nself doctor`**, diagnoses common issues automatically and prints actionable fixes.
2. **Check the [[Errors]] page**, every bracketed error code has a dedicated entry with cause and fix.
3. **Search the [[FAQ]]**, common setup questions are answered there.

---

## Community support (free)

| Channel | Link | Best for |
|---------|------|----------|
| GitHub Discussions | https://github.com/orgs/nself-org/discussions | Questions, feature ideas, general help |
| Community chat | https://chat.nself.org | Real-time chat with the community and maintainers |
| GitHub Issues | https://github.com/nself-org/cli/issues | Bug reports (include `nself version` + `nself doctor` output) |

---

## Paid support

| Plan | SLA | How to reach us |
|------|-----|-----------------|
| Elite ($4.99/mo) | Email response | support@nself.org |
| Business ($9.99/mo) | 24-hour email | support@nself.org (priority queue) |
| Business+ ($49.99/mo) | Dedicated Slack channel | Invite sent after signup |
| Enterprise ($99.99/mo) | Managed DevOps | support@nself.org + shared runbook |

Upgrade at https://nself.org/pricing.

---

## Filing a useful bug report

Include all of the following:

```bash
nself version      # CLI version
nself doctor       # health summary
uname -a           # OS and kernel
docker version     # Docker version
```

Paste the full terminal output. Redact any secrets before posting.

---

## Related pages

- [[Errors]], error code catalog
- [[cmd-doctor]], automated diagnostics
- [[FAQ]], common questions
- [[Plugin-Licensing]], license key issues

---

← [[Home]] | [[_Sidebar]]
