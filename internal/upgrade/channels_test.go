package upgrade

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsValid(t *testing.T) {
	for _, c := range []string{"stable", "rc", "edge"} {
		if !IsValid(c) {
			t.Errorf("IsValid(%q) = false, want true", c)
		}
	}
	for _, c := range []string{"", "beta", "dev", "nightly", "STABLE"} {
		if IsValid(c) {
			t.Errorf("IsValid(%q) = true, want false", c)
		}
	}
}

func TestParse(t *testing.T) {
	got, err := Parse("rc")
	if err != nil {
		t.Fatalf("Parse(rc) err: %v", err)
	}
	if got != ChannelRC {
		t.Errorf("Parse(rc) = %q, want %q", got, ChannelRC)
	}

	if _, err := Parse("beta"); err == nil {
		t.Errorf("Parse(beta) expected error, got nil")
	}
}

func TestFormulaName(t *testing.T) {
	cases := map[Channel]string{
		ChannelStable: "nself",
		ChannelRC:     "nself-rc",
		ChannelEdge:   "nself-edge",
	}
	for c, want := range cases {
		if got := FormulaName(c); got != want {
			t.Errorf("FormulaName(%q) = %q, want %q", c, got, want)
		}
	}
}

func TestTagSuffix(t *testing.T) {
	cases := map[Channel]string{
		ChannelStable: "",
		ChannelRC:     "-rc",
		ChannelEdge:   "-edge",
	}
	for c, want := range cases {
		if got := TagSuffix(c); got != want {
			t.Errorf("TagSuffix(%q) = %q, want %q", c, got, want)
		}
	}
}

func TestClassifyTag(t *testing.T) {
	cases := map[string]Channel{
		"v1.0.9":                       ChannelStable,
		"1.0.9":                        ChannelStable,
		"v1.0.9-rc1":                   ChannelRC,
		"v1.0.9-rc10":                  ChannelRC,
		"v1.0.9-edge.20260418.a3f7c21": ChannelEdge,
		"v2.0.0-beta.1":                ChannelStable, // unknown suffix defaults to stable
		"":                             ChannelStable,
	}
	for tag, want := range cases {
		if got := ClassifyTag(tag); got != want {
			t.Errorf("ClassifyTag(%q) = %q, want %q", tag, got, want)
		}
	}
}

func TestPingAPIEndpoint(t *testing.T) {
	cases := map[Channel]string{
		ChannelStable: "https://ping.nself.org/release/stable/latest",
		ChannelRC:     "https://ping.nself.org/release/rc/latest",
		ChannelEdge:   "https://ping.nself.org/release/edge/latest",
	}
	for c, want := range cases {
		if got := PingAPIEndpoint(c); got != want {
			t.Errorf("PingAPIEndpoint(%q) = %q, want %q", c, got, want)
		}
	}
}

// TestSaveAndLoadChannel uses a temp HOME so it doesn't clobber the real
// ~/.config/nself/channel.json of whoever runs the test.
func TestSaveAndLoadChannel(t *testing.T) {
	tmp := t.TempDir()
	origHome, had := os.LookupEnv("HOME")
	_ = os.Setenv("HOME", tmp)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("HOME", origHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
	})

	// Before anything is saved, LoadChannel must return the default.
	if got := LoadChannel(); got != DefaultChannel {
		t.Errorf("LoadChannel() = %q, want %q (default)", got, DefaultChannel)
	}

	// Save rc, load it back.
	if err := SaveChannel(ChannelRC); err != nil {
		t.Fatalf("SaveChannel(rc) err: %v", err)
	}
	if got := LoadChannel(); got != ChannelRC {
		t.Errorf("LoadChannel() = %q, want %q", got, ChannelRC)
	}

	// Verify file was written where ConfigPath says.
	want := filepath.Join(tmp, ".config", "nself", "channel.json")
	if _, err := os.Stat(want); err != nil {
		t.Errorf("expected %s to exist: %v", want, err)
	}

	// Save edge, load it back.
	if err := SaveChannel(ChannelEdge); err != nil {
		t.Fatalf("SaveChannel(edge) err: %v", err)
	}
	if got := LoadChannel(); got != ChannelEdge {
		t.Errorf("LoadChannel() = %q, want %q", got, ChannelEdge)
	}

	// Invalid channel must error.
	if err := SaveChannel(Channel("bogus")); err == nil {
		t.Errorf("SaveChannel(bogus) expected error, got nil")
	}
}

// TestLoadChannel_CorruptFileFallsBack verifies that a corrupt channel.json
// falls back to DefaultChannel without panicking.
func TestLoadChannel_CorruptFileFallsBack(t *testing.T) {
	tmp := t.TempDir()
	origHome, had := os.LookupEnv("HOME")
	_ = os.Setenv("HOME", tmp)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv("HOME", origHome)
		} else {
			_ = os.Unsetenv("HOME")
		}
	})

	// Write garbage to the config file.
	dir := filepath.Join(tmp, ".config", "nself")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "channel.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := LoadChannel(); got != DefaultChannel {
		t.Errorf("LoadChannel() on corrupt file = %q, want %q", got, DefaultChannel)
	}
}
