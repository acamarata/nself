# nself generate

**Generate type-safe client SDK types from the live Hasura schema.**

Introspects the running ɳSelf Hasura GraphQL endpoint and emits typed client code
for TypeScript, Dart, Swift, Kotlin, and Python. One command keeps all SDK consumers
in sync with the database schema. No hand-maintained type files.

## Synopsis

```
nself generate [flags]
```

## Flags

| Flag | Default | Description |
|------|---------|-------------|
| `--lang` | `all` | Comma-separated target languages: `typescript`, `dart`, `swift`, `kotlin`, `python`, or `all` |
| `--out` | `./generated` | Output directory |
| `--env` | `local` | Target environment: `local`, `staging`, or `prod` |
| `--watch` | false | Re-run every 30 seconds on schema change |
| `--operations` | `src/**/*.graphql` | Path to `.graphql` operation files for query/mutation/subscription types |
| `--no-hooks` | false | Skip React hooks generation (TypeScript only) |
| `--pydantic` | true | Emit Pydantic v2 models (Python only) |

## Output structure

```
generated/
├── typescript/
│   ├── schema.ts         # DB types (interfaces)
│   ├── queries.ts        # typed query documents and return types
│   └── hooks.ts          # React hooks wrapping @nself/client
├── dart/
│   ├── models.dart       # freezed-compatible data classes
│   └── queries.dart      # typed query variables and result classes
├── swift/
│   └── NselfModels.swift # Codable structs
├── kotlin/
│   └── NselfModels.kt    # kotlinx.serialization data classes
└── python/
    └── models.py         # Pydantic v2 models
```

## Codegen pipeline

1. Fetch live Hasura introspection schema via `HASURA_GRAPHQL_ADMIN_SECRET`
2. Parse schema into an internal IR (tables, enums, relationships)
3. Merge `.graphql` operation files into the IR
4. Render per-language output files to `--out` directory
5. Print diff summary: N types, M enums, K operations

## Prerequisites

The ɳSelf stack must be running before generating against `--env local`:

```bash
nself start
nself generate
```

For staging or production, set the corresponding env vars:

```bash
# Staging
export NSELF_HASURA_STAGING_URL=https://api.myproject.com
export NSELF_HASURA_STAGING_ADMIN_SECRET=<secret>
nself generate --env staging

# Production
export NSELF_HASURA_PROD_URL=https://api.myproject.com
export NSELF_HASURA_PROD_ADMIN_SECRET=<secret>
nself generate --env prod
```

## Examples

```bash
# Generate all languages against the local stack
nself generate

# TypeScript and Dart only
nself generate --lang typescript,dart

# TypeScript against staging, custom output dir
nself generate --env staging --lang typescript --out ./src/generated

# Watch mode: regenerate every 30s when schema changes
nself generate --watch

# Skip React hooks (useful for non-React TypeScript projects)
nself generate --lang typescript --no-hooks
```

## Example output

```
Fetching schema from http://localhost:8080 ...
Parsed 42 types, 8 enums
[typescript] 3 files written (0.3s)
[dart] 2 files written (0.2s)
[swift] 1 files written (0.1s)
[kotlin] 1 files written (0.1s)
[python] 1 files written (0.1s)
Done in 0.9s. Output: ./generated
```

## Integration with SDKs

Use generated types with the ɳSelf SDK packages:

| Language | Package | Import |
|----------|---------|--------|
| TypeScript | `@nself/client` | `import type { User } from './generated/typescript/schema'` |
| Dart | `nself_client` | `import 'generated/dart/models.dart'` |
| Swift | `NselfClient` | `import NselfClient; // use NselfModels.swift types` |
| Kotlin | `io.nself:client` | `import io.nself.generated.*` |
| Python | `nself-client` | `from generated.python.models import User` |

See [SDK reference](https://nself.org/docs/sdks/) for full usage.

## Notes

- The command uses the existing `HASURA_GRAPHQL_ADMIN_SECRET` from your `.env` cascade for local environments.
- Hasura aggregate, input, and Hasura-internal types are automatically excluded from the output.
- `--watch` uses time-based polling (30s interval). It does not use filesystem watchers.
- TypeScript output requires `@nself/client` as a peer dependency for hooks.
- Dart output generates `freezed`-compatible annotations; run `dart run build_runner build` to finish code generation.

---

[[Home]] | [[Commands]] | [[Changelog]]
