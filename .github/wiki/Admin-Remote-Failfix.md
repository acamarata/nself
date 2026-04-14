# Admin Remote Fail-Fix Playbook

Common issues with `nself admin connect` and their fixes.

## Tunnel drops every 5 minutes

**Cause:** SSH keepalive not configured.

**Fix:** Add to `~/.ssh/config`:

```
Host *
    ServerAliveInterval 30
    ServerAliveCountMax 3
```

The CLI passes `ServerAliveInterval=30` automatically, but the SSH config override ensures it works for manual connections too.

## Audit rows missing

**Cause:** Middleware ordering. The audit middleware must be the last middleware before the handler.

**Fix:** Check the admin server middleware chain. Audit middleware captures response status and duration, so it must wrap the final handler:

```
auth -> ACL -> [other middleware] -> audit -> handler
```

If audit is before ACL, rejected requests may not be logged. If audit is after the handler, it never runs.

## Admin port bound to 0.0.0.0

**Cause:** Configuration defaults to all interfaces instead of localhost.

**Fix:** Search for the misconfiguration:

```bash
grep -r "0.0.0.0:3021" admin/
```

Replace every instance with `127.0.0.1:3021`. The admin UI must never be network-accessible. External access goes through the SSH tunnel only.

The `AdminPortExternallyReachable` alert fires if the port is reachable from outside, with P1 severity after 5 minutes.
