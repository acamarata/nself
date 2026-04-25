package sdk

import (
	"fmt"
	"strconv"
	"strings"
)

// SemVer represents a parsed semantic version.
type SemVer struct {
	Major, Minor, Patch int
}

// ParseSemVer parses a "X.Y.Z" string (optional "v" prefix). Pre-release /
// build-metadata suffixes are stripped.
func ParseSemVer(s string) (SemVer, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "v")
	if i := strings.IndexAny(s, "-+"); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) < 1 || len(parts) > 3 {
		return SemVer{}, fmt.Errorf("semver: invalid format %q", s)
	}
	out := SemVer{}
	for i, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil {
			return SemVer{}, fmt.Errorf("semver: %q: %w", s, err)
		}
		switch i {
		case 0:
			out.Major = n
		case 1:
			out.Minor = n
		case 2:
			out.Patch = n
		}
	}
	return out, nil
}

// Compare returns -1 if v<o, 0 if v==o, 1 if v>o.
func (v SemVer) Compare(o SemVer) int {
	if v.Major != o.Major {
		if v.Major < o.Major {
			return -1
		}
		return 1
	}
	if v.Minor != o.Minor {
		if v.Minor < o.Minor {
			return -1
		}
		return 1
	}
	if v.Patch != o.Patch {
		if v.Patch < o.Patch {
			return -1
		}
		return 1
	}
	return 0
}

// String renders the version as "X.Y.Z".
func (v SemVer) String() string {
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// CheckMinSDK reports an error if the current SDK version is lower than
// required. Plugins call this at startup to halt with a clear message when
// they're built against a newer SDK than the one linked at runtime (e.g.
// via an override in go.mod).
func CheckMinSDK(required string) error {
	have, err := ParseSemVer(Version)
	if err != nil {
		return fmt.Errorf("sdk version unparsable: %w", err)
	}
	want, err := ParseSemVer(required)
	if err != nil {
		return fmt.Errorf("required version unparsable: %w", err)
	}
	if have.Compare(want) < 0 {
		return fmt.Errorf("plugin-sdk-go %s is older than required %s", have, want)
	}
	return nil
}

// CheckCLICompat reports an error if currentCLI falls outside [minCLI, maxCLI].
// Empty maxCLI means no upper bound. Empty minCLI means no lower bound.
func CheckCLICompat(currentCLI, minCLI, maxCLI string) error {
	cur, err := ParseSemVer(currentCLI)
	if err != nil {
		return fmt.Errorf("current CLI version unparsable: %w", err)
	}
	if minCLI != "" {
		lo, err := ParseSemVer(minCLI)
		if err != nil {
			return fmt.Errorf("min CLI version unparsable: %w", err)
		}
		if cur.Compare(lo) < 0 {
			return fmt.Errorf("CLI %s is older than required %s", cur, lo)
		}
	}
	if maxCLI != "" {
		hi, err := ParseSemVer(maxCLI)
		if err != nil {
			return fmt.Errorf("max CLI version unparsable: %w", err)
		}
		if cur.Compare(hi) > 0 {
			return fmt.Errorf("CLI %s is newer than supported %s", cur, hi)
		}
	}
	return nil
}
