package commands

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/cobra"
)

// newPluginBenchCmd returns an isolated cobra tree for plugin benchmarks.
func newPluginBenchCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	pc := &cobra.Command{
		Use:   "plugin",
		Short: "Manage nSelf plugins",
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	installCmd := &cobra.Command{
		Use:   "install <plugin> [plugin...]",
		Short: "Install one or more plugins",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runPluginInstall,
	}
	installCmd.Flags().Bool("force", false, "Force install even if already installed")
	installCmd.Flags().Bool("skip-deps", false, "Skip dependency resolution")
	installCmd.Flags().Bool("offline", false, "Install from local cache only")

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List available and installed plugins",
		RunE:  runPluginList,
	}
	listCmd.Flags().Bool("installed", false, "Show installed plugins only")
	listCmd.Flags().Bool("available", false, "Show available plugins only")
	listCmd.Flags().Bool("json", false, "Output as JSON")

	pc.AddCommand(installCmd)
	pc.AddCommand(listCmd)
	root.AddCommand(pc)
	return root
}

// BenchmarkPluginInstallCached measures `nself plugin install` dispatch overhead
// when using the local cache (NSELF_PLUGIN_REGISTRY pointing to 127.0.0.1).
//
// The command fails fast (registry unreachable) but exercises: flag parsing,
// NSELF_LICENSE_SKIP_VERIFY gate, registry URL lookup, and context setup —
// the full hot path before actual network I/O.
//
// SLO: cached plugin install p95 < 4s (F16-PERF-SLOS.md — Plugin Install SLOs).
// Run: go test -bench=BenchmarkPluginInstall -benchmem ./cmd/commands/
func BenchmarkPluginInstallCached(b *testing.B) {
	b.ReportAllocs()

	// Point to unreachable local address so the benchmark is network-free.
	// This exercises the full pre-network code path including license validation
	// dispatch logic (skipped because NSELF_LICENSE_SKIP_VERIFY is not set).
	b.Setenv("NSELF_PLUGIN_REGISTRY", "http://127.0.0.1:19998/unreachable-bench")
	b.Setenv("NSELF_LICENSE_SKIP_VERIFY", "")

	root := newPluginBenchCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"plugin", "install", "monitoring"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		// Error expected (no license key). Benchmarks dispatch + gate overhead.
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkPluginInstallCached", b)
}

// BenchmarkPluginList measures `nself plugin list` listing overhead.
// The command reads the local registry; this exercises registry load + format.
func BenchmarkPluginList(b *testing.B) {
	b.ReportAllocs()

	// Use a temp dir so there is no real .nself state to load.
	tmp := b.TempDir()
	b.Chdir(tmp)
	_ = os.MkdirAll(tmp+"/.nself", 0700)

	root := newPluginBenchCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"plugin", "list"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		buf.Reset()
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkPluginList", b)
}

// BenchmarkPluginFlagParsing measures cobra flag-parsing overhead for the plugin
// install subcommand's 3-flag set. Isolates framework from business logic.
func BenchmarkPluginFlagParsing(b *testing.B) {
	b.ReportAllocs()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		root := newPluginBenchCmd()
		var buf bytes.Buffer
		root.SetOut(&buf)
		root.SetErr(&buf)
		root.SetArgs([]string{"plugin", "install", "--force", "--offline", "monitoring"})
		_ = root.Execute()
	}

	emitBenchmarkResult("BenchmarkPluginFlagParsing", b)
}
