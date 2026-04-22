# nself plugin test

Run a plugin's composite test suite: unit tests, smoke install, and smoke uninstall.

## Synopsis

```bash
nself plugin test <name> [flags]
```

## Description

Runs three test phases against a linked plugin:

1. **Unit** — `go test ./...` inside the plugin source (in container or on host)
2. **Smoke install** — installs the plugin via `nself plugin install` and verifies `/healthz` returns 200
3. **Smoke uninstall** — removes the plugin via `nself plugin remove` and verifies clean removal

All three phases must pass for the command to exit 0. Any failure identifies which phase failed.

The plugin must be linked with `nself plugin link` before running tests.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--phase` | `both` | Phases to run: `unit`, `smoke`, or `both` |
| `--host` | false | Run unit tests on host instead of inside a container |
| `--no-cleanup` | false | Skip the uninstall phase (useful when debugging a failed install) |

## Examples

```bash
# Run all phases
nself plugin test myplugin

# Run only unit tests
nself plugin test myplugin --phase=unit

# Run only smoke install/uninstall
nself plugin test myplugin --phase=smoke

# Skip cleanup to inspect state after a failed install
nself plugin test myplugin --phase=smoke --no-cleanup

# Run unit tests on host (faster on macOS)
nself plugin test myplugin --phase=unit --host
```

## Output

```
Running plugin tests for myplugin (phase=both)...
--- Phase 1: unit tests ---
ok  	github.com/nself-org/paid/myplugin/internal/server	0.012s
PASS unit [0.1s]
--- Phase 2: smoke install ---
Installing plugin "myplugin"...
Plugin "myplugin" installed successfully.
PASS smoke-install [3.2s]
--- Phase 3: smoke uninstall ---
Removing plugin "myplugin"...
Plugin "myplugin" removed successfully.
PASS smoke-uninstall [1.8s]
---
Passed: unit, smoke-install, smoke-uninstall
All test phases passed.
```

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | All selected phases passed |
| 1 | One or more phases failed (phase name in error message) |

## Notes

- Smoke install/uninstall has side effects on your local nSelf stack. Use a scratch project
  or `--no-cleanup` for debugging. The cookbook recommends a dedicated test project.
- Unit tests run in a `golang:1.22-alpine` container by default. Pass `--host` to skip Docker.

## See Also

- [[cmd-plugin-link]] — link a plugin directory first
- [[cmd-plugin-dev]] — hot-reload watcher
- [[cmd-plugin-logs]] — inspect logs during smoke test
- [[Home]]
