# Contributing to nSelf CLI

Thank you for your interest in contributing to nSelf. This guide covers bug reports, feature requests, and code contributions.

## Reporting Bugs

1. Search [existing issues](https://github.com/nself-org/cli/issues) first — your bug may already be reported.
2. Open a new issue using the **Bug Report** template.
3. Include: nSelf version (`nself version`), OS, steps to reproduce, expected vs actual behaviour.

## Requesting Features

1. Check the [issue tracker](https://github.com/nself-org/cli/issues) for existing feature requests.
2. Open a new issue using the **Feature Request** template.
3. Describe the use case, not just the solution — this helps us find the best approach.

## Pull Request Process

```bash
# 1. Fork the repository on GitHub
# 2. Clone your fork
git clone https://github.com/YOUR-USERNAME/cli.git
cd cli

# 3. Create a feature branch
git checkout -b feat/my-feature

# 4. Make your changes
# 5. Run tests and linting (must pass)
make test
golangci-lint run

# 6. Commit
git commit -m "feat: add my feature"

# 7. Push and open a PR
git push origin feat/my-feature
```

Open a pull request against the `main` branch. Fill in the PR template completely.

## Local Dev Loop

```bash
make dev        # hot-reload dev server
make test       # run all tests
make lint       # run linters
go vet ./...    # vet code
```

## Code Style

- **Format:** `gofmt` -- run automatically via `make fmt`
- **Lint:** `golangci-lint run` must pass with zero warnings
- **Tests:** all existing tests must pass; new features require new tests
- **Errors:** return errors with context: `fmt.Errorf("doing X: %w", err)`
- **I/O functions:** take `context.Context` as first parameter

## Commit Style

Use conventional commits:

- `feat:` new feature
- `fix:` bug fix
- `chore:` maintenance
- `docs:` documentation
- `test:` tests only

## Commit Sign-off

No AI tool attribution in commit messages or PR descriptions. Contributions must represent your own work.

## Questions

Open a [GitHub Discussion](https://github.com/nself-org/cli/discussions) or join the community at [nself.org](https://nself.org).

## Links

- [[Dev-Setup]] — set up your local development environment
- [[Plugin-Dev-Guide]] — building a new plugin
- [[Release-Process]] — how releases work

---
← [[Home]] | [[_Sidebar]]
