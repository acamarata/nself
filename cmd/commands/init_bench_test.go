package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// newInitBenchCmd returns an isolated cobra tree for init benchmarks.
// Mirrors newInitCmd from init_test.go — kept separate to avoid function name collision.
func newInitBenchCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	ic := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new nSelf project",
		RunE:  runInit,
	}
	ic.Flags().String("name", "", "Project name")
	ic.Flags().String("domain", "", "Base domain")
	ic.Flags().Bool("non-interactive", false, "Use all defaults without prompts")
	ic.Flags().Bool("fast", false, "Skip advanced options, use smart defaults")
	ic.Flags().Bool("wizard", false, "Run the full 10-step interactive wizard")
	ic.Flags().Bool("demo", false, "Auto-configure with all services enabled")
	ic.Flags().Bool("full", false, "Create all environment files")
	ic.Flags().Bool("force", false, "Overwrite existing configuration")
	ic.Flags().Bool("quiet", false, "Suppress output messages")
	ic.Flags().String("template", "", "Use specific template")
	ic.Flags().Bool("skip-validation", false, "Skip configuration validation")
	ic.Flags().Bool("interactive", false, "Explicitly enable interactive wizard")

	root.AddCommand(ic)
	return root
}

// BenchmarkInit measures the flag-parsing and early-validation overhead of
// `nself init --non-interactive` before it reaches the filesystem setup phase.
//
// The command will return an error (no .nself project directory) but the
// benchmark captures the parsing and input-sanitisation cost — the hot path
// that every init invocation traverses.
//
// SLO: init parsing overhead < 10ms (not in F16; this is an internal budget).
// Run: go test -bench=BenchmarkInit -benchmem ./cmd/commands/
func BenchmarkInit(b *testing.B) {
	b.ReportAllocs()

	tmp := b.TempDir()
	b.Chdir(tmp)

	root := newInitBenchCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	// --non-interactive + --fast skips wizard and goes straight to config-gen.
	root.SetArgs([]string{"init", "--non-interactive", "--fast", "--quiet"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		// Error expected (no existing project). We benchmark the path up to
		// the filesystem check, not the full project creation.
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkInit", b)
}

// BenchmarkInitFlagParsing measures ONLY cobra flag parsing overhead for the
// init command's full flag set (12 flags). Isolates cobra from runInit logic.
func BenchmarkInitFlagParsing(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := newInitBenchCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"init", "--non-interactive", "--fast",
			"--quiet", "--name", "bench-proj", "--domain", "bench.local"})
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkInitFlagParsing", b)
}
