// Package deprecation provides the CLI deprecation warning system.
//
// Usage:
//
//	reg, err := deprecation.LoadEmbeddedRegistry()
//	if err != nil {
//	    // warn in debug log, do not crash
//	}
//	if item, ok := reg.Lookup("nself old-cmd"); ok {
//	    reg.Warn(os.Stderr, item)
//	}
package deprecation

import (
	"fmt"
	"io"
	"os"
	"sync"

	"gopkg.in/yaml.v3"
)

// ItemType classifies the deprecated surface.
type ItemType string

const (
	TypeCommand ItemType = "command"
	TypeFlag    ItemType = "flag"
	TypeEnv     ItemType = "env"
)

// Item represents a single deprecated CLI element.
type Item struct {
	Name        string   `yaml:"name"`
	Type        ItemType `yaml:"type"`
	Since       string   `yaml:"since"`
	Replacement string   `yaml:"replacement"`
	DocsURL     string   `yaml:"docs_url"`
	Phase       int      `yaml:"phase"`
}

// registryFile is the YAML envelope.
type registryFile struct {
	Items []Item `yaml:"items"`
}

// Registry holds the loaded deprecation entries.
type Registry struct {
	mu    sync.RWMutex
	index map[string]Item // keyed by Item.Name
}

// LoadRegistry reads the YAML registry file at path and returns a Registry.
// If the file does not exist or fails to parse, it returns an empty
// Registry and a non-nil error (caller should log in debug mode, not crash).
//
// Production code should call LoadEmbeddedRegistry instead: a path-based load
// cannot work for an installed single-file binary. See CLI-R03.
func LoadRegistry(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return &Registry{index: make(map[string]Item)}, fmt.Errorf("deprecation: registry not found at %s: %w", path, err)
	}
	return parseRegistry(data)
}

// parseRegistry builds a Registry from raw YAML bytes. On a parse failure it
// returns an empty (but non-nil, usable) Registry alongside the error.
func parseRegistry(data []byte) (*Registry, error) {
	var rf registryFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return &Registry{index: make(map[string]Item)}, fmt.Errorf("deprecation: failed to parse registry: %w", err)
	}

	reg := &Registry{index: make(map[string]Item, len(rf.Items))}
	for _, item := range rf.Items {
		reg.index[item.Name] = item
	}
	return reg, nil
}

// Len reports how many deprecation entries the registry holds.
func (r *Registry) Len() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.index)
}

// Names returns every registered deprecation name. Order is unspecified.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]string, 0, len(r.index))
	for name := range r.index {
		out = append(out, name)
	}
	return out
}

// Lookup returns the Item for the given name and true if found.
func (r *Registry) Lookup(name string) (Item, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.index[name]
	return item, ok
}

// IsDeprecated returns true if the given name is in the registry.
func (r *Registry) IsDeprecated(name string) bool {
	_, ok := r.Lookup(name)
	return ok
}

// Warn writes a deprecation warning for item to w (should be os.Stderr).
// Format: [DEPRECATED] 'nself <item>' (since v<X.Y.Z>) → use 'nself <replacement>'. Docs: <url>
func (r *Registry) Warn(w io.Writer, item Item) {
	if item.Phase >= 2 {
		_, _ = fmt.Fprintf(w, "[DEPRECATED] '%s' (since %s) → use '%s'. Will be removed in a future release. Docs: %s\n",
			item.Name, item.Since, item.Replacement, item.DocsURL)
	} else {
		_, _ = fmt.Fprintf(w, "[DEPRECATED] '%s' (since %s) → use '%s'. Docs: %s\n",
			item.Name, item.Since, item.Replacement, item.DocsURL)
	}
}

// Warn is a package-level convenience that creates a one-shot warning
// without needing a Registry. Used for inline deprecations.
func Warn(w io.Writer, name, since, replacement, docsURL string) {
	_, _ = fmt.Fprintf(w, "[DEPRECATED] '%s' (since %s) → use '%s'. Docs: %s\n",
		name, since, replacement, docsURL)
}
