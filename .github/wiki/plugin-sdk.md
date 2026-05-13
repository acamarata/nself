# Plugin SDK

## Overview
The nSelf Plugin SDK provides language bindings for building plugins. SDKs are available in Go, Python, TypeScript, and Flutter (Dart).

| SDK | Module path | Registry | Stable |
|-----|-------------|----------|--------|
| Go | github.com/nself-org/cli/sdk/go/v2 | Go module proxy | ✅ |
| Python | nself-sdk | PyPI | ✅ |
| TypeScript | @nself/sdk | npm | ✅ |
| Flutter | nself_sdk | pub.dev | ✅ |

## Breaking Change Policy

### Versioning
All SDKs follow semantic versioning (semver):
- **Patch** (x.y.Z): bug fixes, no API changes
- **Minor** (x.Y.z): new features, backwards-compatible additions
- **Major** (X.y.z): breaking changes — existing plugins may require updates

### What constitutes a breaking change
- Removing or renaming a function, method, type, or constant from a public API
- Changing a function signature (parameters, return types)
- Changing the wire format of plugin RPC messages
- Changing required environment variables
- Changing the plugin manifest schema in a backwards-incompatible way

### Migration path requirements
Every major version bump MUST include:
1. A migration guide in `.github/wiki/sdk-migration-vX.md`
2. A deprecation period of at least one minor version where both old and new API exist
3. Deprecation warnings emitted at plugin load time for old API usage
4. Updated examples in all SDK repos

### Deprecation process
1. Mark symbol as deprecated in current minor version (comment + `//Deprecated:` in Go; `@deprecated` JSDoc in TS)
2. Emit runtime warning when deprecated symbol is used
3. Remove in next major version

### SDK + CLI compatibility matrix
Plugin SDKs are versioned independently of the CLI. A plugin SDK version is compatible with all CLI versions that share the same major plugin protocol version.

| CLI version | Plugin protocol | Compatible SDK versions |
|-------------|-----------------|------------------------|
| v1.0.x      | v1              | SDK Go v1.x, SDK TS v1.x, SDK Py v1.x |
| v1.1.x      | v1              | SDK Go v1.x or v2.x (v2 = Go module path bump only) |

### Submitting SDK changes
PRs that modify public SDK APIs must:
- Update the compatibility matrix above
- Add a changelog entry in the relevant SDK's CHANGELOG
- If breaking: create the migration guide before merging
