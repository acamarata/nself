package bundle

import "testing"

func TestNormalizeChannel(t *testing.T) {
	cases := []struct {
		in   Channel
		want Channel
	}{
		{"", ChannelStable},
		{"stable", ChannelStable},
		{"STABLE", ChannelStable},
		{" beta ", ChannelBeta},
		{"canary", ChannelCanary},
		{"weird", ChannelStable}, // unknown → stable
	}
	for _, tc := range cases {
		got := normalizeChannel(tc.in)
		if got != tc.want {
			t.Errorf("normalizeChannel(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

func TestChannelAllows(t *testing.T) {
	cases := []struct {
		ch     Channel
		status string
		want   bool
	}{
		// stable: only empty or "stable"
		{ChannelStable, "", true},
		{ChannelStable, "stable", true},
		{ChannelStable, "beta", false},
		{ChannelStable, "experimental", false},
		{ChannelStable, "deprecated", false},
		{ChannelStable, "eol", false},
		// beta: stable + beta
		{ChannelBeta, "stable", true},
		{ChannelBeta, "beta", true},
		{ChannelBeta, "experimental", false},
		{ChannelBeta, "deprecated", false},
		// canary: stable + beta + experimental, no deprecated/eol
		{ChannelCanary, "stable", true},
		{ChannelCanary, "beta", true},
		{ChannelCanary, "experimental", true},
		{ChannelCanary, "deprecated", false},
		{ChannelCanary, "eol", false},
	}
	for _, tc := range cases {
		got := channelAllows(tc.ch, tc.status)
		if got != tc.want {
			t.Errorf("channelAllows(%q, %q) = %v; want %v", tc.ch, tc.status, got, tc.want)
		}
	}
}

func TestResolveOnePluginVersion_NilRegistry(t *testing.T) {
	_, err := resolveOnePluginVersion(nil, "ai", ChannelStable)
	if err == nil {
		t.Fatal("expected error for nil registry")
	}
}

// ---------------------------------------------------------------------------
// Tests for cross-bundle MAX-resolution (S2.T22)
// ---------------------------------------------------------------------------

func TestResolveMaxVersion_ReturnsMax(t *testing.T) {
	got, err := ResolveMaxVersion("ai", []string{"1.0.0", "1.2.0", "1.1.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.0" {
		t.Errorf("ResolveMaxVersion = %q; want %q", got, "1.2.0")
	}
}

func TestResolveMaxVersion_SingleVersion(t *testing.T) {
	got, err := ResolveMaxVersion("claw", []string{"2.3.1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "2.3.1" {
		t.Errorf("ResolveMaxVersion = %q; want %q", got, "2.3.1")
	}
}

func TestResolveMaxVersion_WithVPrefix(t *testing.T) {
	got, err := ResolveMaxVersion("ai", []string{"v1.0.0", "v1.2.0", "v1.1.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "v1.2.0" {
		t.Errorf("ResolveMaxVersion = %q; want %q", got, "v1.2.0")
	}
}

func TestResolveMaxVersion_Empty(t *testing.T) {
	_, err := ResolveMaxVersion("ai", nil)
	if err == nil {
		t.Fatal("expected error for empty versions slice")
	}
}

func TestResolveMaxVersion_InvalidSemver(t *testing.T) {
	_, err := ResolveMaxVersion("ai", []string{"1.0.0", "not-a-version"})
	if err == nil {
		t.Fatal("expected error for invalid semver")
	}
}

func TestResolveMaxVersion_PreReleaseRanking(t *testing.T) {
	// "1.2.0" (no pre-release) should beat "1.2.0-beta.1" per semver rules.
	got, err := ResolveMaxVersion("ai", []string{"1.2.0-beta.1", "1.2.0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "1.2.0" {
		t.Errorf("ResolveMaxVersion = %q; want %q", got, "1.2.0")
	}
}

func TestResolveConflicts_MaxResolution(t *testing.T) {
	reqs := map[string][]string{
		"ai":  {"1.0.0", "1.2.0", "1.1.0"},
		"mux": {"2.0.0", "2.0.0"},
	}
	got, err := ResolveConflicts(reqs, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["ai"] != "1.2.0" {
		t.Errorf("ai resolved to %q; want %q", got["ai"], "1.2.0")
	}
	if got["mux"] != "2.0.0" {
		t.Errorf("mux resolved to %q; want %q", got["mux"], "2.0.0")
	}
}

func TestResolveConflicts_StrictModeError(t *testing.T) {
	reqs := map[string][]string{
		"ai": {"1.0.0", "1.2.0"},
	}
	_, err := ResolveConflicts(reqs, true)
	if err == nil {
		t.Fatal("expected error in strict mode with conflicting versions")
	}
}

func TestResolveConflicts_StrictModeNoConflict(t *testing.T) {
	reqs := map[string][]string{
		"ai": {"1.2.0", "1.2.0"},
	}
	got, err := ResolveConflicts(reqs, true)
	if err != nil {
		t.Fatalf("unexpected error in strict mode with identical versions: %v", err)
	}
	if got["ai"] != "1.2.0" {
		t.Errorf("ai resolved to %q; want %q", got["ai"], "1.2.0")
	}
}

func TestCompareSemver(t *testing.T) {
	cases := []struct {
		a, b string
		want int // sign only: -1, 0, or +1
	}{
		{"1.2.0", "1.1.0", 1},
		{"1.1.0", "1.2.0", -1},
		{"1.0.0", "1.0.0", 0},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0", "1.0.0-beta", 1},   // release > pre-release
		{"1.0.0-alpha", "1.0.0", -1}, // pre-release < release
		{"v1.0.0", "1.0.0", 0},       // v-prefix ignored
	}
	for _, tc := range cases {
		got, err := compareSemver(tc.a, tc.b)
		if err != nil {
			t.Errorf("compareSemver(%q, %q) error: %v", tc.a, tc.b, err)
			continue
		}
		gotSign := sign(got)
		if gotSign != tc.want {
			t.Errorf("compareSemver(%q, %q) sign = %d; want %d", tc.a, tc.b, gotSign, tc.want)
		}
	}
}

func sign(n int) int {
	if n > 0 {
		return 1
	}
	if n < 0 {
		return -1
	}
	return 0
}
