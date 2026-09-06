package compose

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

func TestResolveImage_KnownService(t *testing.T) {
	// Explicit caller-supplied image (env/config driven) wins over the pin.
	// Pins overriding configured versions force-migrated existing deployments
	// (P1 EOP staging incident 2026-06-10).
	got := ResolveImage("postgres", "postgres:15")
	want := "postgres:15"
	if got != want {
		t.Errorf("ResolveImage(postgres, explicit) = %q, want %q", got, want)
	}
	// Pin applies only when no image is supplied.
	got = ResolveImage("postgres", "")
	want = "pgvector/pgvector:pg16"
	if got != want {
		t.Errorf("ResolveImage(postgres, empty) = %q, want %q", got, want)
	}
}

func TestResolveImage_UnknownService(t *testing.T) {
	got := ResolveImage("unknown-svc", "my-image:v1")
	want := "my-image:v1"
	if got != want {
		t.Errorf("ResolveImage(unknown) = %q, want %q", got, want)
	}
}

func TestResolveImage_AdminIsLatest(t *testing.T) {
	// Explicit image wins; the :latest pin applies only for empty input.
	got := ResolveImage("admin", AdminImagePath+":v1.0")
	want := AdminImagePath + ":v1.0"
	if got != want {
		t.Errorf("ResolveImage(admin, explicit) = %q, want %q", got, want)
	}
	got = ResolveImage("admin", "")
	want = AdminImagePath + ":latest"
	if got != want {
		t.Errorf("ResolveImage(admin, empty) = %q, want %q", got, want)
	}
}

// TestAdminImagePath_DockerHubNotGHCR verifies the admin image constant uses
// the Docker Hub nself/ namespace and never github.com/nself-org/ paths.
// This closes C1-01 from the Dim 2 undocumented dependency audit (S02.T-UNDEP-01).
func TestAdminImagePath_DockerHubNotGHCR(t *testing.T) {
	if strings.Contains(AdminImagePath, "github.com") {
		t.Errorf("AdminImagePath must not contain github.com — got %q; use Docker Hub nself/ namespace", AdminImagePath)
	}
	if strings.Contains(AdminImagePath, "ghcr.io") {
		t.Errorf("AdminImagePath must not use ghcr.io — got %q; use Docker Hub nself/ namespace", AdminImagePath)
	}
	if !strings.HasPrefix(AdminImagePath, "nself/") {
		t.Errorf("AdminImagePath must start with 'nself/' (Docker Hub namespace) — got %q", AdminImagePath)
	}
}

// TestResolvePostgresImage_ExplicitImageWins verifies precedence branch 1:
// POSTGRES_IMAGE always wins, even when POSTGRES_EXTENSIONS also lists
// pgvector or POSTGRES_VERSION is set (cli#384).
func TestResolvePostgresImage_ExplicitImageWins(t *testing.T) {
	pg := config.PostgresConfig{
		Image:      "pgvector/pgvector:pg16",
		Version:    "16-alpine",
		Extensions: []string{"pgvector"},
	}
	got := ResolvePostgresImage(pg)
	want := "pgvector/pgvector:pg16"
	if got != want {
		t.Errorf("ResolvePostgresImage(explicit image) = %q, want %q", got, want)
	}
}

// TestResolvePostgresImage_PgvectorExtensionImpliesImage verifies precedence
// branch 2: POSTGRES_EXTENSIONS containing pgvector selects the pgvector
// image matching the configured major version when POSTGRES_IMAGE is unset.
// This is the defect at the heart of cli#384 — this pin was previously dead
// code under buildPostgresService's old precedence.
func TestResolvePostgresImage_PgvectorExtensionImpliesImage(t *testing.T) {
	tests := []struct {
		name    string
		version string
		want    string
	}{
		{"alpine suffix", "16-alpine", "pgvector/pgvector:pg16"},
		{"dotted version", "15.4", "pgvector/pgvector:pg15"},
		{"unparseable falls back to pin", "latest", DefaultImageVersions["postgres"]},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pg := config.PostgresConfig{Version: tt.version, Extensions: []string{"pgvector"}}
			got := ResolvePostgresImage(pg)
			if got != tt.want {
				t.Errorf("ResolvePostgresImage(version=%q, pgvector) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

// TestResolvePostgresImage_ExtensionMatchCaseInsensitiveAndTrimmed verifies
// POSTGRES_EXTENSIONS entries are matched case-insensitively and tolerate
// surrounding whitespace from a comma-separated env value.
func TestResolvePostgresImage_ExtensionMatchCaseInsensitiveAndTrimmed(t *testing.T) {
	pg := config.PostgresConfig{
		Version:    "16-alpine",
		Extensions: []string{"uuid-ossp", " PgVector ", "pgcrypto"},
	}
	got := ResolvePostgresImage(pg)
	want := "pgvector/pgvector:pg16"
	if got != want {
		t.Errorf("ResolvePostgresImage(mixed-case extension) = %q, want %q", got, want)
	}
}

// TestResolvePostgresImage_DefaultVersionOnly verifies precedence branch 3:
// with no POSTGRES_IMAGE and no pgvector extension, the plain
// postgres:<POSTGRES_VERSION> default is unchanged from before cli#384.
func TestResolvePostgresImage_DefaultVersionOnly(t *testing.T) {
	pg := config.PostgresConfig{Version: "16-alpine", Extensions: []string{"uuid-ossp", "pgcrypto"}}
	got := ResolvePostgresImage(pg)
	want := "postgres:16-alpine"
	if got != want {
		t.Errorf("ResolvePostgresImage(no image, no pgvector) = %q, want %q", got, want)
	}
}

// TestDefaultImageVersions_NoGoModulePaths asserts that no image in
// DefaultImageVersions contains a Go module–style path (github.com/nself-org/).
// Docker image references use registry/org/image format, never Go module paths.
func TestDefaultImageVersions_NoGoModulePaths(t *testing.T) {
	for service, image := range DefaultImageVersions {
		if strings.Contains(image, "github.com/nself-org/") {
			t.Errorf("service %q: image %q contains Go module path github.com/nself-org/; use Docker registry format", service, image)
		}
	}
}
