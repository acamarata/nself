package plugin

import (
	"testing"
)

func TestSatisfiesRange(t *testing.T) {
	tests := []struct {
		version  string
		rangeStr string
		want     bool
	}{
		{"1.0.3", ">=1.0.0 <2.0.0", true},
		{"1.0.0", ">=1.0.0 <2.0.0", true},
		{"2.0.0", ">=1.0.0 <2.0.0", false},
		{"0.9.9", ">=1.0.0 <2.0.0", false},
		{"1.5.0", ">=1.0.0", true},
		{"0.5.0", ">=1.0.0", false},
		{"14.2.0", ">=14", true},
		{"13.9.0", ">=14", false},
		{"2.30.0", ">=2.30", true},
		{"2.29.0", ">=2.30", false},
		{"v1.0.3", ">=1.0.0 <2.0.0", true},
		{"1.0.3-beta.1", ">=1.0.0 <2.0.0", true},
		{"1.0.0", "!=1.0.0", false},
		{"1.0.1", "!=1.0.0", true},
		{"1.2.3", "=1.2.3", true},
		{"1.2.4", "=1.2.3", false},
		{"1.2.3", "1.2.3", true},
		{"1.0.0", ">0.9.0 <1.1.0", true},
		{"1.0.0", "<=1.0.0", true},
		{"1.0.1", "<=1.0.0", false},
	}

	for _, tt := range tests {
		got, err := SatisfiesRange(tt.version, tt.rangeStr)
		if err != nil {
			t.Errorf("SatisfiesRange(%q, %q) error: %v", tt.version, tt.rangeStr, err)
			continue
		}
		if got != tt.want {
			t.Errorf("SatisfiesRange(%q, %q) = %v, want %v", tt.version, tt.rangeStr, got, tt.want)
		}
	}
}

func TestCheckCLICompat(t *testing.T) {
	// nil compat block: always passes
	if err := CheckCLICompat(nil, "1.0.0"); err != nil {
		t.Errorf("nil compat should pass: %v", err)
	}

	// empty nself field: always passes
	if err := CheckCLICompat(&CompatBlock{}, "1.0.0"); err != nil {
		t.Errorf("empty compat should pass: %v", err)
	}

	// in range
	cb := &CompatBlock{Nself: ">=1.0.0 <2.0.0"}
	if err := CheckCLICompat(cb, "1.0.3"); err != nil {
		t.Errorf("1.0.3 should satisfy >=1.0.0 <2.0.0: %v", err)
	}

	// out of range
	if err := CheckCLICompat(cb, "2.0.0"); err == nil {
		t.Error("2.0.0 should NOT satisfy >=1.0.0 <2.0.0")
	}
}

func TestCheckServiceCompat(t *testing.T) {
	cb := &CompatBlock{
		Requires: map[string]string{
			"postgres": ">=14",
			"hasura":   ">=2.30",
		},
	}

	actual := map[string]string{
		"postgres": "16.2.0",
		"hasura":   "2.35.0",
	}
	failures := CheckServiceCompat(cb, actual)
	if len(failures) != 0 {
		t.Errorf("expected no failures, got: %v", failures)
	}

	actual2 := map[string]string{
		"postgres": "13.0.0",
		"hasura":   "2.35.0",
	}
	failures2 := CheckServiceCompat(cb, actual2)
	if len(failures2) != 1 {
		t.Errorf("expected 1 failure, got %d: %v", len(failures2), failures2)
	}
}

func TestParseSemverTuple(t *testing.T) {
	tests := []struct {
		input string
		want  [3]int
	}{
		{"1.2.3", [3]int{1, 2, 3}},
		{"v1.2.3", [3]int{1, 2, 3}},
		{"14", [3]int{14, 0, 0}},
		{"2.30", [3]int{2, 30, 0}},
		{"1.0.0-beta.1", [3]int{1, 0, 0}},
		{"1.0.0+build123", [3]int{1, 0, 0}},
	}

	for _, tt := range tests {
		got, err := parseSemverTuple(tt.input)
		if err != nil {
			t.Errorf("parseSemverTuple(%q) error: %v", tt.input, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseSemverTuple(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}
