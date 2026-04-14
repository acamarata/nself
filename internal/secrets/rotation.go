// Package secrets — rotation scheduling, dual-key windows, and expiry alerts.
package secrets

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// RotationSchedule defines when and how a secret should be rotated.
type RotationSchedule struct {
	SecretName  string `json:"secret_name"`
	CadenceDays int    `json:"cadence_days"`
	WindowDays  int    `json:"window_days"` // dual-key overlap window
	LastRotated string `json:"last_rotated,omitempty"`
	NextDue     string `json:"next_due,omitempty"`
}

// RotationState is the persisted state for all tracked secret schedules.
type RotationState struct {
	Schedules []RotationSchedule `json:"schedules"`
	UpdatedAt string             `json:"updated_at"`
}

// DefaultSchedules returns the minimum set of tracked secrets per the spec.
func DefaultSchedules() []RotationSchedule {
	return []RotationSchedule{
		{SecretName: "PLUGIN_INTERNAL_SECRET", CadenceDays: 90, WindowDays: 7},
		{SecretName: "HASURA_ADMIN_SECRET", CadenceDays: 180, WindowDays: 3},
		{SecretName: "JWT_SIGNING_KEY", CadenceDays: 180, WindowDays: 1},
		{SecretName: "STRIPE_WEBHOOK_SECRET", CadenceDays: 0, WindowDays: 0}, // on-demand
		{SecretName: "B2_APPLICATION_KEY", CadenceDays: 365, WindowDays: 1},
		{SecretName: "AGE_BACKUP_KEY", CadenceDays: 365, WindowDays: 90},
	}
}

// rotationStatePath returns the path to the rotation state file.
func rotationStatePath(projectRoot string) string {
	return filepath.Join(projectRoot, SecretsDir, "rotation-state.json")
}

// LoadRotationState reads the rotation schedule state from disk.
func LoadRotationState(projectRoot string) (*RotationState, error) {
	path := rotationStatePath(projectRoot)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &RotationState{Schedules: DefaultSchedules()}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading rotation state: %w", err)
	}
	var state RotationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parsing rotation state: %w", err)
	}
	return &state, nil
}

// SaveRotationState persists the rotation schedule state to disk.
func SaveRotationState(projectRoot string, state *RotationState) error {
	state.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(rotationStatePath(projectRoot))
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(rotationStatePath(projectRoot), data, 0600)
}

// ScheduleCheck represents the result of checking one secret's rotation schedule.
type ScheduleCheck struct {
	SecretName  string        `json:"secret_name"`
	CadenceDays int           `json:"cadence_days"`
	WindowDays  int           `json:"window_days"`
	LastRotated string        `json:"last_rotated"`
	NextDue     string        `json:"next_due"`
	DueIn       time.Duration `json:"-"`
	DueInDays   int           `json:"due_in_days"`
	Status      string        `json:"status"` // ok, warning, overdue, missing
}

// CheckSchedule validates all rotation schedules and returns findings.
func CheckSchedule(projectRoot string) ([]ScheduleCheck, error) {
	state, err := LoadRotationState(projectRoot)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	var results []ScheduleCheck

	for _, s := range state.Schedules {
		check := ScheduleCheck{
			SecretName:  s.SecretName,
			CadenceDays: s.CadenceDays,
			WindowDays:  s.WindowDays,
			LastRotated: s.LastRotated,
			NextDue:     s.NextDue,
		}

		// On-demand secrets (cadence 0) are always ok unless explicitly overdue.
		if s.CadenceDays == 0 {
			check.Status = "ok"
			check.DueInDays = -1
			results = append(results, check)
			continue
		}

		if s.NextDue == "" {
			check.Status = "missing"
			results = append(results, check)
			continue
		}

		nextDue, err := time.Parse(time.RFC3339, s.NextDue)
		if err != nil {
			check.Status = "missing"
			results = append(results, check)
			continue
		}

		check.DueIn = nextDue.Sub(now)
		check.DueInDays = int(check.DueIn.Hours() / 24)

		switch {
		case check.DueIn < 0:
			check.Status = "overdue"
		case check.DueIn < 7*24*time.Hour:
			check.Status = "warning"
		default:
			check.Status = "ok"
		}

		results = append(results, check)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].NextDue < results[j].NextDue
	})
	return results, nil
}

// RotateDualWindow generates a new key while keeping the old one as _PREVIOUS.
// The current value moves to KEY_PREVIOUS, and a new value is set as KEY_CURRENT.
func RotateDualWindow(projectRoot, env, key string) error {
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return err
	}

	entry, ok := store.Secrets[key]
	if !ok {
		return fmt.Errorf("secret %q not found in %s environment", key, env)
	}

	oldValue := entry.Value
	newValue, _ := generateRotationValue(key)
	if newValue == "" {
		return fmt.Errorf("secret %q requires manual rotation through provider", key)
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Set _PREVIOUS to old value
	store.Secrets[key+"_PREVIOUS"] = SecretEntry{
		Value:     oldValue,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Set _CURRENT to new value
	store.Secrets[key+"_CURRENT"] = SecretEntry{
		Value:     newValue,
		CreatedAt: now,
		UpdatedAt: now,
		RotatedAt: now,
	}

	// Update the base key to point to new value
	entry.Value = newValue
	entry.UpdatedAt = now
	entry.RotatedAt = now
	store.Secrets[key] = entry

	if err := saveStore(projectRoot, env, store); err != nil {
		return err
	}

	// Update rotation state
	state, err := LoadRotationState(projectRoot)
	if err != nil {
		return err
	}
	for i, s := range state.Schedules {
		if s.SecretName == key {
			state.Schedules[i].LastRotated = now
			nextDue := time.Now().UTC().AddDate(0, 0, s.CadenceDays)
			state.Schedules[i].NextDue = nextDue.Format(time.RFC3339)
			break
		}
	}
	return SaveRotationState(projectRoot, state)
}

// RetireOldKey removes the _PREVIOUS variant of a secret after the dual-key window.
func RetireOldKey(projectRoot, env, key string) error {
	store, err := loadStore(projectRoot, env)
	if err != nil {
		return err
	}

	prevKey := key + "_PREVIOUS"
	if _, ok := store.Secrets[prevKey]; !ok {
		return fmt.Errorf("no previous key %q found — nothing to retire", prevKey)
	}

	delete(store.Secrets, prevKey)

	// Also clean up _CURRENT if present (collapse back to base key)
	if current, ok := store.Secrets[key+"_CURRENT"]; ok {
		entry := store.Secrets[key]
		entry.Value = current.Value
		store.Secrets[key] = entry
		delete(store.Secrets, key+"_CURRENT")
	}

	return saveStore(projectRoot, env, store)
}

// InitSchedules ensures all default schedules are present in the rotation state,
// computing NextDue from LastRotated or setting to now if never rotated.
func InitSchedules(projectRoot string) error {
	state, err := LoadRotationState(projectRoot)
	if err != nil {
		return err
	}

	existing := make(map[string]bool)
	for _, s := range state.Schedules {
		existing[s.SecretName] = true
	}

	now := time.Now().UTC()
	for _, def := range DefaultSchedules() {
		if existing[def.SecretName] {
			continue
		}
		def.LastRotated = now.Format(time.RFC3339)
		if def.CadenceDays > 0 {
			def.NextDue = now.AddDate(0, 0, def.CadenceDays).Format(time.RFC3339)
		}
		state.Schedules = append(state.Schedules, def)
	}

	return SaveRotationState(projectRoot, state)
}
