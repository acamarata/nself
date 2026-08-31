package database

import "testing"

func TestBuildReconcilePlanPure_TracksUntrackedRepoTable(t *testing.T) {
	repo := map[string]hasuraTable{
		"public.np_widgets": {Table: hasuraTableRef{Schema: "public", Name: "np_widgets"}},
	}
	live := map[string]hasuraTable{}

	plan := buildReconcilePlanPure(repo, live)
	if len(plan.BulkOps) != 1 || plan.BulkOps[0]["type"] != "pg_track_table" {
		t.Fatalf("expected one pg_track_table op, got %+v", plan.BulkOps)
	}
}

func TestBuildReconcilePlanPure_CreatesMissingRelationshipBeforePermission(t *testing.T) {
	repo := map[string]hasuraTable{
		"public.np_activity": {
			Table: hasuraTableRef{Schema: "public", Name: "np_activity"},
			ObjectRelationships: []hasuraRelationship{
				{Name: "todo", Using: map[string]interface{}{"foreign_key_constraint_on": "todo_id"}},
			},
			SelectPermissions: []hasuraPermission{
				{Role: "user", Permission: map[string]interface{}{"columns": []interface{}{"id"}}},
			},
		},
	}
	live := map[string]hasuraTable{
		"public.np_activity": {Table: hasuraTableRef{Schema: "public", Name: "np_activity"}},
	}

	plan := buildReconcilePlanPure(repo, live)
	if len(plan.BulkOps) != 2 {
		t.Fatalf("expected relationship + permission create ops, got %+v", plan.BulkOps)
	}
	if plan.BulkOps[0]["type"] != "pg_create_object_relationship" {
		t.Fatalf("relationship must be created before permission, got order %+v", plan.BulkOps)
	}
	if plan.BulkOps[1]["type"] != "pg_create_select_permission" {
		t.Fatalf("expected select permission create second, got %+v", plan.BulkOps[1])
	}
}

func TestBuildReconcilePlanPure_ReplacesDriftedPermissionAsDropThenCreate(t *testing.T) {
	repo := map[string]hasuraTable{
		"public.np_exports": {
			Table: hasuraTableRef{Schema: "public", Name: "np_exports"},
			SelectPermissions: []hasuraPermission{
				{Role: "user", Permission: map[string]interface{}{"columns": []interface{}{"id", "owner_id"}}},
			},
		},
	}
	live := map[string]hasuraTable{
		"public.np_exports": {
			Table: hasuraTableRef{Schema: "public", Name: "np_exports"},
			SelectPermissions: []hasuraPermission{
				{Role: "user", Permission: map[string]interface{}{"columns": []interface{}{"id"}}},
			},
		},
	}

	plan := buildReconcilePlanPure(repo, live)
	if len(plan.BulkOps) != 2 {
		t.Fatalf("expected drop then create, got %+v", plan.BulkOps)
	}
	if plan.BulkOps[0]["type"] != "pg_drop_select_permission" {
		t.Fatalf("expected drop first, got %+v", plan.BulkOps[0])
	}
	if plan.BulkOps[1]["type"] != "pg_create_select_permission" {
		t.Fatalf("expected create second, got %+v", plan.BulkOps[1])
	}
}

func TestBuildReconcilePlanPure_NoOpWhenIdentical(t *testing.T) {
	table := hasuraTable{
		Table: hasuraTableRef{Schema: "public", Name: "np_widgets"},
		SelectPermissions: []hasuraPermission{
			{Role: "user", Permission: map[string]interface{}{"columns": []interface{}{"id"}}},
		},
	}
	repo := map[string]hasuraTable{"public.np_widgets": table}
	live := map[string]hasuraTable{"public.np_widgets": table}

	plan := buildReconcilePlanPure(repo, live)
	if len(plan.BulkOps) != 0 || len(plan.Changes) != 0 {
		t.Fatalf("expected a no-op plan for identical metadata, got %+v", plan)
	}
}

func TestBuildReconcilePlanPure_NeverTouchesLiveOnlyPermission(t *testing.T) {
	// A role present only live (not declared in repo) must be left alone —
	// reconcile only ever adds/replaces what the repo declares, per
	// ntask/backend/scripts/metadata-reconcile.sh's targeted design.
	repo := map[string]hasuraTable{
		"public.np_widgets": {Table: hasuraTableRef{Schema: "public", Name: "np_widgets"}},
	}
	live := map[string]hasuraTable{
		"public.np_widgets": {
			Table: hasuraTableRef{Schema: "public", Name: "np_widgets"},
			SelectPermissions: []hasuraPermission{
				{Role: "legacy_role", Permission: map[string]interface{}{"columns": []interface{}{"id"}}},
			},
		},
	}

	plan := buildReconcilePlanPure(repo, live)
	if len(plan.BulkOps) != 0 {
		t.Fatalf("expected zero ops (live-only permission must not be touched), got %+v", plan.BulkOps)
	}
}
