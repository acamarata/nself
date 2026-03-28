# Contributing to nself

## Development Setup

Requirements: Go 1.22+, Docker 24+.

```bash
git clone https://github.com/nself-org/cli.git
cd cli
go build ./...
go test ./...
```

## Running Tests

```bash
go test ./...                    # all tests
go test ./internal/config/...   # one package
go test -run TestValidate ./...  # one test
```

## Adding a Command

1. Create `cmd/commands/<name>.go`
2. Define a `cobra.Command` var
3. Register it in `cmd/commands/root.go` with `rootCmd.AddCommand`
4. Add tests in `cmd/commands/<name>_test.go`

All commands follow the pattern in `cmd/commands/status.go`.

## Adding a Validator

Validators live in `internal/config/validators.go`. Add a function matching:
```go
func ValidateXxx(cfg *Config) error
```
Then register it in `ValidateAll`.

## Code Style

- `gofmt` before committing (CI enforces this)
- `golangci-lint run ./...` must pass
- No global mutable state outside of cobra command definitions
- All exported functions need a doc comment

## Submitting a PR

Branch naming: `fix/<short-description>` or `feat/<short-description>`.

Every PR needs:
- Tests for the changed behavior
- `go test ./...` passing locally
- `go build ./...` clean

## Reporting Security Issues

Email `security@nself.org`. Do not open a public issue for security vulnerabilities. We respond within 48 hours.
