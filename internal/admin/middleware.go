package admin

import (
	"crypto/sha256"
	"fmt"
	"math/rand"
)

// ShouldAudit determines whether a request should be logged based on
// method. All writes are logged. Reads are sampled at the given rate
// (0.0-1.0, default 0.01 = 1%).
func ShouldAudit(method string, readSampleRate float64) bool {
	switch method {
	case "GET", "HEAD", "OPTIONS":
		return rand.Float64() < readSampleRate //nolint:gosec // sampling, not security
	default:
		// POST, PUT, PATCH, DELETE — always audit
		return true
	}
}

// HashBody returns a hex-encoded SHA-256 hash of the request body.
// Returns empty string for nil or empty bodies.
func HashBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	h := sha256.Sum256(body)
	return fmt.Sprintf("%x", h)
}

// DefaultReadSampleRate is 1 % for read-only requests.
const DefaultReadSampleRate = 0.01
