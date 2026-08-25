package plugin

import (
	"encoding/json"
	"fmt"
	"sort"
)

// EnvVarList is the set of environment variables a plugin declares.
//
// Purpose: accept every shape real plugin.json files use. The field was typed
// `[]EnvVar`, but the manifest template every CLI-R11 extraction was copied
// from writes an object instead:
//
//	"envVars": { "required": [], "optional": ["HCLOUD_TOKEN"] }
//
// Unmarshalling that into a slice fails, and a manifest that fails to parse is
// discarded — readPluginManifest returns nil, ListInstalled skips the plugin.
// So every extracted plugin was absent from `nself plugin list` and from
// anything else that enumerates what is installed, silently. This is the same
// failure the `webhooks` field had, for the same reason: the type described one
// of the shapes in use rather than all of them.
//
// Inputs: the `envVars` value from plugin.json, in any of three shapes.
//
// Outputs: the declared variables, with Required set from whichever shape said
// so.
//
// Constraints: parsing must never fail on a shape a real manifest uses. A
// manifest this code cannot read makes a plugin invisible rather than loud,
// which is the worst of both.
type EnvVarList []EnvVar

// UnmarshalJSON accepts an array of objects, an array of names, an object with
// required/optional name lists, or null.
func (e *EnvVarList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		*e = nil
		return nil
	}

	// Canonical form: [{"name": "X", "required": true, ...}, ...]
	var asObjects []EnvVar
	if err := json.Unmarshal(data, &asObjects); err == nil {
		*e = asObjects
		return nil
	}

	// Shorthand: ["X", "Y"] — names only, requirement unstated.
	var asNames []string
	if err := json.Unmarshal(data, &asNames); err == nil {
		out := make([]EnvVar, 0, len(asNames))
		for _, n := range asNames {
			out = append(out, EnvVar{Name: n})
		}
		*e = out
		return nil
	}

	// Grouped form: {"required": ["A"], "optional": ["B"]}. This is what the
	// plugin template produces, and what broke.
	//
	// Each group is itself either a list of names or an object mapping name to
	// default value — nself-geo writes
	// {"required": ["DATABASE_URL"], "optional": {"GEOCODING_PROVIDERS": "nominatim"}}
	// — so the group values are decoded separately rather than assumed.
	var grouped map[string]json.RawMessage
	if err := json.Unmarshal(data, &grouped); err == nil {
		keys := make([]string, 0, len(grouped))
		for k := range grouped {
			keys = append(keys, k)
		}
		// required first, then optional, then anything else alphabetically, so
		// the result is stable regardless of map iteration order.
		sort.Slice(keys, func(i, j int) bool {
			return groupRank(keys[i], keys[j])
		})

		var out []EnvVar
		for _, k := range keys {
			required := k == "required"
			for _, v := range decodeEnvGroup(grouped[k]) {
				v.Required = required
				out = append(out, v)
			}
		}
		*e = out
		return nil
	}

	return fmt.Errorf("envVars: expected an array of declarations, an array of names, "+
		"or an object with required/optional name lists, got %s", truncateJSON(data))
}

// MarshalJSON always writes the canonical array-of-objects form.
func (e EnvVarList) MarshalJSON() ([]byte, error) {
	if e == nil {
		return []byte("null"), nil
	}
	return json.Marshal([]EnvVar(e))
}

// Names returns every declared variable name, in declaration order.
func (e EnvVarList) Names() []string {
	out := make([]string, 0, len(e))
	for _, v := range e {
		if v.Name != "" {
			out = append(out, v.Name)
		}
	}
	return out
}

// groupRank orders envVars group keys: required, optional, then the rest
// alphabetically. An unrecognised group is still a declaration and is kept —
// dropping one silently is how this class of bug started.
func groupRank(a, b string) bool {
	rank := func(k string) int {
		switch k {
		case "required":
			return 0
		case "optional":
			return 1
		default:
			return 2
		}
	}
	ra, rb := rank(a), rank(b)
	if ra != rb {
		return ra < rb
	}
	return a < b
}

// decodeEnvGroup reads one group's value, which is either a list of names or an
// object mapping name to default value.
func decodeEnvGroup(raw json.RawMessage) []EnvVar {
	var names []string
	if err := json.Unmarshal(raw, &names); err == nil {
		out := make([]EnvVar, 0, len(names))
		for _, n := range names {
			out = append(out, EnvVar{Name: n})
		}
		return out
	}

	var withDefaults map[string]json.RawMessage
	if err := json.Unmarshal(raw, &withDefaults); err == nil {
		keys := make([]string, 0, len(withDefaults))
		for k := range withDefaults {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		out := make([]EnvVar, 0, len(keys))
		for _, k := range keys {
			out = append(out, EnvVar{Name: k, Default: scalarString(withDefaults[k])})
		}
		return out
	}

	return nil
}

// scalarString renders a JSON scalar as the string a default value should be.
// A quoted string loses its quotes; numbers and booleans are kept verbatim.
func scalarString(raw json.RawMessage) string {
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	return string(raw)
}
