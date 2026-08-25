package repoqa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/nself-org/cli/internal/plugin"
)

// TestEveryShippedPluginManifestParses reads every real plugin.json in the
// sibling plugin repos and asserts the CLI can parse it.
//
// Two fields have now shipped with a type that described one of the shapes real
// manifests use rather than all of them: `webhooks` and `envVars`. Both failures
// are silent in the worst way — a manifest that will not parse is discarded, so
// the plugin simply disappears from `nself plugin list` and from everything else
// that enumerates what is installed, with no error printed anywhere.
//
// Unit tests over hand-written JSON did not catch either one, because the
// hand-written JSON was written from the struct. This reads what is actually
// published instead.
//
// Skipped when the sibling repos are not checked out, so a standalone clone of
// the CLI still passes.
func TestEveryShippedPluginManifestParses(t *testing.T) {
	siblings := filepath.Dir(repoRoot(t))

	var manifests []string
	for _, dir := range []string{
		filepath.Join(siblings, "plugins", "free"),
		filepath.Join(siblings, "plugins-pro", "paid"),
	} {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // repo not checked out next to this one
		}
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			p := filepath.Join(dir, e.Name(), "plugin.json")
			if _, err := os.Stat(p); err == nil {
				manifests = append(manifests, p)
			}
		}
	}

	if len(manifests) == 0 {
		t.Skip("no sibling plugin repos checked out")
	}

	failed := 0
	for _, p := range manifests {
		data, err := os.ReadFile(p)
		if err != nil {
			t.Errorf("%s: %v", p, err)
			failed++
			continue
		}
		var m plugin.PluginManifest
		if err := json.Unmarshal(data, &m); err != nil {
			rel, _ := filepath.Rel(siblings, p)
			t.Errorf("%s does not parse, so this plugin is invisible to the CLI: %v", rel, err)
			failed++
		}
	}
	t.Logf("checked %d shipped manifests, %d unparseable", len(manifests), failed)
}
