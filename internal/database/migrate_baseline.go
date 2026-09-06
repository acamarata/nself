package database

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/nself-org/cli/internal/config"
)

// Purpose: the "adopt an already-populated database into the migration
// ledger" verb (cli#386) — planning what a baseline would record, and the
// actual ledger write that never executes the migration's own SQL.
// Inputs: a *config.Config, a migrations directory (or "" to auto-detect),
// and the migration name(s)/file path requested.
// Outputs: BaselinePlan (planning) or a ledger write's error (execution).
// Constraints: BaselineMigration never runs migration SQL — only the two
// ledger INSERTs migrationRecordSQL already builds for the real apply path.
// Planning must not write anything, including creating schema_versions, so a
// missing ledger table is treated as "nothing applied yet", not an error.

// BaselinePlan describes what baselining one migration would record.
type BaselinePlan struct {
	Name           string // ledger key (migrationKey)
	MigrationID    string // nself_ops.migrations id (extractMigrationID)
	Checksum       string // SHA-256 hex of the on-disk file
	FilePath       string
	AlreadyApplied bool
}

// resolveBaselineFiles is pure: matches each requested name to its on-disk
// migration file. Errors on the first name with no match — baseline must
// never record a ledger row for a migration that isn't a real file, since
// recovering from that would need direct SQL, which the nSelf-First doctrine
// forbids.
func resolveBaselineFiles(files []string, names []string) ([]string, error) {
	byKey := make(map[string]string, len(files))
	for _, f := range files {
		byKey[migrationKey(f)] = f
	}
	resolved := make([]string, 0, len(names))
	for _, n := range names {
		f, ok := byKey[n]
		if !ok {
			return nil, fmt.Errorf("migration %q not found on disk", n)
		}
		resolved = append(resolved, f)
	}
	return resolved, nil
}

// buildBaselinePlans is pure aside from reading the already-resolved files:
// no database access, so --dry-run's plan and the real path's plan can never
// disagree about what would be recorded.
func buildBaselinePlans(resolved []string, applied map[string]time.Time) ([]BaselinePlan, error) {
	plans := make([]BaselinePlan, 0, len(resolved))
	for _, f := range resolved {
		name := migrationKey(f)
		data, err := os.ReadFile(f)
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", name, err)
		}
		checksum, err := checksumBytes(data)
		if err != nil {
			return nil, fmt.Errorf("checksum migration %s: %w", name, err)
		}
		_, isApplied := applied[name]
		plans = append(plans, BaselinePlan{
			Name:           name,
			MigrationID:    extractMigrationID(f),
			Checksum:       checksum,
			FilePath:       f,
			AlreadyApplied: isApplied,
		})
	}
	return plans, nil
}

// PlanBaseline resolves each requested migration name against dir (or the
// auto-detected migrations directory) and computes what baselining it would
// record, without writing anything. If the schema_versions ledger does not
// exist yet, every migration is reported as not-yet-applied rather than
// erroring — this keeps --dry-run usable before any migration has ever run.
func PlanBaseline(ctx context.Context, cfg *config.Config, dir string, names []string) ([]BaselinePlan, error) {
	if dir == "" {
		dir = migrationsDir(cfg, "")
	}
	files, err := scanMigrations(dir)
	if err != nil {
		return nil, err
	}
	resolved, err := resolveBaselineFiles(files, names)
	if err != nil {
		return nil, err
	}

	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		applied = map[string]time.Time{}
	}
	return buildBaselinePlans(resolved, applied)
}

// baselineTxSQL builds the ledger-only transaction BaselineMigration sends.
// Factored out so a test can assert it contains only the two ledger INSERTs
// and never the migration file's own SQL — that is what "record without
// executing" means concretely, and it is structurally guaranteed here: the
// migration's SQL body is never even a parameter to this function.
func baselineTxSQL(migrationID, name, checksum string) string {
	legacyRecord, opsRecord := migrationRecordSQL(migrationID, name, checksum)
	return "BEGIN;\n" + legacyRecord + "\n" + opsRecord + "\nCOMMIT;\n"
}

// BaselineMigration records a single on-disk migration as applied WITHOUT
// executing its SQL: it creates schema_versions/nself_ops.migrations if
// absent, then writes the same two ledger INSERTs 'db migrate up' writes
// after successfully running a migration — skipping straight to the record
// step. Errors if the migration is already recorded (callers filter those
// out via BaselinePlan.AlreadyApplied first; this is a defensive re-check).
func BaselineMigration(ctx context.Context, cfg *config.Config, filePath string) error {
	if err := ensureSchemaVersions(ctx, cfg); err != nil {
		return fmt.Errorf("ensure schema_versions: %w", err)
	}
	if err := ensureMigrationsTable(ctx, cfg); err != nil {
		return fmt.Errorf("ensure migrations table: %w", err)
	}

	name := migrationKey(filePath)
	if err := validateMigrationName(name); err != nil {
		return err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}
	checksum, err := checksumBytes(data)
	if err != nil {
		return fmt.Errorf("checksum migration %s: %w", name, err)
	}
	if !sha256HexRegex.MatchString(checksum) {
		return fmt.Errorf("unexpected checksum format for %s", name)
	}

	applied, err := appliedMigrations(ctx, cfg)
	if err != nil {
		return fmt.Errorf("check applied migrations: %w", err)
	}
	if _, ok := applied[name]; ok {
		return fmt.Errorf("migration %s is already recorded as applied", name)
	}

	migrationID := extractMigrationID(filePath)
	if err := validateMigrationName(migrationID); err != nil {
		return fmt.Errorf("migration id from %s: %w", name, err)
	}

	txSQL := baselineTxSQL(migrationID, name, checksum)
	if err := pipeSQLToContainer(ctx, cfg, txSQL); err != nil {
		return fmt.Errorf("baseline %s: %w", name, err)
	}
	return nil
}
