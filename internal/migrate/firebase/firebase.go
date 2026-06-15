// Package firebase implements the Firebase → nSelf migration scaffold.
//
// It reads a Firestore JSON export (produced by `firebase firestore:export`),
// infers a relational schema from the document structure, and generates:
//   - A Drizzle-compatible SQL migration file
//   - A Hasura metadata YAML (table tracking + permissions scaffold)
//   - An optional Firebase Auth → nSelf auth user import script
//   - A human-readable next-steps summary
//
// The command intentionally does NOT call any Firebase HTTP API at runtime.
// The operator runs `firebase firestore:export --format json <outdir>` first,
// then points `nself migrate firebase --export-dir <outdir>` at the output.
// This avoids requiring a live Firebase project during migration.
package firebase

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
)

// ---------------------------------------------------------------------------
// Public types
// ---------------------------------------------------------------------------

// Options holds the user-supplied parameters for a Firebase migration run.
type Options struct {
	// ExportDir is the directory produced by `firebase firestore:export`.
	// It must contain at least one *.json file with Firestore document data.
	ExportDir string

	// AuthExportFile is an optional path to a Firebase Auth users JSON export
	// (produced by `firebase auth:export --format json <file>`).
	// When set, an auth-import.sql file is generated alongside the schema migration.
	AuthExportFile string

	// OutputDir is the directory where migration artifacts are written.
	// Defaults to <ExportDir>/nself-migration if empty.
	OutputDir string

	// ProjectName is used to name the generated migration files and schema.
	// Defaults to "firebase_import" if empty.
	ProjectName string

	// DryRun, when true, prints the generated SQL to stdout instead of writing files.
	DryRun bool
}

// Result holds the paths of all generated migration artifacts.
type Result struct {
	// SchemaSQL is the path to the generated SQL migration file.
	SchemaSQL string
	// HasuraYAML is the path to the generated Hasura metadata YAML.
	HasuraYAML string
	// AuthImportSQL is the path to the auth user import script (may be empty).
	AuthImportSQL string
	// Summary is the path to the human-readable next-steps summary.
	Summary string
	// Tables is the list of inferred table names.
	Tables []string
}

// CollectionInfo holds the inferred schema for a single Firestore collection.
type CollectionInfo struct {
	// Name is the Firestore collection name.
	Name string
	// TableName is the SQL-safe table name derived from Name.
	TableName string
	// Columns holds the inferred column definitions.
	Columns []ColumnDef
	// SampleCount is the number of documents sampled during inference.
	SampleCount int
}

// ColumnDef describes a single inferred column.
type ColumnDef struct {
	// Name is the SQL-safe column name.
	Name string
	// SQLType is the inferred PostgreSQL type.
	SQLType string
	// Nullable indicates whether the column may be NULL (field missing in some docs).
	Nullable bool
}

// ---------------------------------------------------------------------------
// Entry point
// ---------------------------------------------------------------------------

// Run performs the Firebase → nSelf migration scaffold.
// It reads Firestore export JSON files from opts.ExportDir, infers a
// relational schema, and writes migration artifacts to opts.OutputDir.
func Run(_ context.Context, opts Options) (*Result, error) {
	if opts.ExportDir == "" {
		return nil, fmt.Errorf("--export-dir is required")
	}
	if _, err := os.Stat(opts.ExportDir); err != nil {
		return nil, fmt.Errorf("export directory %q not found: %w", opts.ExportDir, err)
	}

	if opts.ProjectName == "" {
		opts.ProjectName = "firebase_import"
	}
	if opts.OutputDir == "" {
		opts.OutputDir = filepath.Join(opts.ExportDir, "nself-migration")
	}

	// 1. Discover and parse export files.
	collections, err := inferSchema(opts.ExportDir)
	if err != nil {
		return nil, fmt.Errorf("inferring schema from export: %w", err)
	}
	if len(collections) == 0 {
		return nil, fmt.Errorf("no Firestore documents found in %q — ensure the export contains *.json files", opts.ExportDir)
	}

	// 2. Generate SQL migration.
	schemaSQLBytes := buildSchemaMigration(collections, opts.ProjectName)

	// 3. Generate Hasura metadata YAML.
	hasuraYAMLBytes := buildHasuraMetadata(collections)

	// 4. Optionally generate auth user import SQL.
	var authImportBytes []byte
	if opts.AuthExportFile != "" {
		authImportBytes, err = buildAuthImport(opts.AuthExportFile)
		if err != nil {
			return nil, fmt.Errorf("generating auth import from %q: %w", opts.AuthExportFile, err)
		}
	}

	// 5. Generate next-steps summary.
	tableNames := make([]string, 0, len(collections))
	for _, c := range collections {
		tableNames = append(tableNames, c.TableName)
	}
	summaryBytes := buildSummary(collections, opts.ProjectName)

	// 6. Write or print.
	if opts.DryRun {
		slog.Info("dry run: schema SQL", "sql", string(schemaSQLBytes))
		if len(authImportBytes) > 0 {
			slog.Info("dry run: auth import SQL", "sql", string(authImportBytes))
		}
		slog.Info("dry run: summary", "summary", string(summaryBytes))
		return &Result{Tables: tableNames}, nil
	}

	if err := os.MkdirAll(opts.OutputDir, 0750); err != nil {
		return nil, fmt.Errorf("creating output directory %q: %w", opts.OutputDir, err)
	}

	ts := time.Now().UTC().Format("20060102150405")
	schemaPath := filepath.Join(opts.OutputDir, fmt.Sprintf("%s_001_firebase_schema.sql", ts))
	hasuraPath := filepath.Join(opts.OutputDir, "hasura_metadata.yaml")
	summaryPath := filepath.Join(opts.OutputDir, "MIGRATION_SUMMARY.md")

	if err := writeFile(schemaPath, schemaSQLBytes); err != nil {
		return nil, fmt.Errorf("writing schema SQL: %w", err)
	}
	if err := writeFile(hasuraPath, hasuraYAMLBytes); err != nil {
		return nil, fmt.Errorf("writing Hasura metadata: %w", err)
	}
	if err := writeFile(summaryPath, summaryBytes); err != nil {
		return nil, fmt.Errorf("writing summary: %w", err)
	}

	res := &Result{
		SchemaSQL:  schemaPath,
		HasuraYAML: hasuraPath,
		Summary:    summaryPath,
		Tables:     tableNames,
	}

	if len(authImportBytes) > 0 {
		authPath := filepath.Join(opts.OutputDir, "auth_import.sql")
		if err := writeFile(authPath, authImportBytes); err != nil {
			return nil, fmt.Errorf("writing auth import SQL: %w", err)
		}
		res.AuthImportSQL = authPath
	}

	return res, nil
}

// ---------------------------------------------------------------------------
// Schema inference
// ---------------------------------------------------------------------------

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

// ---------------------------------------------------------------------------
// SQL generation
// ---------------------------------------------------------------------------

func buildSchemaMigration(collections []CollectionInfo, projectName string) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("-- Firebase → nSelf schema migration\n"))
	sb.WriteString(fmt.Sprintf("-- Project: %s\n", projectName))
	sb.WriteString(fmt.Sprintf("-- Generated: %s\n", time.Now().UTC().Format(time.RFC3339)))
	sb.WriteString("-- Review carefully before applying to production.\n\n")

	sb.WriteString("BEGIN;\n\n")

	for _, coll := range collections {
		sb.WriteString(fmt.Sprintf("-- Collection: %s\n", coll.Name))
		sb.WriteString(fmt.Sprintf("-- Documents sampled: %d\n", coll.SampleCount))
		sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS public.%s (\n", coll.TableName))

		for i, col := range coll.Columns {
			nullStr := " NOT NULL"
			if col.Nullable {
				nullStr = ""
			}
			suffix := ","
			if i == len(coll.Columns)-1 {
				suffix = ""
			}
			if col.Name == "id" {
				sb.WriteString(fmt.Sprintf("  id TEXT NOT NULL PRIMARY KEY%s\n", suffix))
			} else {
				sb.WriteString(fmt.Sprintf("  %s %s%s%s\n", col.Name, col.SQLType, nullStr, suffix))
			}
		}

		sb.WriteString(");\n\n")

		// Basic RLS scaffold.
		sb.WriteString(fmt.Sprintf("ALTER TABLE public.%s ENABLE ROW LEVEL SECURITY;\n", coll.TableName))
		sb.WriteString(fmt.Sprintf("-- TODO: add RLS policies for public.%s (Firebase security rules → Postgres RLS)\n\n", coll.TableName))
	}

	sb.WriteString("COMMIT;\n")
	return []byte(sb.String())
}

// ---------------------------------------------------------------------------
// Hasura metadata YAML generation
// ---------------------------------------------------------------------------

func buildHasuraMetadata(collections []CollectionInfo) []byte {
	var sb strings.Builder
	sb.WriteString("# Hasura metadata — Firebase migration scaffold\n")
	sb.WriteString("# Apply via: nself hasura metadata apply\n\n")
	sb.WriteString("version: 3\n\n")
	sb.WriteString("sources:\n")
	sb.WriteString("  - name: default\n")
	sb.WriteString("    kind: postgres\n")
	sb.WriteString("    tables:\n")

	for _, coll := range collections {
		sb.WriteString(fmt.Sprintf("      - table:\n"))
		sb.WriteString(fmt.Sprintf("          schema: public\n"))
		sb.WriteString(fmt.Sprintf("          name: %s\n", coll.TableName))
		sb.WriteString(fmt.Sprintf("        # Collection: %s (%d documents sampled)\n", coll.Name, coll.SampleCount))
		sb.WriteString(fmt.Sprintf("        # TODO: configure select/insert/update/delete permissions\n"))
	}

	return []byte(sb.String())
}

// ---------------------------------------------------------------------------
// Auth import SQL generation
// ---------------------------------------------------------------------------

// firebaseAuthExport is the top-level structure of a `firebase auth:export` JSON file.
type firebaseAuthExport struct {
	Users []firebaseAuthUser `json:"users"`
}

// firebaseAuthUser holds the fields we care about from a Firebase auth user record.
type firebaseAuthUser struct {
	LocalID       string `json:"localId"`
	Email         string `json:"email"`
	DisplayName   string `json:"displayName"`
	EmailVerified bool   `json:"emailVerified"`
	Disabled      bool   `json:"disabled"`
	CreatedAt     string `json:"createdAt"`
	LastLoginAt   string `json:"lastLoginAt"`
}

func buildAuthImport(authExportFile string) ([]byte, error) {
	f, err := os.Open(authExportFile)
	if err != nil {
		return nil, fmt.Errorf("opening auth export: %w", err)
	}
	defer f.Close()

	var export firebaseAuthExport
	if err := json.NewDecoder(f).Decode(&export); err != nil {
		return nil, fmt.Errorf("parsing auth export: %w", err)
	}

	if len(export.Users) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	sb.WriteString("-- Firebase Auth user import\n")
	sb.WriteString(fmt.Sprintf("-- Source: %s\n", authExportFile))
	sb.WriteString(fmt.Sprintf("-- Users: %d\n", len(export.Users)))
	sb.WriteString("-- Review carefully; passwords are NOT migrated (users must reset).\n\n")
	sb.WriteString("BEGIN;\n\n")

	for _, u := range export.Users {
		if u.LocalID == "" || u.Email == "" {
			continue
		}
		email := sanitizeSQL(u.Email)
		displayName := sanitizeSQL(u.DisplayName)
		sb.WriteString(fmt.Sprintf(
			"INSERT INTO auth.users (id, email, email_confirmed_at, raw_user_meta_data, created_at, is_sso_user)\n"+
				"VALUES (gen_random_uuid(), '%s', %s, '{\"display_name\":\"%s\",\"firebase_uid\":\"%s\"}'::jsonb, NOW(), false)\n"+
				"ON CONFLICT (email) DO NOTHING;\n\n",
			email,
			boolToSQLTimestamp(u.EmailVerified),
			displayName,
			sanitizeSQL(u.LocalID),
		))
	}

	sb.WriteString("COMMIT;\n")
	return []byte(sb.String()), nil
}

func boolToSQLTimestamp(verified bool) string {
	if verified {
		return "NOW()"
	}
	return "NULL"
}

// sanitizeSQL removes single quotes from a value to prevent SQL injection.
// This is used only in auth import which is operator-controlled data, not user input.
func sanitizeSQL(s string) string {
	return strings.ReplaceAll(s, "'", "''")
}

// ---------------------------------------------------------------------------
// Summary generation
// ---------------------------------------------------------------------------

func buildSummary(collections []CollectionInfo, projectName string) []byte {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("# Firebase → nSelf Migration Summary\n\n"))
	sb.WriteString(fmt.Sprintf("**Project:** %s  \n", projectName))
	sb.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().UTC().Format(time.RFC3339)))

	sb.WriteString("## Inferred Tables\n\n")
	sb.WriteString("| Firestore Collection | PostgreSQL Table | Columns | Documents Sampled |\n")
	sb.WriteString("|---|---|---|---|\n")
	for _, c := range collections {
		sb.WriteString(fmt.Sprintf("| `%s` | `%s` | %d | %d |\n", c.Name, c.TableName, len(c.Columns), c.SampleCount))
	}

	sb.WriteString("\n## Next Steps\n\n")
	sb.WriteString("1. **Review the generated SQL** — column types are inferred from a sample and may need adjustment.\n")
	sb.WriteString("2. **Apply the schema migration**: `nself db migrate`\n")
	sb.WriteString("3. **Apply Hasura metadata**: `nself hasura metadata apply`\n")
	sb.WriteString("4. **Migrate data**: write a data migration script that reads your Firestore export and inserts rows.\n")
	sb.WriteString("5. **Add RLS policies**: replace `-- TODO: add RLS policies` with your actual row-level security rules.\n")
	sb.WriteString("6. **Reset user passwords**: Firebase passwords cannot be migrated. Send password-reset emails to all imported auth users.\n")
	sb.WriteString("7. **Test thoroughly** before decommissioning the Firebase project.\n")

	return []byte(sb.String())
}

// ---------------------------------------------------------------------------
// Name normalisation helpers
// ---------------------------------------------------------------------------

var nonAlNum = regexp.MustCompile(`[^a-z0-9]+`)

// toTableName converts a Firestore collection name to a SQL-safe table name.
// Example: "userProfiles" → "user_profiles", "orders-2024" → "orders_2024".
func toTableName(name string) string {
	return normalizeName(name)
}

// toColumnName converts a Firestore field name to a SQL-safe column name.
func toColumnName(name string) string {
	return normalizeName(name)
}

func normalizeName(name string) string {
	// Insert underscore before uppercase letters (camelCase → snake_case).
	var sb strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) && i > 0 {
			sb.WriteRune('_')
		}
		sb.WriteRune(unicode.ToLower(r))
	}
	s := sb.String()
	// Replace non-alphanumeric runs with underscore.
	s = nonAlNum.ReplaceAllString(s, "_")
	// Trim leading/trailing underscores.
	s = strings.Trim(s, "_")
	if s == "" {
		s = "field"
	}
	return s
}

// ---------------------------------------------------------------------------
// File helpers
// ---------------------------------------------------------------------------

func writeFile(path string, data []byte) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	w := bufio.NewWriter(f)
	if _, err := w.Write(data); err != nil {
		_ = f.Close()
		return fmt.Errorf("writing %s: %w", path, err)
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return fmt.Errorf("flushing %s: %w", path, err)
	}
	return f.Close()
}
