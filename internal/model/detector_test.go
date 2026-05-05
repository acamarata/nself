package model

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectModel_ModelB(t *testing.T) {
	home := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".nself", "plugins", "ai"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	opts := resolveOpts{
		homeDir:     home,
		projectDir:  t.TempDir(),
		dockerCheck: func(string) bool { return false },
	}
	got, err := detectWithOpts("ai", opts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != ModelB {
		t.Fatalf("expected ModelB, got %v", got)
	}
}

func TestDetectModel_ModelA_SourceDir(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "plugins-pro-src", "paid", "claw"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	opts := resolveOpts{
		homeDir:     t.TempDir(),
		projectDir:  project,
		dockerCheck: func(string) bool { return false },
	}
	got, err := detectWithOpts("claw", opts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != ModelA {
		t.Fatalf("expected ModelA, got %v", got)
	}
}

func TestDetectModel_ModelA_ComposeService(t *testing.T) {
	opts := resolveOpts{
		homeDir:     t.TempDir(),
		projectDir:  t.TempDir(),
		dockerCheck: func(name string) bool { return name == "mux" },
	}
	got, err := detectWithOpts("mux", opts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != ModelA {
		t.Fatalf("expected ModelA, got %v", got)
	}
}

func TestDetectModel_Mixed(t *testing.T) {
	home := t.TempDir()
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(home, ".nself", "plugins", "ai"), 0o755); err != nil {
		t.Fatalf("mkdir B: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(project, "plugins-pro-src", "paid", "ai"), 0o755); err != nil {
		t.Fatalf("mkdir A: %v", err)
	}

	opts := resolveOpts{
		homeDir:     home,
		projectDir:  project,
		dockerCheck: func(string) bool { return false },
	}
	got, _ := detectWithOpts("ai", opts)
	if got != ModelMixed {
		t.Fatalf("expected ModelMixed, got %v", got)
	}
}

func TestDetectModel_Unknown(t *testing.T) {
	opts := resolveOpts{
		homeDir:     t.TempDir(),
		projectDir:  t.TempDir(),
		dockerCheck: func(string) bool { return false },
	}
	got, err := detectWithOpts("does-not-exist", opts)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != ModelUnknown {
		t.Fatalf("expected ModelUnknown, got %v", got)
	}
}

func TestDetectModel_EmptyName(t *testing.T) {
	if _, err := DetectModel(""); err == nil {
		t.Fatalf("expected error on empty plugin name")
	}
}

func TestModel_String(t *testing.T) {
	cases := []struct {
		m    Model
		want string
	}{
		{ModelA, "A"},
		{ModelB, "B"},
		{ModelMixed, "Mixed"},
		{ModelUnknown, "Unknown"},
	}
	for _, c := range cases {
		if got := c.m.String(); got != c.want {
			t.Errorf("Model %d: got %q, want %q", c.m, got, c.want)
		}
	}
}
