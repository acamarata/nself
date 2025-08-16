# Release Checklist Template for vX.X.X

Copy this template to `RELEASE_CHECKLIST_vX.X.X.MD` for each release.

## Pre-Release Tasks

### Code Quality
- [ ] All tests passing (`cd src/tests && ./run_tests.sh`)
- [ ] ShellCheck clean (no warnings)
- [ ] No TODO/FIXME comments for critical issues
- [ ] Documentation updated

### Version Updates
- [ ] Update `src/VERSION` to `vX.X.X`
- [ ] Update `docs/CHANGELOG.MD` with release notes
- [ ] Update README.md version badge (if shown)
- [ ] Create this checklist as `RELEASE_CHECKLIST_vX.X.X.MD`

### Testing
- [ ] Installation test on clean Linux VM
- [ ] Installation test on clean macOS
- [ ] Installation test on WSL (Windows)
- [ ] Upgrade test from previous version
- [ ] All commands work (`nself help`, `nself doctor`, etc.)

## Release Tasks

### 1. Final Preparation
```bash
# Ensure clean state
git status  # Should be clean
git checkout main
git pull origin main

# Final test run
cd src/tests && ./run_tests.sh
```

### 2. Create Release
```bash
# Use automated script
./scripts/release.sh vX.X.X

# Or manually:
git tag -a vX.X.X -m "Release vX.X.X: <brief description>"
git push origin main
git push origin vX.X.X
```

### 3. GitHub Release
- [ ] Go to: https://github.com/acamarata/nself/releases/new
- [ ] Select tag: vX.X.X
- [ ] Title: nself vX.X.X - <Brief Description>
- [ ] Copy release notes from CHANGELOG.MD
- [ ] Publish as latest release

### 4. Distribution Channels
- [ ] Update Homebrew formula (if applicable)
- [ ] Build and push Docker image (if applicable)
- [ ] Update other package managers as needed

## Post-Release Tasks

### Immediate Verification
- [ ] Installation works: `curl -fsSL .../install.sh | bash`
- [ ] Version correct: `nself version` shows vX.X.X
- [ ] GitHub release visible and tagged correctly
- [ ] No immediate critical issues reported

### Communication
- [ ] Update nself.org website (if applicable)
- [ ] Post announcement on social media
- [ ] Update Discord/Slack channels
- [ ] Email to mailing list (if applicable)

### Monitoring (First 24-48 hours)
- [ ] Monitor GitHub Issues for problems
- [ ] Check download statistics
- [ ] Respond to user feedback
- [ ] Prepare hotfix if critical issues found

## Rollback Plan

If critical issue discovered:
```bash
# Option 1: Hotfix (preferred)
git checkout -b hotfix/vX.X.X.1 vX.X.X
# Make fixes
git commit -am "Hotfix: <description>"
git tag -a vX.X.X.1 -m "Hotfix release"
git push origin vX.X.X.1

# Option 2: Rollback (emergency only)
gh release delete vX.X.X --yes
git tag -d vX.X.X
git push origin :vX.X.X
# Fix and re-release
```

## Sign-off

- [ ] Release completed successfully
- [ ] All verification steps passed
- [ ] Documentation updated
- [ ] Team notified

---

**Release Manager:** _________________
**Date:** _________________
**Version:** vX.X.X
**Status:** [ ] Success [ ] Partial [ ] Rolled Back

## Notes
_Add any release-specific notes, issues, or observations here_