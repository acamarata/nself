package build

import (
	"os"
	"path/filepath"
	"testing"
)

// writeFile is a test helper that writes content to a file, creating
// intermediate directories as needed. It calls t.Fatal on any error.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("creating directory for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// TestPostValidate_ValidCompose verifies that a well-formed docker-compose.yml
// passes all checks.
func TestPostValidate_ValidCompose(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")

	composeContent := `version: "3.8"
services:
  postgres:
    image: postgres:16
    ports:
      - "5432:5432"
  hasura:
    image: hasura/graphql-engine:latest
    ports:
      - "8080:8080"
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
`
	writeFile(t, composePath, composeContent)

	nginxSitesDir := filepath.Join(dir, "nginx", "sites")
	if err := os.MkdirAll(nginxSitesDir, 0755); err != nil {
		t.Fatalf("creating nginx/sites: %v", err)
	}

	result := PostValidate(composePath, nginxSitesDir)

	if !result.ComposeValid {
		t.Errorf("ComposeValid = false, want true; errors: %v", result.Errors)
	}
	if !result.PortsUnique {
		t.Errorf("PortsUnique = false, want true; errors: %v", result.Errors)
	}
	if !result.NamesUnique {
		t.Errorf("NamesUnique = false, want true; errors: %v", result.Errors)
	}

	// Nginx check: will either pass (nginx installed) or produce a warning
	// (nginx not in PATH). Neither case should add an Error.
	for _, e := range result.Errors {
		if contains(e, "nginx") {
			t.Errorf("unexpected nginx error: %s", e)
		}
	}
}

// TestPostValidate_DuplicatePorts verifies that duplicate host-side port
// bindings are caught and reported as errors.
func TestPostValidate_DuplicatePorts(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")

	// Both postgres and hasura expose host port 5432.
	composeContent := `version: "3.8"
services:
  postgres:
    image: postgres:16
    ports:
      - "5432:5432"
  hasura:
    image: hasura/graphql-engine:latest
    ports:
      - "5432:8080"
`
	writeFile(t, composePath, composeContent)

	nginxSitesDir := filepath.Join(dir, "nginx", "sites")
	if err := os.MkdirAll(nginxSitesDir, 0755); err != nil {
		t.Fatalf("creating nginx/sites: %v", err)
	}

	result := PostValidate(composePath, nginxSitesDir)

	if result.PortsUnique {
		t.Error("PortsUnique = true, want false for duplicate port 5432")
	}

	found := false
	for _, e := range result.Errors {
		if contains(e, "5432") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected an error mentioning port 5432; got errors: %v", result.Errors)
	}
}

// TestPostValidate_MissingComposeFile verifies behavior when composePath
// does not exist.
func TestPostValidate_MissingComposeFile(t *testing.T) {
	dir := t.TempDir()
	result := PostValidate(filepath.Join(dir, "missing.yml"), dir)

	if result.ComposeValid {
		t.Error("ComposeValid = true for missing file, want false")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error for missing file")
	}
}

// TestPostValidate_MissingServicesKey verifies that a compose file lacking
// the 'services' key is flagged.
func TestPostValidate_MissingServicesKey(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")

	writeFile(t, composePath, "version: \"3.8\"\n")

	result := PostValidate(composePath, dir)

	if result.ComposeValid {
		t.Error("ComposeValid = true for file without 'services', want false")
	}
	if len(result.Errors) == 0 {
		t.Error("expected at least one error for missing services key")
	}
}

// TestPostValidate_InvalidYAML verifies that a malformed YAML file is caught.
func TestPostValidate_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	composePath := filepath.Join(dir, "docker-compose.yml")

	writeFile(t, composePath, ":::not valid yaml:::\n\t\tbad indent\n")

	result := PostValidate(composePath, dir)

	if result.ComposeValid {
		t.Error("ComposeValid = true for invalid YAML, want false")
	}
}

// TestExtractHostPort validates the host port extraction helper for all
// supported Docker Compose port binding formats.
func TestExtractHostPort(t *testing.T) {
	tests := []struct {
		name    string
		binding string
		want    string
	}{
		{name: "host:container", binding: "8080:80", want: "8080"},
		{name: "host only", binding: "8080", want: "8080"},
		{name: "ip:host:container", binding: "127.0.0.1:80:80", want: "80"},
		{name: "ip:host:container high port", binding: "0.0.0.0:5432:5432", want: "5432"},
		{name: "standard https", binding: "443:443", want: "443"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractHostPort(tt.binding)
			if got != tt.want {
				t.Errorf("extractHostPort(%q) = %q, want %q", tt.binding, got, tt.want)
			}
		})
	}
}

// contains is a simple substring check used in test assertions.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(substr); i++ {
				if s[i:i+len(substr)] == substr {
					return true
				}
			}
			return false
		}())
}

// TestCheckNginxSyntax_PidPermissionNotSyntaxError verifies that when nginx -t
// reports the config syntax is OK but then exits non-zero because it cannot open
// the pid file (the norm for non-root `nself build` runs and CI), the check
// passes rather than reporting a false "syntax check failed". Regression for the
// golden-path E2E step-4 failure (2026-07-06).
func TestCheckNginxSyntax_PidPermissionNotSyntaxError(t *testing.T) {
	dir := t.TempDir()

	// Generated layout: <dir>/nginx.conf + <dir>/nginx/sites/ (caller passes sites/).
	if err := os.WriteFile(filepath.Join(dir, "nginx.conf"), []byte("events{}\nhttp{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	sitesDir := filepath.Join(dir, "nginx", "sites")
	if err := os.MkdirAll(sitesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// findNginxConf looks one level up from sites/, so place nginx.conf at nginx/.
	if err := os.WriteFile(filepath.Join(dir, "nginx", "nginx.conf"), []byte("events{}\nhttp{}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Fake nginx binary: prints the real-world message to stderr and exits 1,
	// exactly like `nginx -t` as a non-root user without a writable pid path.
	binDir := t.TempDir()
	fake := "#!/bin/sh\n" +
		"echo 'nginx: the configuration file ... syntax is ok' 1>&2\n" +
		"echo 'nginx: [emerg] open() \"/var/run/nginx.pid\" failed (13: Permission denied)' 1>&2\n" +
		"exit 1\n"
	fakePath := filepath.Join(binDir, "nginx")
	if err := os.WriteFile(fakePath, []byte(fake), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	result := &PostValidateResult{NginxValid: true}
	checkNginxSyntax(sitesDir, result)

	if !result.NginxValid {
		t.Errorf("expected NginxValid=true (syntax is ok despite pid-file exit 1), got false")
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors when syntax is ok, got: %v", result.Errors)
	}
}
