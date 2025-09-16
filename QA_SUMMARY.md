# nself Cross-Platform QA Summary

## Date: September 16, 2024
## Version: 0.3.9-patch3

## Executive Summary

This document analyzes cross-platform compatibility issues identified during user testing, particularly on Rocky Linux, and provides recommendations for making nself more robust across different environments.

## Identified Issues

### 1. Environment File Detection Problems

#### Issue
- Build command fails to detect `.env.local` files (fixed in patch)
- Loading priority was inconsistent across different commands
- Some users had `.env.local` instead of `.env` from older versions

#### Root Cause
- Original design removed `.env.local` support to simplify
- Not all code paths checked for alternative env file names
- Environment loading cascade was incomplete

#### Status: PARTIALLY FIXED
- Added `.env.local` support back
- Still needs testing across all Linux distributions

### 2. Build Process Silent Failures

#### Issue
- `nself build` exits after "Configuration validated" without creating files
- No error messages displayed when build fails
- Docker-compose.yml not generated despite no errors

#### Root Cause
- `needs_work` variable not properly set when no existing files found
- Complex conditional logic failing silently
- Missing fallback for fresh project detection

#### Status: FIXED (needs verification)
- Added force build for fresh projects
- Added debug output capability
- Still needs Rocky Linux confirmation

### 3. Path Resolution Issues

#### Issue
- nself command may not resolve correctly on different Linux distributions
- Symlink resolution differs between macOS and Linux
- Installation path detection varies by system

#### Potential Problems
```bash
# Current path resolution in bin/nself
SCRIPT_PATH="${BASH_SOURCE[0]}"
while [ -h "$SCRIPT_PATH" ]; do
  DIR="$(cd -P "$(dirname "$SCRIPT_PATH")" && pwd)"
  SCRIPT_PATH="$(readlink "$SCRIPT_PATH")"  # May need readlink -f on Linux
  [[ $SCRIPT_PATH != /* ]] && SCRIPT_PATH="$DIR/$SCRIPT_PATH"
done
```

#### Recommendations
- Use `readlink -f` where available (GNU coreutils)
- Add fallback for systems without `-f` flag
- Test on BSD systems (macOS) vs GNU systems (Linux)

### 4. Shell Compatibility Issues

#### Issue
- Bashisms that may not work in all shells
- Array syntax differences
- String comparison operators vary

#### Specific Problems Found
```bash
# These constructs may fail on some systems:
[[ ]] vs [ ]           # Extended test vs standard test
${var:-default}        # Parameter expansion
array+=("element")     # Array append syntax
$((arithmetic))        # Arithmetic expansion
```

#### Recommendations
- Explicitly require bash 4.0+
- Add version check at startup
- Document minimum bash version

### 5. Docker Command Variations

#### Issue
- `docker compose` (v2) vs `docker-compose` (v1) command differences
- Docker Desktop vs Docker CE behavioral differences
- Rootless Docker complications

#### Current Implementation
```bash
# compose() function in docker.sh tries both but may fail silently
if docker compose version >/dev/null 2>&1; then
  docker compose "$@"
elif command -v docker-compose >/dev/null 2>&1; then
  docker-compose "$@"
```

#### Recommendations
- Better error reporting when neither works
- Check Docker daemon actually running, not just installed
- Handle permission errors explicitly

### 6. File System Assumptions

#### Issue
- Assumes certain directories are writable
- Case sensitivity differences (macOS vs Linux)
- Different temp directory locations

#### Problems
- `/tmp` may not be available in containers
- Home directory structure varies
- Permission models differ between systems

### 7. Network Configuration Variations

#### Issue
- Port availability detection may fail
- Network interfaces named differently
- Firewall rules block differently

#### Current Problems
- Port checking assumes `lsof` or `ss` available
- Network creation may fail in restricted environments
- IPv6 vs IPv4 handling inconsistent

### 8. Environment Variable Handling

#### Issue
- Variable expansion behaves differently
- Export behavior varies
- Sourcing files may fail silently

#### Specific Problems
```bash
set -a; source .env; set +a  # May not work in all shells
export -f function_name       # Bash-specific
${!var}                       # Indirect expansion
```

## Platform-Specific Issues

### Rocky Linux / RHEL / CentOS
- SELinux may block Docker operations
- Different package names (docker-ce vs podman-docker)
- Firewalld default rules
- Different user/group permissions model

### Ubuntu / Debian
- Snap version of Docker has restrictions
- AppArmor conflicts
- Different default shell (dash vs bash)

### macOS
- Docker Desktop resource limits
- Different sed syntax (BSD vs GNU)
- Case-insensitive filesystem by default
- Different readlink behavior

### Alpine / Minimal Distros
- Missing bash by default (uses sh/ash)
- Limited coreutils
- Different package manager

### WSL2 (Windows)
- Path translation issues
- File permission mapping problems
- Docker Desktop integration quirks
- Line ending issues (CRLF vs LF)

## Testing Gaps

### Current Testing Coverage
- ✅ macOS development environment
- ✅ Ubuntu CI/CD (if configured)
- ⚠️ Limited Rocky Linux testing
- ❌ No Alpine testing
- ❌ No BSD testing
- ❌ No WSL2 testing

### Recommended Test Matrix
1. Operating Systems: macOS, Ubuntu LTS, Rocky 8/9, Alpine, WSL2
2. Bash versions: 3.2 (macOS default), 4.x, 5.x
3. Docker versions: 20.x, 24.x, 25.x, Docker Desktop
4. Scenarios: Fresh install, upgrade, .env.local migration

## Immediate Fixes Needed

### High Priority
1. **Better error messages** - Never fail silently
2. **Debug mode everywhere** - Add DEBUG=true support to all commands
3. **Preflight checks** - Validate environment before attempting operations
4. **Path resolution** - Make symlink resolution more robust

### Medium Priority
1. **Shell version check** - Verify bash 4.0+ at startup
2. **Docker compatibility layer** - Better compose v1/v2 handling
3. **Environment file detection** - Support multiple naming conventions
4. **Cross-platform sed** - Detect and use appropriate syntax

### Low Priority
1. **Alternative shell support** - Document bash-only requirement
2. **Podman support** - Consider Docker alternatives
3. **Non-standard paths** - Support custom installation locations

## Proposed Solutions

### 1. Enhanced Startup Checks
```bash
# Add to nself.sh
check_requirements() {
  # Check bash version
  if [[ "${BASH_VERSION%%.*}" -lt 4 ]]; then
    echo "Error: Bash 4.0+ required" >&2
    exit 1
  fi

  # Check for required commands
  for cmd in docker git curl; do
    if ! command -v "$cmd" >/dev/null 2>&1; then
      echo "Error: $cmd is required but not found" >&2
      exit 1
    fi
  done
}
```

### 2. Universal Path Resolution
```bash
# Portable readlink replacement
portable_readlink() {
  if readlink -f "$1" 2>/dev/null; then
    return
  elif python -c "import os; print(os.path.realpath('$1'))" 2>/dev/null; then
    return
  elif perl -e "use Cwd 'abs_path'; print abs_path('$1')" 2>/dev/null; then
    return
  else
    echo "$1"  # Fallback to original path
  fi
}
```

### 3. Better Error Reporting
```bash
# Add to all commands
set -euo pipefail  # Exit on error, undefined vars, pipe failures
trap 'echo "Error at line $LINENO: $BASH_COMMAND" >&2' ERR
```

### 4. Cross-Platform sed Wrapper
```bash
portable_sed() {
  if [[ "$OSTYPE" == "darwin"* ]]; then
    sed -i '' "$@"
  else
    sed -i "$@"
  fi
}
```

## User Feedback Integration

### From GitHub Discussion #16
- User on Rocky Linux cannot generate docker-compose.yml
- Build completes but creates no files
- "System requirements not met" error unclear
- Debug mode not providing enough information

### Action Items
1. ✅ Add `.env.local` support
2. ✅ Force build on fresh projects
3. ✅ Create debug script for troubleshooting
4. ⏳ Await user feedback with debug output
5. 🔄 Iterate based on findings

## Documentation Improvements Needed

### Installation Guide
- Add platform-specific sections
- Document minimum requirements clearly
- Include troubleshooting for each OS

### Error Messages
- Make them actionable
- Include likely causes
- Provide specific commands to fix

### Debug Guide
- How to enable DEBUG mode
- What information to collect
- Where to report issues

## Monitoring & Metrics

### What to Track
- Platform distribution of users
- Common error messages
- Build success/failure rates
- Time to successful deployment

### How to Track
- Anonymous telemetry (opt-in)
- GitHub issue patterns
- Community feedback
- CI/CD test results

## Conclusion

The primary issues stem from:
1. **Silent failures** - Operations fail without clear error messages
2. **Platform assumptions** - Code assumes macOS/Ubuntu-like environment
3. **Missing fallbacks** - No graceful degradation when expected tools missing
4. **Complex detection logic** - Too many conditional paths that can fail

The Rocky Linux issue is likely a combination of:
- Environment file not being detected properly
- Build logic incorrectly determining no work needed
- Possible path resolution issues with the nself installation

Next steps:
1. Wait for debug script output from user
2. Add comprehensive error reporting
3. Simplify detection logic
4. Add platform-specific test coverage
5. Create CI/CD matrix for multiple platforms

---

*This document will be updated as more cross-platform issues are identified and resolved.*