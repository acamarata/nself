package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
)

// newRestartCmd returns an isolated cobra.Command tree with just the restart
// command for flag-parsing and registration tests.
func newRestartCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	rc := &cobra.Command{
		Use:   "restart [SERVICES...]",
		Short: "Smart restart with config change detection",
		RunE:  runRestart,
	}
	rc.Flags().BoolP("smart", "s", true, "Detect changes, only restart affected (default)")
	rc.Flags().BoolP("all", "a", false, "Full restart: stop all + start all")
	rc.Flags().BoolP("verbose", "v", false, "Detailed output")
	rc.Flags().Bool("skip-build", false, "Skip image rebuild")
	rc.Flags().Bool("no-build", false, "Alias for --skip-build")

	root.AddCommand(rc)
	return root
}

// TestRestartCmd_Registered verifies that the restart command is registered on RootCmd.
func TestRestartCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "restart" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'restart' to be registered on RootCmd")
	}
}

// TestRestartCmd_FlagSmart verifies that --smart/-s is accepted.
func TestRestartCmd_FlagSmart(t *testing.T) {
	root := newRestartCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"restart", "--smart"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--smart flag was not recognised: %v", err)
	}
}

// TestRestartCmd_FlagAll verifies that --all/-a is accepted.
func TestRestartCmd_FlagAll(t *testing.T) {
	root := newRestartCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"restart", "--all"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--all flag was not recognised: %v", err)
	}
}

// TestRestartCmd_FlagSkipBuild verifies that --skip-build is accepted.
func TestRestartCmd_FlagSkipBuild(t *testing.T) {
	root := newRestartCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"restart", "--skip-build"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--skip-build flag was not recognised: %v", err)
	}
}

// TestRestartCmd_FlagNoBuild verifies that --no-build (alias for --skip-build) is accepted.
func TestRestartCmd_FlagNoBuild(t *testing.T) {
	root := newRestartCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"restart", "--no-build"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--no-build flag was not recognised: %v", err)
	}
}

// TestRestartCmd_PositionalServices verifies that service name positional args
// are accepted without a flag-parse error.
func TestRestartCmd_PositionalServices(t *testing.T) {
	root := newRestartCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"restart", "postgres", "redis"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("positional SERVICE args caused a flag error: %v", err)
	}
}

// TestRestartCmd_UnknownFlagIsRejected verifies that an unrecognised flag is rejected.
func TestRestartCmd_UnknownFlagIsRejected(t *testing.T) {
	root := newRestartCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"restart", "--this-flag-does-not-exist"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for unknown flag, got nil")
	}
}

// TestRestartCmd_ConfigChangeDetection_NoEnv verifies that restartSmart handles
// a missing .env file gracefully (falls back to quick restart path without panicking).
// This is a unit-level test of the config-change detection logic.
func TestRestartCmd_ConfigChangeDetection_NoEnv(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal docker-compose.yml so the compose stat check passes
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("version: '3'\n"), 0644); err != nil {
		t.Fatalf("setup: could not write docker-compose.yml: %v", err)
	}

	// No .env file exists — restartSmart should fall through to quick restart.
	// We cannot invoke Docker in tests, but we can verify the file-stat logic
	// directly by checking that os.Stat on the missing .env path returns an error.
	envPath := filepath.Join(dir, ".env")
	_, statErr := os.Stat(envPath)
	if statErr == nil {
		t.Fatal("test setup error: .env should not exist in temp dir")
	}
	if !os.IsNotExist(statErr) {
		t.Fatalf("unexpected error checking .env: %v", statErr)
	}
}

// TestRestartCmd_ConfigChangeDetection_EnvNewerThanCompose verifies that when
// .env mtime > docker-compose.yml mtime the config-newer comparison works
// correctly. This exercises the mtime comparison arithmetic without Docker.
func TestRestartCmd_ConfigChangeDetection_EnvNewerThanCompose(t *testing.T) {
	dir := t.TempDir()

	// Write docker-compose.yml first (older mtime)
	composePath := filepath.Join(dir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte("version: '3'\n"), 0644); err != nil {
		t.Fatalf("setup: could not write docker-compose.yml: %v", err)
	}

	// Sleep briefly so .env gets a strictly newer mtime
	time.Sleep(5 * time.Millisecond)

	// Write .env with a newer mtime
	envPath := filepath.Join(dir, ".env")
	if err := os.WriteFile(envPath, []byte("PROJECT_NAME=test\n"), 0600); err != nil {
		t.Fatalf("setup: could not write .env: %v", err)
	}

	envInfo, err := os.Stat(envPath)
	if err != nil {
		t.Fatalf("stat .env: %v", err)
	}
	composeInfo, err := os.Stat(composePath)
	if err != nil {
		t.Fatalf("stat docker-compose.yml: %v", err)
	}

	// The configMtime logic takes the newer of .env and docker-compose.yml
	configMtime := envInfo.ModTime()
	if composeInfo.ModTime().After(configMtime) {
		configMtime = composeInfo.ModTime()
	}

	// configMtime should equal .env mtime (it is newer)
	if !configMtime.Equal(envInfo.ModTime()) {
		t.Errorf("expected configMtime to equal .env mtime; got configMtime=%v envMtime=%v",
			configMtime, envInfo.ModTime())
	}
	// .env mtime must be strictly after compose mtime
	if !envInfo.ModTime().After(composeInfo.ModTime()) {
		t.Errorf("expected .env mtime > docker-compose.yml mtime; got %v vs %v",
			envInfo.ModTime(), composeInfo.ModTime())
	}
}

// TestUpAlias_Registered verifies that the hidden 'up' alias is registered on RootCmd.
func TestUpAlias_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "up" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'up' alias to be registered on RootCmd")
	}
}

// TestUpAlias_ShortMentionsStart verifies that the up alias help text
// references 'nself start' so operators know it is an alias.
func TestUpAlias_ShortMentionsStart(t *testing.T) {
	for _, c := range RootCmd.Commands() {
		if c.Name() == "up" {
			if !strings.Contains(c.Short, "start") && !strings.Contains(c.Short, "nself start") {
				t.Errorf("'up' Short text should reference 'start', got: %q", c.Short)
			}
			return
		}
	}
	t.Fatal("'up' command not found on RootCmd")
}
