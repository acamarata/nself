package plugin

import (
	"encoding/json"
	"testing"
)

// TestEnvVarListAcceptsEveryShapeInUse pins the fix for a bug that made every
// CLI-R11 plugin invisible.
//
// The field was typed []EnvVar, but the manifest template every extraction was
// copied from writes an object with required/optional name lists. Unmarshalling
// that into a slice fails, and a manifest that fails to parse is discarded —
// so the plugin vanished from `nself plugin list` and from everything else that
// enumerates installed plugins, with no error anywhere.
func TestEnvVarListAcceptsEveryShapeInUse(t *testing.T) {
	tests := []struct {
		name  string
		json  string
		want  []string
		reqd  map[string]bool
		fails bool
	}{
		{
			name: "grouped form — what the plugin template writes, and what broke",
			json: `{"required":["A"],"optional":["B"]}`,
			want: []string{"A", "B"},
			reqd: map[string]bool{"A": true, "B": false},
		},
		{
			name: "canonical array of declarations",
			json: `[{"name":"A","required":true},{"name":"B"}]`,
			want: []string{"A", "B"},
			reqd: map[string]bool{"A": true, "B": false},
		},
		{
			name: "shorthand array of names",
			json: `["A","B"]`,
			want: []string{"A", "B"},
			reqd: map[string]bool{"A": false, "B": false},
		},
		{
			name: "empty groups",
			json: `{"required":[],"optional":[]}`,
			want: []string{},
		},
		{
			name: "null",
			json: `null`,
			want: []string{},
		},
		{
			name:  "genuine nonsense is still an error",
			json:  `42`,
			fails: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got EnvVarList
			err := json.Unmarshal([]byte(tt.json), &got)
			if tt.fails {
				if err == nil {
					t.Fatal("expected an error for a shape no manifest uses")
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshal: %v", err)
			}
			names := got.Names()
			if len(names) != len(tt.want) {
				t.Fatalf("names = %v, want %v", names, tt.want)
			}
			for i, n := range names {
				if n != tt.want[i] {
					t.Errorf("names[%d] = %q, want %q", i, n, tt.want[i])
				}
			}
			for _, v := range got {
				if want, ok := tt.reqd[v.Name]; ok && v.Required != want {
					t.Errorf("%s Required = %v, want %v", v.Name, v.Required, want)
				}
			}
		})
	}
}

// TestManifestWithGroupedEnvVarsParses is the end the user actually sees: a
// whole plugin.json in the shape every extracted plugin ships.
func TestManifestWithGroupedEnvVarsParses(t *testing.T) {
	raw := `{
	  "name": "infra",
	  "version": "1.0.0",
	  "pluginType": "cli",
	  "binaryName": "nself-infra",
	  "envVars": {"required": [], "optional": ["HETZNER_NSELF_TOKEN", "HCLOUD_TOKEN"]},
	  "webhooks": {},
	  "tables": []
	}`
	var m PluginManifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("a real plugin manifest failed to parse, which makes the plugin invisible: %v", err)
	}
	if got := cliBinaryName("infra", &m); got != "nself-infra" {
		t.Errorf("cliBinaryName = %q, want nself-infra", got)
	}
	if len(m.EnvVars) != 2 {
		t.Errorf("EnvVars = %v, want 2 entries", m.EnvVars)
	}
}
