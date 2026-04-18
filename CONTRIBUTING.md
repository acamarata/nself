# Contributing to nSelf CLI

## Prerequisites

- Go 1.22+
- Docker (for integration tests)
- Make

## Setup

```bash
git clone https://github.com/nself-org/cli
cd cli
make build
```

## Development

```bash
make dev        # hot-reload dev server
make test       # run all tests
make lint       # run linters
go vet ./...    # vet code
```

## Pull Requests

1. Fork the repo and create a branch: `git checkout -b feat/my-feature`
2. Write tests for new behavior
3. Run `make test` — all tests must pass
4. Run `go vet ./...` — must pass clean
5. Submit a PR against `main`

## Commit Style

Use conventional commits:

- `feat:` new feature
- `fix:` bug fix
- `chore:` maintenance
- `docs:` documentation
- `test:` tests only

## Questions

Open a GitHub Discussion or join the community at nself.org.
