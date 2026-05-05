// Package flags — flag lifecycle helpers (sunset and removal date enforcement).
package flags

import "time"

// LifecycleWarning describes a sunset/removal enforcement event for a flag.
type LifecycleWarning struct {
	FlagKey string
	Level   string // "warn" or "error"
	Message string
}

// CheckFlagLifecycle inspects each flag's SunsetAt and RemovalDate fields
// against the supplied "now" timestamp and returns warnings for flags that
// are past sunset (level=warn) or past removal (level=error).
//
// A flag with no SunsetAt/RemovalDate produces no warning. A flag with
// SunsetAt in the future produces no warning.
func CheckFlagLifecycle(flags []Flag, now time.Time) []LifecycleWarning {
	var out []LifecycleWarning
	for _, f := range flags {
		if f.RemovalDate != nil && !f.RemovalDate.After(now) {
			out = append(out, LifecycleWarning{
				FlagKey: f.Key,
				Level:   "error",
				Message: "flag is past removal_date — should be removed from code",
			})
			continue
		}
		if f.SunsetAt != nil && !f.SunsetAt.After(now) {
			out = append(out, LifecycleWarning{
				FlagKey: f.Key,
				Level:   "warn",
				Message: "flag is past sunset_at — schedule for removal",
			})
		}
	}
	return out
}

// ApplyLifecycleDisable mutates the flags slice in place: any flag whose
// RemovalDate has passed is force-disabled (Enabled=false). Returns the
// number of flags that were disabled.
func ApplyLifecycleDisable(flags []Flag, now time.Time) int {
	n := 0
	for i := range flags {
		f := &flags[i]
		if f.RemovalDate != nil && !f.RemovalDate.After(now) && f.Enabled {
			f.Enabled = false
			n++
		}
	}
	return n
}
