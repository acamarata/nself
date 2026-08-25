package firebase

// firebase_schema.go — Firestore export parsing and schema inference.
//
// Purpose: walk a Firestore JSON export, infer a relational schema from the document structure and classify field types, used by Run in firebase.go, split out for file size.
// Inputs: the export directory produced by `firebase firestore:export`.
// Outputs: inferred CollectionInfo/ColumnDef values consumed by the migration and Hasura metadata builders.
// Constraints: pure move from firebase.go (CLI-R12 Batch E); no behaviour change.

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// inferSchema reads all *.json files under exportDir and infers a per-collection schema.
// Firestore export format: a single JSON object with a top-level "collections" key,
// or an array of documents each with "__name__" (document path) and arbitrary fields.
// We support both the gcloud/firebase CLI export formats.
func inferSchema(exportDir string) ([]CollectionInfo, error) {
	paths, err := findJSONFiles(exportDir)
	if err != nil {
		return nil, err
	}

	// collectionFields maps collection name → field name → set of observed types.
	collectionFields := make(map[string]map[string]map[string]int)
	collectionSamples := make(map[string]int)

	for _, p := range paths {
		if err := parseExportFile(p, collectionFields, collectionSamples); err != nil {
			// Log and continue — partial imports are better than a full abort.
			fmt.Fprintf(os.Stderr, "warning: skipping %s: %v\n", p, err)
		}
	}

	if len(collectionFields) == 0 {
		return nil, nil
	}

	collections := make([]CollectionInfo, 0, len(collectionFields))
	for collName, fields := range collectionFields {
		info := CollectionInfo{
			Name:        collName,
			TableName:   toTableName(collName),
			SampleCount: collectionSamples[collName],
		}

		// Always include the document ID as primary key.
		info.Columns = append(info.Columns, ColumnDef{
			Name:     "id",
			SQLType:  "TEXT",
			Nullable: false,
		})

		// Sort fields for deterministic output.
		sortedFields := make([]string, 0, len(fields))
		for f := range fields {
			sortedFields = append(sortedFields, f)
		}
		slices.Sort(sortedFields)

		for _, fieldName := range sortedFields {
			typeCounts := fields[fieldName]
			sqlType := inferSQLType(typeCounts)
			nullable := collectionSamples[collName] > 1 && len(typeCounts) > 0
			info.Columns = append(info.Columns, ColumnDef{
				Name:     toColumnName(fieldName),
				SQLType:  sqlType,
				Nullable: nullable,
			})
		}

		collections = append(collections, info)
	}

	// Sort by table name for stable output.
	slices.SortFunc(collections, func(a, b CollectionInfo) int {
		return strings.Compare(a.TableName, b.TableName)
	})

	return collections, nil
}

// findJSONFiles returns all *.json paths under dir (non-recursive for the top level,
// plus any immediate subdirectories that match Firestore collection naming).
func findJSONFiles(dir string) ([]string, error) {
	var paths []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading export directory: %w", err)
	}
	for _, e := range entries {
		name := e.Name()
		fullPath := filepath.Join(dir, name)
		if e.IsDir() {
			// Walk one level into subdirectories (Firestore collections are directories).
			subPaths, _ := findJSONFilesDir(fullPath)
			paths = append(paths, subPaths...)
			continue
		}
		if strings.HasSuffix(name, ".json") {
			paths = append(paths, fullPath)
		}
	}
	return paths, nil
}

func findJSONFilesDir(dir string) ([]string, error) {
	var paths []string
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			paths = append(paths, filepath.Join(dir, e.Name()))
		}
	}
	return paths, nil
}

// parseExportFile parses a single Firestore export JSON file.
// It handles two formats:
//  1. Array of documents: [{"__name__": "col/docID", "field": value, ...}, ...]
//  2. Object with "documents" key: {"documents": [...]}
//  3. Single object (one document per file): {"__name__": "col/docID", "field": value}
func parseExportFile(path string, fields map[string]map[string]map[string]int, samples map[string]int) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close()

	var raw json.RawMessage
	if err := json.NewDecoder(f).Decode(&raw); err != nil {
		return fmt.Errorf("parsing %s: %w", path, err)
	}

	// Determine top-level shape.
	trimmed := strings.TrimSpace(string(raw))
	if len(trimmed) == 0 {
		return nil
	}

	switch trimmed[0] {
	case '[':
		var docs []map[string]json.RawMessage
		if err := json.Unmarshal(raw, &docs); err != nil {
			return fmt.Errorf("parsing document array in %s: %w", path, err)
		}
		for _, doc := range docs {
			processDocument(doc, filepath.Base(filepath.Dir(path)), fields, samples)
		}
	case '{':
		var obj map[string]json.RawMessage
		if err := json.Unmarshal(raw, &obj); err != nil {
			return fmt.Errorf("parsing document object in %s: %w", path, err)
		}
		// Check for "documents" wrapper.
		if docsRaw, ok := obj["documents"]; ok {
			var docs []map[string]json.RawMessage
			if err := json.Unmarshal(docsRaw, &docs); err != nil {
				return fmt.Errorf("parsing documents array in %s: %w", path, err)
			}
			for _, doc := range docs {
				processDocument(doc, filepath.Base(filepath.Dir(path)), fields, samples)
			}
			return nil
		}
		// Single document.
		processDocument(obj, filepath.Base(filepath.Dir(path)), fields, samples)
	}

	return nil
}

// processDocument extracts field type information from a single Firestore document.
// collHint is the inferred collection name from the file's parent directory.
func processDocument(doc map[string]json.RawMessage, collHint string, fields map[string]map[string]map[string]int, samples map[string]int) {
	// Derive collection name: try __name__ (e.g. "projects/p/databases/d/documents/users/uid"),
	// fall back to collHint (parent directory name).
	collName := collHint
	if nameRaw, ok := doc["__name__"]; ok {
		var name string
		if err := json.Unmarshal(nameRaw, &name); err == nil {
			parts := strings.Split(name, "/documents/")
			if len(parts) == 2 {
				seg := strings.Split(strings.TrimPrefix(parts[1], "/"), "/")
				if len(seg) >= 1 {
					collName = seg[0]
				}
			}
		}
	}

	if collName == "" || collName == "." {
		collName = "documents"
	}

	if fields[collName] == nil {
		fields[collName] = make(map[string]map[string]int)
	}
	samples[collName]++

	for k, v := range doc {
		if k == "__name__" {
			continue
		}
		colName := k
		if fields[collName][colName] == nil {
			fields[collName][colName] = make(map[string]int)
		}
		typeName := jsonTypeName(v)
		fields[collName][colName][typeName]++
	}
}

// jsonTypeName returns a simple type label for a JSON value.
func jsonTypeName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return "null"
	}
	switch raw[0] {
	case '"':
		// Detect timestamp strings.
		var s string
		if json.Unmarshal(raw, &s) == nil {
			if looksLikeTimestamp(s) {
				return "timestamp"
			}
		}
		return "string"
	case 't', 'f':
		return "boolean"
	case 'n':
		return "null"
	case '[':
		return "array"
	case '{':
		return "object"
	default:
		// Number.
		var f float64
		if json.Unmarshal(raw, &f) == nil {
			if strings.ContainsAny(string(raw), ".eE") {
				return "float"
			}
			return "integer"
		}
		return "string"
	}
}

var timestampRE = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`)

func looksLikeTimestamp(s string) bool {
	return timestampRE.MatchString(s)
}

// inferSQLType picks the best PostgreSQL type given observed JSON types.
// When multiple types are seen for the same field, JSONB is used as a fallback.
func inferSQLType(typeCounts map[string]int) string {
	seen := make([]string, 0, len(typeCounts))
	for t := range typeCounts {
		if t != "null" {
			seen = append(seen, t)
		}
	}
	if len(seen) == 0 {
		return "TEXT"
	}
	if len(seen) > 1 {
		return "JSONB"
	}
	switch seen[0] {
	case "integer":
		return "BIGINT"
	case "float":
		return "DOUBLE PRECISION"
	case "boolean":
		return "BOOLEAN"
	case "timestamp":
		return "TIMESTAMPTZ"
	case "array", "object":
		return "JSONB"
	default:
		return "TEXT"
	}
}
