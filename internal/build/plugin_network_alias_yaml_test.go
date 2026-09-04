package build

import "gopkg.in/yaml.v3"

// assertValidComposeYAML parses s as generic YAML and returns an error if it
// does not parse. This is a syntax-only check (it cannot catch compose-spec
// schema violations like a `${VAR}` map key, which is syntactically valid
// YAML — see docker/compose-network-alias tests for the schema-level
// regression guard) but it does catch structural corruption such as bad
// indentation or duplicate/misaligned keys introduced by a byte-offset bug
// in the rewrite logic.
func assertValidComposeYAML(s string) error {
	var out map[string]any
	return yaml.Unmarshal([]byte(s), &out)
}
