package commands

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// newVersionTestCmd returns a fresh isolated cobra tree with only the version
// command wired, matching the pattern established in build_test.go.
func newVersionTestCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	vc := &cobra.Command{
		Use:   "version",
		Short: "Show version and system information",
		RunE:  versionCmd.RunE,
	}
	vc.Flags().Bool("short", false, "Print version number only")
	vc.Flags().Bool("json", false, "Print version info as JSON")

	root.AddCommand(vc)
	return root
}

// BenchmarkVersion measures cobra startup + version string formatting overhead.
//
// SLO: p95 < 150ms (F16-PERF-SLOS.md — CLI Operations: nself version).
// Run: go test -bench=BenchmarkVersion -benchmem ./cmd/commands/
func BenchmarkVersion(b *testing.B) {
	b.ReportAllocs()

	root := newVersionTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"version", "--short"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := root.Execute(); err != nil {
			b.Fatalf("version --short failed: %v", err)
		}
	}

	emitBenchmarkResult("BenchmarkVersion", b)
}

// BenchmarkVersionJSON measures version command with JSON output serialisation.
// JSON output adds allocation overhead; this sub-benchmark isolates it.
func BenchmarkVersionJSON(b *testing.B) {
	b.ReportAllocs()

	root := newVersionTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"version", "--json"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		if err := root.Execute(); err != nil {
			b.Fatalf("version --json failed: %v", err)
		}
	}

	emitBenchmarkResult("BenchmarkVersionJSON", b)
}

// BenchmarkVersionHelp measures cobra --help flag traversal overhead.
// The help path exercises cobra's flag registration + usage string building.
func BenchmarkVersionHelp(b *testing.B) {
	b.ReportAllocs()

	root := newVersionTestCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"version", "--help"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = root.Execute() // --help returns nil in cobra.
	}

	emitBenchmarkResult("BenchmarkVersionHelp", b)
}

// emitBenchmarkResult writes a NDJSON line to $BENCH_RESULTS_FILE when set.
// The CI pipeline (perf.yml) sets BENCH_RESULTS_FILE so perf-compare.sh can
// aggregate all benchmark outputs into results.json.
func emitBenchmarkResult(name string, b *testing.B) {
	b.Helper()

	outPath := os.Getenv("BENCH_RESULTS_FILE")
	if outPath == "" {
		return
	}

	nsPerOp := int64(0)
	if b.N > 0 {
		nsPerOp = b.Elapsed().Nanoseconds() / int64(b.N)
	}

	type entry struct {
		Name    string  `json:"name"`
		NsPerOp int64   `json:"ns_per_op"`
		MsPerOp float64 `json:"ms_per_op"`
		Allocs  int64   `json:"allocs_per_op"`
	}

	e := entry{
		Name:    name,
		NsPerOp: nsPerOp,
		MsPerOp: float64(nsPerOp) / 1e6,
	}

	data, err := json.Marshal(e)
	if err != nil {
		return
	}

	f, err := os.OpenFile(outPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(data, '\n'))
}
