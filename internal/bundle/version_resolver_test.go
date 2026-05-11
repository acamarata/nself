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
