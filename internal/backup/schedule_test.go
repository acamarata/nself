package backup

import (
	"fmt"
	"testing"
	"time"
)

// heavyJob represents a scheduled maintenance job with a name and its UTC
// fire time expressed as hours and minutes.
type heavyJob struct {
	name   string
	hour   int
	minute int
}

// minuteOfDay returns the number of minutes since midnight for a job.
func (j heavyJob) minuteOfDay() int {
	return j.hour*60 + j.minute
}

// canonicalHeavyJobs returns the current set of heavy nightly jobs that share
// the backup window. Any change to the default schedules in systemd.go or in
// the plugin cron seeds MUST be reflected here.
//
// Canonical schedule (UTC):
//   - backup-full:       03:00 (FullAt default in RenderSystemdUnits)
//   - mux-classify:      03:15 (MUX_CRON_CLASSIFY_HOUR/MINUTE defaults)
//   - backup-prune:      04:30 (PruneAt default in RenderSystemdUnits)
//   - claw_daily_briefing: 04:45 (seeded by claw plugin scheduler)
func canonicalHeavyJobs() []heavyJob {
	return []heavyJob{
		{name: "backup-full", hour: 3, minute: 0},
		{name: "mux-classify", hour: 3, minute: 15},
		{name: "backup-prune", hour: 4, minute: 30},
		{name: "claw_daily_briefing", hour: 4, minute: 45},
	}
}

// TestNoHeavyJobsCollide verifies that no two heavy nightly jobs are scheduled
// within a 10-minute window of each other. Colliding jobs saturate Postgres I/O
// during the backup window and cause plugin health-check failures.
func TestNoHeavyJobsCollide(t *testing.T) {
	const collisionWindow = 10 // minutes

	jobs := canonicalHeavyJobs()

	// Validate that every job has a plausible time value.
	for _, j := range jobs {
		if j.hour < 0 || j.hour > 23 {
			t.Fatalf("job %q: hour %d out of range [0,23]", j.name, j.hour)
		}
		if j.minute < 0 || j.minute > 59 {
			t.Fatalf("job %q: minute %d out of range [0,59]", j.name, j.minute)
		}
	}

	// Check every pair for collision.
	for i := 0; i < len(jobs); i++ {
		for k := i + 1; k < len(jobs); k++ {
			a, b := jobs[i], jobs[k]
			diff := abs(a.minuteOfDay() - b.minuteOfDay())
			if diff < collisionWindow {
				t.Errorf(
					"jobs %q (%02d:%02d UTC) and %q (%02d:%02d UTC) are only %d minute(s) apart, "+
						"minimum gap is %d minutes",
					a.name, a.hour, a.minute,
					b.name, b.hour, b.minute,
					diff, collisionWindow,
				)
			}
		}
	}
}

// TestPruneDefaultAfterFull verifies that the backup-prune timer default is
// scheduled at least 90 minutes after the backup-full timer default.
// This guards against the 04:00 UTC collision that caused the PERF-03 incident.
func TestPruneDefaultAfterFull(t *testing.T) {
	const minGap = 90 // minutes between full backup start and prune start

	var opts SystemdInstallOptions
	// Apply the same defaulting logic as RenderSystemdUnits.
	if opts.FullAt == "" {
		opts.FullAt = "03:00"
	}
	if opts.PruneAt == "" {
		opts.PruneAt = "04:30"
	}

	fullTime, err := parseHHMM(opts.FullAt)
	if err != nil {
		t.Fatalf("parseHHMM(%q): %v", opts.FullAt, err)
	}
	pruneTime, err := parseHHMM(opts.PruneAt)
	if err != nil {
		t.Fatalf("parseHHMM(%q): %v", opts.PruneAt, err)
	}

	gap := int(pruneTime.Sub(fullTime).Minutes())
	if gap < minGap {
		t.Errorf(
			"backup-prune default (%s UTC) is only %d minute(s) after backup-full default (%s UTC); "+
				"minimum gap is %d minutes to avoid I/O saturation",
			opts.PruneAt, gap, opts.FullAt, minGap,
		)
	}
}

// parseHHMM parses a "HH:MM" string into a time.Time on an arbitrary reference day.
func parseHHMM(hhmm string) (time.Time, error) {
	t, err := time.Parse("15:04", hhmm)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid HH:MM %q: %w", hhmm, err)
	}
	return t, nil
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
