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
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"
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
