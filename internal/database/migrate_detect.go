package database

import (
	"regexp"
	"strings"
)

// Purpose: parse migration SQL for the schema objects it creates, and
// classify a migration against a live-catalog presence map. Pure text/data
// functions only — see migrate_detect_query.go for the live pg_catalog query
// that supplies the presence map DetectMigrations uses.
// Inputs: raw SQL text (extraction) or an ObjectRef list + presence map
// (classification).
// Outputs: []ObjectRef, or (DetectClass, present, missing).
// Constraints: extraction is best-effort regex, not a SQL parser — it is
// intentionally conservative (may under-detect quoted/computed identifiers)
// but must never fabricate an object that isn't in the text. Classification
// is honest by construction: BASELINE requires every object present, APPLY
// requires none present; anything else is CONFLICT, never silently resolved.

// ObjectKind identifies the kind of schema object a CREATE statement makes.
type ObjectKind string

const (
	ObjectTable      ObjectKind = "table"
	ObjectIndex      ObjectKind = "index"
	ObjectType       ObjectKind = "type"
	ObjectView       ObjectKind = "view"
	ObjectMatView    ObjectKind = "materialized view"
	ObjectSequence   ObjectKind = "sequence"
	ObjectSchemaKind ObjectKind = "schema"
)

// ObjectRef is one schema object a migration file creates.
type ObjectRef struct {
	Kind ObjectKind
	Name string // as written in the SQL; may be schema-qualified
}

// Key returns the unique map key used to dedupe/look up presence for o.
func (o ObjectRef) Key() string {
	return string(o.Kind) + ":" + strings.ToLower(o.Name)
}

var (
	createTableRe   = regexp.MustCompile(`(?i)\bCREATE\s+TABLE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][\w]*(?:\.[A-Za-z_][\w]*)?)`)
	createIndexRe   = regexp.MustCompile(`(?i)\bCREATE\s+(?:UNIQUE\s+)?INDEX\s+(?:CONCURRENTLY\s+)?(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][\w]*)`)
	createTypeRe    = regexp.MustCompile(`(?i)\bCREATE\s+TYPE\s+([A-Za-z_][\w]*(?:\.[A-Za-z_][\w]*)?)`)
	createMatViewRe = regexp.MustCompile(`(?i)\bCREATE\s+MATERIALIZED\s+VIEW\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][\w]*(?:\.[A-Za-z_][\w]*)?)`)
	createViewRe    = regexp.MustCompile(`(?i)\bCREATE\s+(?:OR\s+REPLACE\s+)?VIEW\s+([A-Za-z_][\w]*(?:\.[A-Za-z_][\w]*)?)`)
	createSeqRe     = regexp.MustCompile(`(?i)\bCREATE\s+SEQUENCE\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][\w]*(?:\.[A-Za-z_][\w]*)?)`)
	createSchemaRe  = regexp.MustCompile(`(?i)\bCREATE\s+SCHEMA\s+(?:IF\s+NOT\s+EXISTS\s+)?([A-Za-z_][\w]*)`)
)

// ExtractCreatedObjects returns every schema object sqlContent creates.
func ExtractCreatedObjects(sqlContent string) []ObjectRef {
	var out []ObjectRef
	collect := func(kind ObjectKind, re *regexp.Regexp) {
		for _, m := range re.FindAllStringSubmatch(sqlContent, -1) {
			out = append(out, ObjectRef{Kind: kind, Name: m[1]})
		}
	}
	collect(ObjectTable, createTableRe)
	collect(ObjectIndex, createIndexRe)
	collect(ObjectType, createTypeRe)
	collect(ObjectMatView, createMatViewRe)
	collect(ObjectView, createViewRe)
	collect(ObjectSequence, createSeqRe)
	collect(ObjectSchemaKind, createSchemaRe)
	return out
}

// DetectClass is the honest three-way (plus unknown) classification of a
// pending migration against the live schema.
type DetectClass string

const (
	DetectApply    DetectClass = "APPLY"
	DetectBaseline DetectClass = "BASELINE"
	DetectConflict DetectClass = "CONFLICT"
	DetectUnknown  DetectClass = "UNKNOWN"
)

// ClassifyByPresence classifies a migration by how many of the objects it
// creates already exist (existing, keyed by ObjectRef.Key()). BASELINE is
// only reachable when every object is present and APPLY only when none
// are — any partial match is CONFLICT, which callers must never
// auto-resolve. UNKNOWN means extraction found no objects to check (e.g. a
// data-only migration) — callers should fall back to ledger-only status.
func ClassifyByPresence(objects []ObjectRef, existing map[string]bool) (class DetectClass, present, missing []ObjectRef) {
	if len(objects) == 0 {
		return DetectUnknown, nil, nil
	}
	for _, o := range objects {
		if existing[o.Key()] {
			present = append(present, o)
		} else {
			missing = append(missing, o)
		}
	}
	if len(missing) == 0 {
		return DetectBaseline, present, missing
	}
	if len(present) == 0 {
		return DetectApply, present, missing
	}
	return DetectConflict, present, missing
}
