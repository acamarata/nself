# Deprecated CLI Commands

This directory contains backward compatibility wrappers for commands that have been consolidated into the `service` command.

## Command Consolidation (v1.0)

All service-related commands have been consolidated under `nself service` for better organization and discoverability.

### Deprecated Commands → New Commands

| Old Command | New Command | Status |
|-------------|-------------|--------|
| `nself storage` | `nself service storage` | Deprecated |
| `nself email` | `nself service email` | Deprecated |
| `nself search` | `nself service search` | Deprecated |
| `nself redis` | `nself service redis` | Deprecated |
| `nself functions` | `nself service functions` | Deprecated |
| `nself mlflow` | `nself service mlflow` | Deprecated |
| `nself realtime` | `nself service realtime` | Deprecated |
| `nself admin` | `nself service admin` | Deprecated |
| `nself admin-dev` | `nself service admin dev` | Deprecated |

## Behavior

Each wrapper:
1. Shows a deprecation warning
2. Forwards all arguments to the new `nself service` command
3. Maintains full backward compatibility

## Example

```bash
# Old way (still works but shows warning)
nself email test admin@example.com

# Output:
# [DEPRECATED] 'nself email' is deprecated.
# Use: nself service email instead
#
# [then runs the command]

# New way (recommended)
nself service email test admin@example.com
```

## Timeline

- **v0.9.x**: Old commands work without warnings
- **v1.0.0**: Old commands work with deprecation warnings (current)
- **v1.1.0**: Old commands will be removed (breaking change)

## Migration Guide

Update your scripts by replacing:
- `nself storage` → `nself service storage`
- `nself email` → `nself service email`
- `nself search` → `nself service search`
- `nself redis` → `nself service redis`
- `nself functions` → `nself service functions`
- `nself mlflow` → `nself service mlflow`
- `nself realtime` → `nself service realtime`
- `nself admin` → `nself service admin`
- `nself admin-dev` → `nself service admin dev`

All functionality remains identical - only the command structure changed.
