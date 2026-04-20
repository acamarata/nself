package commands

import (
	"bytes"
	"testing"

	"github.com/spf13/cobra"
)

// newDoctorBenchCmd returns an isolated cobra tree for doctor benchmarks.
// Uses a stub RunE (mirrors doctor_test.go pattern) so no Docker is required.
func newDoctorBenchCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	dc := &cobra.Command{
		Use:   "doctor",
		Short: "Run system diagnostics",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Stub: exercises flag parsing + cobra dispatch overhead without
			// requiring Docker or a live nSelf stack.
			return nil
		},
	}
	dc.Flags().BoolP("verbose", "v", false, "Show verbose output")
	dc.Flags().Bool("full", false, "Run all checks including optional services")
	dc.Flags().Bool("deep", false, "Run deep diagnostics (12 subsystems)")
	dc.Flags().Bool("fix", false, "Attempt to auto-fix issues")
	dc.Flags().Bool("json", false, "Output as JSON")
	dc.Flags().String("format", "", "Output format: table|json|plain")
	dc.Flags().String("only", "", "Run only named check section")
	dc.Flags().Bool("ai", false, "Include AI model checks")
	dc.Flags().Bool("yes", false, "Auto-confirm fix prompts")
	dc.Flags().Bool("skip-ollama", false, "Skip Ollama availability check")
	dc.Flags().Bool("skip-pool", false, "Skip pool sizing check (PERF-POOL-01)")

	root.AddCommand(dc)
	return root
}

// BenchmarkDoctor measures cobra dispatch + flag-parse overhead for `nself doctor`.
//
// The stub RunE returns immediately, so this isolates cobra framework cost from
// actual Docker/system check cost. Use as a baseline: if this number regresses
// the cobra framework was modified.
//
// SLO: nself doctor p95 < 8s (F16-PERF-SLOS.md — CLI Operations: nself doctor).
// This benchmark measures pre-check overhead only, not full check duration.
// Run: go test -bench=BenchmarkDoctor -benchmem ./cmd/commands/
func BenchmarkDoctor(b *testing.B) {
	b.ReportAllocs()

	root := newDoctorBenchCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"doctor"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := root.Execute(); err != nil {
			b.Fatalf("doctor command failed: %v", err)
		}
	}

	emitBenchmarkResult("BenchmarkDoctor", b)
}

// BenchmarkDoctorDeep measures dispatch overhead for `nself doctor --deep`.
// The --deep flag enables 12 subsystem checks; this benchmarks the flag
// traversal and dispatch path (stub only, no actual checks).
func BenchmarkDoctorDeep(b *testing.B) {
	b.ReportAllocs()

	root := newDoctorBenchCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"doctor", "--deep", "--json"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := root.Execute(); err != nil {
			b.Fatalf("doctor --deep failed: %v", err)
		}
	}

	emitBenchmarkResult("BenchmarkDoctorDeep", b)
}

// BenchmarkDoctorFlagParsing measures cobra flag-parsing overhead for doctor's
// full 11-flag set. Isolates framework from any application logic.
func BenchmarkDoctorFlagParsing(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := newDoctorBenchCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"doctor", "--verbose", "--deep", "--json",
			"--only", "system", "--skip-ollama", "--skip-pool"})
		if err := root.Execute(); err != nil {
			b.Fatalf("doctor flag parsing failed: %v", err)
		}
	}

	emitBenchmarkResult("BenchmarkDoctorFlagParsing", b)
}
