# Release Checklist for v0.3.0

## Pre-Release Tasks ✅

- [x] Update version in `/src/VERSION` to `0.3.0`
- [x] Update version in README.md badge
- [x] Create/update CHANGELOG.md in `/docs`
- [x] Create comprehensive RELEASES.md with all version history
- [x] Fix all critical bugs identified in testing
- [x] Ensure all commands work with new architecture
- [x] Test installation and upgrade paths
- [x] Update documentation to reflect new structure

## Code Changes Summary

### Architecture Changes
- [x] Refactored from monolithic to modular src-centric architecture
- [x] Moved all implementation to `/src`
- [x] Created thin shims in `/bin`
- [x] Organized templates, certs, and libraries under `/src`

### Bug Fixes
- [x] Fixed HASURA_GRAPHQL_ADMIN_SECRET validation
- [x] Fixed auto-fix subsystem to use absolute paths
- [x] Fixed bash 3.2 compatibility for macOS
- [x] Fixed port conflict detection
- [x] Fixed Go module build errors
- [x] Fixed safety checks for repo detection

### New Features
- [x] Comprehensive error handling system
- [x] Auto-fix capabilities for common issues
- [x] Enhanced doctor command
- [x] Improved status and logging commands
- [x] Interactive recovery options

## Git Tasks (For User to Complete)

### 1. Stage Changes
```bash
git add .
git status  # Review all changes
```

### 2. Commit with Version Tag
```bash
git commit -m "Release v0.3.0: Major architectural refactor to modular src-centric design

BREAKING CHANGES:
- Complete restructuring of codebase from monolithic to modular
- All implementation moved to /src directory
- /bin now contains only thin delegation scripts

Features:
- Comprehensive error handling with auto-fix capabilities
- Modular build system (1078 lines → organized modules)
- Enhanced doctor, status, and logs commands
- Interactive recovery for port conflicts and build errors
- Full bash 3.2 compatibility for macOS

Fixes:
- HASURA_GRAPHQL_ADMIN_SECRET validation
- Auto-fix subsystem path resolution
- Port conflict detection using configured values
- Go module dependency resolution
- Installation upgrade path from v0.2.x

See docs/RELEASES.md for full details"
```

### 3. Create Git Tag
```bash
git tag -a v0.3.0 -m "Release v0.3.0: Major architectural refactor"
```

### 4. Push Changes
```bash
git push origin main
git push origin v0.3.0
```

## GitHub Release (For User to Complete)

### 1. Go to GitHub Releases
Navigate to: https://github.com/acamarata/nself/releases/new

### 2. Create Release
- **Tag:** v0.3.0
- **Title:** v0.3.0 - Major Architectural Refactor
- **Description:** Copy from `/docs/RELEASES.md` v0.3.0 section

### 3. Attach Assets (Optional)
- Consider creating a tarball of the release

## Post-Release Tasks

- [ ] Verify installation script works with new release
- [ ] Update any external documentation or wikis
- [ ] Announce release (if applicable)
- [ ] Monitor for any immediate issues

## Testing Commands

Run these to verify everything works:
```bash
# Test version
nself version  # Should show 0.3.0

# Test in clean environment
cd /tmp
rm -rf test-nself
mkdir test-nself && cd test-nself
nself init --name test --domain localhost
nself build
nself doctor
```

## Notes

This is a **BREAKING CHANGE** release. Users upgrading from v0.2.x will have their installations automatically updated when running `nself update`, but they should be aware of the significant structural changes.