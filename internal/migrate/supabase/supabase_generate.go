package supabase

// supabase_generate.go — generating nSelf artifacts from a pulled Supabase project.
//
// Purpose: render a pulled PullResult into a Drizzle SQL migration, Hasura metadata, an auth import script and a human-readable summary, split out of supabase.go for file size.
// Inputs: a PullResult from supabase_pull.go.
// Outputs: GeneratedArtifacts containing the rendered files.
// Constraints: pure move from supabase.go (CLI-R12 Batch E); no behaviour change.

import (
	"bytes"
	"fmt"
	"strings"
	"time"
)

// Generate converts a PullResult into nSelf-compatible artifacts.
func Generate(projectRef string, r *PullResult) *GeneratedArtifacts {
	return &GeneratedArtifacts{
		MigrationSQL:     generateMigrationSQL(r),
		HasuraMetadata:   generateHasuraMetadata(projectRef, r),
		AuthImportScript: generateAuthImportScript(r.Users),
		Summary:          generateSummary(projectRef, r),
	}
}

// generateMigrationSQL produces a Drizzle-compatible CREATE TABLE migration.
func generateMigrationSQL(r *PullResult) string {
	var b bytes.Buffer
	ts := time.Now().Format("20060102150405")
	fmt.Fprintf(&b, "-- nSelf migration generated from Supabase export\n")
	fmt.Fprintf(&b, "-- Generated: %s\n", ts)
	fmt.Fprintf(&b, "-- Tables: %d  |  RLS policies: %d\n\n", len(r.Tables), len(r.Policies))

	for _, t := range r.Tables {
		fmt.Fprintf(&b, "CREATE TABLE IF NOT EXISTS %s.%s (\n", quoteIdent(t.Schema), quoteIdent(t.Name))
		for i, col := range t.Columns {
			nullable := ""
			if !col.IsNullable {
				nullable = " NOT NULL"
			}
			def := ""
			if col.Default != "" {
				def = fmt.Sprintf(" DEFAULT %s", col.Default)
			}
			comma := ","
			if i == len(t.Columns)-1 {
				comma = ""
			}
			fmt.Fprintf(&b, "  %s %s%s%s%s\n", quoteIdent(col.Name), col.DataType, nullable, def, comma)
		}
		fmt.Fprintf(&b, ");\n\n")
	}

	if len(r.Policies) == 0 {
		fmt.Fprintf(&b, "-- NOTE: RLS policies could not be fetched automatically via PostgREST.\n")
		fmt.Fprintf(&b, "-- Review your Supabase RLS policies and re-apply them using:\n")
		fmt.Fprintf(&b, "--   ALTER TABLE <table> ENABLE ROW LEVEL SECURITY;\n")
		fmt.Fprintf(&b, "--   CREATE POLICY ... ON <table> ...\n\n")
	}

	for _, p := range r.Policies {
		roles := strings.Join(p.Roles, ", ")
		permissive := "PERMISSIVE"
		if !p.Permissive {
			permissive = "RESTRICTIVE"
		}
		fmt.Fprintf(&b, "-- RLS: %s.%s  cmd=%s  %s  roles=[%s]\n", p.TableSchema, p.TableName, p.Command, permissive, roles)
		fmt.Fprintf(&b, "ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;\n", quoteIdent(p.TableSchema), quoteIdent(p.TableName))
		if p.Using != "" {
			fmt.Fprintf(&b, "CREATE POLICY %s ON %s.%s AS %s FOR %s USING (%s);\n",
				quoteIdent(p.PolicyName), quoteIdent(p.TableSchema), quoteIdent(p.TableName),
				permissive, p.Command, p.Using)
		}
		fmt.Fprintf(&b, "\n")
	}

	return b.String()
}

// generateHasuraMetadata produces a minimal Hasura metadata YAML for the tables.
func generateHasuraMetadata(projectRef string, r *PullResult) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "# Hasura metadata generated from Supabase project %s\n", projectRef)
	fmt.Fprintf(&b, "# Apply with: hasura metadata apply\n\n")
	fmt.Fprintf(&b, "version: 3\n")
	fmt.Fprintf(&b, "sources:\n")
	fmt.Fprintf(&b, "  - name: default\n")
	fmt.Fprintf(&b, "    kind: postgres\n")
	fmt.Fprintf(&b, "    tables:\n")
	for _, t := range r.Tables {
		fmt.Fprintf(&b, "      - table:\n")
		fmt.Fprintf(&b, "          schema: %s\n", t.Schema)
		fmt.Fprintf(&b, "          name: %s\n", t.Name)
		fmt.Fprintf(&b, "        select_permissions:\n")
		fmt.Fprintf(&b, "          - role: user\n")
		fmt.Fprintf(&b, "            permission:\n")
		fmt.Fprintf(&b, "              filter: {}\n")
		fmt.Fprintf(&b, "              columns: \"*\"\n")
	}
	return b.String()
}

// generateAuthImportScript produces a shell script to import Supabase auth users.
func generateAuthImportScript(users []AuthUser) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "#!/usr/bin/env bash\n")
	fmt.Fprintf(&b, "# nSelf auth user import — generated from Supabase export\n")
	fmt.Fprintf(&b, "# Users: %d\n", len(users))
	fmt.Fprintf(&b, "# Run: bash import-auth-users.sh\n\n")
	fmt.Fprintf(&b, "set -euo pipefail\n\n")

	if len(users) == 0 {
		fmt.Fprintf(&b, "echo 'No auth users to import.'\n")
		return b.String()
	}

	fmt.Fprintf(&b, "NSELF_AUTH_URL=\"${NSELF_AUTH_URL:-http://localhost:4000}\"\n\n")
	fmt.Fprintf(&b, "echo \"Importing %d users via nSelf auth API...\"\n\n", len(users))

	for _, u := range users {
		email := strings.ReplaceAll(u.Email, `"`, `\"`)
		fmt.Fprintf(&b, "# User: %s\n", email)
		fmt.Fprintf(&b, "curl -sf -X POST \"${NSELF_AUTH_URL}/admin/v1/users\" \\\n")
		fmt.Fprintf(&b, "  -H 'Content-Type: application/json' \\\n")
		fmt.Fprintf(&b, "  -d '{\"email\":\"%s\",\"id\":\"%s\",\"email_confirm\":true}' || true\n\n", email, u.ID)
	}

	fmt.Fprintf(&b, "echo 'Import complete. Verify with: nself auth users'\n")
	return b.String()
}

// generateSummary returns a human-readable summary string.
func generateSummary(projectRef string, r *PullResult) string {
	var b bytes.Buffer
	fmt.Fprintf(&b, "Migration from Supabase project %q complete.\n\n", projectRef)
	fmt.Fprintf(&b, "Pulled:\n")
	fmt.Fprintf(&b, "  Tables:          %d\n", len(r.Tables))
	fmt.Fprintf(&b, "  RLS policies:    %d (see migration SQL for manual steps)\n", len(r.Policies))
	fmt.Fprintf(&b, "  Auth users:      %d\n", len(r.Users))
	fmt.Fprintf(&b, "  Storage buckets: %d\n\n", len(r.Buckets))
	fmt.Fprintf(&b, "Generated files:\n")
	fmt.Fprintf(&b, "  supabase-migration.sql       — run via: nself db migrate\n")
	fmt.Fprintf(&b, "  hasura-metadata.yaml         — apply via: hasura metadata apply\n")
	fmt.Fprintf(&b, "  import-auth-users.sh         — run to import users\n\n")
	fmt.Fprintf(&b, "Next steps:\n")
	fmt.Fprintf(&b, "  1. nself init                   — create a new nSelf project (if not done)\n")
	fmt.Fprintf(&b, "  2. nself start                  — boot the stack\n")
	fmt.Fprintf(&b, "  3. psql -f supabase-migration.sql  — apply schema\n")
	fmt.Fprintf(&b, "  4. hasura metadata apply           — load Hasura metadata\n")
	fmt.Fprintf(&b, "  5. bash import-auth-users.sh       — import auth users\n")
	if len(r.Buckets) > 0 {
		fmt.Fprintf(&b, "  6. Configure MinIO buckets to match Supabase Storage buckets:\n")
		for _, bucket := range r.Buckets {
			visibility := "private"
			if bucket.Public {
				visibility = "public"
			}
			fmt.Fprintf(&b, "       nself storage create-bucket %s --visibility=%s\n", bucket.Name, visibility)
		}
	}
	return b.String()
}

// quoteIdent wraps a SQL identifier in double-quotes.
func quoteIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}
