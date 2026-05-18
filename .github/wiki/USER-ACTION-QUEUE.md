# User Action Queue

Items in this queue require manual action by a repository administrator. Each item has a unique ID (UA-NN) and describes the configuration needed to unlock a CI/CD feature.

Items are ordered by priority. Complete them at your own pace — CI passes cleanly without any of them (steps that depend on absent secrets skip gracefully).

---

## UA-07 — Provision secrets for dispatch-monitor workflow

**Status:** Pending  
**Priority:** Low  
**Workflow:** `.github/workflows/ci.yml` — `dispatch-monitor` job  

### What this unlocks

Three optional CI features activate when the secrets below are provisioned:

| Feature | Requires |
|---------|---------|
| Slack build-status notifications | `SLACK_WEBHOOK` |
| GitHub App installation token generation | `GITHUB_APP_PRIVATE_KEY` + `NSELF_DISPATCH_APP_ID` |
| Downstream `repository_dispatch` to `nself-org/plugins` on main-branch push | Both App secrets |

Without these secrets, the `dispatch-monitor` job runs and exits cleanly — each step logs `INFO: … not configured — skipping (see UA-07)`. No failures, no red checks.

### Steps to provision

**1. Slack webhook**

1. Go to your Slack workspace, open **Settings** for the target channel.
2. Under **Integrations**, add an **Incoming Webhook** app.
3. Copy the generated URL (`https://hooks.slack.com/services/…`).
4. In GitHub: **Settings** > **Secrets and variables** > **Actions** > **New repository secret**.
   - Name: `SLACK_WEBHOOK`
   - Value: the Incoming Webhook URL

**2. GitHub App secrets**

These two secrets come from the same GitHub App.

1. If no App exists yet: go to **GitHub** > **Settings** > **Developer settings** > **GitHub Apps** > **New GitHub App**.
   - Name it something like `nself-ci-dispatch`.
   - Set **Repository permissions** > **Contents: Read** and **Metadata: Read** on the target repos (`nself-org/plugins`).
   - Under **Webhook**, uncheck "Active" (this App does not need webhooks).
   - Click **Create GitHub App**.
2. On the App's settings page:
   - Note the **App ID** integer — this goes into `NSELF_DISPATCH_APP_ID`.
   - Scroll to **Private keys** > **Generate a private key**. Download the `.pem` file.
   - Install the App on the `nself-org` organization (or the specific repos it needs to dispatch to).
3. In GitHub: **Settings** > **Secrets and variables** > **Actions** > add two secrets:
   - `NSELF_DISPATCH_APP_ID`: the integer App ID (e.g. `123456`)
   - `GITHUB_APP_PRIVATE_KEY`: the full contents of the downloaded `.pem` file (including `-----BEGIN RSA PRIVATE KEY-----` header/footer)

### Verification

After provisioning, push any commit to `main`. The `dispatch-monitor` job should:

- Send a Slack message to the configured channel.
- Log `INFO: installation token generated`.
- Log `INFO: repository_dispatch sent to nself-org/plugins`.

If only `SLACK_WEBHOOK` is provisioned, only the Slack step runs; the App steps still skip.

### Related

- Workflow: `.github/workflows/ci.yml` — `dispatch-monitor` job
- Sprint: S26.T12 (CI workflow scaffold — secret-conditional steps)

---

## Adding new items

When a new user-action item is identified, append it here in the format:

```markdown
## UA-NN — Short description

**Status:** Pending | Done  
**Priority:** Critical | High | Medium | Low  
**Workflow / Component:** …

### What this unlocks
…

### Steps to provision
…
```

---

[[Home]] | [[Commands]] | [[Architecture]]
