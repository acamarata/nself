# Gemini OAuth Re-Auth, Operations Quick Reference

> Operator quick-reference for re-authorizing Gemini accounts in the Gemini Free Pool (GFP) after token expiry. For the full how-to with screenshots and consent flow walkthrough, see [[Plugins-AI-OAuth]].

The GFP rotates Google OAuth-linked Gemini accounts to spread free-tier traffic. Tokens are not permanent and require re-auth on a predictable set of triggers.

## When Re-Auth Is Required

| Trigger | Pool status | Cause |
|---------|-------------|-------|
| 401 from Gemini API | `expired` | Refresh token revoked or stale |
| `invalid_grant` from Google token endpoint | `invalid` | Refresh token no longer accepted |
| Scope drift | `scope_mismatch` | Required Gemini scope added or rotated |
| Operator-initiated rotation | `revoked` | Account removed by operator |

## Re-Auth Flow

The pool detects a 401 on the next key test, invalidates the cached token, and surfaces the affected account in `nself ai pool status --verbose`. Operators run:

```bash
nself plugin auth gemini --account user@example.com
```

The CLI opens the Google OAuth consent URL, displays the device code on stdout, and waits for the operator to authorize. On success the new refresh token is encrypted with the project secret and persisted to the pool. Pool tests resume on the next scheduled cycle.

## Verification

```bash
nself ai pool status --verbose          # status field returns to active
nself ai chat --provider gemini "ping"  # round-trips a real Gemini call
```

## Cross-References

- [[Plugins-AI-OAuth]], full reference (consent flow, scope list, troubleshooting)
- [[plugin-google]], google plugin and shared OAuth client
- [[ai-pool]], `nself ai pool` command reference
- [[Home]]
