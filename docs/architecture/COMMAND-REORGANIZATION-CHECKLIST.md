# Command Reorganization Implementation Checklist

**Reference:** [COMMAND-REORGANIZATION-PROPOSAL.md](./COMMAND-REORGANIZATION-PROPOSAL.md)

---

## Phase 1: Add New Commands (Weeks 1-2)

### Create New Command Files

- [ ] **Create `src/cli/observe.sh`** - Consolidate observability
  - [ ] Absorb logs.sh functionality
  - [ ] Absorb metrics.sh functionality
  - [ ] Absorb monitor.sh functionality
  - [ ] Absorb health.sh functionality
  - [ ] Absorb doctor.sh functionality
  - [ ] Absorb history.sh functionality
  - [ ] Absorb audit.sh functionality
  - [ ] Absorb urls.sh functionality
  - [ ] Absorb exec.sh functionality
  - [ ] Add comprehensive help text
  - [ ] Add subcommand routing

- [ ] **Create `src/cli/secure.sh`** - Consolidate security
  - [ ] Absorb security.sh functionality
  - [ ] Absorb secrets.sh functionality
  - [ ] Absorb vault.sh functionality
  - [ ] Absorb ssl.sh functionality
  - [ ] Absorb trust.sh functionality
  - [ ] Add comprehensive help text
  - [ ] Add subcommand routing

### Expand Existing Command Files

- [ ] **Expand `src/cli/auth.sh`** - Auth & authorization hub
  - [ ] Add `oauth` subcommand (from oauth.sh)
  - [ ] Add `mfa` subcommand (from mfa.sh)
  - [ ] Add `devices` subcommand (from devices.sh)
  - [ ] Add `roles` subcommand (from roles.sh)
  - [ ] Add `webhooks` subcommand (from webhooks.sh)
  - [ ] Update help text with all subcommands
  - [ ] Add routing logic for subcommands

- [ ] **Expand `src/cli/service.sh`** - All optional services
  - [ ] Add `admin` subcommand (from admin.sh, admin-dev.sh)
  - [ ] Add `email` subcommand (from email.sh)
  - [ ] Add `search` subcommand (from search.sh)
  - [ ] Add `functions` subcommand (from functions.sh)
  - [ ] Add `mlflow` subcommand (from mlflow.sh)
  - [ ] Add `storage` subcommand (from storage.sh)
  - [ ] Add `cache` subcommand (from redis.sh)
  - [ ] Add `realtime` subcommand (from realtime.sh)
  - [ ] Add `rate-limit` subcommand (from rate-limit.sh)
  - [ ] Update help text
  - [ ] Add routing logic

- [ ] **Expand `src/cli/deploy.sh`** - All deployment operations
  - [ ] Add `env` subcommand (from env.sh)
  - [ ] Add `sync` subcommand (from sync.sh)
  - [ ] Add `validate` subcommand (from validate.sh)
  - [ ] Keep existing: staging, production, rollback, preview, canary, blue-green
  - [ ] Update help text
  - [ ] Add routing logic

- [ ] **Expand `src/cli/cloud.sh`** - Infrastructure hub
  - [ ] Add `k8s` subcommand (from k8s.sh)
  - [ ] Add `helm` subcommand (from helm.sh)
  - [ ] Keep existing: provider, server, cost, deploy
  - [ ] Update help text
  - [ ] Add routing logic

- [ ] **Expand `src/cli/dev.sh`** - Developer tools
  - [ ] Add `perf` subcommand (from perf.sh)
  - [ ] Add `bench` subcommand (from bench.sh)
  - [ ] Add `scale` subcommand (from scale.sh)
  - [ ] Add `frontend` subcommand (from frontend.sh)
  - [ ] Add `ci` subcommand (from ci.sh)
  - [ ] Add `completion` subcommand (from completion.sh)
  - [ ] Update help text
  - [ ] Add routing logic

- [ ] **Expand `src/cli/config.sh`** - Configuration management
  - [ ] Add `clean` subcommand (from clean.sh)
  - [ ] Add `reset` subcommand (from reset.sh)
  - [ ] Keep existing: show, get, set, list, edit, validate, diff, export, import
  - [ ] Update help text
  - [ ] Add routing logic

- [ ] **Update `src/cli/db.sh`** - Minor enhancements
  - [ ] Ensure `backup` subcommand exists
  - [ ] Ensure `restore` subcommand exists
  - [ ] Ensure `migrate` subcommand exists
  - [ ] Update help text if needed

- [ ] **Update `src/cli/tenant.sh`** - Minor enhancements
  - [ ] Ensure `billing` subcommand is clear
  - [ ] Ensure `branding` subcommand is clear
  - [ ] Update help text if needed

### Create Legacy Alias System

- [ ] **Create `src/cli/legacy/` directory**
  - [ ] Create README explaining legacy alias system

- [ ] **Create legacy alias handler script**
  - [ ] `src/cli/legacy/alias-handler.sh`
  - [ ] Function to show deprecation warning
  - [ ] Function to redirect to new command
  - [ ] Telemetry tracking (opt-in)

- [ ] **Create individual legacy alias files**
  - [ ] `src/cli/legacy/oauth.sh` → auth oauth
  - [ ] `src/cli/legacy/mfa.sh` → auth mfa
  - [ ] `src/cli/legacy/devices.sh` → auth devices
  - [ ] `src/cli/legacy/roles.sh` → auth roles
  - [ ] `src/cli/legacy/webhooks.sh` → auth webhooks
  - [ ] `src/cli/legacy/admin.sh` → service admin
  - [ ] `src/cli/legacy/admin-dev.sh` → service admin dev
  - [ ] `src/cli/legacy/email.sh` → service email
  - [ ] `src/cli/legacy/search.sh` → service search
  - [ ] `src/cli/legacy/functions.sh` → service functions
  - [ ] `src/cli/legacy/mlflow.sh` → service mlflow
  - [ ] `src/cli/legacy/storage.sh` → service storage
  - [ ] `src/cli/legacy/redis.sh` → service cache
  - [ ] `src/cli/legacy/realtime.sh` → service realtime
  - [ ] `src/cli/legacy/rate-limit.sh` → service rate-limit
  - [ ] `src/cli/legacy/staging.sh` → deploy staging
  - [ ] `src/cli/legacy/prod.sh` → deploy production
  - [ ] `src/cli/legacy/rollback.sh` → deploy rollback
  - [ ] `src/cli/legacy/env.sh` → deploy env
  - [ ] `src/cli/legacy/sync.sh` → deploy sync
  - [ ] `src/cli/legacy/validate.sh` → deploy validate
  - [ ] `src/cli/legacy/providers.sh` → cloud provider
  - [ ] `src/cli/legacy/provision.sh` → cloud server create
  - [ ] `src/cli/legacy/server.sh` → cloud server
  - [ ] `src/cli/legacy/servers.sh` → cloud server list
  - [ ] `src/cli/legacy/k8s.sh` → cloud k8s
  - [ ] `src/cli/legacy/helm.sh` → cloud helm
  - [ ] `src/cli/legacy/logs.sh` → observe logs
  - [ ] `src/cli/legacy/metrics.sh` → observe metrics
  - [ ] `src/cli/legacy/monitor.sh` → observe monitor
  - [ ] `src/cli/legacy/health.sh` → observe health
  - [ ] `src/cli/legacy/doctor.sh` → observe doctor
  - [ ] `src/cli/legacy/history.sh` → observe history
  - [ ] `src/cli/legacy/audit.sh` → observe audit
  - [ ] `src/cli/legacy/urls.sh` → observe urls
  - [ ] `src/cli/legacy/exec.sh` → observe exec
  - [ ] `src/cli/legacy/security.sh` → secure
  - [ ] `src/cli/legacy/secrets.sh` → secure secrets
  - [ ] `src/cli/legacy/vault.sh` → secure vault
  - [ ] `src/cli/legacy/ssl.sh` → secure ssl
  - [ ] `src/cli/legacy/trust.sh` → secure ssl trust
  - [ ] `src/cli/legacy/perf.sh` → dev perf
  - [ ] `src/cli/legacy/bench.sh` → dev bench
  - [ ] `src/cli/legacy/scale.sh` → dev scale
  - [ ] `src/cli/legacy/frontend.sh` → dev frontend
  - [ ] `src/cli/legacy/ci.sh` → dev ci
  - [ ] `src/cli/legacy/completion.sh` → dev completion
  - [ ] `src/cli/legacy/clean.sh` → config clean
  - [ ] `src/cli/legacy/reset.sh` → config reset
  - [ ] `src/cli/legacy/billing.sh` → tenant billing
  - [ ] `src/cli/legacy/whitelabel.sh` → tenant branding
  - [ ] `src/cli/legacy/org.sh` → tenant
  - [ ] `src/cli/legacy/backup.sh` → db backup
  - [ ] `src/cli/legacy/restore.sh` → db restore
  - [ ] `src/cli/legacy/migrate.sh` → db migrate

### Update Main CLI Dispatcher

- [ ] **Update `src/cli/nself.sh`**
  - [ ] Add routing for new categories (observe, secure)
  - [ ] Add legacy alias detection
  - [ ] Add deprecation warning system
  - [ ] Update command list

### Update Help System

- [ ] **Update `src/cli/help.sh`**
  - [ ] Show 13 categories instead of 77 commands
  - [ ] Add category-based help
  - [ ] Add search functionality
  - [ ] Update help text examples
  - [ ] Add "For help: nself help <category>"

### Update Shell Completion

- [ ] **Update `src/cli/completion.sh`**
  - [ ] Add completions for new structure
  - [ ] Support both old and new commands
  - [ ] Add category completions
  - [ ] Add subcommand completions for each category

### Testing

- [ ] **Test new command structure**
  - [ ] Test all `observe` subcommands
  - [ ] Test all `secure` subcommands
  - [ ] Test expanded `auth` subcommands
  - [ ] Test expanded `service` subcommands
  - [ ] Test expanded `deploy` subcommands
  - [ ] Test expanded `cloud` subcommands
  - [ ] Test expanded `dev` subcommands
  - [ ] Test expanded `config` subcommands

- [ ] **Test legacy aliases**
  - [ ] Test all 50+ legacy aliases redirect correctly
  - [ ] Verify deprecation warnings don't appear yet (Phase 1)
  - [ ] Test legacy aliases work identically to new commands

- [ ] **Test help system**
  - [ ] `nself help` shows 13 categories
  - [ ] `nself help auth` shows auth subcommands
  - [ ] `nself help observe` shows observe subcommands
  - [ ] `nself help secure` shows secure subcommands
  - [ ] All category help works

- [ ] **Test shell completion**
  - [ ] Tab completion for categories
  - [ ] Tab completion for subcommands
  - [ ] Tab completion for legacy commands
  - [ ] Works in bash, zsh, fish

### Documentation

- [ ] **Update main README**
  - [ ] Show new command structure
  - [ ] Add migration note
  - [ ] Link to migration guide

- [ ] **Create migration guide**
  - [ ] `docs/guides/COMMAND-MIGRATION.md`
  - [ ] Before/after examples
  - [ ] Full command mapping table
  - [ ] FAQs

- [ ] **Update command documentation**
  - [ ] Update `docs/commands/COMMANDS.md`
  - [ ] Create category-specific docs:
    - [ ] `docs/commands/OBSERVE.md`
    - [ ] `docs/commands/SECURE.md`
  - [ ] Update existing category docs:
    - [ ] `docs/commands/AUTH.md`
    - [ ] `docs/commands/SERVICE.md`
    - [ ] `docs/commands/DEPLOY.md`
    - [ ] `docs/commands/CLOUD.md`
    - [ ] `docs/commands/DEV.md`
    - [ ] `docs/commands/CONFIG.md`

- [ ] **Update release notes**
  - [ ] Add to `docs/releases/vX.Y.Z.md`
  - [ ] Mention new structure
  - [ ] Link to migration guide
  - [ ] Emphasize backward compatibility

---

## Phase 2: Add Deprecation Warnings (Weeks 3-6)

### Update Legacy Aliases

- [ ] **Enable deprecation warnings**
  - [ ] Modify `src/cli/legacy/alias-handler.sh`
  - [ ] Add environment variable to control warnings: `NSELF_HIDE_DEPRECATION_WARNINGS`
  - [ ] Format warning message clearly
  - [ ] Show new command syntax

### Example Deprecation Message

```bash
⚠️  DEPRECATED: 'nself logs' will be removed in v1.0
    Use: nself observe logs

    To hide this warning: export NSELF_HIDE_DEPRECATION_WARNINGS=1
    Migration guide: nself help migrate

[command continues normally...]
```

### Update Documentation

- [ ] **Emphasize new commands**
  - [ ] Update all examples to use new syntax
  - [ ] Add deprecation timeline to docs
  - [ ] Update quick start guide

- [ ] **Create deprecation announcement**
  - [ ] Blog post / release notes
  - [ ] GitHub discussion
  - [ ] Update website

### Testing

- [ ] **Verify deprecation warnings**
  - [ ] All legacy commands show warning
  - [ ] Warning format is consistent
  - [ ] Environment variable suppresses warning
  - [ ] Command still works correctly

---

## Phase 3: Legacy Alias System (Ongoing)

### Telemetry (Optional)

- [ ] **Add opt-in telemetry**
  - [ ] Track legacy command usage
  - [ ] Track new command usage
  - [ ] Privacy-preserving (no PII)
  - [ ] Help inform sunset timeline

### Monitor Usage

- [ ] **Create dashboard**
  - [ ] Show legacy vs new command usage
  - [ ] Track migration progress
  - [ ] Identify heavily-used legacy commands

### User Feedback

- [ ] **Collect feedback**
  - [ ] GitHub issues
  - [ ] User surveys
  - [ ] Community discussions
  - [ ] Adjust timeline if needed

---

## Phase 4: Remove Old Commands (6-12 months, v1.0+)

### Preparation

- [ ] **Final migration push**
  - [ ] Send migration reminders
  - [ ] Update all official examples
  - [ ] Update all documentation
  - [ ] Create migration tool if needed

### Remove Legacy Commands

- [ ] **Delete legacy alias files**
  - [ ] Remove `src/cli/legacy/` directory
  - [ ] Remove all old command files (if separate)
  - [ ] Keep only new structure

- [ ] **Update main dispatcher**
  - [ ] Remove legacy alias routing
  - [ ] Add helpful error messages for old commands
  - [ ] Point to migration guide

### Example Error Message

```bash
$ nself logs
Error: 'nself logs' has been removed in v1.0

Use:   nself observe logs

For full migration guide: nself help migrate
Or visit: https://nself.org/docs/migration
```

### Update Documentation

- [ ] **Remove legacy command docs**
  - [ ] Archive old command reference
  - [ ] Update all links
  - [ ] Add migration guide to archive

- [ ] **Major version release notes**
  - [ ] Highlight command structure changes
  - [ ] Emphasize migration
  - [ ] Provide support resources

### Testing

- [ ] **Verify removal**
  - [ ] All old commands return helpful error
  - [ ] All new commands work correctly
  - [ ] Help system only shows new structure
  - [ ] Shell completion only shows new commands

---

## General Guidelines

### Code Quality

- [ ] All new code follows existing patterns
- [ ] All functions have error handling
- [ ] All functions have help text
- [ ] Code is cross-platform compatible (Bash 3.2+)
- [ ] No use of `echo -e` (use `printf`)
- [ ] No Bash 4+ features

### Testing Standards

- [ ] Unit tests for new functions
- [ ] Integration tests for workflows
- [ ] Test on macOS (Bash 3.2)
- [ ] Test on Linux (Bash 5.x)
- [ ] Test in CI/CD pipeline

### Documentation Standards

- [ ] All new commands documented
- [ ] All examples tested
- [ ] Migration guide complete
- [ ] Help text is clear and concise

---

## Success Criteria

### Phase 1 Success

- [ ] All new commands work
- [ ] All legacy aliases work
- [ ] No breaking changes
- [ ] Tests pass
- [ ] Documentation updated

### Phase 2 Success

- [ ] Deprecation warnings appear
- [ ] Users can suppress warnings
- [ ] Commands still work
- [ ] Migration guide available

### Phase 3 Success

- [ ] Telemetry shows migration progress
- [ ] User feedback is positive
- [ ] Community adopts new structure

### Phase 4 Success

- [ ] Old commands removed
- [ ] Users migrated
- [ ] Documentation updated
- [ ] Clean codebase

---

## Rollback Plan

### If Issues Arise in Phase 1

- [ ] Disable new commands
- [ ] Keep only legacy commands
- [ ] Review feedback
- [ ] Iterate and retry

### If Issues Arise in Phase 2

- [ ] Remove deprecation warnings
- [ ] Extend timeline
- [ ] Address user concerns
- [ ] Improve migration guide

### If Issues Arise in Phase 3/4

- [ ] Delay removal
- [ ] Provide migration support
- [ ] Consider keeping legacy longer
- [ ] Community consultation

---

## Resources

- **Proposal**: [COMMAND-REORGANIZATION-PROPOSAL.md](./COMMAND-REORGANIZATION-PROPOSAL.md)
- **Visual Guide**: [COMMAND-REORGANIZATION-VISUAL.md](./COMMAND-REORGANIZATION-VISUAL.md)
- **Main Commands Doc**: [docs/commands/COMMANDS.md](../../commands/COMMANDS.md)

---

**Last Updated**: 2026-01-30
**Status**: Ready for implementation
**Estimated Timeline**: 6-12 months for full migration
