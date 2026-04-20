package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// newBuildBenchCmd returns an isolated cobra tree for build benchmarks.
// Avoids name collision with newBuildCmd from build_test.go.
func newBuildBenchCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	bc := &cobra.Command{
		Use:   "build",
		Short: "Compose your infrastructure from .env",
		RunE:  runBuild,
	}
	bc.Flags().BoolP("force", "f", false, "Force rebuild all components")
	bc.Flags().Bool("no-cache", false, "Disable build cache")
	bc.Flags().BoolP("verbose", "v", false, "Show environment cascade")
	bc.Flags().Bool("debug", false, "Enable debug mode")
	bc.Flags().Bool("security-report", false, "Generate security analysis")
	bc.Flags().Bool("allow-insecure", false, "Allow insecure config (dev only)")
	bc.Flags().Bool("check", false, "Validate only, don't build")
	bc.Flags().BoolP("quiet", "q", false, "Suppress non-error output")
	bc.Flags().Bool("no-monorepo", false, "Disable automatic monorepo backend detection")
	bc.Flags().Bool("no-migration-check", false, "Skip v1 artifact detection")

	root.AddCommand(bc)
	return root
}

// BenchmarkBuild measures config-generation overhead for `nself build`.
//
// The command fails early (no .nself project) but exercises the full flag-parse
// and FindNSelfRoot path — the hot path that every build invocation traverses.
//
// SLO: nself build p95 < 5s (F16-PERF-SLOS.md — CLI Operations).
// Run: go test -bench=BenchmarkBuild -benchmem ./cmd/commands/
func BenchmarkBuild(b *testing.B) {
	b.ReportAllocs()

	tmp := b.TempDir()
	b.Chdir(tmp)

	root := newBuildBenchCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"build", "--no-monorepo", "--no-migration-check", "--quiet"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		// Error expected (no project dir). Benchmarks the path to FindNSelfRoot.
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkBuild", b)
}

// BenchmarkBuildCheck measures `nself build --check` (validate-only mode).
// Skips Docker/nginx generation; isolates config-load and validation overhead.
func BenchmarkBuildCheck(b *testing.B) {
	b.ReportAllocs()

	tmp := b.TempDir()
	b.Chdir(tmp)

	root := newBuildBenchCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"build", "--check", "--no-monorepo", "--no-migration-check", "--quiet"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkBuildCheck", b)
}

// BenchmarkBuildFlagParsing measures ONLY cobra flag-parsing overhead for
// the build command's 10-flag set. Isolates framework from business logic.
func BenchmarkBuildFlagParsing(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := newBuildBenchCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"build", "--force", "--no-cache", "--verbose",
			"--security-report", "--quiet"})
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkBuildFlagParsing", b)
}
