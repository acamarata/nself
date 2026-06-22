# nself plugin test

> Run unit and smoke tests for a plugin.

## Synopsis

```
nself plugin test <name> [flags]
```

## Description

`nself plugin test` runs the test suite for a named plugin. It supports two test phases: `unit` (fast, no Docker required) and `smoke` (integration tests against a live container). The default `--phase both` runs unit tests first, then smoke tests.

Unit tests are the plugin's own `go test ./...` suite. Smoke tests start the plugin container and run a minimal end-to-end check (health probe, request–response roundtrip). Use `--host` to run tests in host-process mode, bypassing the Docker container layer for faster iteration.

After the smoke phase, the plugin container is stopped and cleaned up. Use `--no-cleanup` to leave the container running for post-failure inspection.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--phase` | `both` | Test phase to run: `unit`, `smoke`, or `both` |
| `--host` | false | Run tests in host-process mode instead of a container |
| `--no-cleanup` | false | Skip cleanup after the smoke test (useful for debugging failures) |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Run all tests (unit + smoke)
nself plugin test my-plugin
```

```bash
# Run only unit tests
nself plugin test my-plugin --phase unit
```

```bash
# Run smoke tests in host-process mode
nself plugin test my-plugin --phase smoke --host
```

```bash
# Run smoke tests and leave the container running after failure
nself plugin test my-plugin --phase smoke --no-cleanup
```

## See Also

- [[cmd-plugin-dev]] — start dev mode with live reload
- [[cmd-plugin-debug]] — attach a debugger to a running plugin
- [[cmd-plugin-link]] — link a local plugin directory into the stack
- [[cmd-plugin]] — plugin command overview
- [[Commands]] — full command index

← [[Commands]] | [[Home]] →
