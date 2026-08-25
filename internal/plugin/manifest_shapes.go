package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
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

// PermissionSet is what a plugin declares it needs.
//
// Two vocabularies are in use and they do not describe the same things:
//
//   - The canonical one this CLI validates: "db:read", "fs:write",
//     "network:internet", "system:exec" — an array of strings, checked
//     fail-closed against an allowlist (S71-T01).
//   - A descriptive one used by 121 shipped manifests: an object such as
//     {"database": ["create","read"], "filesystem": ["logs"],
//     "network": ["*"]}. None of its tokens appear in the allowlist.
//
// Flattening the second into the first would produce "database:create" and be
// rejected, turning an invisible plugin into one that refuses to install.
// Mapping between them — deciding that "database:create" means "db:write" —
// would be inventing security policy, which is not a parser's job.
//
// So both are kept, separately. Canonical is validated. Grouped is recorded and
// displayed, and the divergence is a real finding for someone to settle.
type PermissionSet struct {
	// Canonical holds array-form entries, validated against the allowlist.
	Canonical []string
	// Grouped holds object-form entries verbatim, in their own vocabulary.
	Grouped map[string][]string
}

// UnmarshalJSON accepts the array form, the object form, or null.
func (p *PermissionSet) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*p = PermissionSet{}
		return nil
	}

	var asList []string
	if err := json.Unmarshal(data, &asList); err == nil {
		*p = PermissionSet{Canonical: asList}
		return nil
	}

	var asMap map[string][]string
	if err := json.Unmarshal(data, &asMap); err == nil {
		*p = PermissionSet{Grouped: asMap}
		return nil
	}

	return fmt.Errorf("permissions: expected an array of canonical permissions "+
		"or an object grouping actions by category, got %s", truncateJSON(data))
}

// MarshalJSON writes back whichever form was read, preferring the canonical
// array when both are somehow present.
func (p PermissionSet) MarshalJSON() ([]byte, error) {
	if len(p.Canonical) > 0 || p.Grouped == nil {
		if p.Canonical == nil {
			return []byte("null"), nil
		}
		return json.Marshal(p.Canonical)
	}
	return json.Marshal(p.Grouped)
}

// Len reports how many permissions were declared, in either vocabulary.
func (p PermissionSet) Len() int {
	n := len(p.Canonical)
	for _, actions := range p.Grouped {
		n += len(actions)
	}
	return n
}

// Strings renders every declared permission for display. Grouped entries are
// shown as "category:action" so a human can read them, which is NOT the same as
// treating them as canonical — nothing validates these.
func (p PermissionSet) Strings() []string {
	out := append([]string(nil), p.Canonical...)
	cats := make([]string, 0, len(p.Grouped))
	for c := range p.Grouped {
		cats = append(cats, c)
	}
	sort.Strings(cats)
	for _, c := range cats {
		for _, a := range p.Grouped[c] {
			out = append(out, c+":"+a)
		}
	}
	return out
}

// UsesLegacyVocabulary reports whether this plugin declared permissions in the
// descriptive form, which the canonical allowlist does not cover.
func (p PermissionSet) UsesLegacyVocabulary() bool { return len(p.Grouped) > 0 }

// APIEndpointList is the set of HTTP endpoints a plugin exposes.
//
// Published manifests use two shapes: an array of "METHOD /path" strings, and
// an array of {method, path, description} objects. The field was typed for the
// first, so the one plugin using the second was unparseable and therefore
// invisible.
type APIEndpointList []string

// UnmarshalJSON accepts an array of strings, an array of objects, or null.
func (a *APIEndpointList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*a = nil
		return nil
	}

	var asStrings []string
	if err := json.Unmarshal(data, &asStrings); err == nil {
		*a = asStrings
		return nil
	}

	var asObjects []struct {
		Method string `json:"method"`
		Path   string `json:"path"`
	}
	if err := json.Unmarshal(data, &asObjects); err == nil {
		out := make([]string, 0, len(asObjects))
		for _, e := range asObjects {
			switch {
			case e.Method != "" && e.Path != "":
				out = append(out, e.Method+" "+e.Path)
			case e.Path != "":
				out = append(out, e.Path)
			}
		}
		*a = out
		return nil
	}

	return fmt.Errorf("apiEndpoints: expected an array of \"METHOD /path\" strings "+
		"or an array of {method, path} objects, got %s", truncateJSON(data))
}
