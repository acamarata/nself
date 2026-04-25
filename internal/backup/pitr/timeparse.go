package pitr

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ParseTargetTime parses a target time string in one of two forms:
//
//   - RFC3339 absolute:   "2026-04-22T15:30:00Z"
//   - Relative duration:  "30 minutes ago", "2 hours ago", "1 day ago"
//
// Returns the resolved UTC time or an error.
func ParseTargetTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("target time is required")
	}

	// Try RFC3339 first.
	if t, err := time.Parse(time.RFC3339, s); err == nil {
		return t.UTC(), nil
	}

	// Try relative form: "<N> <unit> ago"
	lower := strings.ToLower(s)
	if strings.HasSuffix(lower, " ago") {
		body := strings.TrimSuffix(lower, " ago")
		body = strings.TrimSpace(body)
		parts := strings.SplitN(body, " ", 2)
		if len(parts) == 2 {
			n, err := strconv.Atoi(parts[0])
			if err != nil {
				return time.Time{}, fmt.Errorf("invalid number in relative time %q: %w", s, err)
			}
			unit := strings.TrimSuffix(parts[1], "s") // normalise plural
			var d time.Duration
			switch unit {
			case "second":
				d = time.Duration(n) * time.Second
			case "minute":
				d = time.Duration(n) * time.Minute
			case "hour":
				d = time.Duration(n) * time.Hour
			case "day":
				d = time.Duration(n) * 24 * time.Hour
			case "week":
				d = time.Duration(n) * 7 * 24 * time.Hour
			default:
				return time.Time{}, fmt.Errorf("unrecognised time unit %q in %q", unit, s)
			}
			return time.Now().UTC().Add(-d), nil
		}
	}

	return time.Time{}, fmt.Errorf("cannot parse target time %q: use RFC3339 (e.g. 2026-04-22T15:30:00Z) or relative (e.g. \"30 minutes ago\")", s)
}
