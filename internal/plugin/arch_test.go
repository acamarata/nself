package plugin

import (
	"errors"
	"testing"
)

// TestArchForPlatform_AllFiveCombinations verifies that all five supported
// GOOS/GOARCH pairs resolve to the correct canonical arch string.
func TestArchForPlatform_AllFiveCombinations(t *testing.T) {
	cases := []struct {
		goos   string
		goarch string
		want   string
	}{
		{"darwin", "arm64", "darwin-arm64"},
		{"darwin", "amd64", "darwin-amd64"},
		{"linux", "amd64", "linux-amd64"},
		{"linux", "arm64", "linux-arm64"},
		{"windows", "amd64", "windows-amd64"},
	}
	for _, tc := range cases {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			got, err := archForPlatform(tc.goos, tc.goarch)
			if err != nil {
				t.Fatalf("archForPlatform(%q, %q): unexpected error: %v", tc.goos, tc.goarch, err)
			}
			if got != tc.want {
				t.Errorf("archForPlatform(%q, %q): got %q, want %q", tc.goos, tc.goarch, got, tc.want)
			}
		})
	}
}

// TestArchForPlatform_UnsupportedPairs verifies that unsupported combinations
// return ErrUnsupportedArch.
func TestArchForPlatform_UnsupportedPairs(t *testing.T) {
	unsupported := []struct {
		goos   string
		goarch string
	}{
		{"linux", "386"},
		{"windows", "arm64"},
		{"freebsd", "amd64"},
		{"", ""},
	}
	for _, tc := range unsupported {
		t.Run(tc.goos+"/"+tc.goarch, func(t *testing.T) {
			_, err := archForPlatform(tc.goos, tc.goarch)
			if err == nil {
				t.Fatalf("archForPlatform(%q, %q): expected error, got nil", tc.goos, tc.goarch)
			}
			if !errors.Is(err, ErrUnsupportedArch) {
				t.Errorf("archForPlatform(%q, %q): want ErrUnsupportedArch, got %v", tc.goos, tc.goarch, err)
			}
		})
	}
}
