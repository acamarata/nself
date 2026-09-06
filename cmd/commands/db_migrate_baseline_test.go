package commands

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
	"github.com/nself-org/cli/internal/database"
)

var errBaselineWriteFailed = errors.New("write failed")

func TestBaselineConfirmed_YesFlagSkipsPrompt(t *testing.T) {
	out := &bytes.Buffer{}
	confirmed, err := baselineConfirmed(true, strings.NewReader(""), out, "prompt: ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confirmed {
		t.Fatal("expected confirmed=true when --yes is set")
	}
	if out.Len() != 0 {
		t.Errorf("expected no prompt printed when --yes is set, got %q", out.String())
	}
}

func TestBaselineConfirmed_InteractiveYes(t *testing.T) {
	out := &bytes.Buffer{}
	confirmed, err := baselineConfirmed(false, strings.NewReader("yes\n"), out, "prompt: ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !confirmed {
		t.Fatal(`expected confirmed=true for typed "yes"`)
	}
	if !strings.Contains(out.String(), "prompt: ") {
		t.Error("expected the prompt to be printed")
	}
}

func TestBaselineConfirmed_InteractiveRefusal(t *testing.T) {
	out := &bytes.Buffer{}
	for _, input := range []string{"no\n", "\n", "Y\n", "sure\n"} {
		confirmed, err := baselineConfirmed(false, strings.NewReader(input), out, "prompt: ")
		if err != nil {
			t.Fatalf("unexpected error for input %q: %v", input, err)
		}
		if confirmed {
			t.Errorf("input %q must not confirm (only exact \"yes\" may)", input)
		}
	}
}

func TestRunBaselinePlans_DryRunWritesNothing(t *testing.T) {
	calls := 0
	fakeWriter := func(_ context.Context, _ *config.Config, _ string) error {
		calls++
		return nil
	}
	plans := []database.BaselinePlan{
		{Name: "20260101_a.sql", MigrationID: "20260101_a", Checksum: "abc", FilePath: "/x/20260101_a.sql"},
		{Name: "20260102_b.sql", MigrationID: "20260102_b", Checksum: "def", FilePath: "/x/20260102_b.sql"},
	}
	out := &bytes.Buffer{}
	applied, err := runBaselinePlans(out, context.Background(), nil, plans, true, fakeWriter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if calls != 0 {
		t.Fatalf("dry-run must never call the writer, got %d calls", calls)
	}
	if applied != 0 {
		t.Fatalf("dry-run applied count = %d, want 0", applied)
	}
	if !strings.Contains(out.String(), "would record: 20260101_a.sql") {
		t.Error("expected the plan to be printed even in dry-run")
	}
}

func TestRunBaselinePlans_ConfirmedRunWritesOnlyPending(t *testing.T) {
	var calledWith []string
	fakeWriter := func(_ context.Context, _ *config.Config, filePath string) error {
		calledWith = append(calledWith, filePath)
		return nil
	}
	plans := []database.BaselinePlan{
		{Name: "20260101_a.sql", AlreadyApplied: true, FilePath: "/x/20260101_a.sql"},
		{Name: "20260102_b.sql", MigrationID: "20260102_b", Checksum: "def", FilePath: "/x/20260102_b.sql"},
	}
	out := &bytes.Buffer{}
	applied, err := runBaselinePlans(out, context.Background(), nil, plans, false, fakeWriter)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if applied != 1 {
		t.Fatalf("applied = %d, want 1 (only the non-applied plan)", applied)
	}
	if len(calledWith) != 1 || calledWith[0] != "/x/20260102_b.sql" {
		t.Fatalf("writer called with %v, want exactly the pending file", calledWith)
	}
	if !strings.Contains(out.String(), "already applied: 20260101_a.sql (skipped)") {
		t.Error("expected the already-applied plan to be reported as skipped")
	}
}

func TestRunBaselinePlans_WriterErrorPropagates(t *testing.T) {
	fakeWriter := func(_ context.Context, _ *config.Config, _ string) error {
		return errBaselineWriteFailed
	}
	plans := []database.BaselinePlan{{Name: "x.sql", FilePath: "/x/x.sql"}}
	_, err := runBaselinePlans(&bytes.Buffer{}, context.Background(), nil, plans, false, fakeWriter)
	if err == nil {
		t.Fatal("expected the writer's error to propagate")
	}
}

func TestPendingBaselineCount(t *testing.T) {
	plans := []database.BaselinePlan{
		{AlreadyApplied: true},
		{AlreadyApplied: false},
		{AlreadyApplied: false},
	}
	if n := pendingBaselineCount(plans); n != 2 {
		t.Fatalf("pendingBaselineCount = %d, want 2", n)
	}
}
