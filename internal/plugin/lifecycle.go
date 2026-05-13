package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// LifecycleState represents the current state of a plugin's license lifecycle.
type LifecycleState string

const (
	// StateActive means the plugin license is valid.
	StateActive LifecycleState = "active"
	// StateDormant means the license expired but the grace period is still active.
	StateDormant LifecycleState = "dormant"
	// StateExpired means the grace period exhausted — plugin is an auto-remove candidate.
	StateExpired LifecycleState = "expired"

	// DefaultGracePeriod is the time a plugin stays dormant before auto-removal.
	DefaultGracePeriod = 7 * 24 * time.Hour
)

// PluginLifecycleRecord tracks a single plugin's license lifecycle state.
type PluginLifecycleRecord struct {
	Name          string         `json:"name"`
	State         LifecycleState `json:"state"`
	LicenseExpiry time.Time      `json:"license_expiry"`
	DormantSince  *time.Time     `json:"dormant_since,omitempty"`
	// GracePeriod is stored per-record so that future config changes cannot
	// retroactively shorten grace periods that are already in progress.
	GracePeriod time.Duration `json:"grace_period"`
}

// LifecycleStore is the persistent state file for plugin lifecycle tracking.
type LifecycleStore struct {
	Version int                               `json:"version"`
	Records map[string]*PluginLifecycleRecord `json:"records"`
}

// lifecycleStorePath returns ~/.config/nself/plugin-lifecycle.json.
func lifecycleStorePath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "nself", "plugin-lifecycle.json")
}

// LoadLifecycleStore reads the persistent store, creating an empty one if missing.
func LoadLifecycleStore() (*LifecycleStore, error) {
	path := lifecycleStorePath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &LifecycleStore{Version: 1, Records: make(map[string]*PluginLifecycleRecord)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("lifecycle: read store: %w", err)
	}
	var store LifecycleStore
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, fmt.Errorf("lifecycle: parse store: %w", err)
	}
	if store.Records == nil {
		store.Records = make(map[string]*PluginLifecycleRecord)
	}
	return &store, nil
}

// Save writes the store atomically via a temp file + rename to prevent corruption
// on concurrent CLI invocations (TOCTOU protection).
func (s *LifecycleStore) Save() error {
	path := lifecycleStorePath()
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

// CheckExpiry transitions plugins to DORMANT or EXPIRED based on the given time.
// It returns two slices: plugins newly transitioned to DORMANT, and plugins now
// EXPIRED (auto-remove candidates). The caller must call Save() to persist transitions.
//
// Renewal check: MarkRenewal must be called before CheckExpiry to clear dormant
// state for any plugin that was renewed; this prevents a renewed plugin from
// immediately cycling back to EXPIRED.
func (s *LifecycleStore) CheckExpiry(now time.Time) (dormant, autoRemove []string) {
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

// DormantBanner returns the warning text for a dormant plugin showing days remaining.
func DormantBanner(rec *PluginLifecycleRecord, now time.Time) string {
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

// MarkRenewal transitions a plugin back to active state with a new expiry.
// If the record does not exist yet, it is created.
func (s *LifecycleStore) MarkRenewal(name string, newExpiry time.Time) {
	rec, ok := s.Records[name]
	if !ok {
		rec = &PluginLifecycleRecord{Name: name, GracePeriod: DefaultGracePeriod}
		s.Records[name] = rec
	}
	rec.State = StateActive
	rec.LicenseExpiry = newExpiry
	rec.DormantSince = nil
}
