package plugin

import "testing"

// TestVersionPattern_Valid verifies that a well-formed semver string matches.
func TestVersionPattern_Valid(t *testing.T) {
	if !versionPattern.MatchString("1.2.3") {
		t.Error("expected \"1.2.3\" to match versionPattern, but it did not")
	}
}

// TestVersionPattern_PreRelease verifies that pre-release suffixes are accepted.
func TestVersionPattern_PreRelease(t *testing.T) {
	if !versionPattern.MatchString("1.2.3-alpha") {
		t.Error("expected \"1.2.3-alpha\" to match versionPattern, but it did not")
	}
}

// TestVersionPattern_ShellInjection verifies that shell metacharacters are rejected.
func TestVersionPattern_ShellInjection(t *testing.T) {
	if versionPattern.MatchString("1.2.3; rm -rf /") {
		t.Error("expected \"1.2.3; rm -rf /\" to NOT match versionPattern, but it did")
	}
}

// TestVersionPattern_FourParts verifies that four-component version strings are rejected.
func TestVersionPattern_FourParts(t *testing.T) {
	if versionPattern.MatchString("1.2.3.4") {
		t.Error("expected \"1.2.3.4\" to NOT match versionPattern, but it did")
	}
}

// TestVersionPattern_ZeroVersion verifies that 0.0.0 is accepted as a valid semver string.
func TestVersionPattern_ZeroVersion(t *testing.T) {
	if !versionPattern.MatchString("0.0.0") {
		t.Error("expected \"0.0.0\" to match versionPattern, but it did not")
	}
}

// TestVersionPattern_LargeNumbers verifies that large numeric components are accepted.
func TestVersionPattern_LargeNumbers(t *testing.T) {
	if !versionPattern.MatchString("100.200.300") {
		t.Error("expected \"100.200.300\" to match versionPattern, but it did not")
	}
}
