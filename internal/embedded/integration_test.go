//go:build integration

// Package embedded integration tests validate the full embedded-PG lifecycle:
// WASM boot, socket bridge, migration runner, backup, and cold-start budget.
//
// Run with:
//
//	INTEGRATION=1 CGO_ENABLED=1 go test -mod=vendor -tags integration -timeout 120s \
//	    ./internal/embedded/...
package embedded

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// shortTempDir creates a temp dir under /tmp to keep AF_UNIX paths short
// (macOS limit: 104 chars).
func shortIntegTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "epg-integ-")
	if err != nil {
		t.Fatalf("mktemp: %v", err)
	}
	t.Cleanup(func() { os.RemoveAll(dir) })
	return dir
}

// quarantined reports whether the embedded-PG integration tests should be
// skipped. pglite v0.2.17 is an Emscripten SIDE_MODULE whose GOT.mem/GOT.func
// relocations are never applied host-side, so it traps with an out-of-bounds
// access before Postgres starts. These tests have never passed on main.
//
// Tracked in https://github.com/nself-org/cli/issues/231. Removing this skip is
// part of closing that issue. Set EMBEDDED_PG_WASM_FIXED=1 to run them while
// working on the fix.
func quarantined(t *testing.T) {
	t.Helper()
	if os.Getenv("EMBEDDED_PG_WASM_FIXED") == "" {
		t.Skip("embedded pglite WASM runtime does not boot yet — quarantined, see issue #231 (set EMBEDDED_PG_WASM_FIXED=1 to run)")
	}
}

func requireWasmPath(t *testing.T) string {
	t.Helper()
	quarantined(t)
	wp := os.Getenv("PGLITE_WASM_PATH")
	if wp == "" {
		// Use a fixed known-good path for local development.
		home, _ := os.UserHomeDir()
		wp = filepath.Join(home, ".nself", "cache", "pglite", DefaultPGliteVersion, "pglite.wasm")
	}
	if _, err := os.Stat(wp); err != nil {
		t.Skipf("pglite WASM not found at %q — set PGLITE_WASM_PATH or run FetchOrCached first: %v", wp, err)
	}
	return wp
}

// TestEmbeddedPGBootCycle verifies that the embedded runtime starts, exposes a
// socket, accepts a basic query, and stops cleanly.
func TestEmbeddedPGBootCycle(t *testing.T) {
	wasmPath := requireWasmPath(t)
	runtimeDir := shortIntegTempDir(t)

	// On a cold cache this does a full wasmtime compile of the pglite Postgres
	// WASM before Postgres even starts, which is CPU-bound and takes minutes on
	// 2-core CI runners. The compiled module is then cached on disk (and by the
	// workflow's cache step), so only the first run pays this. Liveness bound,
	// not a perf assertion.
	//
	// The bound must sit UNDER the `go test -timeout` in
	// .github/workflows/embedded-pg-matrix.yml (2400s), which in turn sits under
	// the job's timeout-minutes (45), but comfortably ABOVE a cold
	// compile. It was 900s, which is only 15 of those 45 minutes: on any cache
	// miss the compile outran the bound and the test died at exactly 900.00s
	// while the job still had 30 minutes of budget left. 2100s (35 min) keeps
	// 10 minutes of headroom for the runner to report the failure cleanly.
	// Override with NSELF_EMBEDDED_BOOT_TIMEOUT for a faster local loop.
	bootTimeout := 2100 * time.Second
	if v := os.Getenv("NSELF_EMBEDDED_BOOT_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			bootTimeout = d
		} else {
			t.Fatalf("NSELF_EMBEDDED_BOOT_TIMEOUT=%q is not a valid duration: %v", v, err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), bootTimeout)
	defer cancel()

	rt, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}

	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Stop must be clean even on early return.
	t.Cleanup(func() { _ = rt.Stop() })

	if !rt.Healthy() {
		t.Error("runtime is not Healthy after Start")
	}

	// Verify basic query via the UDS socket.
	dsn := fmt.Sprintf("host=%s dbname=nself sslmode=disable", runtimeDir)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close() //nolint:errcheck

	var answer int
	if err := db.QueryRowContext(ctx, "SELECT 1").Scan(&answer); err != nil {
		t.Fatalf("SELECT 1: %v", err)
	}
	if answer != 1 {
		t.Errorf("expected 1, got %d", answer)
	}
}

// TestSocketBridgeProxy verifies that a PGSocketBridge in front of the runtime
// transparently proxies a simple SELECT while blocking COPY TO.
func TestSocketBridgeProxy(t *testing.T) {
	wasmPath := requireWasmPath(t)
	runtimeDir := shortIntegTempDir(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	rt, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = rt.Stop() })

	// Bridge listens on a separate socket path.
	bridgeSock := filepath.Join(runtimeDir, "bridge.sock")
	bridge := &PGSocketBridge{}
	if err := bridge.Listen(ctx, bridgeSock, rt); err != nil {
		t.Fatalf("bridge.Listen: %v", err)
	}
	t.Cleanup(func() { _ = bridge.Close() })

	// The bridge directory acts as the "host" in the libpq-style DSN.
	bridgeDSN := fmt.Sprintf("host=%s dbname=nself sslmode=disable", runtimeDir)
	db, err := sql.Open("pgx", bridgeDSN)
	if err != nil {
		t.Fatalf("sql.Open bridge: %v", err)
	}
	defer db.Close() //nolint:errcheck

	// Simple SELECT should pass through the bridge.
	var val int
	if err := db.QueryRowContext(ctx, "SELECT 42").Scan(&val); err != nil {
		t.Fatalf("SELECT 42 via bridge: %v", err)
	}
	if val != 42 {
		t.Errorf("expected 42, got %d", val)
	}
}

// TestMigrationRunnerEmbedded verifies that DDL statements execute against the
// embedded runtime via the database/sql + pgx wire path.
func TestMigrationRunnerEmbedded(t *testing.T) {
	wasmPath := requireWasmPath(t)
	runtimeDir := shortIntegTempDir(t)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	rt, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime: %v", err)
	}
	if err := rt.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}
	t.Cleanup(func() { _ = rt.Stop() })

	dsn := fmt.Sprintf("host=%s dbname=nself sslmode=disable", runtimeDir)
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close() //nolint:errcheck

	// Run a DDL migration statement.
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS _migration_test (id serial PRIMARY KEY, label TEXT NOT NULL)`); err != nil {
		t.Fatalf("CREATE TABLE: %v", err)
	}

	if _, err := db.ExecContext(ctx, `INSERT INTO _migration_test (label) VALUES ('hello')`); err != nil {
		t.Fatalf("INSERT: %v", err)
	}

	var label string
	if err := db.QueryRowContext(ctx, `SELECT label FROM _migration_test LIMIT 1`).Scan(&label); err != nil {
		t.Fatalf("SELECT: %v", err)
	}
	if label != "hello" {
		t.Errorf("expected 'hello', got %q", label)
	}
}

// TestColdStartBudget verifies that the embedded runtime (with pre-compiled
// module cache) boots in under 10 seconds. The first boot (cold compile) may
// take longer and is explicitly excluded from the budget check.
func TestColdStartBudget(t *testing.T) {
	wasmPath := requireWasmPath(t)
	runtimeDir := shortIntegTempDir(t)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// First boot — cold compile; no budget check.
	rt1, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime (cold): %v", err)
	}
	if err := rt1.Start(ctx); err != nil {
		t.Fatalf("Start (cold): %v", err)
	}
	if err := rt1.Stop(); err != nil {
		t.Logf("Stop (cold): %v", err)
	}

	// Second boot — module cache exists; apply budget.
	const warmBudget = 10 * time.Second
	start := time.Now()
	rt2, err := NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if err != nil {
		t.Fatalf("NewEmbeddedPGRuntime (warm): %v", err)
	}
	if err := rt2.Start(ctx); err != nil {
		t.Fatalf("Start (warm): %v", err)
	}
	elapsed := time.Since(start)
	t.Cleanup(func() { _ = rt2.Stop() })

	if elapsed > warmBudget {
		t.Errorf("warm boot took %v, budget is %v", elapsed, warmBudget)
	} else {
		t.Logf("warm boot: %v (budget %v)", elapsed, warmBudget)
	}
}
