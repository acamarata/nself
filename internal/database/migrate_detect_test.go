package database

import (
	"reflect"
	"testing"
)

func TestExtractCreatedObjects(t *testing.T) {
	sql := `
CREATE SCHEMA IF NOT EXISTS app;
CREATE TABLE app.tasks (id uuid PRIMARY KEY);
CREATE TABLE IF NOT EXISTS app.projects (id uuid PRIMARY KEY);
CREATE INDEX idx_tasks_project ON app.tasks (project_id);
CREATE TYPE app.task_status AS ENUM ('open', 'done');
CREATE MATERIALIZED VIEW app.task_counts AS SELECT 1;
CREATE VIEW app.active_tasks AS SELECT 1;
CREATE SEQUENCE app.task_seq;
`
	objs := ExtractCreatedObjects(sql)

	want := map[ObjectKind]int{
		ObjectTable:      2,
		ObjectIndex:      1,
		ObjectType:       1,
		ObjectMatView:    1,
		ObjectView:       1,
		ObjectSequence:   1,
		ObjectSchemaKind: 1,
	}
	got := map[ObjectKind]int{}
	for _, o := range objs {
		got[o.Kind]++
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("object kind counts = %+v, want %+v", got, want)
	}
}

func TestExtractCreatedObjects_NoFalsePositiveOnMaterializedView(t *testing.T) {
	objs := ExtractCreatedObjects("CREATE MATERIALIZED VIEW app.counts AS SELECT 1;")
	views, matViews := 0, 0
	for _, o := range objs {
		switch o.Kind {
		case ObjectView:
			views++
		case ObjectMatView:
			matViews++
		}
	}
	if matViews != 1 || views != 0 {
		t.Fatalf("matViews=%d views=%d, want matViews=1 views=0 (no double count)", matViews, views)
	}
}

func TestClassifyByPresence(t *testing.T) {
	tbl := ObjectRef{Kind: ObjectTable, Name: "app.tasks"}
	idx := ObjectRef{Kind: ObjectIndex, Name: "idx_tasks"}

	tests := []struct {
		name     string
		objects  []ObjectRef
		existing map[string]bool
		want     DetectClass
	}{
		{"no objects is unknown", nil, map[string]bool{}, DetectUnknown},
		{"none present is apply", []ObjectRef{tbl, idx}, map[string]bool{}, DetectApply},
		{"all present is baseline", []ObjectRef{tbl, idx}, map[string]bool{tbl.Key(): true, idx.Key(): true}, DetectBaseline},
		{"partial is conflict", []ObjectRef{tbl, idx}, map[string]bool{tbl.Key(): true}, DetectConflict},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			class, present, missing := ClassifyByPresence(tt.objects, tt.existing)
			if class != tt.want {
				t.Fatalf("class = %s, want %s", class, tt.want)
			}
			if len(present)+len(missing) != len(tt.objects) {
				t.Fatalf("present(%d)+missing(%d) != objects(%d)", len(present), len(missing), len(tt.objects))
			}
		})
	}
}

func TestClassifyByPresence_NeverAutoResolvesConflict(t *testing.T) {
	objects := []ObjectRef{
		{Kind: ObjectTable, Name: "a"},
		{Kind: ObjectTable, Name: "b"},
	}
	existing := map[string]bool{objects[0].Key(): true} // only "a" exists
	class, present, missing := ClassifyByPresence(objects, existing)
	if class != DetectConflict {
		t.Fatalf("class = %s, want CONFLICT", class)
	}
	if len(present) != 1 || len(missing) != 1 {
		t.Fatalf("present=%d missing=%d, want 1 and 1", len(present), len(missing))
	}
}

func TestExistsExprFor(t *testing.T) {
	cases := []struct {
		o    ObjectRef
		want string
	}{
		{ObjectRef{Kind: ObjectSchemaKind, Name: "app"}, "EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'app')"},
		{ObjectRef{Kind: ObjectType, Name: "app.status"}, "to_regtype('app.status') IS NOT NULL"},
		{ObjectRef{Kind: ObjectTable, Name: "app.tasks"}, "to_regclass('app.tasks') IS NOT NULL"},
	}
	for _, c := range cases {
		if got := existsExprFor(c.o); got != c.want {
			t.Errorf("existsExprFor(%+v) = %q, want %q", c.o, got, c.want)
		}
	}
}

func TestSQLLiteral_EscapesQuotes(t *testing.T) {
	if got, want := sqlLiteral("o'brien"), "'o''brien'"; got != want {
		t.Errorf("sqlLiteral = %q, want %q", got, want)
	}
}
