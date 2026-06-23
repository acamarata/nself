// Purpose: Tests for claw proxy — port constant, CORS helpers, and origin checks.
//
// Acceptance: claw proxy port constant is 3760 (nself-ai-cc canonical gateway).
// SPORT: F02-COMMAND-INVENTORY row: claw proxy

package commands

import (
	"testing"
)

func TestClawAICCPort(t *testing.T) {
	if clawAICCPort != 3760 {
		t.Errorf("clawAICCPort = %d, want 3760 (nself-ai-cc canonical gateway)", clawAICCPort)
	}
}

func TestParseAllowedOrigins(t *testing.T) {
	cases := []struct {
		name  string
		env   string
		want  map[string]bool
		count int
	}{
		{
			name:  "empty",
			env:   "",
			count: 0,
		},
		{
			name:  "single origin",
			env:   "https://app.example.com",
			count: 1,
		},
		{
			name:  "csv origins",
			env:   "https://app.example.com,https://other.example.com",
			count: 2,
		},
		{
			name:  "wildcard is excluded",
			env:   "*",
			count: 0,
		},
		{
			name:  "whitespace trimmed",
			env:   "  https://app.example.com , https://b.example.com  ",
			count: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseAllowedOrigins(tc.env)
			if len(got) != tc.count {
				t.Errorf("parseAllowedOrigins(%q) returned %d origins, want %d", tc.env, len(got), tc.count)
			}
		})
	}
}

func TestIsLocalhostOrigin(t *testing.T) {
	cases := []struct {
		origin string
		want   bool
	}{
		{"http://localhost", true},
		{"https://localhost", true},
		{"http://localhost:3000", true},
		{"http://127.0.0.1", true},
		{"http://127.0.0.1:8080", true},
		{"https://127.0.0.1:9000", true},
		{"https://example.com", false},
		{"http://myapp.localhost.evil.com", false},
		{"", false},
	}

	for _, tc := range cases {
		got := isLocalhostOrigin(tc.origin)
		if got != tc.want {
			t.Errorf("isLocalhostOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}
