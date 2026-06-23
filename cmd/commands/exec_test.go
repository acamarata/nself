package commands

import (
	"bytes"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newExecCmd returns an isolated cobra.Command tree with just the exec command
// for flag-parsing and registration tests.
func newExecCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	ec := &cobra.Command{
		Use:   "exec [FLAGS] <SERVICE> [COMMAND...]",
		Short: "Execute a command inside a service container",
		Args:  cobra.MinimumNArgs(1),
		RunE:  runExec,
	}
	ec.Flags().BoolP("no-tty", "T", false, "Disable pseudo-TTY allocation")
	ec.Flags().StringP("user", "u", "", "Run command as this user")
	ec.Flags().StringP("workdir", "w", "", "Working directory inside the container")
	ec.Flags().StringSliceP("env", "e", nil, "Set environment variables (KEY=VALUE)")

	root.AddCommand(ec)
	return root
}

// TestExecCmd_Registered verifies that the exec command is registered on RootCmd.
func TestExecCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "exec" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'exec' to be registered on RootCmd")
	}
}

// TestExecCmd_RequiresServiceArg verifies that running exec without a SERVICE
// positional argument returns a cobra usage error, not a panic.
func TestExecCmd_RequiresServiceArg(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"exec"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error when exec is called without SERVICE arg, got nil")
	}
}

// TestExecCmd_FlagNoTTY verifies that --no-tty / -T is accepted.
func TestExecCmd_FlagNoTTY(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	// Provide a SERVICE arg to pass Args validation; command will fail on Docker
	// but the flag must be parsed without error.
	root.SetArgs([]string{"exec", "--no-tty", "postgres"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--no-tty flag was not recognised: %v", err)
	}
}

// TestExecCmd_FlagNoTTYShort verifies the -T shorthand is accepted.
func TestExecCmd_FlagNoTTYShort(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"exec", "-T", "postgres"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("-T flag was not recognised: %v", err)
	}
}

// TestExecCmd_FlagUser verifies that --user/-u is accepted.
func TestExecCmd_FlagUser(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"exec", "--user=root", "nginx"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--user flag was not recognised: %v", err)
	}
}

// TestExecCmd_FlagWorkdir verifies that --workdir/-w is accepted.
func TestExecCmd_FlagWorkdir(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"exec", "--workdir=/tmp", "postgres"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--workdir flag was not recognised: %v", err)
	}
}

// TestExecCmd_FlagEnv verifies that --env/-e accepts KEY=VALUE pairs.
func TestExecCmd_FlagEnv(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"exec", "--env=FOO=bar", "postgres"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--env flag was not recognised: %v", err)
	}
}

// TestExecCmd_PositionalServiceAndCommand verifies that SERVICE + COMMAND args
// are accepted without a flag-parse error.
func TestExecCmd_PositionalServiceAndCommand(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"exec", "postgres", "--", "pg_dump", "mydb"})
	err := root.Execute()
	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("positional args caused a flag error: %v", err)
	}
}

// TestExecCmd_UnknownFlagIsRejected verifies that an unrecognised flag is rejected.
func TestExecCmd_UnknownFlagIsRejected(t *testing.T) {
	root := newExecCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs([]string{"exec", "--this-flag-does-not-exist", "postgres"})
	err := root.Execute()
	if err == nil {
		t.Fatal("expected an error for unknown flag, got nil")
	}
}
