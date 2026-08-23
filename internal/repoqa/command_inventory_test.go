package repoqa

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// inventoryEntry mirrors the top-level shape emitted by tools/cmdinventory.
type inventoryEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// TestCommandInventoryIsCurrent fails when .github/command-inventory.json no
// longer matches the cobra tree.
//
// Before CLI-R06 every published command count was hand-typed and every one of
// them was wrong: SPORT F02 and the PRI both claimed 84 while the binary
// registered 92. Regenerating from the tree makes the drift impossible; this
// test makes forgetting to regenerate impossible too.
func TestCommandInventoryIsCurrent(t *testing.T) {
	root := repoRoot(t)
	committedPath := filepath.Join(root, ".github", "command-inventory.json")

	committed, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatalf("read committed inventory: %v (run `make cmd-inventory`)", err)
	}

	gen := exec.Command("go", "run", "-mod=vendor", "./tools/cmdinventory", "-format", "json", "-depth", "2")
	gen.Dir = root
	gen.Env = append(os.Environ(), "CGO_ENABLED=0")
	live, err := gen.Output()
	if err != nil {
		t.Fatalf("regenerate inventory: %v", err)
	}

	if string(committed) != string(live) {
		t.Fatalf("command inventory is stale — run `make cmd-inventory` and commit the result.\n"+
			"committed %d bytes, regenerated %d bytes", len(committed), len(live))
	}
}

// TestPublishedCommandCountMatchesBinary checks the human-readable count in the
// generated wiki index against the machine-readable inventory, so the two
// generated artifacts can never disagree with each other.
func TestPublishedCommandCountMatchesBinary(t *testing.T) {
	root := repoRoot(t)

	data, err := os.ReadFile(filepath.Join(root, ".github", "command-inventory.json"))
	if err != nil {
		t.Fatalf("read inventory: %v", err)
	}
	var entries []inventoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse inventory: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("inventory is empty")
	}

	wiki, err := os.ReadFile(filepath.Join(root, ".github", "wiki", "COMMANDS.md"))
	if err != nil {
		t.Fatalf("read wiki index: %v", err)
	}

	want := "**Total top-level commands: " + strconv.Itoa(len(entries)) + "**"
	if !strings.Contains(string(wiki), want) {
		t.Fatalf(".github/wiki/COMMANDS.md does not state %q — run `make cmd-inventory`", want)
	}

	// Every inventory entry must have a row in the index.
	for _, e := range entries {
		if !strings.Contains(string(wiki), "`"+e.Path+"`") {
			t.Errorf("command %q is missing from the generated wiki index", e.Path)
		}
	}
}
