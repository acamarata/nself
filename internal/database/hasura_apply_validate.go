package database

// Purpose: the list-wrapped-!include'd-file guard used by
// applyViaIncludeResolver (hasura_apply.go) before sending resolved metadata
// to the Hasura API.
// Inputs: the yaml.v3-decoded document tree (map[string]interface{} or
// []interface{}, matching resolveIncludes' output shape).
// Outputs: a clear, actionable error the first time an array section
// contains a list where a bare mapping was expected; nil otherwise.
// Constraints: split out of hasura_apply.go (CLI-R12 300-line file cap) as a
// pure move; no behavior changed. FIX-CLI-3 (P6 2026-09-04).

import "fmt"

// hasuraArraySections are the top-level metadata.yaml/tables.yaml keys whose
// value must be an array of single-object mappings, one per !include'd file.
// This is the exact shape the historical incident (43/48 !include'd table
// files list-wrapped since a9eb61bf, found 2026-09-03) violated: each
// included file was itself `- table: ...` (a one-element list) instead of a
// bare `table: ...` mapping, silently nesting a list inside the array.
var hasuraArraySections = []string{"tables", "actions", "remote_schemas", "cron_triggers", "query_collections", "rest_endpoints"}

// validateNoListWrappedEntries walks the known metadata array sections and
// returns a clear, actionable error the first time an entry is itself a list
// rather than a single mapping — instead of letting the malformed shape
// reach the Hasura API as an opaque 400.
//
// Handles both metadata root shapes applyViaIncludeResolver accepts: a
// mapping with named array sections (config.yaml's "tables: !include
// tables.yaml" etc.), and a bare top-level array (projects that use
// tables.yaml directly at the metadata root, per its own doc comment above).
func validateNoListWrappedEntries(doc interface{}) error {
	if root, ok := doc.(map[string]interface{}); ok {
		for _, section := range hasuraArraySections {
			items, ok := root[section].([]interface{})
			if !ok {
				continue
			}
			if err := checkNoListWrappedItems(section, items); err != nil {
				return err
			}
		}
		return nil
	}
	if items, ok := doc.([]interface{}); ok {
		return checkNoListWrappedItems("tables", items)
	}
	return nil
}

// checkNoListWrappedItems is the shared per-item check used by both
// validateNoListWrappedEntries branches (named section, and bare top-level
// array).
func checkNoListWrappedItems(section string, items []interface{}) error {
	for i, item := range items {
		if _, isList := item.([]interface{}); isList {
			return fmt.Errorf(
				"hasura metadata: %s[%d] is a YAML list, not an object — the !include'd file is "+
					"list-wrapped (starts with \"- \" instead of a bare mapping); unwrap it to a single object",
				section, i)
		}
	}
	return nil
}
