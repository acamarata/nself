package database

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/errs"
)

func TestExtractAlteredObjects_BasicTarget(t *testing.T) {
	objs := ExtractAlteredObjects("ALTER TABLE licenses ADD COLUMN tier VARCHAR(20);")
	if len(objs) != 1 || objs[0].Kind != ObjectTable || objs[0].Name != "licenses" {
		t.Fatalf("ExtractAlteredObjects = %+v, want one table %q", objs, "licenses")
	}
}

func TestExtractAlteredObjects_SkipsGuardedIfExists(t *testing.T) {
	objs := ExtractAlteredObjects("ALTER TABLE IF EXISTS licenses ADD COLUMN tier VARCHAR(20);")
	if len(objs) != 0 {
		t.Fatalf("ExtractAlteredObjects on a guarded ALTER = %+v, want none (IF EXISTS cannot fail on a missing table)", objs)
	}
}

func TestExtractAlteredObjects_HandlesOnlyAndSchemaQualified(t *testing.T) {
	objs := ExtractAlteredObjects("ALTER TABLE ONLY app.tasks ADD COLUMN done boolean;")
	if len(objs) != 1 || objs[0].Name != "app.tasks" {
		t.Fatalf("ExtractAlteredObjects = %+v, want one table %q", objs, "app.tasks")
	}
}

func TestClassifyAlterPrerequisites_SatisfiedByLiveSchema(t *testing.T) {
	files := []parsedMigration{
		{Name: "009_licensing_tiers.sql", Altered: []ObjectRef{{Kind: ObjectTable, Name: "licenses"}}},
	}
	existing := map[string]bool{ObjectRef{Kind: ObjectTable, Name: "licenses"}.Key(): true}

	missing := classifyAlterPrerequisites(files, existing)
	if len(missing) != 0 {
		t.Fatalf("classifyAlterPrerequisites = %+v, want none (table already exists live)", missing)
	}
}

func TestClassifyAlterPrerequisites_SatisfiedByEarlierFileInBatch(t *testing.T) {
	files := []parsedMigration{
		{Name: "001_create_widgets.sql", Created: []ObjectRef{{Kind: ObjectTable, Name: "widgets"}}},
		{Name: "002_alter_widgets.sql", Altered: []ObjectRef{{Kind: ObjectTable, Name: "widgets"}}},
	}
	missing := classifyAlterPrerequisites(files, map[string]bool{})
	if len(missing) != 0 {
		t.Fatalf("classifyAlterPrerequisites = %+v, want none (widgets is created earlier in the same batch)", missing)
	}
}

// TestClassifyAlterPrerequisites_SatisfiedByOwnFile is the regression this
// fix locks: a single migration file that both CREATEs a table and then
// ALTERs the same table (the common "CREATE ...; ALTER ... ADD CONSTRAINT
// ...;" authoring shape) must never be flagged as missing a prerequisite —
// the table it needs is created earlier in the very same file.
func TestClassifyAlterPrerequisites_SatisfiedByOwnFile(t *testing.T) {
	files := []parsedMigration{
		{
			Name:    "20260901_bad_constraint.sql",
			Created: []ObjectRef{{Kind: ObjectTable, Name: "accounts"}},
			Altered: []ObjectRef{{Kind: ObjectTable, Name: "accounts"}},
		},
	}
	missing := classifyAlterPrerequisites(files, map[string]bool{})
	if len(missing) != 0 {
		t.Fatalf("classifyAlterPrerequisites = %+v, want none (accounts is created earlier in the same file)", missing)
	}
}

func TestClassifyAlterPrerequisites_RefusesWhenNeitherLiveNorInBatch(t *testing.T) {
	files := []parsedMigration{
		{Name: "009_licensing_tiers.sql", Altered: []ObjectRef{{Kind: ObjectTable, Name: "licenses"}}},
	}
	missing := classifyAlterPrerequisites(files, map[string]bool{})
	if len(missing) != 1 {
		t.Fatalf("classifyAlterPrerequisites = %+v, want exactly one missing prerequisite", missing)
	}
	if missing[0].Object.Name != "licenses" || missing[0].MigrationID != "009_licensing_tiers.sql" {
		t.Fatalf("missing prerequisite = %+v, want licenses needed by 009_licensing_tiers.sql", missing[0])
	}
}

// TestClassifyAlterPrerequisites_LaterCreateDoesNotSatisfyEarlierAlter proves
// ordering matters: a CREATE TABLE that only appears in a LATER file must not
// satisfy an ALTER TABLE in an earlier one, since the earlier migration would
// run first and still hit a missing relation.
func TestClassifyAlterPrerequisites_LaterCreateDoesNotSatisfyEarlierAlter(t *testing.T) {
	files := []parsedMigration{
		{Name: "001_alter_widgets.sql", Altered: []ObjectRef{{Kind: ObjectTable, Name: "widgets"}}},
		{Name: "002_create_widgets.sql", Created: []ObjectRef{{Kind: ObjectTable, Name: "widgets"}}},
	}
	missing := classifyAlterPrerequisites(files, map[string]bool{})
	if len(missing) != 1 || missing[0].MigrationID != "001_alter_widgets.sql" {
		t.Fatalf("classifyAlterPrerequisites = %+v, want one missing prerequisite against 001_alter_widgets.sql", missing)
	}
}

func TestClassifyAlterPrerequisites_DedupesRepeatedTarget(t *testing.T) {
	files := []parsedMigration{
		{Name: "009_a.sql", Altered: []ObjectRef{{Kind: ObjectTable, Name: "licenses"}}},
		{Name: "010_b.sql", Altered: []ObjectRef{{Kind: ObjectTable, Name: "licenses"}}},
	}
	missing := classifyAlterPrerequisites(files, map[string]bool{})
	if len(missing) != 1 {
		t.Fatalf("classifyAlterPrerequisites = %+v, want a single deduped entry for the repeated target", missing)
	}
}

// TestClassifyAlterPrerequisites_EmptyBatchIsIdempotent covers the
// idempotency case at the pure-function layer: an empty pending batch (every
// migration already applied) must never report a missing prerequisite —
// exercised again end-to-end via checkAlterPrerequisites below.
func TestClassifyAlterPrerequisites_EmptyBatchIsIdempotent(t *testing.T) {
	missing := classifyAlterPrerequisites(nil, map[string]bool{})
	if len(missing) != 0 {
		t.Fatalf("classifyAlterPrerequisites(nil, ...) = %+v, want none", missing)
	}
}

// TestCheckAlterPrerequisites_SkipsEntirelyWhenNothingPending is the
// idempotency guarantee at the checkAlterPrerequisites layer: once every
// migration in a batch is applied, the caller passes an empty pending slice
// and this must return immediately without touching the database (a nil
// *config.Config would panic if it tried).
func TestCheckAlterPrerequisites_SkipsEntirelyWhenNothingPending(t *testing.T) {
	missing, err := checkAlterPrerequisites(context.Background(), nil, nil)
	if err != nil {
		t.Fatalf("checkAlterPrerequisites(nil files) error = %v, want nil", err)
	}
	if len(missing) != 0 {
		t.Fatalf("checkAlterPrerequisites(nil files) = %+v, want none", missing)
	}
}

func TestHasuraMigrationsDirIfPresent_NestedLayout(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(filepath.Join("hasura", "migrations", "default"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	// hasuraMigrationsDirIfPresent returns its candidate list's literal
	// forward-slash form (matching migrationsDir's own candidates in
	// migrate_ledger.go), not a filepath.Join result — os.Stat accepts "/"
	// on Windows too, but filepath.Join there would produce "\"-separated
	// segments that never equal the literal the function actually returns.
	const want = "hasura/migrations/default"
	got, ok := hasuraMigrationsDirIfPresent()
	if !ok || got != want {
		t.Fatalf("hasuraMigrationsDirIfPresent() = (%q, %v), want (%q, true)", got, ok, want)
	}
}

func TestHasuraMigrationsDirIfPresent_AbsentReturnsFalse(t *testing.T) {
	t.Chdir(t.TempDir())
	if _, ok := hasuraMigrationsDirIfPresent(); ok {
		t.Fatalf("hasuraMigrationsDirIfPresent() = true in an empty directory, want false")
	}
}

// TestPrerequisiteError_NamesObjectAndHasuraCommand is the refuse-first
// contract's message check: it must name the missing table, that it comes
// from Hasura's migrations, the specific Hasura migration that creates it
// when one is found on disk, and the exact remediation command.
func TestPrerequisiteError_NamesObjectAndHasuraCommand(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	hasuraMigDir := filepath.Join("hasura", "migrations", "default", "1706140802000_licenses_and_telemetry")
	if err := os.MkdirAll(hasuraMigDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(hasuraMigDir, "up.sql"), []byte("CREATE TABLE licenses (id uuid PRIMARY KEY);"), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	err := prerequisiteError([]MissingPrerequisite{
		{Object: ObjectRef{Kind: ObjectTable, Name: "licenses"}, MigrationID: "009_licensing_tiers.sql"},
	})
	if err == nil {
		t.Fatal("prerequisiteError(...) = nil, want a non-nil refusal")
	}
	if !errors.Is(err, errs.ErrMigrationPrerequisiteMissing) {
		t.Fatalf("prerequisiteError(...) does not wrap ErrMigrationPrerequisiteMissing: %v", err)
	}

	msg := err.Error()
	for _, want := range []string{
		`"licenses"`,
		"009_licensing_tiers.sql",
		"1706140802000_licenses_and_telemetry",
		"hasura migrate apply --database-name default",
		"nself db migrate up",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("prerequisiteError message %q does not contain %q", msg, want)
		}
	}
}

// TestPrerequisiteError_NoHasuraDirFallsBackGenerically covers a project with
// no Hasura migrations at all (e.g. ntask's postgres/migrations layout):
// the refusal must still name the object without falsely claiming Hasura
// ownership.
func TestPrerequisiteError_NoHasuraDirFallsBackGenerically(t *testing.T) {
	t.Chdir(t.TempDir())

	err := prerequisiteError([]MissingPrerequisite{
		{Object: ObjectRef{Kind: ObjectTable, Name: "widgets"}, MigrationID: "002_alter_widgets.sql"},
	})
	msg := err.Error()
	if !strings.Contains(msg, `"widgets"`) || !strings.Contains(msg, "002_alter_widgets.sql") {
		t.Fatalf("prerequisiteError message %q does not name the missing object/migration", msg)
	}
	if strings.Contains(msg, "hasura migrate apply") {
		t.Fatalf("prerequisiteError message %q claims a Hasura fix command with no Hasura directory present", msg)
	}
}

// TestParseMigrationFile_MatchesExtractHelpers is a smoke test that
// parseMigrationFile wires ExtractCreatedObjects/ExtractAlteredObjects
// together correctly rather than diverging from them.
func TestParseMigrationFile_MatchesExtractHelpers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "20260101_widgets.sql")
	sql := "CREATE TABLE widgets (id serial primary key);\nALTER TABLE gadgets ADD COLUMN done boolean;"
	if err := os.WriteFile(path, []byte(sql), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	p, err := parseMigrationFile(path)
	if err != nil {
		t.Fatalf("parseMigrationFile: %v", err)
	}
	if p.Name != "20260101_widgets.sql" {
		t.Errorf("Name = %q, want 20260101_widgets.sql", p.Name)
	}
	if len(p.Created) != 1 || p.Created[0].Name != "widgets" {
		t.Errorf("Created = %+v, want one table %q", p.Created, "widgets")
	}
	if len(p.Altered) != 1 || p.Altered[0].Name != "gadgets" {
		t.Errorf("Altered = %+v, want one table %q", p.Altered, "gadgets")
	}
}
