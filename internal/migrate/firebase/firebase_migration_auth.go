package firebase

// firebase_migration_auth.go — SQL/Hasura generation and Firebase Auth import.
//
// Purpose: render the inferred schema into a Drizzle SQL migration and Hasura metadata, and build the optional Firebase Auth to nSelf auth user import, used by Run in firebase.go, split out for file size.
// Inputs: the inferred CollectionInfo/ColumnDef values and, for auth import, a Firebase Auth export file.
// Outputs: a SQL migration file, Hasura metadata YAML, and an optional auth import script.
// Constraints: pure move from firebase.go (CLI-R12 Batch E); no behaviour change.

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

func buildSchemaMigration(collections []CollectionInfo, projectName string) []byte {
	var sb strings.Builder
	sb.WriteString("-- Firebase → nSelf schema migration\n")
	fmt.Fprintf(&sb, "-- Project: %s\n", projectName)
	fmt.Fprintf(&sb, "-- Generated: %s\n", time.Now().UTC().Format(time.RFC3339))
	sb.WriteString("-- Review carefully before applying to production.\n\n")

	sb.WriteString("BEGIN;\n\n")

	for _, coll := range collections {
		fmt.Fprintf(&sb, "-- Collection: %s\n", coll.Name)
		fmt.Fprintf(&sb, "-- Documents sampled: %d\n", coll.SampleCount)
		fmt.Fprintf(&sb, "CREATE TABLE IF NOT EXISTS public.%s (\n", coll.TableName)

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
				fmt.Fprintf(&sb, "  id TEXT NOT NULL PRIMARY KEY%s\n", suffix)
			} else {
				fmt.Fprintf(&sb, "  %s %s%s%s\n", col.Name, col.SQLType, nullStr, suffix)
			}
		}

		sb.WriteString(");\n\n")

		// Basic RLS scaffold.
		fmt.Fprintf(&sb, "ALTER TABLE public.%s ENABLE ROW LEVEL SECURITY;\n", coll.TableName)
		fmt.Fprintf(&sb, "-- TODO: add RLS policies for public.%s (Firebase security rules → Postgres RLS)\n\n", coll.TableName)
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
		sb.WriteString("      - table:\n")
		sb.WriteString("          schema: public\n")
		fmt.Fprintf(&sb, "          name: %s\n", coll.TableName)
		fmt.Fprintf(&sb, "        # Collection: %s (%d documents sampled)\n", coll.Name, coll.SampleCount)
		sb.WriteString("        # TODO: configure select/insert/update/delete permissions\n")
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
	defer func() { _ = f.Close() }()

	var export firebaseAuthExport
	if err := json.NewDecoder(f).Decode(&export); err != nil {
		return nil, fmt.Errorf("parsing auth export: %w", err)
	}

	if len(export.Users) == 0 {
		return nil, nil
	}

	var sb strings.Builder
	sb.WriteString("-- Firebase Auth user import\n")
	fmt.Fprintf(&sb, "-- Source: %s\n", authExportFile)
	fmt.Fprintf(&sb, "-- Users: %d\n", len(export.Users))
	sb.WriteString("-- Review carefully; passwords are NOT migrated (users must reset).\n\n")
	sb.WriteString("BEGIN;\n\n")

	for _, u := range export.Users {
		if u.LocalID == "" || u.Email == "" {
			continue
		}
		email := sanitizeSQL(u.Email)
		displayName := sanitizeSQL(u.DisplayName)
		fmt.Fprintf(&sb,
			"INSERT INTO auth.users (id, email, email_confirmed_at, raw_user_meta_data, created_at, is_sso_user)\n"+
				"VALUES (gen_random_uuid(), '%s', %s, '{\"display_name\":\"%s\",\"firebase_uid\":\"%s\"}'::jsonb, NOW(), false)\n"+
				"ON CONFLICT (email) DO NOTHING;\n\n",
			email,
			boolToSQLTimestamp(u.EmailVerified),
			displayName,
			sanitizeSQL(u.LocalID),
		)
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
