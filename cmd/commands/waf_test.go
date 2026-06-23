package commands

import (
	"os"
	"path/filepath"
	"testing"
)

// TestWAFCmd_Structure verifies subcommands are registered.
func TestWAFCmd_Structure(t *testing.T) {
	subs := wafCmd.Commands()
	names := make(map[string]bool)
	for _, cmd := range subs {
		names[cmd.Name()] = true
	}
	for _, name := range []string{"enable", "mode", "report"} {
		if !names[name] {
			t.Errorf("missing subcommand: %s", name)
		}
	}
}

// TestWAFModeCmd_Args verifies waf mode requires exactly one argument.
func TestWAFModeCmd_Args(t *testing.T) {
	if wafModeCmd.Args == nil {
		t.Fatal("waf mode should enforce ExactArgs(1)")
	}
	// ExactArgs(1) — valid with exactly one arg
	if err := wafModeCmd.Args(wafModeCmd, []string{"detection"}); err != nil {
		t.Errorf("unexpected error for valid arg: %v", err)
	}
	// invalid — too many args
	if err := wafModeCmd.Args(wafModeCmd, []string{"detection", "extra"}); err == nil {
		t.Error("expected error for too many args")
	}
}

// TestWAFReportCmd_SinceFlag verifies --since flag registration.
func TestWAFReportCmd_SinceFlag(t *testing.T) {
	f := wafReportCmd.Flags().Lookup("since")
	if f == nil {
		t.Fatal("--since flag not registered on waf report")
	}
	if f.DefValue != "24h" {
		t.Errorf("--since default = %q, want 24h", f.DefValue)
	}
}

// makeProjectDir creates a temp dir with a .env file so FindNSelfRoot resolves.
func makeProjectDir(t *testing.T) string {
	t.Helper()
	tmp := t.TempDir()
	// FindNSelfRoot resolves when a .env file is present in the dir.
	if err := os.WriteFile(filepath.Join(tmp, ".env"), []byte("# nself project\n"), 0600); err != nil {
		t.Fatal(err)
	}
	return tmp
}

// TestRunWAFEnable_CreatesConfig verifies that runWAFEnable writes coraza.conf
// and custom.conf into nginx/waf/ under a temp project dir.
func TestRunWAFEnable_CreatesConfig(t *testing.T) {
	tmp := makeProjectDir(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := runWAFEnable(wafEnableCmd, nil); err != nil {
		t.Fatalf("runWAFEnable: %v", err)
	}

	corazaPath := filepath.Join(tmp, "nginx", "waf", "coraza.conf")
	if _, err := os.Stat(corazaPath); err != nil {
		t.Errorf("coraza.conf not created: %v", err)
	}

	customPath := filepath.Join(tmp, "nginx", "waf", "custom.conf")
	if _, err := os.Stat(customPath); err != nil {
		t.Errorf("custom.conf not created: %v", err)
	}
}

// TestRunWAFEnable_CorazaContainsOWASP verifies generated config references OWASP CRS.
func TestRunWAFEnable_CorazaContainsOWASP(t *testing.T) {
	tmp := makeProjectDir(t)

	origDir, _ := os.Getwd()
	if err := os.Chdir(tmp); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	if err := runWAFEnable(wafEnableCmd, nil); err != nil {
		t.Fatalf("runWAFEnable: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tmp, "nginx", "waf", "coraza.conf"))
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	checks := []string{"SecRuleEngine", "OWASP CRS", "SecAuditLog"}
	for _, c := range checks {
		found := false
		for i := 0; i < len(content)-len(c); i++ {
			if content[i:i+len(c)] == c {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("coraza.conf missing expected string: %q", c)
		}
	}
}
