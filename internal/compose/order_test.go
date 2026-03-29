package compose

import (
	"strings"
	"testing"

	"nself/internal/config"
)

func TestServiceOrder(t *testing.T) {
	cfg := &config.Config{
		ProjectName: "test",
		Minio:       config.MinioConfig{Enabled: true},
		Search:      config.SearchConfig{Enabled: true, Engine: "meilisearch"},
	}
	cfg = config.ApplyDefaults(cfg)

	gen := NewGenerator(cfg)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	yaml := string(data)
	
	// minio-init MUST appear before minio
	initIdx := strings.Index(yaml, "\n    minio-init:")
	svcIdx := strings.Index(yaml, "\n    minio:")
	if initIdx < 0 || svcIdx < 0 {
		t.Fatal("minio-init or minio not found in yaml")
	}
	if initIdx > svcIdx {
		t.Errorf("minio-init (pos %d) appears AFTER minio (pos %d)", initIdx, svcIdx)
	} else {
		t.Logf("✅ minio-init (pos %d) appears BEFORE minio (pos %d)", initIdx, svcIdx)
	}

	// postgres MUST appear before hasura
	pgIdx := strings.Index(yaml, "\n    postgres:")
	haIdx := strings.Index(yaml, "\n    hasura:")
	if pgIdx > haIdx {
		t.Errorf("postgres (pos %d) appears AFTER hasura (pos %d)", pgIdx, haIdx)
	} else {
		t.Logf("✅ postgres (pos %d) appears BEFORE hasura (pos %d)", pgIdx, haIdx)
	}
	
	// Print first few service names
	lines := strings.Split(yaml, "\n")
	t.Log("Service order:")
	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		if strings.HasSuffix(trimmed, ":") && !strings.Contains(trimmed, " ") && len(trimmed) > 2 {
			if !strings.HasSuffix(trimmed, "_network:") && !strings.HasSuffix(trimmed, "_data:") && !strings.HasSuffix(trimmed, "_cache:") {
				t.Logf("  %s", trimmed)
			}
		}
	}
}
