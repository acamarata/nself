package commands

import (
	"bytes"
	"errors"
	"reflect"
	"testing"

	"github.com/nself-org/cli/internal/errs"
	"github.com/nself-org/cli/internal/plugin"
	"github.com/spf13/pflag"
)

// TestStripRootPersistentFlags pins the fix for a regression that affected
// every command moved out of core.
//
// Before a command became a plugin, cobra consumed nself's persistent flags at
// the root and the subcommand never saw them. Once proxied, they were handed
// straight to a plugin binary that has no reason to know about them, so a
// script that had always run
//
//	nself soak --no-deprecation-warnings run
//
// started failing with "unknown flag: --no-deprecation-warnings".
func TestStripRootPersistentFlags(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "the flag that broke real scripts",
			args: []string{"--no-deprecation-warnings", "status"},
			want: []string{"status"},
		},
		{
			name: "flag after the subcommand is stripped too",
			args: []string{"status", "--no-monorepo"},
			want: []string{"status"},
		},
		{
			name: "inline value form",
			args: []string{"--no-monorepo=true", "run"},
			want: []string{"run"},
		},
		{
			name: "plugin flags are left alone",
			args: []string{"run", "--provider", "hetzner", "--force"},
			want: []string{"run", "--provider", "hetzner", "--force"},
		},
		{
			name: "a plugin flag whose name merely starts the same is kept",
			args: []string{"--no-monorepo-really", "run"},
			want: []string{"--no-monorepo-really", "run"},
		},
		{
			name: "everything after -- is positional and passes verbatim",
			args: []string{"exec", "--", "--no-monorepo", "arg"},
			want: []string{"exec", "--", "--no-monorepo", "arg"},
		},
		{
			name: "empty",
			args: []string{},
			want: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := stripRootPersistentFlags(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("stripRootPersistentFlags(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

// TestStripRootPersistentFlagsCoversEveryRootFlag guards the derivation itself.
// The list is read from RootCmd rather than hardcoded precisely so that adding
// a persistent flag to the CLI does not quietly start breaking plugins; this
// asserts that property rather than trusting it.
func TestStripRootPersistentFlagsCoversEveryRootFlag(t *testing.T) {
	RootCmd.PersistentFlags().VisitAll(func(f *pflag.Flag) {
		args := []string{"sub", "--" + f.Name}
		if f.Value.Type() != "bool" {
			args = append(args, "value")
		}
		got := stripRootPersistentFlags(args)
		if len(got) != 1 || got[0] != "sub" {
			t.Errorf("persistent flag --%s (%s) survived stripping: %q", f.Name, f.Value.Type(), got)
		}
	})
}

// TestReportProxyFailure pins which plugin failures the CLI speaks about.
//
// A plugin that ran and exited non-zero already printed its own error to the
// inherited stderr. The CLI adding "Plugin error: plugin exited with code 1"
// underneath was pure noise on every extracted command, and contradicted
// ExitCodeError.Silent(), which had promised the opposite since it was written.
func TestReportProxyFailure(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantPrint string
		wantCode  int
	}{
		{
			name:      "plugin ran and failed: say nothing, mirror the code",
			err:       &plugin.ExitCodeError{Code: 1},
			wantPrint: "",
			wantCode:  1,
		},
		{
			name:      "a plugin's own exit code is preserved, not flattened to 1",
			err:       &plugin.ExitCodeError{Code: 42},
			wantPrint: "",
			wantCode:  42,
		},
		{
			name:      "proxy could not run anything: the user has seen nothing yet",
			err:       errors.New("unknown command \"frobnicate\""),
			wantPrint: "Plugin error: unknown command \"frobnicate\"\n",
			wantCode:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := reportProxyFailure(&buf, tt.err)

			if buf.String() != tt.wantPrint {
				t.Errorf("printed %q, want %q", buf.String(), tt.wantPrint)
			}

			var coder errs.ExitCoder
			if !errors.As(got, &coder) {
				t.Fatalf("returned %v, which carries no exit code", got)
			}
			if coder.ExitCode() != tt.wantCode {
				t.Errorf("exit code = %d, want %d", coder.ExitCode(), tt.wantCode)
			}
		})
	}
}
