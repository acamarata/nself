# plugin-sdk-go Compatibility Matrix

The shared Go SDK for nSelf plugins. Every pro plugin (ai, mux, claw, voice,
browser, notify, cron, chat, livekit, etc.) depends on this module.

## Current versions

| Component | Version | Notes |
| --- | --- | --- |
| `plugin-sdk-go` | **0.1.0** | See [`doc.go`](doc.go) `Version` constant |
| nSelf CLI (min) | **1.0.9** | Plugins declaring `minNselfVersion` below this are rejected |
| Go toolchain | **1.23.0+** | `go.mod` declares `go 1.23.0` |

## Compatibility guarantees

`plugin-sdk-go` follows strict SemVer.

| Change class | Major | Minor | Patch |
| --- | --- | --- | --- |
| Add new package (e.g. `tracing/`) | | x | |
| Add new exported function or type | | x | |
| Add new optional field on an `Options` struct | | x | |
| Fix bug without changing signatures | | | x |
| Rename / remove any exported symbol | x | | |
| Change behavior of existing function in a way that breaks callers | x | | |
| Bump min Go version | x | | |

Plugins specify their minimum SDK version via `sdk.CheckMinSDK(required)` at
startup. Plugin manifests (`plugin.json`) also declare `minSdkVersion` for
offline verification by the CLI.

## CLI ↔ plugin compatibility

Plugins declare their CLI range via `plugin.json`:

```json
{
  "minNselfVersion": "1.0.9",
  "maxNselfVersion": ""
}
```

Empty `maxNselfVersion` means "no upper bound". CLI refuses to install a
plugin whose declared range does not include the running CLI. Use
`sdk.CheckCLICompat(currentCLI, minCLI, maxCLI)` inside a plugin to enforce
the same check at runtime.

## Supported Go versions

| Go | plugin-sdk-go support |
| --- | --- |
| 1.22 | not supported (missing `log/slog` stability + `slices` generics) |
| 1.23 | **required minimum** |
| 1.24 | supported |
| 1.25+ | supported (best-effort) |

## Supported CLI versions

plugin-sdk-go 0.1.x targets CLI **1.0.9 and newer**. CLI 1.0.5-1.0.8 predate
the Go plugin runtime contract and are not supported.

## Release policy

- Patch releases every ~2 weeks or on critical fix.
- Minor releases bundled with CLI LTS ticks.
- Major releases only when breaking changes cannot be absorbed by a minor.
- Every release ships a migration note in [`CHANGELOG.md`](CHANGELOG.md) when
  downstream plugins must update.

## Runtime version checks

Every plugin's `main.go` should include:

```go
if err := sdk.CheckMinSDK("0.1.0"); err != nil {
    log.Fatalf("incompatible plugin-sdk-go: %v", err)
}
if err := sdk.CheckCLICompat(os.Getenv("NSELF_CLI_VERSION"), "1.0.9", ""); err != nil {
    log.Fatalf("incompatible nSelf CLI: %v", err)
}
```

## Deprecation policy

Deprecated symbols stay for a minimum of **two minor releases** before removal.
They are marked with a `// Deprecated: ...` comment and surface a `go vet`
warning through the standard lint path.
