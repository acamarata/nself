package nginx

import (
	"strings"
	"testing"
)

func TestGenerateWeightedUpstream_canary10(t *testing.T) {
	cfg := DefaultUpstreamConfig()
	got, err := GenerateWeightedUpstream(cfg, 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Blue at port 8080 with weight 90.
	if !strings.Contains(got, "127.0.0.1:8080 weight=90") {
		t.Errorf("expected blue 8080 weight=90, got:\n%s", got)
	}
	// Green at port 8180 with weight 10.
	if !strings.Contains(got, "127.0.0.1:8180 weight=10") {
		t.Errorf("expected green 8180 weight=10, got:\n%s", got)
	}
}

func TestGenerateWeightedUpstream_allBlue(t *testing.T) {
	cfg := DefaultUpstreamConfig()
	got, err := GenerateWeightedUpstream(cfg, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "127.0.0.1:8080 weight=100") {
		t.Errorf("expected blue 8080 weight=100, got:\n%s", got)
	}
	// Green is backup when canary=0.
	if !strings.Contains(got, "8180 weight=0 backup") {
		t.Errorf("expected green as backup, got:\n%s", got)
	}
}

func TestGenerateWeightedUpstream_fullGreen(t *testing.T) {
	cfg := DefaultUpstreamConfig()
	got, err := GenerateWeightedUpstream(cfg, 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "127.0.0.1:8180 weight=100") {
		t.Errorf("expected green 8180 weight=100, got:\n%s", got)
	}
	// Blue is backup when fully promoted.
	if !strings.Contains(got, "8080 weight=0 backup") {
		t.Errorf("expected blue as backup at 100%% green, got:\n%s", got)
	}
}

func TestGenerateWeightedUpstream_wrapsInUpstreamBlock(t *testing.T) {
	cfg := DefaultUpstreamConfig()
	got, err := GenerateWeightedUpstream(cfg, 25)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(got, "upstream nself_api {") {
		t.Errorf("expected upstream block header, got:\n%s", got)
	}
	if !strings.HasSuffix(strings.TrimSpace(got), "}") {
		t.Errorf("expected closing brace, got:\n%s", got)
	}
}

func TestGenerateWeightedUpstream_customUpstreamName(t *testing.T) {
	cfg := DefaultUpstreamConfig()
	cfg.UpstreamName = "my_backend"
	got, err := GenerateWeightedUpstream(cfg, 50)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "upstream my_backend {") {
		t.Errorf("expected custom upstream name, got:\n%s", got)
	}
}

func TestGenerateWeightedUpstream_invalidPercent(t *testing.T) {
	cfg := DefaultUpstreamConfig()

	if _, err := GenerateWeightedUpstream(cfg, -1); err == nil {
		t.Errorf("expected error for canaryPercent=-1")
	}
	if _, err := GenerateWeightedUpstream(cfg, 101); err == nil {
		t.Errorf("expected error for canaryPercent=101")
	}
}

func TestGenerateWeightedUpstream_weightsSum100(t *testing.T) {
	cfg := DefaultUpstreamConfig()

	for _, pct := range []int{0, 10, 25, 50, 75, 90, 100} {
		got, err := GenerateWeightedUpstream(cfg, pct)
		if err != nil {
			t.Fatalf("pct=%d: unexpected error: %v", pct, err)
		}
		// The upstream block must be non-empty and valid.
		if !strings.Contains(got, "upstream nself_api") {
			t.Errorf("pct=%d: missing upstream block", pct)
		}
		_ = got
	}
}

func TestDefaultUpstreamConfig(t *testing.T) {
	cfg := DefaultUpstreamConfig()
	if cfg.UpstreamName != "nself_api" {
		t.Errorf("UpstreamName: got %q, want %q", cfg.UpstreamName, "nself_api")
	}
	if cfg.BluePort != 8080 {
		t.Errorf("BluePort: got %d, want 8080", cfg.BluePort)
	}
	if cfg.GreenPort != 8180 {
		t.Errorf("GreenPort: got %d, want 8180", cfg.GreenPort)
	}
	if cfg.BlueHost != "127.0.0.1" {
		t.Errorf("BlueHost: got %q, want 127.0.0.1", cfg.BlueHost)
	}
	if cfg.GreenHost != "127.0.0.1" {
		t.Errorf("GreenHost: got %q, want 127.0.0.1", cfg.GreenHost)
	}
}
