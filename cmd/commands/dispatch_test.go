package commands

import (
	"reflect"
	"testing"

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
