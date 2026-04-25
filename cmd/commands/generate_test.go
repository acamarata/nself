package commands

import (
	"testing"
)

func TestParseGenerateLangs_All(t *testing.T) {
	langs, err := parseGenerateLangs("all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(langs) != len(generateLangs) {
		t.Errorf("expected %d langs, got %d", len(generateLangs), len(langs))
	}
}

func TestParseGenerateLangs_Subset(t *testing.T) {
	langs, err := parseGenerateLangs("typescript,python")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(langs) != 2 {
		t.Errorf("expected 2 langs, got %d", len(langs))
	}
	if langs[0] != "typescript" || langs[1] != "python" {
		t.Errorf("unexpected langs: %v", langs)
	}
}

func TestParseGenerateLangs_Invalid(t *testing.T) {
	_, err := parseGenerateLangs("cobol")
	if err == nil {
		t.Fatal("expected error for unsupported language")
	}
}

func TestParseGenerateLangs_Empty(t *testing.T) {
	_, err := parseGenerateLangs("")
	if err == nil {
		t.Fatal("expected error for empty lang list")
	}
}

func TestParseGenerateLangs_Whitespace(t *testing.T) {
	langs, err := parseGenerateLangs(" typescript , dart ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(langs) != 2 {
		t.Errorf("expected 2 langs, got %d: %v", len(langs), langs)
	}
}

func TestGenerateCmd_HelpExits(t *testing.T) {
	// Verify the command is registered and has expected flags.
	cmd := generateCmd
	if cmd.Use != "generate" {
		t.Errorf("expected Use=generate, got %s", cmd.Use)
	}

	flags := []string{"lang", "out", "env", "watch", "operations", "no-hooks", "pydantic"}
	for _, f := range flags {
		if cmd.Flags().Lookup(f) == nil {
			t.Errorf("expected flag --%s to be registered", f)
		}
	}
}
