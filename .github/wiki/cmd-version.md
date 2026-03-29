# nself version

> Show version and system information.

## Synopsis

```
nself version [flags]
nself -v
nself --version
```

## Description

`nself version` prints the CLI version, Go build version, git commit hash, and build date. This information is embedded at compile time using Go linker flags (`-ldflags`).

Use `--short` for scripts that only need the version number. Use `--json` for structured output in monitoring systems or CI pipelines.

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--short` | false | Print version number only |
| `--json` | false | Print version info as JSON |
| `--help`, `-h` | — | Show help |

## Examples

```bash
# Show full version info
nself version

# Version number only
nself version --short

# JSON output
nself version --json
```

**Sample output:**

```
nself version 1.0.0
  Go version:  go1.23.4
  Git commit:  a1b2c3d
  Build date:  2026-03-28T00:00:00Z
  OS/Arch:     darwin/arm64
```

**JSON output:**

```json
{
  "version": "1.0.0",
  "go_version": "go1.23.4",
  "git_commit": "a1b2c3d",
  "build_date": "2026-03-28T00:00:00Z",
  "os_arch": "darwin/arm64"
}
```

← [[Commands]] | [[Home]] →
