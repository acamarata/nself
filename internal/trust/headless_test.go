package trust

import (
	"testing"
)

// TestIsHeadless covers all documented truthy values plus falsy values.
func TestIsHeadless(t *testing.T) {
	tests := []struct {
		name  string
		env   string
		want  bool
	}{
		// Truthy values
		{name: "value_1", env: "1", want: true},
		{name: "value_true_lower", env: "true", want: true},
		{name: "value_true_upper", env: "TRUE", want: true},
		{name: "value_true_title", env: "True", want: true},
		{name: "value_yes_lower", env: "yes", want: true},
		{name: "value_yes_upper", env: "YES", want: true},
		{name: "value_yes_title", env: "Yes", want: true},
		{name: "value_on_lower", env: "on", want: true},
		{name: "value_on_upper", env: "ON", want: true},
		{name: "value_on_title", env: "On", want: true},
		// Falsy values
		{name: "empty", env: "", want: false},
		{name: "value_0", env: "0", want: false},
		{name: "value_false", env: "false", want: false},
		{name: "value_FALSE", env: "FALSE", want: false},
		{name: "value_no", env: "no", want: false},
		{name: "value_off", env: "off", want: false},
		{name: "value_random", env: "random-string", want: false},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("NSELF_HEADLESS", tc.env)
			got := isHeadless()
			if got != tc.want {
				t.Errorf("isHeadless() with NSELF_HEADLESS=%q = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}
