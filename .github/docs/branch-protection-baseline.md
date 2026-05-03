# Branch Protection Baseline

Documents the required status checks and protection settings for `main` on all
active nSelf repos. Last updated by Sprint S98-02 (T16) on 2026-05-03.

---

## nself-org/cli — main

| Setting | Value |
|---------|-------|
| Require status checks to pass | Yes |
| Require branches to be up to date | No (strict: false) |
| Require PR reviews | Yes — 1 approving review |
| Dismiss stale reviews | No |
| Require code owner reviews | No |
| Enforce admins | Yes |
| Require linear history | Yes |
| Allow force pushes | No |
| Allow deletions | No |

### Required status checks

| Check name | Workflow file | Job id | Purpose |
|------------|--------------|--------|---------|
| `Gitleaks` | (app-level — gitleaks GitHub App) | — | Secret leak scanner |
| `Verify all 12 version files are in lockstep` | `.github/workflows/version-lockstep.yml` | `lockstep` | Ensures cli/.github/VERSION matches every downstream version constant; blocks PRs that bump version in only one place |
| `Forbid disallowed files (root + tracked junk)` | `.github/workflows/clean-root.yml` | `check` | Enforces Clean Repo Root Hard Rule; blocks .DS_Store, stray .md at root, tracked .env secrets, missing .gitignore baseline patterns; also verifies cli/sdk/ 4-language layout |

### To update required checks via CLI

```bash
gh api repos/nself-org/cli/branches/main/protection/required_status_checks \
  -X PATCH \
  --field 'strict=false' \
  -f 'contexts[]=Gitleaks' \
  -f 'contexts[]=Verify all 12 version files are in lockstep' \
  -f 'contexts[]=Forbid disallowed files (root + tracked junk)'
```

---

## nself-org/admin — main

| Setting | Value |
|---------|-------|
| Require status checks to pass | Yes |
| Require branches to be up to date | No (strict: false) |
| Require PR reviews | Yes — 1 approving review |
| Dismiss stale reviews | No |
| Require code owner reviews | No |
| Enforce admins | Yes |
| Require linear history | Yes |
| Allow force pushes | No |
| Allow deletions | No |

### Required status checks

| Check name | Workflow file | Job id | Purpose |
|------------|--------------|--------|---------|
| `Gitleaks` | (app-level — gitleaks GitHub App) | — | Secret leak scanner |
| `Verify admin version == CLI version` | `.github/workflows/version-lockstep.yml` | `lockstep` | Asserts admin `package.json` version == CLI version constant in `lib/cli-version.ts`; enforces cli=admin lockstep from P93 |
| `Forbid disallowed files (root + tracked junk)` | `.github/workflows/clean-root.yml` | `check` | Same Clean Repo Root gate as cli |

### To update required checks via CLI

```bash
gh api repos/nself-org/admin/branches/main/protection/required_status_checks \
  -X PATCH \
  --field 'strict=false' \
  -f 'contexts[]=Gitleaks' \
  -f 'contexts[]=Verify admin version == CLI version' \
  -f 'contexts[]=Forbid disallowed files (root + tracked junk)'
```

---

## Adding a new required check

1. Add the workflow to `.github/workflows/` and verify it runs on `pull_request` targeting `main`.
2. Note the job `name:` field — that is the check context string GitHub registers.
3. Run the PATCH command above with the new context appended to the `-f 'contexts[]='` list.
4. Update this doc with the new row in the relevant table.

Do NOT use the GitHub web UI to manage required checks — it resets on org policy changes.
CLI management via `gh api` is the source of truth.

---

## Minimum baseline for any new nSelf repo

Every new repo added to `nself-org` must reach this baseline within the first PR:

1. `Gitleaks` — install the gitleaks GitHub App on the repo in org settings.
2. `clean-root` (job: `Forbid disallowed files (root + tracked junk)`) — copy `.github/workflows/clean-root.yml` from `cli/`.
3. 1 required PR review.
4. Enforce admins: on.
5. Require linear history: on.
6. Allow force pushes: off.

Register checks via:

```bash
gh api repos/nself-org/<REPO>/branches/main/protection/required_status_checks \
  -X PATCH \
  --field 'strict=false' \
  -f 'contexts[]=Gitleaks' \
  -f 'contexts[]=Forbid disallowed files (root + tracked junk)'
```
