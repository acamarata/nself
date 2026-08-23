package repoqa

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// TestParityMatrixIsCurrent fails when .github/surface-parity.{md,json} no
// longer match tools/parity's output (CLI-R17), the same drift guard
// TestCommandInventoryIsCurrent applies to the command inventory. The matrix
// is built from that same committed inventory, so this test only asserts the
// matrix itself is current — it does not re-derive the command list.
func TestParityMatrixIsCurrent(t *testing.T) {
	root := repoRoot(t)

	gen := exec.Command("go", "run", "-mod=vendor", "./tools/parity", "-check")
	gen.Dir = root
	gen.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := gen.CombinedOutput()
	if err != nil {
		t.Fatalf("parity matrix is stale — run `make parity` and commit the result:\n%s", out)
	}
}

// TestParityMatrixCommandsHaveWikiPages is a direct content assertion on the
// committed matrix, independent of the -check regeneration above: every
// top-level command it lists must be marked wiki_page: true. Docs are the
// minimum bar (CLI-R17) — this is the hard gate; MCP tool coverage, env var
// docs, and OpenAPI routes are informational only (see .github/surface-parity.md
// header for why).
func TestParityMatrixCommandsHaveWikiPages(t *testing.T) {
	root := repoRoot(t)

	data, err := os.ReadFile(filepath.Join(root, ".github", "surface-parity.json"))
	if err != nil {
		t.Fatalf("read surface-parity.json: %v (run `make parity`)", err)
	}

	var doc struct {
		Rows []struct {
			Path     string `json:"path"`
			WikiPage bool   `json:"wiki_page"`
		} `json:"rows"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("parse surface-parity.json: %v", err)
	}
	if len(doc.Rows) == 0 {
		t.Fatal("surface-parity.json has no rows")
	}
	for _, r := range doc.Rows {
		if !r.WikiPage {
			t.Errorf("%s has no wiki page (wiki_page: false in surface-parity.json)", r.Path)
		}
	}
}
