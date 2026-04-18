package commands

import (
	"testing"
)

func TestResolveTarget(t *testing.T) {
	cases := []struct {
		in      string
		want    string
		wantErr bool
	}{
		{"local", "local", false},
		{"staging", "staging", false},
		{"prod", "prod", false},
		{"production", "prod", false},
		{"PRODUCTION", "prod", false},
		{"  staging  ", "staging", false},
		{"preview", "", true},
		{"", "", true},
	}
	for _, c := range cases {
		got, err := resolveTarget(c.in)
		if c.wantErr {
			if err == nil {
				t.Errorf("resolveTarget(%q): expected error, got %q", c.in, got)
			}
			continue
		}
		if err != nil {
			t.Errorf("resolveTarget(%q): unexpected error: %v", c.in, err)
			continue
		}
		if got != c.want {
			t.Errorf("resolveTarget(%q): got %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDeployStrategies(t *testing.T) {
	for _, s := range []string{"rolling", "blue-green", "canary", "preview"} {
		if !deployStrategies[s] {
			t.Errorf("expected strategy %q to be allowed", s)
		}
	}
	for _, s := range []string{"", "recreate", "green"} {
		if deployStrategies[s] {
			t.Errorf("did not expect strategy %q to be allowed", s)
		}
	}
}

func TestStepStatus(t *testing.T) {
	if got := stepStatus(true, "done"); got != "pending" {
		t.Errorf("stepStatus(dryRun=true): got %q, want pending", got)
	}
	if got := stepStatus(false, "done"); got != "done" {
		t.Errorf("stepStatus(dryRun=false): got %q, want done", got)
	}
}
