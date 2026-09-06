package database

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// Purpose: the live-catalog half of --detect — querying pg_catalog /
// information_schema for whether each object a pending migration creates
// already exists, and assembling the per-migration DetectResult the `db
// migrate status --detect` command prints.
// Inputs: a *config.Config and a migrations directory (or "" to auto-detect).
// Outputs: []DetectResult, one per pending (not-yet-in-ledger) migration.
// Constraints: read-only. Never writes to the ledger or the schema; acting on
// a BASELINE-classified result is always a separate, explicit command
// (nself db migrate baseline), never an automatic side effect of detection.

// DetectResult is one pending migration's classification against the live
// schema, plus its bulk-unguarded-CREATE-TABLE lint finding (nil if clean).
type DetectResult struct {
	Name    string
	Class   DetectClass
	Present []ObjectRef
	Missing []ObjectRef
	Lint    *LintFinding
}

// existsExprFor returns the SQL boolean expression that tests whether o
// already exists in the live catalog.
func existsExprFor(o ObjectRef) string {
	switch o.Kind {
	case ObjectSchemaKind:
		return fmt.Sprintf("EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = %s)", sqlLiteral(o.Name))
	case ObjectType:
		return fmt.Sprintf("to_regtype(%s) IS NOT NULL", sqlLiteral(o.Name))
	default:
		// Tables, views, materialized views, sequences, and indexes are all
		// relations — to_regclass resolves any of them via search_path.
		return fmt.Sprintf("to_regclass(%s) IS NOT NULL", sqlLiteral(o.Name))
	}
}

// sqlLiteral quotes s as a SQL string literal, doubling embedded quotes.
func sqlLiteral(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

// queryExistingObjects checks every distinct object in objects against the
// live catalog in a single round trip. Returns a map keyed by ObjectRef.Key().
func queryExistingObjects(ctx context.Context, cfg *config.Config, objects []ObjectRef) (map[string]bool, error) {
	result := make(map[string]bool, len(objects))
	if len(objects) == 0 {
		return result, nil
	}

	seen := make(map[string]bool, len(objects))
	var selects []string
	for _, o := range objects {
		key := o.Key()
		if seen[key] {
			continue
		}
		seen[key] = true
		selects = append(selects, fmt.Sprintf("SELECT %s || '|' || (%s)::text", sqlLiteral(key), existsExprFor(o)))
	}

	db := cfg.Postgres.DB
	if db == "" {
		db = "nself"
	}
	out, err := querySQL(ctx, cfg, db, strings.Join(selects, "\nUNION ALL\n"))
	if err != nil {
		return nil, fmt.Errorf("query existing objects: %w", err)
	}

	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			continue
		}
		result[parts[0]] = parts[1] == "t" || parts[1] == "true"
	}
	return result, nil
}

// DetectMigrations classifies every pending (not-yet-in-ledger) migration in
// dir (or the auto-detected directory when dir is "") against the live
// schema.
func DetectMigrations(ctx context.Context, cfg *config.Config, dir string) ([]DetectResult, error) {
	if dir == "" {
		dir = migrationsDir(cfg, "")
	}
	files, err := scanMigrations(dir)
	if err != nil {
		return nil, err
	}

	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		// Ledger table likely doesn't exist yet — detection is read-only, so
		// treat this as "nothing applied" rather than creating it.
		applied = map[string]time.Time{}
	}

	type candidate struct {
		name    string
		objects []ObjectRef
		content string
	}
	var pending []candidate
	var allObjects []ObjectRef
	for _, f := range files {
		name := migrationKey(f)
		if _, ok := applied[name]; ok {
			continue
		}
		data, readErr := os.ReadFile(f)
		if readErr != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, readErr)
		}
		content := string(data)
		objs := ExtractCreatedObjects(content)
		pending = append(pending, candidate{name: name, objects: objs, content: content})
		allObjects = append(allObjects, objs...)
	}

	existing, err := queryExistingObjects(ctx, cfg, allObjects)
	if err != nil {
		return nil, err
	}

	results := make([]DetectResult, 0, len(pending))
	for _, c := range pending {
		class, present, missing := ClassifyByPresence(c.objects, existing)
		results = append(results, DetectResult{
			Name:    c.name,
			Class:   class,
			Present: present,
			Missing: missing,
			Lint:    LintUnguardedBulkCreateTables(c.content),
		})
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results, nil
}
