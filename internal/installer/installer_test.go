package installer

import (
	"errors"
	"strings"
	"testing"
)

// TestInstallerError_Error verifies the Error() string format.
func TestInstallerError_Error(t *testing.T) {
	cases := []struct {
		name     string
		err      *InstallerError
		wantSubs []string
	}{
		{
			name:     "with wrapped error",
			err:      errf(ErrOllamaInstallFailed, "download failed", errors.New("connection refused")),
			wantSubs: []string{ErrOllamaInstallFailed, "download failed", "connection refused"},
		},
		{
			name:     "without wrapped error",
			err:      errf(ErrUnsupportedOS, "only linux/macOS supported", nil),
			wantSubs: []string{ErrUnsupportedOS, "only linux/macOS supported"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := tc.err.Error()
			for _, sub := range tc.wantSubs {
				if !strings.Contains(s, sub) {
					t.Errorf("Error() = %q, want substring %q", s, sub)
				}
			}
		})
	}
}

// TestInstallerError_Codes verifies all error code constants are non-empty.
func TestInstallerError_Codes(t *testing.T) {
	codes := []string{
		ErrOllamaInstallFailed,
		ErrSystemdUnavailable,
		ErrIptablesNoPermission,
		ErrPortBindConflict,
		ErrRAMInsufficient,
		ErrUnsupportedOS,
	}
	for _, c := range codes {
		if c == "" {
			t.Errorf("error code constant is empty string")
		}
	}
}

// TestExpectedOllamaInstallChecksum_NonEmpty verifies that the pinned checksum
// is always a non-empty hex string. An empty value disables integrity
// verification and is a supply-chain security risk.
func TestExpectedOllamaInstallChecksum_NonEmpty(t *testing.T) {
	checksum := ExpectedOllamaInstallChecksum()
	if checksum == "" {
		t.Error("ExpectedOllamaInstallChecksum() returned empty string: checksum pinning is required")
	}
	if len(checksum) != 64 {
		t.Errorf("ExpectedOllamaInstallChecksum() returned %q (len=%d), want a 64-char SHA-256 hex string", checksum, len(checksum))
	}
}

// TestDefaultChecksumVersion_NonEmpty verifies the version constant is set.
func TestDefaultChecksumVersion_NonEmpty(t *testing.T) {
	if DefaultChecksumVersion == "" {
		t.Error("DefaultChecksumVersion should not be empty")
	}
}

// TestRecommendationMatrix_HasCurrentVersion verifies the matrix includes an
// entry for the default checksum version.
func TestRecommendationMatrix_HasCurrentVersion(t *testing.T) {
	tiers, ok := RecommendationMatrix[DefaultChecksumVersion]
	if !ok {
		t.Fatalf("RecommendationMatrix missing entry for DefaultChecksumVersion=%q", DefaultChecksumVersion)
	}
	if len(tiers) == 0 {
		t.Error("RecommendationMatrix[current] is empty")
	}
}

// TestModelRec_TierKeys verifies all declared tier keys exist in the matrix.
func TestModelRec_TierKeys(t *testing.T) {
	allTiers := []TierKey{TierNone, TierTiny, TierSmall, TierBalanced, TierMedium, TierLarge, TierXL, TierXXL}
	tiers, ok := RecommendationMatrix[DefaultChecksumVersion]
	if !ok {
		t.Skipf("no matrix for version %q", DefaultChecksumVersion)
	}
	for _, tier := range allTiers {
		if _, exists := tiers[tier]; !exists {
			t.Errorf("matrix[%q] missing tier %q", DefaultChecksumVersion, tier)
		}
	}
}

// TestModelRec_AllModelsHaveTasks verifies every ModelRec in the matrix has
// at least one task assigned.
func TestModelRec_AllModelsHaveTasks(t *testing.T) {
	tiers, ok := RecommendationMatrix[DefaultChecksumVersion]
	if !ok {
		t.Skipf("no matrix for version %q", DefaultChecksumVersion)
	}
	for tier, models := range tiers {
		for _, m := range models {
			if m.Name == "" {
				t.Errorf("tier %q: model with empty name", tier)
			}
			if len(m.Tasks) == 0 {
				t.Errorf("tier %q model %q: no tasks assigned", tier, m.Name)
			}
		}
	}
}
