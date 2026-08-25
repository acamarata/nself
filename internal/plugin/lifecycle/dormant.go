// Package lifecycle provides the dormant-plugin state machine for the nSelf plugin
// license lifecycle. It handles active → dormant → expired transitions, persistent
// JSON state storage, and the 7-day default grace period.
//
// TOCTOU protection: Save uses an atomic temp-file rename so concurrent CLI
// invocations cannot corrupt the store. The caller must ensure it holds the
// store in memory for only the duration of a single operation — no long-held
// in-memory caches — to avoid read/check/write races across processes.
package lifecycle

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// State represents the current license lifecycle state of a plugin.
type State string

const (
	// StateActive means the plugin license is valid and the plugin runs normally.
	StateActive State = "active"
	// StateDormant means the license expired but the grace period is still active.
	// The plugin continues to function during this window with a daily warning banner.
	StateDormant State = "dormant"
	// StateExpired means the grace period is exhausted. The plugin is disabled and
	// queued for auto-removal on the next `nself build`.
	StateExpired State = "expired"

	// DefaultGracePeriod is the time a plugin stays dormant before auto-removal.
	DefaultGracePeriod = 7 * 24 * time.Hour

	// storeVersion is written to the JSON file so future schema migrations can
	// detect and upgrade old store files.
	storeVersion = 1
)

// Record tracks a single plugin's license lifecycle state.
type Record struct {
	Name          string     `json:"name"`
	State         State      `json:"state"`
	LicenseExpiry time.Time  `json:"license_expiry"`
	DormantSince  *time.Time `json:"dormant_since,omitempty"`
	// GracePeriod is persisted per-record so that global config changes cannot
	// retroactively shorten a grace period already in progress.
	GracePeriod time.Duration `json:"grace_period"`
}

// Store is the persistent state for all plugin lifecycle records.
// It is serialised to ~/.config/nself/plugin-lifecycle.json.
type Store struct {
	Version int                `json:"version"`
	Records map[string]*Record `json:"records"`
}

// storePath returns the canonical path of the persistent store file.
func storePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nself", "plugin-lifecycle.json")
}

// Load reads the persistent store from disk. If the file does not exist a new
// empty store is returned so the caller can populate it without special-casing
// first-run behaviour.
func Load() (*Store, error) {
	path := storePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Store{Version: storeVersion, Records: make(map[string]*Record)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lifecycle: read store: %w", err)
	}
	var s Store
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("lifecycle: parse store: %w", err)
	}
	if s.Records == nil {
		s.Records = make(map[string]*Record)
	}
	return &s, nil
}

// Save persists the store to disk using an atomic temp-file rename.
// The rename is the OS-level TOCTOU guard: either the full new content lands or
// the old file is untouched — there is no window where the file is partially
// written.
func (s *Store) Save() error {
	path := storePath()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("lifecycle: create store dir: %w", err)
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return fmt.Errorf("lifecycle: marshal store: %w", err)
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return fmt.Errorf("lifecycle: write temp store: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("lifecycle: atomic rename: %w", err)
	}
	return nil
}

// CheckExpiry transitions plugins from active → dormant → expired based on now.
// It returns two slices:
//   - dormant: plugins newly transitioned to StateDormant this call.
//   - autoRemove: plugins transitioned to StateExpired (auto-removal candidates).
//
// The caller must call Save after CheckExpiry to persist any transitions.
// Call MarkRenewal before CheckExpiry for any plugin whose license was renewed
// this session so it cannot immediately cycle back to expired.
func (s *Store) CheckExpiry(now time.Time) (dormant, autoRemove []string) {
	for name, rec := range s.Records {
		switch rec.State {
		case StateActive:
			if now.After(rec.LicenseExpiry) {
				t := now
				rec.State = StateDormant
				rec.DormantSince = &t
				if rec.GracePeriod == 0 {
					rec.GracePeriod = DefaultGracePeriod
				}
				dormant = append(dormant, name)
			}
		case StateDormant:
			if rec.DormantSince != nil && now.After(rec.DormantSince.Add(rec.GracePeriod)) {
				rec.State = StateExpired
				autoRemove = append(autoRemove, name)
			}
		}
	}
	return
}

// DormantBanner returns the user-facing warning text for a dormant plugin,
// showing the number of days remaining in the grace period.
// Returns an empty string if rec is nil or has no DormantSince timestamp.
func DormantBanner(rec *Record, now time.Time) string {
	if rec == nil || rec.DormantSince == nil {
		return ""
	}
	deadline := rec.DormantSince.Add(rec.GracePeriod)
	remaining := deadline.Sub(now)
	days := int(remaining.Hours() / 24)
	if days < 0 {
		days = 0
	}
	return fmt.Sprintf(
		"Plugin %q is DORMANT — license expired. %d day(s) until auto-removal. Renew at nself.org",
		rec.Name, days,
	)
}

// MarkRenewal transitions a plugin back to StateActive with a new expiry.
// If no record exists for name it is created. DormantSince is cleared so the
// grace period counter resets on the next CheckExpiry call.
func (s *Store) MarkRenewal(name string, newExpiry time.Time) {
	rec, ok := s.Records[name]
	if !ok {
		rec = &Record{Name: name, GracePeriod: DefaultGracePeriod}
		s.Records[name] = rec
	}
	rec.State = StateActive
	rec.LicenseExpiry = newExpiry
	rec.DormantSince = nil
}
