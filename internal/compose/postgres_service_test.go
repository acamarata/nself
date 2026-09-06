package compose

import (
	"strings"
	"testing"
)

// TestBuildPostgresService_PgvectorExtensionPreservesUidAndPGDATA verifies
// cli#384 end-to-end: a build with POSTGRES_EXTENSIONS=pgvector (and no
// explicit POSTGRES_IMAGE) must emit the pgvector image, uid 999:999, and a
// PGDATA subdir — matching what the pgvector/pgvector image actually runs as,
// so a regen against an already-running pgvector container does not crash
// the container or lose the extension on restart.
func TestBuildPostgresService_PgvectorExtensionPreservesUidAndPGDATA(t *testing.T) {
	cfg := minimalConfig()
	cfg.Postgres.Version = "16-alpine"
	cfg.Postgres.Extensions = []string{"pgvector"}
	g := NewGenerator(cfg)

	svc := g.buildPostgresService()

	if svc.Image != "pgvector/pgvector:pg16" {
		t.Errorf("Image = %q, want pgvector/pgvector:pg16", svc.Image)
	}
	if svc.User != "999:999" {
		t.Errorf("User = %q, want 999:999 for a pgvector (debian-family) image", svc.User)
	}
	if got := svc.Environment["PGDATA"]; got != "/var/lib/postgresql/data/pgdata" {
		t.Errorf("PGDATA = %q, want /var/lib/postgresql/data/pgdata", got)
	}
}

// TestBuildPostgresService_ExplicitImageWinsOverExtensions verifies
// POSTGRES_IMAGE overrides both the version default and the pgvector
// inference from POSTGRES_EXTENSIONS.
func TestBuildPostgresService_ExplicitImageWinsOverExtensions(t *testing.T) {
	cfg := minimalConfig()
	cfg.Postgres.Version = "16-alpine"
	cfg.Postgres.Extensions = []string{"pgvector"}
	cfg.Postgres.Image = "postgres:16" // debian, not alpine, not pgvector
	g := NewGenerator(cfg)

	svc := g.buildPostgresService()

	if svc.Image != "postgres:16" {
		t.Errorf("Image = %q, want postgres:16 (explicit POSTGRES_IMAGE)", svc.Image)
	}
	if svc.User != "999:999" {
		t.Errorf("User = %q, want 999:999 for a debian-family image", svc.User)
	}
	if _, ok := svc.Environment["PGDATA"]; !ok {
		t.Error("PGDATA missing, want set for a debian-family image")
	}
}

// TestBuildPostgresService_PlainAlpineDefault_NoExtensions verifies the
// unchanged default: no POSTGRES_IMAGE, no pgvector extension, plain alpine
// postgres image, uid 70:70, and no PGDATA (matches upstream alpine image
// behavior — this is not the regression case).
func TestBuildPostgresService_PlainAlpineDefault_NoExtensions(t *testing.T) {
	cfg := minimalConfig()
	cfg.Postgres.Version = "16-alpine"
	cfg.Postgres.Extensions = []string{"uuid-ossp", "pgcrypto"}
	g := NewGenerator(cfg)

	svc := g.buildPostgresService()

	if svc.Image != "postgres:16-alpine" {
		t.Errorf("Image = %q, want postgres:16-alpine", svc.Image)
	}
	if svc.User != "70:70" {
		t.Errorf("User = %q, want 70:70 for a plain alpine image", svc.User)
	}
	if _, ok := svc.Environment["PGDATA"]; ok {
		t.Error("PGDATA set, want absent for a plain alpine image")
	}
}

// TestBuildPostgresService_ExtensionMatchIsCaseInsensitive guards the
// POSTGRES_EXTENSIONS=pgvector matching used by buildPostgresService via
// ResolvePostgresImage against case/whitespace variants a hand-edited .env
// might contain.
func TestBuildPostgresService_ExtensionMatchIsCaseInsensitive(t *testing.T) {
	cfg := minimalConfig()
	cfg.Postgres.Version = "15-alpine"
	cfg.Postgres.Extensions = []string{" PGVECTOR "}
	g := NewGenerator(cfg)

	svc := g.buildPostgresService()

	if !strings.HasPrefix(svc.Image, "pgvector/pgvector:pg15") {
		t.Errorf("Image = %q, want pgvector/pgvector:pg15", svc.Image)
	}
}
