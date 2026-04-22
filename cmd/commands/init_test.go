package commands

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// newInitCmd returns a fresh cobra.Command tree wired with just the init
// command so tests can execute it in isolation without side effects from the
// global RootCmd state.
func newInitCmd() *cobra.Command {
	root := &cobra.Command{Use: "nself", RunE: func(cmd *cobra.Command, args []string) error { return nil }}

	ic := &cobra.Command{
		Use:   "init",
		Short: "Initialize a new nSelf project",
		RunE:  runInit,
	}
	ic.Flags().String("name", "", "Project name")
	ic.Flags().String("domain", "", "Base domain")
	ic.Flags().Bool("non-interactive", false, "Use all defaults without prompts")
	ic.Flags().Bool("fast", false, "Skip advanced options, use smart defaults")
	ic.Flags().Bool("wizard", false, "Run the full 10-step interactive wizard")
	ic.Flags().Bool("demo", false, "Auto-configure with all services enabled")
	ic.Flags().Bool("full", false, "Create all environment files")
	ic.Flags().Bool("force", false, "Overwrite existing configuration")
	ic.Flags().Bool("quiet", false, "Suppress output messages")
	ic.Flags().String("template", "", "Use specific template")
	ic.Flags().Bool("skip-validation", false, "Skip configuration validation")
	ic.Flags().Bool("interactive", false, "Explicitly enable interactive wizard")

	root.AddCommand(ic)
	return root
}

// TestInitCmd_Registered verifies that the init subcommand is registered on
// the root command.
func TestInitCmd_Registered(t *testing.T) {
	found := false
	for _, c := range RootCmd.Commands() {
		if c.Name() == "init" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected 'init' to be registered on RootCmd")
	}
}

// TestInitCmd_FlagName verifies that the --name flag is accepted by the init
// command without a parse error.
func TestInitCmd_FlagName(t *testing.T) {
	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	// The command will fail because there is no nself project in the temp dir,
	// but the flag must be recognised (not an "unknown flag" error).
	root.SetArgs([]string{"init", "--name", "myproject", "--non-interactive"})
	err := root.Execute()

	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--name flag was not recognised: %v", err)
	}
}

// TestInitCmd_FlagDomain verifies that the --domain flag is accepted.
func TestInitCmd_FlagDomain(t *testing.T) {
	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--domain", "myapp.dev", "--non-interactive"})
	err := root.Execute()

	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--domain flag was not recognised: %v", err)
	}
}

// TestInitCmd_FlagNonInteractive verifies that the --non-interactive flag is
// accepted.
func TestInitCmd_FlagNonInteractive(t *testing.T) {
	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive"})
	err := root.Execute()

	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--non-interactive flag was not recognised: %v", err)
	}
}

// TestInitCmd_UnknownFlagRejected verifies that an unrecognised flag is
// rejected with a cobra error rather than silently ignored.
func TestInitCmd_UnknownFlagRejected(t *testing.T) {
	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--this-flag-does-not-exist"})
	err := root.Execute()

	if err == nil {
		t.Fatal("expected an error for unknown flag, got nil")
	}
}

// TestInitCmd_FlagFast verifies that the --fast flag is accepted.
func TestInitCmd_FlagFast(t *testing.T) {
	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--fast"})
	err := root.Execute()

	if err != nil && strings.Contains(err.Error(), "unknown flag") {
		t.Fatalf("--fast flag was not recognised: %v", err)
	}
}

// TestInitCmd_NonInteractiveCreatesEnvFile verifies that running
// "init --non-interactive --quiet" in a fresh directory creates a .env file.
// This exercises the success path of runInit including setup.Initialize.
func TestInitCmd_NonInteractiveCreatesEnvFile(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive", "--quiet"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("init --non-interactive --quiet failed: %v", err)
	}

	if _, statErr := os.Stat(".env"); os.IsNotExist(statErr) {
		t.Fatal("expected .env to be created by init, but it does not exist")
	}
}

// TestInitCmd_NameFlagSetsProjectName verifies that --name sets PROJECT_NAME
// in the generated .env file.
func TestInitCmd_NameFlagSetsProjectName(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive", "--quiet", "--name", "myproject"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("init with --name failed: %v", err)
	}

	content, readErr := os.ReadFile(".env")
	if readErr != nil {
		t.Fatalf("could not read .env: %v", readErr)
	}
	if !strings.Contains(string(content), "PROJECT_NAME=myproject") {
		t.Errorf("expected PROJECT_NAME=myproject in .env, got:\n%s", string(content))
	}
}

// TestInitCmd_DomainFlagSetsBaseDomain verifies that --domain sets BASE_DOMAIN
// in the generated .env file.
func TestInitCmd_DomainFlagSetsBaseDomain(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive", "--quiet", "--domain", "myapp.dev"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("init with --domain failed: %v", err)
	}

	content, readErr := os.ReadFile(".env")
	if readErr != nil {
		t.Fatalf("could not read .env: %v", readErr)
	}
	if !strings.Contains(string(content), "BASE_DOMAIN=myapp.dev") {
		t.Errorf("expected BASE_DOMAIN=myapp.dev in .env, got:\n%s", string(content))
	}
}

// TestInitCmd_ForceOverwritesExistingEnv verifies that --force allows init to
// run when a .env file already exists.
func TestInitCmd_ForceOverwritesExistingEnv(t *testing.T) {
	t.Chdir(t.TempDir())

	// Create a pre-existing .env so a plain init would fail.
	if err := os.WriteFile(".env", []byte("EXISTING=1\n"), 0600); err != nil {
		t.Fatalf("writing seed .env: %v", err)
	}

	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive", "--quiet", "--force"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("init --force failed: %v", err)
	}
}

// TestInitCmd_InvalidNameRejected verifies that --name with all invalid
// characters is rejected with a clear error message.
func TestInitCmd_InvalidNameRejected(t *testing.T) {
	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	// "!!!" has no valid characters after sanitization.
	root.SetArgs([]string{"init", "--name", "!!!", "--non-interactive"})
	err := root.Execute()

	if err == nil {
		t.Fatal("expected error for invalid --name, got nil")
	}
	if !strings.Contains(err.Error(), "--name") {
		t.Errorf("expected error to mention --name, got: %v", err)
	}
}

// ── validateTemplate — unit tests ────────────────────────────────────────────

// TestValidateTemplate_Valid verifies that each supported template name is
// accepted without error.
func TestValidateTemplate_Valid(t *testing.T) {
	templates := []string{"postgres", "express", "fastapi", "go", "rust"}
	for _, tmpl := range templates {
		if err := validateTemplate(tmpl); err != nil {
			t.Errorf("validateTemplate(%q) returned unexpected error: %v", tmpl, err)
		}
	}
}

// TestValidateTemplate_Invalid verifies that an unrecognised template name is
// rejected and the error message lists available templates.
func TestValidateTemplate_Invalid(t *testing.T) {
	err := validateTemplate("laravel")
	if err == nil {
		t.Fatal("expected an error for unknown template 'laravel', got nil")
	}
	if !strings.Contains(err.Error(), "laravel") {
		t.Errorf("expected error to mention the bad template name, got: %v", err)
	}
	if !strings.Contains(err.Error(), "Available templates") {
		t.Errorf("expected error to mention 'Available templates', got: %v", err)
	}
}

// TestValidateTemplate_Empty verifies that an empty string is rejected.
func TestValidateTemplate_Empty(t *testing.T) {
	if err := validateTemplate(""); err == nil {
		t.Fatal("expected an error for empty template name, got nil")
	}
}

// ── validateDomain — unit tests ───────────────────────────────────────────────

// TestValidateDomain_Valid verifies that typical valid domain values are
// accepted by validateDomain.
func TestValidateDomain_Valid(t *testing.T) {
	cases := []string{
		"myapp.dev",
		"local.nself.org",
		"127.0.0.1.nip.io",
		"example.com",
		"localhost",
	}
	for _, d := range cases {
		if err := validateDomain(d); err != nil {
			t.Errorf("validateDomain(%q) returned unexpected error: %v", d, err)
		}
	}
}

// TestValidateDomain_Empty verifies that an empty string is rejected.
func TestValidateDomain_Empty(t *testing.T) {
	if err := validateDomain(""); err == nil {
		t.Fatal("expected an error for empty domain, got nil")
	}
}

// TestValidateDomain_WithSpace verifies that a domain containing a space is
// rejected.
func TestValidateDomain_WithSpace(t *testing.T) {
	if err := validateDomain("my app.dev"); err == nil {
		t.Fatal("expected an error for domain with space, got nil")
	}
}

// TestValidateDomain_WithTab verifies that a domain containing a tab is
// rejected.
func TestValidateDomain_WithTab(t *testing.T) {
	if err := validateDomain("my\tapp.dev"); err == nil {
		t.Fatal("expected an error for domain with tab, got nil")
	}
}

// ── init --template flag — integration tests ──────────────────────────────────

// TestInitCmd_TemplateFlag_Valid verifies that --template go is accepted and
// produces a .env file in a fresh directory.
func TestInitCmd_TemplateFlag_Valid(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive", "--quiet", "--template", "go"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("init --template go failed: %v", err)
	}

	if _, statErr := os.Stat(".env"); os.IsNotExist(statErr) {
		t.Fatal("expected .env to be created by init --template go, but it does not exist")
	}
}

// TestInitCmd_TemplateFlag_Invalid verifies that an unknown --template value
// is rejected before any file is created.
func TestInitCmd_TemplateFlag_Invalid(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive", "--template", "laravel"})
	err := root.Execute()

	if err == nil {
		t.Fatal("expected an error for invalid --template value, got nil")
	}

	// The .env file must NOT be created when the template is rejected.
	if _, statErr := os.Stat(".env"); !os.IsNotExist(statErr) {
		t.Error("expected .env NOT to be created for invalid template")
	}
}

// ── classifyInitError — unit tests ───────────────────────────────────────────

// TestClassifyInitError_Nil verifies that a nil error returns an empty string.
func TestClassifyInitError_Nil(t *testing.T) {
	got := classifyInitError(nil)
	if got != "" {
		t.Errorf("classifyInitError(nil) = %q, want %q", got, "")
	}
}

// TestClassifyInitError_Timeout verifies that timeout-like errors are classified
// as "timeout".
func TestClassifyInitError_Timeout(t *testing.T) {
	cases := []string{
		"context deadline exceeded",
		"operation timeout",
		"deadline exceeded: connection refused",
	}
	for _, msg := range cases {
		got := classifyInitError(fmt.Errorf("%s", msg))
		if got != "timeout" {
			t.Errorf("classifyInitError(%q) = %q, want %q", msg, got, "timeout")
		}
	}
}

// TestClassifyInitError_PermissionDenied verifies that permission-related errors
// are classified as "permission-denied".
func TestClassifyInitError_PermissionDenied(t *testing.T) {
	cases := []string{
		"permission denied: /etc/nself",
		"access denied writing config",
		"EACCES: no permission",
	}
	for _, msg := range cases {
		got := classifyInitError(fmt.Errorf("%s", msg))
		if got != "permission-denied" {
			t.Errorf("classifyInitError(%q) = %q, want %q", msg, got, "permission-denied")
		}
	}
}

// TestClassifyInitError_DockerNotFound verifies that Docker-related errors are
// classified as "docker-not-found".
func TestClassifyInitError_DockerNotFound(t *testing.T) {
	cases := []string{
		"cannot connect to the Docker daemon",
		"Docker daemon is not running",
		"docker: command not found",
	}
	for _, msg := range cases {
		got := classifyInitError(fmt.Errorf("%s", msg))
		if got != "docker-not-found" {
			t.Errorf("classifyInitError(%q) = %q, want %q", msg, got, "docker-not-found")
		}
	}
}

// TestClassifyInitError_Other verifies that unrecognised errors fall back to
// "other" and that no file paths or user text leak into the category string.
func TestClassifyInitError_Other(t *testing.T) {
	cases := []string{
		"unexpected EOF",
		"disk full",
		"some completely unknown failure",
	}
	for _, msg := range cases {
		got := classifyInitError(fmt.Errorf("%s", msg))
		if got != "other" {
			t.Errorf("classifyInitError(%q) = %q, want %q", msg, got, "other")
		}
	}
}

// TestClassifyInitError_CategoryIsEnum verifies that classifyInitError only
// ever returns one of the four documented enum values.
func TestClassifyInitError_CategoryIsEnum(t *testing.T) {
	allowed := map[string]bool{
		"":                 true, // nil case
		"timeout":          true,
		"permission-denied": true,
		"docker-not-found": true,
		"other":            true,
	}
	inputs := []error{
		nil,
		fmt.Errorf("timeout"),
		fmt.Errorf("permission denied"),
		fmt.Errorf("docker not running"),
		fmt.Errorf("some unknown error xyz"),
	}
	for _, err := range inputs {
		got := classifyInitError(err)
		if !allowed[got] {
			t.Errorf("classifyInitError returned undocumented category %q for error %v", got, err)
		}
	}
}

// TestInitCmd_DemoFlag verifies that --demo runs successfully and creates a
// .env file with demo defaults.
func TestInitCmd_DemoFlag(t *testing.T) {
	t.Chdir(t.TempDir())

	root := newInitCmd()
	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)

	root.SetArgs([]string{"init", "--non-interactive", "--quiet", "--demo"})
	err := root.Execute()
	if err != nil {
		t.Fatalf("init --demo failed: %v", err)
	}

	if _, statErr := os.Stat(".env"); os.IsNotExist(statErr) {
		t.Fatal("expected .env to be created by init --demo, but it does not exist")
	}
}
