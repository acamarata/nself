package database

import "testing"

func TestNormalizePermission_FillsDefaultsAndSortsColumns(t *testing.T) {
	live := map[string]interface{}{
		"columns": []interface{}{"b", "a"},
		"filter":  map[string]interface{}{"owner_id": "X-Hasura-User-Id"},
	}
	// repo states allow_aggregations explicitly even though it's the default —
	// this must NOT be reported as drift.
	repo := map[string]interface{}{
		"columns":            []interface{}{"a", "b"},
		"filter":             map[string]interface{}{"owner_id": "X-Hasura-User-Id"},
		"allow_aggregations": false,
	}
	if !permissionEqual(live, repo) {
		t.Fatalf("expected permissions to be equal after normalization, got diff fields: %v",
			diffPermissionFields(live, repo))
	}
}

func TestPermissionEqual_DetectsRealDifference(t *testing.T) {
	live := map[string]interface{}{"columns": []interface{}{"a", "b"}}
	repo := map[string]interface{}{"columns": []interface{}{"a", "b", "c"}}
	if permissionEqual(live, repo) {
		t.Fatal("expected permissions to differ (repo has an extra column)")
	}
	fields := diffPermissionFields(live, repo)
	if len(fields) != 1 || fields[0] != "columns" {
		t.Fatalf("expected diff on [columns], got %v", fields)
	}
}

func TestDiffMetadataTables_IgnoresHasuraAuthOwnedTables(t *testing.T) {
	repo := map[string]hasuraTable{}
	live := map[string]hasuraTable{
		"auth.refresh_tokens": {Table: hasuraTableRef{Schema: "auth", Name: "refresh_tokens"}},
	}
	findings := diffMetadataTables(repo, live)
	if len(findings) != 0 {
		t.Fatalf("expected zero false-positive findings for hasura-auth-owned tables, got %+v", findings)
	}
}

func TestDiffMetadataTables_FlagsUnexpectedLiveOnlyTable(t *testing.T) {
	repo := map[string]hasuraTable{}
	live := map[string]hasuraTable{
		"public.np_widgets": {Table: hasuraTableRef{Schema: "public", Name: "np_widgets"}},
	}
	findings := diffMetadataTables(repo, live)
	if len(findings) != 1 || findings[0].Kind != "table" {
		t.Fatalf("expected one table finding for an untracked-in-repo live table, got %+v", findings)
	}
}

func TestDiffMetadataTables_FlagsUntrackedRepoTable(t *testing.T) {
	repo := map[string]hasuraTable{
		"public.np_widgets": {Table: hasuraTableRef{Schema: "public", Name: "np_widgets"}},
	}
	live := map[string]hasuraTable{}
	findings := diffMetadataTables(repo, live)
	if len(findings) != 1 || findings[0].Detail == "" {
		t.Fatalf("expected one table finding for a repo table not tracked live, got %+v", findings)
	}
}

func TestDiffMetadataTables_FlagsMissingLivePermission(t *testing.T) {
	repo := map[string]hasuraTable{
		"public.np_exports": {
			Table: hasuraTableRef{Schema: "public", Name: "np_exports"},
			SelectPermissions: []hasuraPermission{
				{Role: "user", Permission: map[string]interface{}{
					"columns": []interface{}{"id", "storage_key"},
					"filter":  map[string]interface{}{"owner_id": "X-Hasura-User-Id"},
				}},
			},
		},
	}
	live := map[string]hasuraTable{
		"public.np_exports": {
			Table: hasuraTableRef{Schema: "public", Name: "np_exports"},
			// select permission absent entirely on live — the exact class of
			// gap the 2026-08-22 report proved exploitable.
		},
	}
	findings := diffMetadataTables(repo, live)
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding, got %+v", findings)
	}
	f := findings[0]
	if f.Table != "public.np_exports" || f.Role != "user" || f.Kind != "select" {
		t.Fatalf("finding did not name the exact table/role/kind: %+v", f)
	}
}

func TestDiffMetadataTables_FlagsChangedColumnsPermission(t *testing.T) {
	repo := map[string]hasuraTable{
		"public.np_exports": {
			Table: hasuraTableRef{Schema: "public", Name: "np_exports"},
			SelectPermissions: []hasuraPermission{
				{Role: "user", Permission: map[string]interface{}{
					"columns": []interface{}{"id", "storage_key", "owner_id"},
				}},
			},
		},
	}
	live := map[string]hasuraTable{
		"public.np_exports": {
			Table: hasuraTableRef{Schema: "public", Name: "np_exports"},
			SelectPermissions: []hasuraPermission{
				{Role: "user", Permission: map[string]interface{}{
					// live is missing "owner_id" — an intentionally-introduced
					// drift, per this ticket's acceptance criteria.
					"columns": []interface{}{"id", "storage_key"},
				}},
			},
		},
	}
	findings := diffMetadataTables(repo, live)
	if len(findings) != 1 {
		t.Fatalf("expected exactly one finding, got %+v", findings)
	}
	if findings[0].Detail == "" {
		t.Fatal("expected a non-empty detail naming the drifted field")
	}
}

func TestDiffMetadataTables_CleanIsEmpty(t *testing.T) {
	table := hasuraTable{
		Table: hasuraTableRef{Schema: "public", Name: "np_widgets"},
		SelectPermissions: []hasuraPermission{
			{Role: "user", Permission: map[string]interface{}{"columns": []interface{}{"id"}}},
		},
	}
	repo := map[string]hasuraTable{"public.np_widgets": table}
	live := map[string]hasuraTable{"public.np_widgets": table}
	if findings := diffMetadataTables(repo, live); len(findings) != 0 {
		t.Fatalf("expected zero findings for identical metadata, got %+v", findings)
	}
}
