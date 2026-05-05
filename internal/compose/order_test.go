package compose

import (
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/config"
)

func TestServiceOrder(t *testing.T) {
	cfg := &config.Config{
		ProjectName: "test",
		Minio:       config.MinioConfig{Enabled: true},
		Search:      config.SearchConfig{Enabled: true, Engine: "meilisearch"},
	}
	var applyErr error
	cfg, applyErr = config.ApplyDefaults(cfg)
	if applyErr != nil {
		t.Fatalf("ApplyDefaults() error: %v", applyErr)
	}

	gen := NewGenerator(cfg)
	data, err := gen.Generate()
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}

	yaml := string(data)

	// minio MUST be present in the yaml
	if strings.Index(yaml, "\n    minio:") < 0 {
		t.Fatal("minio not found in yaml")
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
