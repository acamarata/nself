# Plugin SDK

The nSelf Plugin SDK is available in four languages. Each SDK provides the same core contracts — lifecycle hooks, schema migration helpers, metrics, health checks, and structured logging — adapted to the idioms of its language.

All four SDKs are MIT-licensed and published to their respective registries.

---

## Language SDKs

| Language | Package | Registry | Import |
|---|---|---|---|
| **Go** | `github.com/nself-org/cli/sdk/go/v2` | pkg.go.dev | `"github.com/nself-org/cli/sdk/go/v2/plugin"` |
| **TypeScript** | `@nself/plugin-sdk` | npm | `import { Plugin } from '@nself/plugin-sdk'` |
| **Python** | `nself-plugin-sdk` | PyPI | `from nself_plugin import Plugin` |

### Go — v2

Go is the reference implementation. All other SDKs maintain behavioral parity with the Go SDK.

```bash
go get github.com/nself-org/cli/sdk/go/v2
```

Source: `cli/sdk/go/` — published automatically when a `sdk-go/v*` tag is pushed.

### TypeScript — 2.0

Targets Node.js 18+ with ESM and CJS exports. Full TypeScript types included.

```bash
npm install @nself/plugin-sdk
```

Source: `cli/sdk/ts/` — published automatically when a `sdk-ts/v*` tag is pushed.

### Python — 2.0

Targets Python 3.11+. Built on FastAPI and asyncpg. Published as a wheel and sdist.

```bash
pip install nself-plugin-sdk
# with optional Postgres support:
pip install nself-plugin-sdk[db]
```

Source: `cli/sdk/py/` — published automatically when a `sdk-py/v*` tag is pushed.

---

## Version Policy

All four SDKs follow [Semantic Versioning](https://semver.org/).

| Bump | When |
|---|---|
| **MAJOR** | Breaking changes to public API, hook signatures, or wire formats |
| **MINOR** | New optional fields, new hooks with defaults, new helper utilities |
| **PATCH** | Bug fixes with no API changes |

MAJOR versions are synchronized across all four SDKs. When one SDK ships a breaking change, all four bump to the same MAJOR in the same release cycle. MINOR and PATCH versions may diverge.

---

## Breaking Change Process

Before any breaking change ships:

1. **Deprecation notice** — mark the old API deprecated in the current MINOR release. Keep the old API working. Use `//Deprecated:` in Go, `@deprecated` JSDoc in TypeScript, Python deprecation warnings, and Dart `@Deprecated`.
2. **Migration guide** — commit `MIGRATION.md` to the SDK directory before the MAJOR release tag.
3. **30-day minimum notice** — publish a deprecation notice in the changelog and announce to plugin authors at least 30 days before the MAJOR release.
4. **RC tags required** — tag at least one release candidate (e.g. `sdk-go/v3.0.0-rc.1`) and allow a two-week soak window before promoting to final.

Full policy: [[sdk/breaking-change-policy]]

---

## Publishing Workflows

Each SDK publishes via a dedicated GitHub Actions workflow triggered by a scoped tag:

| SDK | Tag pattern | Workflow |
|---|---|---|
| Go | `sdk-go/vX.Y.Z` | `cli/sdk/go/.github/workflows/sdk-go-publish.yml` |
| TypeScript | `sdk-ts/vX.Y.Z` | `cli/sdk/ts/.github/workflows/sdk-ts-publish.yml` |
| Python | `sdk-py/vX.Y.Z` | `cli/sdk/py/.github/workflows/sdk-py-publish.yml` |

Required repository secrets:

| Secret | Used by |
|---|---|
| `NPM_TOKEN` | TypeScript SDK — npm publish |

Go and Python publish via OIDC Trusted Publisher — no long-lived secret required.

---

## SDK + CLI Compatibility

Plugin SDKs version independently of the CLI. Compatibility is determined by the plugin protocol major version, not the CLI version.

| CLI version | Plugin protocol | Compatible SDK major |
|---|---|---|
| v1.0.x | v1 | SDK v1.x |
| v1.1.x | v1 | SDK v1.x or v2.x |

---

## See Also

- [[sdk/breaking-change-policy]] — full deprecation cycle, migration guide requirements, RC process
- [[Home]]
