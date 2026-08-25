package main

import "testing"

func TestScoreEnvVars(t *testing.T) {
	doc := []byte("The `PROJECT_NAME` variable controls the docker namespace. See also `BASE_DOMAIN`.")

	cases := []struct {
		name string
		vars []string
		want string
	}{
		{"no vars found", nil, "n/a"},
		{"all documented", []string{"PROJECT_NAME", "BASE_DOMAIN"}, "documented"},
		{"one undocumented", []string{"PROJECT_NAME", "NSELF_MADE_UP"}, "undocumented: NSELF_MADE_UP"},
		{"both undocumented", []string{"NSELF_A", "NSELF_B"}, "undocumented: NSELF_A, NSELF_B"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := scoreEnvVars(c.vars, doc)
			if got != c.want {
				t.Errorf("scoreEnvVars(%v) = %q, want %q", c.vars, got, c.want)
			}
		})
	}
}
