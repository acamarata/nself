package config

import (
	"path/filepath"
	"testing"
)

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty input", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "hyphens only", input: "---", wantErr: true},
		{name: "valid lowercase", input: "myproject", want: "myproject"},
		{name: "uppercase becomes lowercase", input: "MyProject", want: "myproject"},
		{name: "spaces become hyphens", input: "my project", want: "my-project"},
		{name: "underscores become hyphens", input: "my_project", want: "my-project"},
		{name: "leading/trailing spaces trimmed", input: "  myproject  ", want: "myproject"},
		{name: "leading hyphens trimmed", input: "--myproject", want: "myproject"},
		{name: "trailing hyphens trimmed", input: "myproject--", want: "myproject"},
		{name: "consecutive hyphens collapsed", input: "my---project", want: "my-project"},
		{name: "invalid chars removed", input: "my@project!", want: "myproject"},
		{name: "mixed spaces underscores", input: "my_great project", want: "my-great-project"},
		{name: "numbers allowed", input: "project123", want: "project123"},
		{name: "idempotent: valid name unchanged", input: "my-project", want: "my-project"},
		{name: "idempotent: run twice same result", input: "my-project-2", want: "my-project-2"},
		{name: "special chars only", input: "!@#$%", wantErr: true},
		{name: "unicode stripped", input: "mypröject", want: "myprject"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeName(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SanitizeName(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("SanitizeName(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
			// Idempotency check.
			got2, err2 := SanitizeName(got)
			if err2 != nil {
				t.Errorf("SanitizeName(%q) second call error: %v", got, err2)
				return
			}
			if got2 != got {
				t.Errorf("SanitizeName not idempotent: first=%q second=%q", got, got2)
			}
		})
	}
}

func TestSanitizeRoute(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty input", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "slashes only", input: "///", wantErr: true},
		{name: "valid route", input: "api", want: "api"},
		{name: "uppercase lowercased", input: "API", want: "api"},
		{name: "leading slash stripped", input: "/api", want: "api"},
		{name: "trailing slash stripped", input: "api/", want: "api"},
		{name: "both slashes stripped", input: "/api/", want: "api"},
		{name: "spaces become hyphens", input: "my route", want: "my-route"},
		{name: "leading/trailing whitespace trimmed", input: "  auth  ", want: "auth"},
		{name: "multi-segment route preserved", input: "api/v1", want: "api/v1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizeRoute(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("sanitizeRoute(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("sanitizeRoute(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("sanitizeRoute(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizePath(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty input", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "absolute path cleaned", input: "/var/www//html", want: "/var/www/html"},
		{name: "relative path cleaned", input: "data/./files", want: "data/files"},
		{name: "double slashes removed", input: "/tmp//foo//bar", want: "/tmp/foo/bar"},
		{name: "valid simple path", input: "/etc/nginx", want: "/etc/nginx"},
		{name: "dot-dot preserved in clean", input: "/a/b/../c", want: "/a/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizePath(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("sanitizePath(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("sanitizePath(%q) unexpected error: %v", tt.input, err)
				return
			}
			// Normalize to forward slashes so the test asserts semantic equality
			// across POSIX and Windows (filepath.Clean uses OS-native separators).
			gotNorm := filepath.ToSlash(got)
			if gotNorm != tt.want {
				t.Errorf("sanitizePath(%q) = %q, want %q", tt.input, gotNorm, tt.want)
			}
		})
	}
}

func TestSanitizePort(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty input", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "non-numeric", input: "abc", wantErr: true},
		{name: "float not allowed", input: "80.5", wantErr: true},
		{name: "valid port", input: "8080", want: "8080"},
		{name: "whitespace trimmed", input: "  443  ", want: "443"},
		{name: "port 80", input: "80", want: "80"},
		{name: "high port", input: "65535", want: "65535"},
		{name: "negative not numeric?", input: "-1", want: "-1"}, // strconv.Atoi("-1") succeeds
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := sanitizePort(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("sanitizePort(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("sanitizePort(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("sanitizePort(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSanitizeDomain(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "empty input", input: "", wantErr: true},
		{name: "whitespace only", input: "   ", wantErr: true},
		{name: "trailing dot only", input: ".", wantErr: true},
		{name: "valid domain", input: "example.com", want: "example.com"},
		{name: "uppercase lowercased", input: "Example.COM", want: "example.com"},
		{name: "trailing dot trimmed", input: "example.com.", want: "example.com"},
		{name: "leading/trailing whitespace trimmed", input: "  example.com  ", want: "example.com"},
		{name: "localhost", input: "localhost", want: "localhost"},
		{name: "subdomain", input: "api.example.com", want: "api.example.com"},
		{name: "multiple trailing dots", input: "example.com...", want: "example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := SanitizeDomain(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SanitizeDomain(%q) expected error, got %q", tt.input, got)
				}
				return
			}
			if err != nil {
				t.Errorf("SanitizeDomain(%q) unexpected error: %v", tt.input, err)
				return
			}
			if got != tt.want {
				t.Errorf("SanitizeDomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
