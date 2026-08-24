package featureflags

// Purpose: project-local state persistence (.nself/features.json) and per-flag resolution against a Registry, split out of registry.go's catalog loading.
// Inputs: a project directory, a *Registry loaded by Load, and a *State loaded/saved via StatePath.
// Outputs: resolved Status values reflecting each flag's default and any override, and persisted state files.
// Constraints: split out of registry.go as a pure move (CLI-R12); no behavior change.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// --- State (project-local .nself/features.json) -------------------------

// StatePath returns the absolute path the state file would occupy for the
// given project directory.
func StatePath(projectDir string) string {
	return filepath.Join(projectDir, stateFileRel)
}

// LoadState reads the state file at <projectDir>/.nself/features.json.
//
// A missing file is NOT an error: the returned State has an empty overrides
// map. Use it as the working baseline.
func LoadState(projectDir string) (*State, error) {
	p := StatePath(projectDir)
	b, err := os.ReadFile(p) // #nosec G304 — path derived from caller-supplied projectDir; reading our own state file is intended.
	if err != nil {
		if os.IsNotExist(err) {
			return &State{Overrides: map[string]bool{}}, nil
		}
		return nil, fmt.Errorf("featureflags: read state %s: %w", p, err)
	}
	var s State
	if err := json.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("featureflags: parse state %s: %w", p, err)
	}
	if s.Overrides == nil {
		s.Overrides = map[string]bool{}
	}
	return &s, nil
}

// SaveState writes the state file atomically (write+rename) with 0600 perms.
//
// 0600 follows the project-wide rule for .env-equivalent state files: state
// includes which features the operator has enabled, which is sensitive enough
// to keep readable only by the owning user.
func SaveState(projectDir string, s *State) error {
	if s == nil {
		return fmt.Errorf("featureflags: nil state")
	}
	if s.Overrides == nil {
		s.Overrides = map[string]bool{}
	}
	s.UpdatedAt = time.Now().UTC()

	dir := filepath.Join(projectDir, ".nself")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("featureflags: mkdir %s: %w", dir, err)
	}
	final := StatePath(projectDir)
	tmp, err := os.CreateTemp(dir, "features.json.tmp.*")
	if err != nil {
		return fmt.Errorf("featureflags: create tmp: %w", err)
	}
	tmpName := tmp.Name()
	enc := json.NewEncoder(tmp)
	enc.SetIndent("", "  ")
	if err := enc.Encode(s); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpName)
		return fmt.Errorf("featureflags: encode state: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("featureflags: close tmp: %w", err)
	}
	if err := os.Chmod(tmpName, 0o600); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("featureflags: chmod tmp: %w", err)
	}
	if err := os.Rename(tmpName, final); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("featureflags: rename %s: %w", final, err)
	}
	return nil
}

// --- Combined resolution ------------------------------------------------

// Resolve returns the effective Status for every registry flag, given the
// supplied state.
func (r *Registry) Resolve(s *State) []Status {
	if s == nil {
		s = &State{Overrides: map[string]bool{}}
	}
	all := r.All()
	out := make([]Status, 0, len(all))
	for _, f := range all {
		st := Status{Flag: f, Enabled: f.Default, Source: "default", UpdatedAt: s.UpdatedAt}
		if v, ok := s.Overrides[f.Key]; ok {
			st.Enabled = v
			st.Overridden = true
			st.Source = "override"
		}
		out = append(out, st)
	}
	return out
}

// ResolveOne returns the Status for a single flag.
//
// Returns ok=false when the key is not in the registry. An override for a
// key that doesn't exist is ignored — registry membership is authoritative.
func (r *Registry) ResolveOne(s *State, key string) (Status, bool) {
	f, ok := r.Get(key)
	if !ok {
		return Status{}, false
	}
	if s == nil {
		s = &State{Overrides: map[string]bool{}}
	}
	st := Status{Flag: f, Enabled: f.Default, Source: "default", UpdatedAt: s.UpdatedAt}
	if v, present := s.Overrides[key]; present {
		st.Enabled = v
		st.Overridden = true
		st.Source = "override"
	}
	return st, true
}

// SetOverride mutates s by recording an override for key. The key must
// exist in the registry; otherwise an error is returned.
func (r *Registry) SetOverride(s *State, key string, enabled bool) error {
	if s == nil {
		return fmt.Errorf("featureflags: nil state")
	}
	if _, ok := r.Get(key); !ok {
		return fmt.Errorf("featureflags: unknown flag %q (run `nself feature list` to see registered flags)", key)
	}
	if s.Overrides == nil {
		s.Overrides = map[string]bool{}
	}
	s.Overrides[key] = enabled
	return nil
}

// ClearOverride removes an override for key. No-op if no override exists.
// Returns an error if the key is not in the registry.
func (r *Registry) ClearOverride(s *State, key string) error {
	if s == nil {
		return fmt.Errorf("featureflags: nil state")
	}
	if _, ok := r.Get(key); !ok {
		return fmt.Errorf("featureflags: unknown flag %q", key)
	}
	delete(s.Overrides, key)
	return nil
}
