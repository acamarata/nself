# nself CLI Help Contract (v0.9.8)

This document defines the canonical help behavior contract for all nself CLI commands.

## Requirements

### 1. Exit Code

Every command MUST exit `0` when invoked with `--help` or `-h`.

### 2. No Side Effects

Help output MUST NOT:
- Require project initialization (`.env.local` existence)
- Require Docker to be running
- Create, modify, or delete any files
- Execute any pre-command hooks (init guards, Docker checks, env loading)
- Make any network requests

### 3. Guard Bypass

The `--help` / `-h` check MUST execute BEFORE any `pre_command` call. Pattern:

```bash
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  # Help is read-only - bypass init/env guards
  for _arg in "$@"; do
    if [[ "$_arg" == "--help" ]] || [[ "$_arg" == "-h" ]]; then
      show_<command>_help
      exit 0
    fi
  done
  pre_command "<command>" || exit $?
  # ... normal execution
fi
```

### 4. Output Schema

Help output SHOULD include these elements (in order):

1. **Usage line** (REQUIRED): `Usage: nself <command> [subcommand] [options]`
2. **Description** (REQUIRED): Brief one-line summary of what the command does
3. **Subcommands** (if applicable): List of available subcommands
4. **Options** (REQUIRED): Available flags and options including `-h, --help`
5. **Examples** (recommended): Common usage examples

### 5. Deprecated Commands

Deprecated wrapper commands MUST:
- Show a deprecation notice pointing to the new command location
- Still exit `0`
- Include usage information for the new command
- Pattern: `"<old> is deprecated. Use 'nself <new parent> <subcommand>' instead."`

### 6. pre_command Exempt Commands

Commands in the `no_init_commands` list in `pre-command.sh` do not require
`.env.local` to exist. Current exempt list:

```
init help version update reset checklist doctor completion
```

### 7. Enforcement

The help contract is enforced by:
- `src/tests/test-help-contract.sh` - CI regression gate
- Manual smoke test: `for f in src/cli/*.sh; do bash "$f" --help >/dev/null 2>&1 || echo "FAIL: $f"; done`

## Version History

- v0.9.8: Contract defined and enforced across all 31 TLCs + deprecated wrappers
