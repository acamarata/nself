// S9.T08-T10 — unit tests for the Loki build-pipeline integration.
//
// Cases (per spec):
//
//  1. default config rendered, retention 720h, no S3 backend
//  2. S3 backend enabled, retention left at default
//  3. retention override propagated end-to-end
//  4. validate() rejects missing ProjectName / S3Region / TenantID
//  5. WriteLokiConfigs is idempotent — second call produces byte-identical files
package build

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRenderLokiBundle_Default: zero overrides → default retention (720h),
// filesystem chunk storage (no S3 stanza in output).
func TestRenderLokiBundle_Default(t *testing.T) {
	loki, prom, err := RenderLokiBundle(LokiBuildOptions{
		ProjectName: "nself-test",
	})
	if err != nil {
		t.Fatalf("default render: %v", err)
	}
	if !strings.Contains(string(loki), "retention_period: 720h") {
		t.Errorf("default loki.yml missing retention_period: 720h\n%s", loki)
	}
	if strings.Contains(string(loki), "storage_config:\n  aws:") {
		t.Errorf("default loki.yml should not contain S3 stanza:\n%s", loki)
	}
	if !strings.Contains(string(prom), "nself_project: nself-test") {
		t.Errorf("default promtail.yml missing project label\n%s", prom)
	}
}

// TestRenderLokiBundle_S3Backend: S3 bucket + region set → s3 storage_config
// stanza appended; default retention preserved.
func TestRenderLokiBundle_S3Backend(t *testing.T) {
	loki, _, err := RenderLokiBundle(LokiBuildOptions{
		ProjectName: "nself-test",
		S3Bucket:    "nself-loki-prod",
		S3Region:    "auto",
	})
	if err != nil {
		t.Fatalf("S3 render: %v", err)
	}
	if !strings.Contains(string(loki), "s3://nself-loki-prod") {
		t.Errorf("S3 stanza missing bucket reference:\n%s", loki)
	}
	if !strings.Contains(string(loki), "region: auto") {
		t.Errorf("S3 stanza missing region:\n%s", loki)
	}
	if !strings.Contains(string(loki), "retention_period: 720h") {
		t.Errorf("retention should remain default when only S3 set:\n%s", loki)
	}
}

// TestRenderLokiBundle_RetentionOverride: explicit retention propagates and S3
// stanza absent (proves the two override knobs are independent).
func TestRenderLokiBundle_RetentionOverride(t *testing.T) {
	loki, _, err := RenderLokiBundle(LokiBuildOptions{
		ProjectName:     "nself-test",
		RetentionPeriod: "2160h", // 90 days
	})
	if err != nil {
		t.Fatalf("retention override render: %v", err)
	}
	if !strings.Contains(string(loki), "retention_period: 2160h") {
		t.Errorf("retention override not applied:\n%s", loki)
	}
}

// TestLokiBuildOptions_Validate: each invalid combination returns an error
// instead of silently producing broken YAML.
func TestLokiBuildOptions_Validate(t *testing.T) {
	cases := []struct {
		name    string
		opts    LokiBuildOptions
		wantErr string
	}{
		{
			name:    "missing project",
			opts:    LokiBuildOptions{},
			wantErr: "ProjectName is required",
		},
		{
			name:    "S3 without region",
			opts:    LokiBuildOptions{ProjectName: "p", S3Bucket: "b"},
			wantErr: "S3Region required",
		},
		{
			name:    "multi-tenant without ID",
			opts:    LokiBuildOptions{ProjectName: "p", MultiTenant: true},
			wantErr: "TenantID required",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := RenderLokiBundle(tc.opts)
			if err == nil {
				t.Fatalf("want error %q, got nil", tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error %q does not mention %q", err.Error(), tc.wantErr)
			}
		})
	}
}

// TestWriteLokiConfigs_Idempotent: two calls produce byte-identical files,
// proving downstream Docker bind-mount hashes won't churn on re-build.
func TestWriteLokiConfigs_Idempotent(t *testing.T) {
	workdir := t.TempDir()
	opts := LokiBuildOptions{
		ProjectName:     "idempotent-test",
		RetentionPeriod: "720h",
	}

	n1, err := WriteLokiConfigs(workdir, opts)
	if err != nil {
		t.Fatalf("first write: %v", err)
	}
	if n1 != 2 {
		t.Errorf("first write: want 2 files, got %d", n1)
	}

	loki1, err := os.ReadFile(filepath.Join(workdir, "monitoring", "loki.yml"))
	if err != nil {
		t.Fatalf("read loki.yml after first write: %v", err)
	}
	prom1, err := os.ReadFile(filepath.Join(workdir, "monitoring", "promtail.yml"))
	if err != nil {
		t.Fatalf("read promtail.yml after first write: %v", err)
	}

	n2, err := WriteLokiConfigs(workdir, opts)
	if err != nil {
		t.Fatalf("second write: %v", err)
	}
	if n2 != 2 {
		t.Errorf("second write: want 2 files, got %d", n2)
	}

	loki2, err := os.ReadFile(filepath.Join(workdir, "monitoring", "loki.yml"))
	if err != nil {
		t.Fatalf("read loki.yml after second write: %v", err)
	}
	prom2, err := os.ReadFile(filepath.Join(workdir, "monitoring", "promtail.yml"))
	if err != nil {
		t.Fatalf("read promtail.yml after second write: %v", err)
	}

	if !bytes.Equal(loki1, loki2) {
		t.Errorf("loki.yml differs across rebuilds — not idempotent\nfirst:\n%s\nsecond:\n%s", loki1, loki2)
	}
	if !bytes.Equal(prom1, prom2) {
		t.Errorf("promtail.yml differs across rebuilds — not idempotent")
	}
}
