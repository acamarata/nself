package docker

// Purpose: Regression tests for PCI ntask-hasura-container-hash-name:
//          stale Compose rename-leftovers (b6d7b59a1c78_ntask_hasura) must be
//          detected so cleanup restores the clean <project>_<service> names.

import "testing"

func TestIsRenamedComposeLeftover(t *testing.T) {
	cases := []struct {
		name    string
		project string
		want    bool
	}{
		{"b6d7b59a1c78_ntask_hasura", "ntask", true},
		{"0123abcd_ntask_postgres", "ntask", true},
		{"ntask_hasura", "ntask", false},                   // clean name
		{"b6d7b59a1c78_other_hasura", "ntask", false},      // different project
		{"deadbeef1234_ntaskextra_hasura", "ntask", false}, // prefix, not project
		{"hasura", "ntask", false},
		{"b6d7_ntask_hasura", "ntask", false}, // hex too short (<8)
	}
	for _, c := range cases {
		if got := isRenamedComposeLeftover(c.name, c.project); got != c.want {
			t.Errorf("isRenamedComposeLeftover(%q, %q) = %v, want %v", c.name, c.project, got, c.want)
		}
	}
}
