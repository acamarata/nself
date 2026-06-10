package compose

import (
	"strings"
	"testing"
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
