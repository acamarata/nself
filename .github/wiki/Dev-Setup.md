# Dev Setup

Set up a local development environment for the ɳSelf CLI.

## Prerequisites

| Tool | Minimum Version | Install |
|------|----------------|---------|
| Go | 1.21 | [go.dev/dl](https://go.dev/dl/) |
| Docker | 24.0 | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Make | 3.81 | `brew install make` (macOS) or system package |
| golangci-lint | 1.57 | `brew install golangci-lint` |

## Clone and Build

```bash
git clone https://github.com/nself-org/cli.git
cd cli

# Build the binary
make build

# Verify
./nself version
```

## Common Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build binary for current platform |
| `make test` | Run all tests |
| `make lint` | Run golangci-lint |
| `make fmt` | Run gofmt on all files |
| `make cross` | Cross-compile for Linux/macOS/Windows (arm64 + amd64) |
| `make dist` | Full release build + archives |
| `make clean` | Remove build artifacts |

## Development Loop

```bash
# Edit source files
# Then rebuild and test
make build && ./nself {command}

# Run tests for a specific package
go test ./internal/sanitize/...

# Run all tests with verbose output
make test VERBOSE=1
```

## Running Tests

```bash
make test          # all tests
go test ./...      # equivalent

# With race detector
go test -race ./...
```

Tests must pass before submitting a PR. New features require test coverage.

## Linting

```bash
golangci-lint run          # must return zero issues
golangci-lint run ./...    # equivalent
```

The `.golangci.yml` config at the repo root defines enabled linters.

## Cross-Compilation

```bash
make cross
# Outputs: dist/nself-linux-amd64, dist/nself-linux-arm64,
#          dist/nself-darwin-amd64, dist/nself-darwin-arm64,
#          dist/nself-windows-amd64.exe
```

## See Also

- [[Contributing]], contribution guidelines
- [[Release-Process]], how releases are built and published
- [[Architecture]], codebase structure

---
← [[Home]] | [[_Sidebar]]
