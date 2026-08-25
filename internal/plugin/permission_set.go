package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
)

// The permission field, which carries two vocabularies.
//
// Split out of manifest_shapes.go: that file is about JSON shapes varying
// between published manifests, and this is about what those permissions MEAN
// and how the descriptive form reduces to the canonical one. The file-size
// ratchet forced the split and picked a better boundary than I had.
//
// See permission_derive.go for the reduction itself.

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
// descriptive form.
func (p PermissionSet) UsesLegacyVocabulary() bool { return len(p.Grouped) > 0 }

// Effective returns the permissions to validate: the canonical entries as
// written, plus the canonical reduction of any descriptive ones.
//
// Both are checked against the same allowlist. The descriptive form used to be
// recorded and skipped, which left 121 plugins outside a fail-closed check;
// see DeriveCanonicalPermissions for why reducing it is sound and which
// direction each mapping moves in.
func (p PermissionSet) Effective() []string {
	out := append([]string(nil), p.Canonical...)
	return append(out, DeriveCanonicalPermissions(p.Grouped)...)
}

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
