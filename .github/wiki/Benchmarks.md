# CLI Benchmarks

The ɳSelf CLI ships with Go benchmark tests for the top-5 commands. These benchmarks measure cobra dispatch overhead, flag-parsing cost, and early business-logic traversal, not external I/O (Docker, network, filesystem beyond a temp dir).

## Running Benchmarks

```bash
# Run all benchmarks once (quick check)
go test -mod=vendor -bench=. -benchtime=1x -benchmem ./cmd/commands/

# Run a specific benchmark (3s per b.N, 3 runs)
go test -mod=vendor -bench=BenchmarkVersion -benchtime=3s -count=3 -benchmem ./cmd/commands/

# Run all 5 primary benchmarks
go test -mod=vendor \
  -bench='BenchmarkVersion$|BenchmarkInit$|BenchmarkBuild$|BenchmarkPluginInstallCached$|BenchmarkDoctor$' \
  -benchtime=3s -count=3 -benchmem \
  ./cmd/commands/
```

## Benchmark Files

| File | Benchmarks | What it measures |
|------|-----------|-----------------|
| `version_bench_test.go` | `BenchmarkVersion`, `BenchmarkVersionJSON`, `BenchmarkVersionHelp` | Cobra startup + version string format |
| `init_bench_test.go` | `BenchmarkInit`, `BenchmarkInitFlagParsing` | Flag parse + early input sanitisation for `nself init` |
| `build_bench_test.go` | `BenchmarkBuild`, `BenchmarkBuildCheck`, `BenchmarkBuildFlagParsing` | Flag parse + FindNSelfRoot for `nself build` |
| `plugin_install_bench_test.go` | `BenchmarkPluginInstallCached`, `BenchmarkPluginList`, `BenchmarkPluginFlagParsing` | Plugin dispatch + gate overhead |
| `doctor_bench_test.go` | `BenchmarkDoctor`, `BenchmarkDoctorDeep`, `BenchmarkDoctorFlagParsing` | Doctor command dispatch (stub RunE, no Docker required) |

## Emitting Results for CI

Set `BENCH_RESULTS_FILE=/path/to/output.ndjson` before running benchmarks to emit
NDJSON result entries. The CI pipeline (`perf.yml`) uses this to feed `perf-compare.sh`.

```bash
BENCH_RESULTS_FILE=/tmp/bench.ndjson \
  go test -mod=vendor -bench=. -benchtime=3s ./cmd/commands/
```

## SLO Targets

Per F16-PERF-SLOS.md:

| Command | p95 SLO |
|---------|---------|
| `nself version` | ≤ 150ms |
| `nself build` | ≤ 5s |
| `nself doctor` | ≤ 8s |

These benchmarks measure pre-check overhead only. Full command execution (including Docker and network) is measured by the integration test suite.

## CI Pipeline

`cli/.github/workflows/perf.yml` runs on every PR and push to main:
- Builds the `darwin/arm64` binary and checks against hard limits: ≤35 MB uncompressed, ≤6 MB compressed.
- Runs the 5 benchmark suites and uploads results as artifacts.
- A >10% binary size growth without the `size-grow-approved` PR label fails the gate.

← [[Contributing]] | [[Commands]] | [[Home]] →
