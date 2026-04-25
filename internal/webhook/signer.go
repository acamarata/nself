// Package webhook provides HMAC-SHA256 signing and dispatch for nSelf webhook deliveries.
package webhook

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// GenerateSecret returns a 32-byte hex-encoded HMAC signing secret.
func GenerateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("webhook: generating secret: %w", err)
	}
	return hex.EncodeToString(b), nil
}

// Sign computes the HMAC-SHA256 signature for an outbound webhook payload.
//
// Signature input: "<unix_seconds>.<body_bytes>"
// Header: X-Nself-Signature-256: sha256=<hex_hmac>
func Sign(secret string, unixSeconds int64, body []byte) string {
	msg := []byte(strconv.FormatInt(unixSeconds, 10) + ".")
	msg = append(msg, body...)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(msg)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

// Verify checks whether the X-Nself-Signature-256 header value matches the payload.
// It uses a constant-time comparison to prevent timing attacks.
func Verify(secret string, unixSeconds int64, body []byte, headerValue string) bool {
	expected := Sign(secret, unixSeconds, body)
	return hmac.Equal([]byte(expected), []byte(headerValue))
}

// InjectHeaders adds the nSelf webhook authentication headers to r.
//
//   - X-Nself-Signature-256: sha256=<hmac>
//   - X-Nself-Delivery:      <delivery_uuid>
//   - X-Nself-Event:         <event_name>
//   - X-Nself-Timestamp:     <unix_seconds>
//
// If secret is empty, the signature header is omitted (unsigned delivery).
func InjectHeaders(r *http.Request, secret, deliveryID, eventName string, ts time.Time, body []byte) {
	unix := ts.Unix()
	r.Header.Set("X-Nself-Delivery", deliveryID)
	r.Header.Set("X-Nself-Event", eventName)
	r.Header.Set("X-Nself-Timestamp", strconv.FormatInt(unix, 10))
	if secret != "" {
		r.Header.Set("X-Nself-Signature-256", Sign(secret, unix, body))
	}
}
