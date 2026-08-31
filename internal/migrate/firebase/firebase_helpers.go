package firebase

// firebase_helpers.go — naming and output helpers for the Firebase migration.
//
// Purpose: build the human-readable migration summary and normalize Firestore collection/field names into SQL-safe table/column names, used throughout firebase.go, split out for file size.
// Inputs: raw Firestore collection or field names, and the migration Result to summarize.
// Outputs: SQL-safe identifiers, and a written summary file.
// Constraints: pure move from firebase.go (CLI-R12 Batch E); no behaviour change.

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"
	"unicode"
)

func buildSummary(collections []CollectionInfo, projectName string) []byte {
	var sb strings.Builder
	sb.WriteString("# Firebase → nSelf Migration Summary\n\n")
	fmt.Fprintf(&sb, "**Project:** %s  \n", projectName)
	fmt.Fprintf(&sb, "**Generated:** %s\n\n", time.Now().UTC().Format(time.RFC3339))

	sb.WriteString("## Inferred Tables\n\n")
	sb.WriteString("| Firestore Collection | PostgreSQL Table | Columns | Documents Sampled |\n")
	sb.WriteString("|---|---|---|---|\n")
	for _, c := range collections {
		fmt.Fprintf(&sb, "| `%s` | `%s` | %d | %d |\n", c.Name, c.TableName, len(c.Columns), c.SampleCount)
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
