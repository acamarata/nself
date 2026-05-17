//go:build bench

// Package embedded benchmarks measure the embedded-PG boot latency and simple
// query throughput against the pglite runtime.
//
// Run with:
//
//	CGO_ENABLED=1 go test -mod=vendor -tags bench -bench=. -benchtime=3x \
//	    -timeout 300s ./internal/embedded/...
package embedded

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

// requireBenchWasmPath returns the pglite WASM path for benchmarks; skips
// when not found.
func requireBenchWasmPath(b *testing.B) string {
	b.Helper()
	wp := os.Getenv("PGLITE_WASM_PATH")
	if wp == "" {
		home, _ := os.UserHomeDir()
		wp = filepath.Join(home, ".nself", "cache", "pglite", DefaultPGliteVersion, "pglite.wasm")
	}
	if _, err := os.Stat(wp); err != nil {
		b.Skipf("pglite WASM not found at %q — set PGLITE_WASM_PATH: %v", wp, err)
	}
	return wp
}

// BenchmarkEmbeddedPGWarmBoot measures the time to start an EmbeddedPGRuntime
// when the compiled module cache already exists (warm path).
func BenchmarkEmbeddedPGWarmBoot(b *testing.B) {
	wasmPath := requireBenchWasmPath(b)

	// Use a fixed benchmark runtime dir to preserve the compiled cache across
	// iterations. Lives under /tmp so AF_UNIX path stays short.
	runtimeDir, err := os.MkdirTemp("/tmp", "epg-bench-")
	if err != nil {
		b.Fatalf("mktemp: %v", err)
	}
	b.Cleanup(func() { os.RemoveAll(runtimeDir) })

	// Prime the compiled module cache (cold boot — not measured).
	ctx0 := context.Background()
	rt0, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if err != nil {
		b.Fatalf("cold NewEmbeddedPGRuntime: %v", err)
	}
	ctxPrime, cancelPrime := context.WithTimeout(ctx0, 120*time.Second)
	if err := rt0.Start(ctxPrime); err != nil {
		cancelPrime()
		b.Fatalf("cold Start: %v", err)
	}
	_ = rt0.Stop()
	cancelPrime()

	b.ResetTimer()

	for range b.N {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		rt, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
		if err != nil {
			cancel()
			b.Fatalf("NewEmbeddedPGRuntime: %v", err)
		}
		if err := rt.Start(ctx); err != nil {
			cancel()
			b.Fatalf("Start: %v", err)
		}
		_ = rt.Stop()
		cancel()
	}
}

// BenchmarkEmbeddedPGSelectThroughput measures simple SELECT throughput over
// the Unix-domain socket once the runtime is running.
func BenchmarkEmbeddedPGSelectThroughput(b *testing.B) {
	wasmPath := requireBenchWasmPath(b)
	runtimeDir, err := os.MkdirTemp("/tmp", "epg-bench-thr-")
	if err != nil {
		b.Fatalf("mktemp: %v", err)
	}
	b.Cleanup(func() { os.RemoveAll(runtimeDir) })

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	b.Cleanup(cancel)

	rt, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if err != nil {
		b.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}
	if err := rt.Start(ctx); err != nil {
		b.Fatalf("Start: %v", err)
	}
	b.Cleanup(func() { _ = rt.Stop() })

	dsn := fmt.Sprintf("host=%s dbname=nself sslmode=disable", runtimeDir)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		b.Fatalf("sql.Open: %v", err)
	}
	b.Cleanup(func() { db.Close() }) //nolint:errcheck

	b.ResetTimer()

	for range b.N {
		var v int
		if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&v); err != nil {
			b.Fatalf("SELECT 1: %v", err)
		}
	}
}
