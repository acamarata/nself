// Package featureflags provides a built-in, file-backed feature flag registry
// for the nSelf CLI binary.
//
// This package is distinct from internal/flags/, which is a thin HTTP client
// over the runtime feature-flags plugin (port 3305). This package serves
// build-time / install-time capability gates that ship inside the CLI binary
// itself — no plugin required, no network calls.
//
// Layout:
//   - registry.yaml is the canonical catalog (embedded via go:embed).
//   - State (per-flag enabled/disabled overrides) lives in `.nself/features.json`
//     inside the project working directory.
//
// Spec source: .claude/docs/ARCHITECTURE.md § Feature Flags.
package featureflags

import (
	_ "embed"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// stateFileRel is the project-relative location of the persisted state file.
// The .nself/ directory is the conventional project-local cache/state dir
// already created by `nself init` / `nself build`.
const stateFileRel = ".nself/features.json"

//go:embed registry.yaml
var registryYAML []byte

// FlagType narrows the allowed flag categories.
type FlagType string

const (
	// FlagTypeRelease toggles new behavior that will eventually be permanent.
	FlagTypeRelease FlagType = "release"
	// FlagTypeOps toggles operational behavior (rate limits, strict modes).
	FlagTypeOps FlagType = "ops"
	// FlagTypeKillSwitch is an emergency disable path; never auto-enables.
	FlagTypeKillSwitch FlagType = "kill_switch"
)

// Flag is a single registry entry as declared in registry.yaml.
//
// Fields use struct tags for both yaml (registry source) and json (state file
// + machine output).
type Flag struct {
	Key         string   `yaml:"key" json:"key"`
	Type        FlagType `yaml:"type" json:"type"`
	Surface     string   `yaml:"surface" json:"surface"`
	Description string   `yaml:"description" json:"description"`
	Default     bool     `yaml:"default" json:"default"`
	Introduced  string   `yaml:"introduced" json:"introduced"`
	AutoKill    string   `yaml:"auto_kill,omitempty" json:"auto_kill,omitempty"`
	Owner       string   `yaml:"owner,omitempty" json:"owner,omitempty"`
	DocsURL     string   `yaml:"docs_url,omitempty" json:"docs_url,omitempty"`
}

// fileSchema is the on-disk shape of registry.yaml.
type fileSchema struct {
	Version int    `yaml:"version"`
	Flags   []Flag `yaml:"flags"`
}

// State is the persisted per-flag override map.
//
// Keys are flag keys; values are the user's requested enabled state.
// The absence of a key means "use registry default".
type State struct {
	Overrides map[string]bool `json:"overrides"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// Status is the resolved view of a flag: its registry entry plus the
// effective enabled state (override or default) and a flag indicating
// whether an override is in play.
type Status struct {
	Flag       Flag      `json:"flag"`
	Enabled    bool      `json:"enabled"`
	Overridden bool      `json:"overridden"`
	Source     string    `json:"source"` // "default" | "override"
	UpdatedAt  time.Time `json:"updated_at,omitempty"`
}

// Registry is the loaded set of flags from registry.yaml.
//
// The zero value is not usable; obtain one via Load.
type Registry struct {
	mu      sync.RWMutex
	version int
	byKey   map[string]Flag
	order   []string // preserve registry.yaml order for stable List output
}

// Load parses the embedded registry.yaml.
//
// Returns an error if the YAML is malformed or any entry fails validation.
// Validation rejects: empty key, unknown type, empty surface, empty description,
// duplicate key, and malformed auto_kill date (when present).
func Load() (*Registry, error) {
	return loadFromBytes(registryYAML)
}

// loadFromBytes is exported only via Load; split out so tests can feed
// alternate registry contents without juggling embed.FS in tests.
func loadFromBytes(b []byte) (*Registry, error) {
	if len(b) == 0 {
		return nil, fmt.Errorf("featureflags: empty registry data")
	}
	var fs fileSchema
	if err := yaml.Unmarshal(b, &fs); err != nil {
		return nil, fmt.Errorf("featureflags: parse registry: %w", err)
	}
	if fs.Version != 1 {
		return nil, fmt.Errorf("featureflags: unsupported registry version %d (want 1)", fs.Version)
	}
	r := &Registry{
		version: fs.Version,
		byKey:   make(map[string]Flag, len(fs.Flags)),
		order:   make([]string, 0, len(fs.Flags)),
	}
	for i, f := range fs.Flags {
		if err := validateFlag(f); err != nil {
			return nil, fmt.Errorf("featureflags: registry entry %d (%q): %w", i, f.Key, err)
		}
		if _, dup := r.byKey[f.Key]; dup {
			return nil, fmt.Errorf("featureflags: duplicate registry key %q", f.Key)
		}
		r.byKey[f.Key] = f
		r.order = append(r.order, f.Key)
	}
	return r, nil
}

func validateFlag(f Flag) error {
	if strings.TrimSpace(f.Key) == "" {
		return fmt.Errorf("key is required")
	}
	switch f.Type {
	case FlagTypeRelease, FlagTypeOps, FlagTypeKillSwitch:
	default:
		return fmt.Errorf("invalid type %q (want release|ops|kill_switch)", f.Type)
	}
	if strings.TrimSpace(f.Surface) == "" {
		return fmt.Errorf("surface is required")
	}
	if strings.TrimSpace(f.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if strings.TrimSpace(f.Introduced) == "" {
		return fmt.Errorf("introduced is required")
	}
	if f.AutoKill != "" {
		if _, err := time.Parse("2006-01-02", f.AutoKill); err != nil {
			return fmt.Errorf("auto_kill must be YYYY-MM-DD: %w", err)
		}
	}
	return nil
}

// All returns the registry flags in declaration order.
func (r *Registry) All() []Flag {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Flag, 0, len(r.order))
	for _, k := range r.order {
		out = append(out, r.byKey[k])
	}
	return out
}

// Get returns the flag for key, or ok=false if not in the registry.
func (r *Registry) Get(key string) (Flag, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	f, ok := r.byKey[key]
	return f, ok
}

// Keys returns the sorted set of registry keys.
func (r *Registry) Keys() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.byKey))
	for k := range r.byKey {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
