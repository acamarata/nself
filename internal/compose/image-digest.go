// Base image digest pinning for docker-compose generation.
//
// Per S34-T04 (base image pinning): every base image referenced in the
// generated docker-compose.yml can be pinned to a specific sha256 digest, not
// just a tag. Tags are mutable on Docker Hub / registries — a rebuild today
// and a rebuild tomorrow can pull different content under the same tag. Pinning
// to `image:tag@sha256:<digest>` freezes the exact bytes until the user (or
// `nself update images`) refreshes the pins.
//
// This file houses digest-specific helpers; `images.go` retains the default
// image-version map and the ResolveImage entry point which calls into here.
//
// Cross-ref:
//   - S32-T04 (overlap — coordinate image pinning across subsystems)
//   - internal/compose/images.go (entry point: ResolveImage)
//   - cmd/commands/update_images.go (if present — refresh pins)

package compose

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// DigestPrefix is the fixed algorithm prefix for OCI image digests.
const DigestPrefix = "sha256:"

// DigestLength is the hex length of a sha256 digest.
const DigestLength = 64

// IsValidDigest returns true if s is a well-formed sha256 digest (64 hex chars,
// with or without the "sha256:" prefix).
func IsValidDigest(s string) bool {
	s = strings.TrimPrefix(s, DigestPrefix)
	if len(s) != DigestLength {
		return false
	}
	// Must be valid hex.
	if _, err := hex.DecodeString(s); err != nil {
		return false
	}
	return true
}

// NormalizeDigest strips a leading "sha256:" prefix and lower-cases the hex.
// Returns an error if the input is not a valid sha256 digest.
func NormalizeDigest(s string) (string, error) {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, DigestPrefix)
	s = strings.ToLower(s)
	if !IsValidDigest(s) {
		return "", fmt.Errorf("not a valid sha256 digest: %q", s)
	}
	return s, nil
}

// PinImage returns the fully-pinned image reference for the given image:tag
// and digest. If digest is empty the image is returned unchanged (no pin).
//
//	PinImage("postgres:16", "abc...") -> "postgres:16@sha256:abc..."
//	PinImage("postgres:16", "")       -> "postgres:16"
//
// If the image already contains an @sha256: suffix, the existing pin is
// replaced with the new one (trust the caller — they asked to repin).
func PinImage(imageTag, digest string) string {
	if digest == "" {
		return imageTag
	}
	norm, err := NormalizeDigest(digest)
	if err != nil {
		// Caller handed us bad data; return the image unpinned rather than
		// emit an invalid compose line. `nself update images` should surface
		// the error during refresh, not here during render.
		return imageTag
	}
	// Strip any existing digest suffix.
	if at := strings.Index(imageTag, "@"); at != -1 {
		imageTag = imageTag[:at]
	}
	return imageTag + "@" + DigestPrefix + norm
}

// SplitImageRef splits an image reference into its constituent parts:
//
//	SplitImageRef("postgres:16@sha256:abc...")
//	-> name="postgres", tag="16", digest="abc..."
//
//	SplitImageRef("ghcr.io/foo/bar:v1")
//	-> name="ghcr.io/foo/bar", tag="v1", digest=""
//
//	SplitImageRef("alpine")
//	-> name="alpine", tag="latest", digest=""
//
// Registries with port numbers (localhost:5000/foo:v1) are handled by scanning
// right-to-left for the tag separator.
func SplitImageRef(ref string) (name, tag, digest string) {
	// Extract digest first — always uses "@".
	if at := strings.Index(ref, "@"); at != -1 {
		digest = strings.TrimPrefix(ref[at+1:], DigestPrefix)
		ref = ref[:at]
	}

	// Tag is separated by the LAST ":" that is not inside a host:port
	// segment. A port is distinguished by being followed by "/" before any
	// further ":". Scan backwards.
	lastColon := strings.LastIndex(ref, ":")
	if lastColon == -1 {
		return ref, "latest", digest
	}
	// If there's a "/" after this colon, the colon is part of a host:port.
	if strings.Contains(ref[lastColon:], "/") {
		return ref, "latest", digest
	}
	return ref[:lastColon], ref[lastColon+1:], digest
}

// DigestForContent returns the sha256 hex digest for the given bytes. Primarily
// useful for tests and for computing digests of content the CLI itself
// generates (manifests, configs). NOT a replacement for pulling an authoritative
// digest from a registry — that requires talking to the registry API.
func DigestForContent(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
