package database

// Purpose: detects Hasura METADATA drift (table tracking + permissions +
// relationships) between a project's committed hasura/metadata/** files and
// a live Hasura instance. This is distinct from ScanSchemaDrift
// (schema_drift.go), which only checks np_* table COLUMN conventions.
// Inputs: a context, *config.Config, and the project directory containing
// hasura/metadata/**.
// Outputs: []MetadataDriftFinding naming the exact table/role/field that
// differs, or an error.
// Constraints: ported from ntask/backend/scripts/metadata-diff.sh's proven
// design — read-only, never mutates. Two footguns closed here on purpose:
//  1. Hasura omits keys that hold their default value (e.g.
//     allow_aggregations: false), while repo YAML often states them
//     explicitly. Comparing raw values reports phantom differences that get
//     the tool ignored, which is exactly how the real drift this ticket
//     responds to (msg-2026-08-22) stayed invisible. See normalizePermission.
//  2. hasura-auth creates tables in the `auth` schema (refresh_tokens,
//     roles, user_providers, ...) that legitimately exist live and are never
//     declared in project metadata. Flagging them is the other class of
//     false positive that got the original tool ignored. See
//     hasuraAuthOwnedTables (metadata_types.go).
//
// Live metadata fetch (incl. the admin-secret-via-docker-inspect footgun
// fix) lives in metadata_live.go; shared types live in metadata_types.go.

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/nself-org/cli/internal/config"
)

// permissionDefaults are the values Hasura omits from export_metadata when a
// permission field holds its default. Both sides must be normalized to these
// before comparison or every export produces phantom drift.
var permissionDefaults = map[string]interface{}{
	"allow_aggregations": false,
	"computed_fields":    []interface{}{},
	"columns":            []interface{}{},
	"set":                map[string]interface{}{},
	"backend_only":       false,
	"filter":             map[string]interface{}{},
	"check":              map[string]interface{}{},
}

// normalizePermission fills in Hasura's default-omitted keys, drops the
// comment field (metadata-only, never functional), and sorts the columns
// list, so that semantically-identical permissions compare equal regardless
// of which side omitted a default.
func normalizePermission(p map[string]interface{}) map[string]interface{} {
	out := make(map[string]interface{}, len(p)+len(permissionDefaults))
	for k, v := range p {
		out[k] = v
	}
	for k, def := range permissionDefaults {
		if _, ok := out[k]; !ok {
			out[k] = def
		}
	}
	delete(out, "comment")
	if cols, ok := out["columns"].([]interface{}); ok {
		strs := make([]string, 0, len(cols))
		for _, c := range cols {
			strs = append(strs, fmt.Sprint(c))
		}
		sort.Strings(strs)
		colsOut := make([]interface{}, len(strs))
		for i, s := range strs {
			colsOut[i] = s
		}
		out["columns"] = colsOut
	}
	return out
}

// permissionEqual reports whether two permission maps are equal after
// normalization. json.Marshal sorts map keys, giving a stable comparison key.
func permissionEqual(a, b map[string]interface{}) bool {
	aj, _ := json.Marshal(normalizePermission(a))
	bj, _ := json.Marshal(normalizePermission(b))
	return string(aj) == string(bj)
}

// diffPermissionFields returns the sorted list of top-level keys that differ
// between two normalized permission maps, e.g. ["columns","filter"] — used to
// name the exact field that drifted.
func diffPermissionFields(live, repo map[string]interface{}) []string {
	nl, nr := normalizePermission(live), normalizePermission(repo)
	keys := map[string]bool{}
	for k := range nl {
		keys[k] = true
	}
	for k := range nr {
		keys[k] = true
	}
	var diffs []string
	for k := range keys {
		lj, _ := json.Marshal(nl[k])
		rj, _ := json.Marshal(nr[k])
		if string(lj) != string(rj) {
			diffs = append(diffs, k)
		}
	}
	sort.Strings(diffs)
	return diffs
}

// MetadataDriftFinding is one row of Hasura metadata drift: a specific
// table/role/kind combination where committed metadata and the live Hasura
// instance disagree.
type MetadataDriftFinding struct {
	Table  string // "schema.name"
	Role   string // permission role; empty for a table/relationship finding
	Kind   string // "table" | "relationship" | "insert" | "select" | "update" | "delete"
	Detail string
}

// ScanMetadataDrift compares committed hasura/metadata/** against a live
// Hasura instance and returns every finding. An empty result means clean.
func ScanMetadataDrift(ctx context.Context, cfg *config.Config, projectDir string) ([]MetadataDriftFinding, error) {
	repo, err := loadRepoTables(projectDir)
	if err != nil {
		return nil, err
	}
	live, err := fetchLiveTables(ctx, cfg)
	if err != nil {
		return nil, err
	}
	return diffMetadataTables(repo, live), nil
}

// diffMetadataTables is the pure comparison at the heart of ScanMetadataDrift,
// split out so it can be unit tested without a live Hasura instance.
func diffMetadataTables(repo, live map[string]hasuraTable) []MetadataDriftFinding {
	var findings []MetadataDriftFinding

	liveKeys := make([]string, 0, len(live))
	for k := range live {
		liveKeys = append(liveKeys, k)
	}
	sort.Strings(liveKeys)
	for _, key := range liveKeys {
		if _, ok := repo[key]; !ok && !hasuraAuthOwnedTables[key] {
			findings = append(findings, MetadataDriftFinding{
				Table: key, Kind: "table",
				Detail: "tracked live, not declared in repo metadata",
			})
		}
	}

	repoKeys := make([]string, 0, len(repo))
	for k := range repo {
		repoKeys = append(repoKeys, k)
	}
	sort.Strings(repoKeys)

	for _, key := range repoKeys {
		rt := repo[key]
		lt, exists := live[key]
		if !exists {
			findings = append(findings, MetadataDriftFinding{
				Table: key, Kind: "table",
				Detail: "declared in repo metadata, not tracked live",
			})
			continue
		}
		findings = append(findings, diffTableRelationships(key, lt, rt)...)
		findings = append(findings, diffTablePermissions(key, lt, rt)...)
	}

	return findings
}

func diffTableRelationships(key string, lt, rt hasuraTable) []MetadataDriftFinding {
	var findings []MetadataDriftFinding
	for _, relSet := range [][2][]hasuraRelationship{
		{lt.ObjectRelationships, rt.ObjectRelationships},
		{lt.ArrayRelationships, rt.ArrayRelationships},
	} {
		have := map[string]bool{}
		for _, r := range relSet[0] {
			have[r.Name] = true
		}
		for _, r := range relSet[1] {
			if !have[r.Name] {
				findings = append(findings, MetadataDriftFinding{
					Table: key, Kind: "relationship",
					Detail: fmt.Sprintf("relationship %q declared in repo, missing live", r.Name),
				})
			}
		}
	}
	return findings
}

func diffTablePermissions(key string, lt, rt hasuraTable) []MetadataDriftFinding {
	var findings []MetadataDriftFinding
	for _, pk := range permissionKinds {
		rperm := indexPermissionsByRole(rt.tablePermissions(pk.short))
		lperm := indexPermissionsByRole(lt.tablePermissions(pk.short))
		roles := make([]string, 0, len(rperm))
		for role := range rperm {
			roles = append(roles, role)
		}
		sort.Strings(roles)
		for _, role := range roles {
			rp := rperm[role]
			lp, ok := lperm[role]
			if !ok {
				findings = append(findings, MetadataDriftFinding{
					Table: key, Role: role, Kind: pk.short,
					Detail: fmt.Sprintf("%s permission for role %q declared in repo, missing live", pk.short, role),
				})
				continue
			}
			if !permissionEqual(lp, rp) {
				fields := diffPermissionFields(lp, rp)
				findings = append(findings, MetadataDriftFinding{
					Table: key, Role: role, Kind: pk.short,
					Detail: fmt.Sprintf("%s permission for role %q differs live vs repo: %s",
						pk.short, role, strings.Join(fields, ", ")),
				})
			}
		}
	}
	return findings
}

func indexPermissionsByRole(perms []hasuraPermission) map[string]map[string]interface{} {
	out := make(map[string]map[string]interface{}, len(perms))
	for _, p := range perms {
		out[p.Role] = p.Permission
	}
	return out
}
