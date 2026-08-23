package commands

import (
	"sort"
	"strings"
	"testing"
)

// TestEveryCommandHasAGroup is the CLI-R10 gate. A command with no GroupID is
// rendered by cobra under "Additional Commands", which is exactly the flat wall
// grouping was added to remove — so a new command that forgets its group would
// silently undo the work for itself.
func TestEveryCommandHasAGroup(t *testing.T) {
	ApplyCommandGroups()

	var ungrouped []string
	for _, c := range RootCmd.Commands() {
		if c.Hidden || c.Name() == "help" || c.Name() == "completion" {
			continue
		}
		if c.GroupID == "" {
			ungrouped = append(ungrouped, c.Name())
		}
	}
	sort.Strings(ungrouped)

	if len(ungrouped) > 0 {
		t.Fatalf("%d command(s) have no group — add them to commandGroupAssignments "+
			"in groups.go:\n  %s", len(ungrouped), strings.Join(ungrouped, "\n  "))
	}
}

// TestEveryAssignedGroupExists stops a typo in the assignment table from
// producing a command cobra cannot render at all.
func TestEveryAssignedGroupExists(t *testing.T) {
	known := map[string]bool{}
	for _, g := range commandGroups {
		known[g.ID] = true
	}

	for name, id := range commandGroupAssignments {
		if !known[id] {
			t.Errorf("command %q is assigned to unknown group %q", name, id)
		}
	}
}

// TestNoStaleGroupAssignments keeps the table from accumulating entries for
// commands that no longer exist — which matters most during CLI-R11, when
// families move out to plugins and their rows must move with them.
func TestNoStaleGroupAssignments(t *testing.T) {
	registered := map[string]bool{}
	for _, c := range RootCmd.Commands() {
		registered[c.Name()] = true
	}

	var stale []string
	for name := range commandGroupAssignments {
		if !registered[name] {
			stale = append(stale, name)
		}
	}
	sort.Strings(stale)

	if len(stale) > 0 {
		t.Fatalf("commandGroupAssignments names %d command(s) that are not registered "+
			"— remove them:\n  %s", len(stale), strings.Join(stale, "\n  "))
	}
}

// TestEveryGroupIsUsed catches a group that has been emptied out, which would
// print an empty heading in help.
func TestEveryGroupIsUsed(t *testing.T) {
	used := map[string]bool{}
	for _, id := range commandGroupAssignments {
		used[id] = true
	}

	for _, g := range commandGroups {
		if !used[g.ID] {
			t.Errorf("group %q (%q) has no commands — remove it or assign to it", g.ID, g.Title)
		}
	}
}

// TestCoreGroupHoldsTheGoldenPath pins the promise the help text makes at the
// top of the page: init, build and start are the three commands a new user runs.
func TestCoreGroupHoldsTheGoldenPath(t *testing.T) {
	for _, name := range []string{"init", "build", "start"} {
		if got := commandGroupAssignments[name]; got != groupCore {
			t.Errorf("%q is in group %q; the golden path must be in %q", name, got, groupCore)
		}
	}
}
