package compose

import "testing"

func TestResolveImage_KnownService(t *testing.T) {
	got := ResolveImage("postgres", "postgres:15")
	want := "pgvector/pgvector:pg16"
	if got != want {
		t.Errorf("ResolveImage(postgres) = %q, want %q", got, want)
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
	got := ResolveImage("admin", "nself/nself-admin:v1.0")
	want := "nself/nself-admin:latest"
	if got != want {
		t.Errorf("ResolveImage(admin) = %q, want %q", got, want)
	}
}
