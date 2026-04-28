# Scoop Packaging (stub)

Status: **stub** — publish target is v1.1.0.

This directory holds the Scoop bucket manifest for the `nself` CLI on Windows. The publish target is a separate bucket repo: `nself-org/scoop-nself` (deferred).

`nself.json` is wired with `checkver` + `autoupdate` so once the bucket repo is created, Scoop's `scoop bucket update` will pick up new GitHub releases automatically — no manual version bump after the initial publish.

## Publish-day checklist (do not run yet)

- [ ] Create `nself-org/scoop-nself` GitHub repo (public).
- [ ] Copy `nself.json` to the bucket repo root as `bucket/nself.json` (or `nself.json` at root, depending on chosen layout).
- [ ] Run `scoop bucket add nself-org https://github.com/nself-org/scoop-nself`.
- [ ] Verify `scoop install nself` from a clean Windows VM.
- [ ] Add the bucket to https://github.com/ScoopInstaller/Scoop/discussions/categories/buckets-tracker (community discoverability).

## Blockers (carry to user)

- `nself-org/scoop-nself` repo not yet created. User action required.
- No API key needed (Scoop bucket = plain Git repo), so once the repo exists this is fully automated.

## Local validation

```powershell
# In a Windows shell, after filling version + hashes:
scoop install ./nself.json
scoop list nself
```
