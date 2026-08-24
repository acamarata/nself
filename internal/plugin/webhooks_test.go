package plugin

import (
	"encoding/json"
	"testing"
)

// TestWebhookNames_AcceptsBothManifestShapes is the regression guard for a bug
// that made plugins silently disappear.
//
// The field was typed []string, but real plugin.json files use an object as
// often as an array. Unmarshalling an object into []string fails, and
// ListInstalled skips any plugin whose manifest will not parse — so those
// plugins vanished from `nself plugin list` and `nself costs` with no error at
// all. Both shapes were present in plugins/free/ at the time of the fix.
func TestWebhookNames_AcceptsBothManifestShapes(t *testing.T) {
	cases := []struct {
		name string
		json string
		want []string
	}{
		{"array form", `["acquisition.subscribed","acquisition.new_release"]`,
			[]string{"acquisition.subscribed", "acquisition.new_release"}},
		{"object form", `{"progress.updated":"Update playback position","progress.cleared":"Clear it"}`,
			[]string{"progress.cleared", "progress.updated"}}, // sorted
		{"empty object", `{}`, []string{}},
		{"empty array", `[]`, []string{}},
		{"null", `null`, nil},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got WebhookNames
			if err := json.Unmarshal([]byte(tc.json), &got); err != nil {
				t.Fatalf("a shape real manifests use must never fail to parse: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
			for i := range tc.want {
				if got[i] != tc.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

// TestWebhookNames_RejectsNonsense keeps the leniency bounded: a shape no
// manifest uses should still be an error rather than silently empty.
func TestWebhookNames_RejectsNonsense(t *testing.T) {
	var got WebhookNames
	if err := json.Unmarshal([]byte(`42`), &got); err == nil {
		t.Error("a number should not parse as a webhook list")
	}
}

// TestManifestWithObjectWebhooksStillParses is the end-to-end shape of the bug:
// a whole manifest must survive, not just the field.
func TestManifestWithObjectWebhooksStillParses(t *testing.T) {
	raw := `{
	  "name": "content-progress",
	  "version": "1.0.0",
	  "webhooks": {"progress.updated": "Update playback position"}
	}`
	var m PluginManifest
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("manifest with object-shaped webhooks failed to parse — the plugin would vanish from plugin list: %v", err)
	}
	if m.Name != "content-progress" {
		t.Errorf("name = %q", m.Name)
	}
	if len(m.Webhooks) != 1 || m.Webhooks[0] != "progress.updated" {
		t.Errorf("webhooks = %v", m.Webhooks)
	}
}
