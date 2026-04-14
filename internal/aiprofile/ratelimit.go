package aiprofile

import (
	"sync"
	"time"
)

// TokenBucket implements a per-key rate limiter capping at 15 req/min (T-05-10).
// Thread-safe. Each key gets its own bucket.
type TokenBucket struct {
	mu      sync.Mutex
	buckets map[int]*bucket
	rate    float64       // tokens per second
	burst   int           // max tokens
	window  time.Duration // refill window
}

type bucket struct {
	tokens    float64
	lastCheck time.Time
	cooldown  time.Time // if non-zero, bucket is rate-limited until this time
}

const (
	defaultRPM   = 15
	defaultBurst = 15
)

// NewTokenBucket creates a rate limiter with 15 RPM per key.
func NewTokenBucket() *TokenBucket {
	return &TokenBucket{
		buckets: make(map[int]*bucket),
		rate:    float64(defaultRPM) / 60.0, // tokens per second
		burst:   defaultBurst,
		window:  time.Minute,
	}
}

// Allow checks whether a request for the given key ID is allowed.
// Returns true if allowed, false if rate-limited.
func (tb *TokenBucket) Allow(keyID int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	b, ok := tb.buckets[keyID]
	if !ok {
		b = &bucket{
			tokens:    float64(tb.burst),
			lastCheck: now,
		}
		tb.buckets[keyID] = b
	}

	// If in cooldown, check if it has expired.
	if !b.cooldown.IsZero() && now.Before(b.cooldown) {
		return false
	}
	b.cooldown = time.Time{}

	// Refill tokens based on elapsed time.
	elapsed := now.Sub(b.lastCheck).Seconds()
	b.tokens += elapsed * tb.rate
	if b.tokens > float64(tb.burst) {
		b.tokens = float64(tb.burst)
	}
	b.lastCheck = now

	if b.tokens < 1 {
		return false
	}

	b.tokens--
	return true
}

// MarkRateLimited marks a key as rate-limited. If retryAfter > 0, uses that
// exact duration. Otherwise applies exponential backoff: 60s, 120s, 240s, up to 1h.
func (tb *TokenBucket) MarkRateLimited(keyID int, retryAfter time.Duration) {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	b, ok := tb.buckets[keyID]
	if !ok {
		b = &bucket{lastCheck: now}
		tb.buckets[keyID] = b
	}

	if retryAfter > 0 {
		b.cooldown = now.Add(retryAfter)
	} else {
		// Exponential backoff: double from last cooldown duration.
		var backoff time.Duration
		if !b.cooldown.IsZero() && b.cooldown.After(now) {
			remaining := b.cooldown.Sub(now)
			backoff = remaining * 2
		} else {
			backoff = 60 * time.Second
		}
		if backoff > time.Hour {
			backoff = time.Hour
		}
		b.cooldown = now.Add(backoff)
	}
	b.tokens = 0
}

// IsAvailable returns true if the key is not currently in cooldown.
func (tb *TokenBucket) IsAvailable(keyID int) bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	b, ok := tb.buckets[keyID]
	if !ok {
		return true
	}
	if !b.cooldown.IsZero() && time.Now().Before(b.cooldown) {
		return false
	}
	return true
}
