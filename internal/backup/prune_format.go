package backup

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// PruneJSONEntry is one entry in the would_delete or kept arrays.
type PruneJSONEntry struct {
	Path    string `json:"path"`
	AgeDays int    `json:"age_days"`
}

// PruneJSONReport is the machine-readable output of `backup prune --format json`.
type PruneJSONReport struct {
	DryRun      bool             `json:"dry_run"`
	KeepDaily   int              `json:"keep_daily"`
	WouldDelete []PruneJSONEntry `json:"would_delete"`
	Kept        []PruneJSONEntry `json:"kept"`
}

// FormatPruneJSON prints a PruneResult as JSON to stdout.
func FormatPruneJSON(result *PruneResult, keepDaily int) error {
	report := PruneJSONReport{
		DryRun:      result.DryRun,
		KeepDaily:   keepDaily,
		WouldDelete: toEntries(result.Pruned),
		Kept:        toEntries(result.Kept),
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal prune report: %w", err)
	}
	fmt.Fprintln(os.Stdout, string(data))
	return nil
}

func toEntries(paths []string) []PruneJSONEntry {
	out := make([]PruneJSONEntry, 0, len(paths))
	now := time.Now()
	for _, p := range paths {
		entry := PruneJSONEntry{Path: p}
		if info, err := os.Stat(p); err == nil {
			entry.AgeDays = int(now.Sub(info.ModTime()).Hours() / 24)
		} else {
			// Fall back: try basename timestamp parse.
			entry.AgeDays = ageFromFilename(filepath.Base(p), now)
		}
		out = append(out, entry)
	}
	return out
}

// ageFromFilename extracts a YYYYMMDD_HHMMSS token and computes age in days.
// Returns -1 if no parseable timestamp is found.
func ageFromFilename(name string, now time.Time) int {
	// Expected token form: _YYYYMMDD_HHMMSS
	for i := 0; i+15 <= len(name); i++ {
		if name[i] != '_' {
			continue
		}
		candidate := name[i+1:]
		if len(candidate) < 15 {
			continue
		}
		if candidate[8] != '_' {
			continue
		}
		t, err := time.Parse("20060102_150405", candidate[:15])
		if err == nil {
			return int(now.Sub(t).Hours() / 24)
		}
	}
	return -1
}
