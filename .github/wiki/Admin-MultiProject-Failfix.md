# Admin Multi-Project Fail-Fix Playbook

Common issues with the admin multi-project switcher and their fixes.

## Dropdown empty

**Cause:** `~/.config/nself/admin/projects.json` is missing or malformed.

**Fix:**

1. Check if the file exists: `cat ~/.config/nself/admin/projects.json`
2. If missing, re-register projects:
   ```bash
   nself admin projects add --name nself --host nself-prod --ssh-user ops
   nself admin projects add --name unity --host unity-prod
   nself admin projects add --name ummat --host ummat-prod
   nself admin projects add --name camarata --host nclaw-prod --ssh-user ops
   ```
3. If malformed, delete and re-register: `rm ~/.config/nself/admin/projects.json`

## ACL bypass possible

**Cause:** Middleware ordering. The ACL check must run before the project loader.

**Fix:** Verify middleware order in admin server:

```
auth -> ACL check -> project loader -> audit -> handler
```

If the project loader runs before ACL, it loads project context for unauthorized users. The ACL middleware must return 403 `project_not_authorized` before any project data is touched.

## Switch leaves stale data

**Cause:** Client-side store not cleared on project switch.

**Fix:** The admin UI must dispatch a `reset()` action on the client-side store when the project selector changes. Every store slice (env vars, plugins, metrics) must be cleared before loading new project data.

Verify: switch between two projects with different plugin lists. If the old project's plugins flash briefly, the store is not being reset.
