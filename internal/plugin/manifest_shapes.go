package plugin

import (
	"encoding/json"
	"fmt"
)

// Reading the plugin.json shapes that are actually published.
//
// Purpose: 181 of the 184 manifests shipped in the plugins and plugins-pro
// repos could not be parsed by this package. A manifest that fails to parse is
// discarded — parseManifest returns an error, ListInstalled skips the plugin —
// so almost the entire catalogue was invisible to `nself plugin list`,
// `nself plugin info`, and everything else that enumerates what is installed.
// Nothing was ever printed about it.
//
// Four fields were involved, each typed for one of the shapes in use rather
// than all of them: `webhooks`, `envVars`, `dependencies`, `permissions`, plus
// `systemDependencies` when written as an empty array.
//
// Inputs: a plugin.json document.
//
// Outputs: a PluginManifest, with the grouped forms normalised.
//
// Constraints: parsing must never fail on a shape a real manifest uses. Where
// a shape carries a DIFFERENT VOCABULARY rather than a different layout — which
// is the case for permissions — it is recorded as-is and kept out of the
// canonical validated set. Inventing a mapping between two permission
// vocabularies would be writing security policy by guesswork.

// manifestAlias exists solely to give PluginManifest.UnmarshalJSON a type to
// delegate to without recursing.
type manifestAlias PluginManifest

// groupedDeps is the {"required": [...], "optional": [...]} form that 169 of
// the shipped manifests use for `dependencies`.
type groupedDeps struct {
	Required []string `json:"required"`
	Optional []string `json:"optional"`
}

// UnmarshalJSON reads a manifest, normalising the field shapes that vary
// between published plugins.
//
// Dependencies needs handling here rather than on the field, because the
// grouped form carries optional entries that belong in a DIFFERENT field
// (OptionalDependencies), and a field-level unmarshaller cannot write to a
// sibling. 31 shipped plugins have non-empty optional dependencies, so
// flattening them into one list or dropping them both change behaviour.
func (m *PluginManifest) UnmarshalJSON(data []byte) error {
	// Take dependencies out before the main decode, so the alias never sees a
	// shape its []string field cannot hold.
	var probe struct {
		Dependencies json.RawMessage `json:"dependencies"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return err
	}

	var required, optional []string
	haveGrouped := false
	if len(probe.Dependencies) > 0 && string(probe.Dependencies) != "null" {
		var g groupedDeps
		if err := json.Unmarshal(probe.Dependencies, &g); err == nil {
			required, optional = g.Required, g.Optional
			haveGrouped = true
		}
	}

	stripped := data
	if haveGrouped {
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(data, &raw); err != nil {
			return err
		}
		delete(raw, "dependencies")
		var err error
		stripped, err = json.Marshal(raw)
		if err != nil {
			return err
		}
	}

	var alias manifestAlias
	if err := json.Unmarshal(stripped, &alias); err != nil {
		return err
	}
	*m = PluginManifest(alias)

	if haveGrouped {
		m.Dependencies = required
		// Merge rather than overwrite: a manifest may carry both
		// dependencies.optional and a top-level optionalDependencies.
		m.OptionalDependencies = mergeUnique(m.OptionalDependencies, optional)
	}
	return nil
}

// MarshalJSON writes the canonical shapes, so anything this package re-emits
// round-trips through its own parser.
func (m PluginManifest) MarshalJSON() ([]byte, error) {
	return json.Marshal(manifestAlias(m))
}

// mergeUnique appends b to a, skipping entries already present, preserving
// order.
func mergeUnique(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	for _, v := range a {
		seen[v] = true
	}
	for _, v := range b {
		if !seen[v] {
			a = append(a, v)
			seen[v] = true
		}
	}
	return a
}

// UnmarshalJSON lets SystemDependencies accept the empty-array form.
//
// Four shipped manifests write `"systemDependencies": []` where this package
// expects an object. An empty array means "none", which is the zero value.
func (s *SystemDependencies) UnmarshalJSON(data []byte) error {
	trimmed := string(data)
	if trimmed == "null" || trimmed == "[]" {
		*s = SystemDependencies{}
		return nil
	}

	type alias SystemDependencies
	var a alias
	if err := json.Unmarshal(data, &a); err == nil {
		*s = SystemDependencies(a)
		return nil
	}

	// A non-empty array of dependency objects is the other plausible reading of
	// the array form: treat them as required.
	var asList []SystemDependency
	if err := json.Unmarshal(data, &asList); err == nil {
		*s = SystemDependencies{Required: asList}
		return nil
	}

	// Bare names: ["libsodium"]. Required, with nothing else stated.
	var asNames []string
	if err := json.Unmarshal(data, &asNames); err == nil {
		reqs := make([]SystemDependency, 0, len(asNames))
		for _, n := range asNames {
			reqs = append(reqs, SystemDependency{Name: n})
		}
		*s = SystemDependencies{Required: reqs}
		return nil
	}

	return fmt.Errorf("systemDependencies: expected an object with required/recommended, "+
		"or an array of dependencies, got %s", truncateJSON(data))
}
