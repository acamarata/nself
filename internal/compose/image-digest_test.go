package compose

import "testing"

func TestIsValidDigest(t *testing.T) {
	valid := []string{
		"abc1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd",
		"sha256:abc1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd",
	}
	for _, d := range valid {
		if !IsValidDigest(d) {
			t.Errorf("IsValidDigest(%q) = false, want true", d)
		}
	}
	invalid := []string{
		"",
		"abc",
		"sha256:",
		"sha256:abc",
		"abc1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcz", // z is not hex
	}
	for _, d := range invalid {
		if IsValidDigest(d) {
			t.Errorf("IsValidDigest(%q) = true, want false", d)
		}
	}
}

func TestNormalizeDigest(t *testing.T) {
	hex := "ABC1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCD"
	want := "abc1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd"

	// Without prefix, upper-case.
	got, err := NormalizeDigest(hex)
	if err != nil {
		t.Fatalf("NormalizeDigest err: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// With prefix, whitespace.
	got, err = NormalizeDigest("  sha256:" + hex + "  ")
	if err != nil {
		t.Fatalf("NormalizeDigest err: %v", err)
	}
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	// Invalid.
	if _, err := NormalizeDigest("bogus"); err == nil {
		t.Errorf("NormalizeDigest(bogus) expected error")
	}
}

func TestPinImage(t *testing.T) {
	hex := "abc1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd"

	cases := []struct {
		image, digest, want string
	}{
		{"postgres:16", "", "postgres:16"},
		{"postgres:16", hex, "postgres:16@sha256:" + hex},
		{"postgres:16", "sha256:" + hex, "postgres:16@sha256:" + hex},
		// Already-pinned image should have its pin REPLACED.
		{"postgres:16@sha256:0000000000000000000000000000000000000000000000000000000000000000", hex, "postgres:16@sha256:" + hex},
		// Bad digest falls back to unpinned (safer than emitting junk).
		{"postgres:16", "not-a-digest", "postgres:16"},
	}
	for _, tc := range cases {
		if got := PinImage(tc.image, tc.digest); got != tc.want {
			t.Errorf("PinImage(%q, %q) = %q, want %q", tc.image, tc.digest, got, tc.want)
		}
	}
}

func TestSplitImageRef(t *testing.T) {
	hex := "abc1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcd"

	cases := []struct {
		ref, name, tag, digest string
	}{
		{"alpine", "alpine", "latest", ""},
		{"postgres:16", "postgres", "16", ""},
		{"postgres:16@sha256:" + hex, "postgres", "16", hex},
		{"ghcr.io/foo/bar:v1", "ghcr.io/foo/bar", "v1", ""},
		{"localhost:5000/foo", "localhost:5000/foo", "latest", ""},
		{"localhost:5000/foo:v1", "localhost:5000/foo", "v1", ""},
	}
	for _, tc := range cases {
		name, tag, digest := SplitImageRef(tc.ref)
		if name != tc.name || tag != tc.tag || digest != tc.digest {
			t.Errorf("SplitImageRef(%q) = (%q, %q, %q), want (%q, %q, %q)",
				tc.ref, name, tag, digest, tc.name, tc.tag, tc.digest)
		}
	}
}

func TestDigestForContent(t *testing.T) {
	// Known sha256("hello") = 2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824
	got := DigestForContent([]byte("hello"))
	want := "2cf24dba5fb0a30e26e83b2ac5b9e29e1b161e5c1fa7425e73043362938b9824"
	if got != want {
		t.Errorf("DigestForContent(hello) = %q, want %q", got, want)
	}
	if !IsValidDigest(got) {
		t.Errorf("DigestForContent output is not a valid digest")
	}
}
