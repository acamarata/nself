package repoqa

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// surfaceBudgetFile holds the number of top-level commands the core binary is
// currently allowed to register.
//
// CLI-R11 extracts roughly thirty command families out to plugins, taking the
// core from 85 towards the approved target of ~35. That cannot land in one
// change, and while it is in progress the easiest way to lose ground is for a
// new top-level command to be added without anyone noticing. This is the same
// ratchet CLI-R12 uses for file sizes: the number may fall, never rise.
//
// Raising it requires a deliberate edit and an explanation in the commit
// message. The approved core list lives in the DECISIONS section of
// .claude/tasks/cli-review-tickets-2026-08-23.md.
const surfaceBudgetFile = ".github/command-surface-budget.txt"

func readSurfaceBudget(t *testing.T, root string) int {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, surfaceBudgetFile))
	if err != nil {
		t.Fatalf("read %s: %v", surfaceBudgetFile, err)
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		n, err := strconv.Atoi(line)
		if err != nil {
			t.Fatalf("%s: %q is not an integer", surfaceBudgetFile, line)
		}
		return n
	}
	t.Fatalf("%s contains no budget number", surfaceBudgetFile)
	return 0
}

// topLevelCommandNames reads the generated inventory rather than importing
// cmd/commands, which would make this package depend on the whole CLI.
func topLevelCommandNames(t *testing.T, root string) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, ".github", "command-inventory.json"))
	if err != nil {
		t.Fatalf("read command inventory: %v (run `make cmd-inventory`)", err)
	}
	var entries []inventoryEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parse command inventory: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	sort.Strings(names)
	return names
}

// TestCommandSurfaceBudgetNotExceeded fails when a new top-level command lands
// while the core is supposed to be shrinking.
func TestCommandSurfaceBudgetNotExceeded(t *testing.T) {
	root := repoRoot(t)
	names := topLevelCommandNames(t, root)
	budget := readSurfaceBudget(t, root)

	if len(names) > budget {
		t.Fatalf("the core registers %d top-level commands but the budget is %d.\n"+
			"CLI-R11 is shrinking this surface towards ~35 — a new top-level command\n"+
			"needs to justify itself, or belongs in a plugin.\nCommands: %s",
			len(names), budget, strings.Join(names, " "))
	}
}

// TestCommandSurfaceBudgetIsTight fails when the budget is left above reality,
// which would quietly bank room for commands to creep back in.
func TestCommandSurfaceBudgetIsTight(t *testing.T) {
	root := repoRoot(t)
	names := topLevelCommandNames(t, root)
	budget := readSurfaceBudget(t, root)

	if len(names) < budget {
		t.Fatalf("budget is %d but the core registers %d top-level commands — "+
			"lower the number in %s to %d so the ratchet keeps holding",
			budget, len(names), surfaceBudgetFile, len(names))
	}
}

// TestGoldenPathCommandsAreInCore is the floor beneath the ratchet. Whatever
// else moves out to a plugin, a fresh install must be able to create, build and
// run a stack, diagnose it, and extend itself — with no plugin installed.
func TestGoldenPathCommandsAreInCore(t *testing.T) {
	root := repoRoot(t)
	have := map[string]bool{}
	for _, n := range topLevelCommandNames(t, root) {
		have[n] = true
	}

	mustKeep := []string{
		"init", "build", "start", "stop", "restart", "status", "logs", "urls",
		"doctor", "config", "env", "secrets", "db", "backup", "deploy",
		"plugin", "install", "version",
	}

	var missing []string
	for _, n := range mustKeep {
		if !have[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) > 0 {
		t.Fatalf("CLI-R11 extracted %d command(s) that must stay in core — "+
			"a fresh install cannot work without them: %s",
			len(missing), strings.Join(missing, " "))
	}
}
