//go:build darwin || linux

package maintenance

import (
	"strings"
	"testing"
)

// TestHasuraMetadataPlistContainsTZUTC verifies that the macOS LaunchDaemon
// plist template enforces TZ=UTC so the 02:00 schedule fires at UTC time
// regardless of the host's local timezone (matches systemd timer behaviour).
func TestHasuraMetadataPlistContainsTZUTC(t *testing.T) {
	// Build the plist string the same way installHasuraMetadataLaunchDaemon
	// does, substituting placeholder values for the label and binary path.
	plist := buildHasuraMetadataPlist("com.nself.hasura-metadata-backup", "/usr/local/bin/nself")

	checks := []struct {
		desc    string
		snippet string
	}{
		{"EnvironmentVariables key present", "<key>EnvironmentVariables</key>"},
		{"TZ key present", "<key>TZ</key>"},
		{"TZ value is UTC", "<string>UTC</string>"},
	}

	for _, c := range checks {
		if !strings.Contains(plist, c.snippet) {
			t.Errorf("plist missing %s: expected to find %q", c.desc, c.snippet)
		}
	}
}
