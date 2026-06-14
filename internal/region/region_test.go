package region

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestAddAndPrimary(t *testing.T) {
	cfg := &Config{RoutingStrategy: RoutingPrimaryOnly}

	r1 := Region{ID: "fsn1", Primary: true, PgURL: "postgres://localhost/db"}
	if err := cfg.Add(r1); err != nil {
		t.Fatalf("Add fsn1: %v", err)
	}

	r2 := Region{ID: "hel1", PgURL: "postgres://hel1/db"}
	if err := cfg.Add(r2); err != nil {
		t.Fatalf("Add hel1: %v", err)
	}

	// Duplicate should fail.
	if err := cfg.Add(r1); err == nil {
		t.Fatal("expected error adding duplicate region")
	}

	p := cfg.Primary()
	if p == nil || p.ID != "fsn1" {
		t.Fatalf("expected primary=fsn1, got %v", p)
	}

	replicas := cfg.ReplicaIDs()
	if replicas != "hel1" {
		t.Fatalf("expected replicas=hel1, got %q", replicas)
	}
}

func TestPromote(t *testing.T) {
	cfg := &Config{RoutingStrategy: RoutingLatency}
	cfg.Regions = []Region{
		{ID: "fsn1", Primary: true, AddedAt: time.Now()},
		{ID: "hel1", Primary: false, AddedAt: time.Now()},
	}

	if err := cfg.Promote("hel1"); err != nil {
		t.Fatalf("Promote: %v", err)
	}

	p := cfg.Primary()
	if p == nil || p.ID != "hel1" {
		t.Fatalf("expected primary=hel1 after promote, got %v", p)
	}

	// fsn1 must now be non-primary.
	for _, r := range cfg.Regions {
		if r.ID == "fsn1" && r.Primary {
			t.Fatal("fsn1 should not be primary after hel1 promoted")
		}
	}
}

func TestPromoteNotFound(t *testing.T) {
	cfg := &Config{}
	if err := cfg.Promote("nonexistent"); err == nil {
		t.Fatal("expected error promoting unknown region")
	}
}

func TestSaveLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "regions.json")
	t.Setenv("NSELF_REGION_CONFIG", path)

	cfg := &Config{
		RoutingStrategy: RoutingLatency,
		Regions: []Region{
			{ID: "fsn1", Primary: true, PgURL: "postgres://a/db", AddedAt: time.Now()},
		},
	}
	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify mode is 0600 (Unix only — Windows always reports 0666).
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("stat: %v", err)
		}
		if info.Mode().Perm() != 0600 {
			t.Fatalf("expected mode 0600, got %v", info.Mode().Perm())
		}
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.RoutingStrategy != RoutingLatency {
		t.Fatalf("expected latency routing, got %v", loaded.RoutingStrategy)
	}
	if len(loaded.Regions) != 1 || loaded.Regions[0].ID != "fsn1" {
		t.Fatalf("unexpected regions: %+v", loaded.Regions)
	}
}

func TestLoadMissing(t *testing.T) {
	t.Setenv("NSELF_REGION_CONFIG", "/tmp/nself-test-region-missing-9999.json")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load of missing file should not error: %v", err)
	}
	if cfg.RoutingStrategy != RoutingPrimaryOnly {
		t.Fatalf("expected primary-only default, got %v", cfg.RoutingStrategy)
	}
}

func TestEnvVars(t *testing.T) {
	cfg := &Config{
		RoutingStrategy: RoutingLatency,
		Regions: []Region{
			{ID: "fsn1", Primary: true},
			{ID: "hel1", Primary: false},
		},
	}
	vars := cfg.EnvVars()
	if vars["NSELF_REGION"] != "fsn1" {
		t.Errorf("NSELF_REGION: got %q", vars["NSELF_REGION"])
	}
	if vars["NSELF_PRIMARY_REGION"] != "fsn1" {
		t.Errorf("NSELF_PRIMARY_REGION: got %q", vars["NSELF_PRIMARY_REGION"])
	}
	if vars["NSELF_REPLICA_REGIONS"] != "hel1" {
		t.Errorf("NSELF_REPLICA_REGIONS: got %q", vars["NSELF_REPLICA_REGIONS"])
	}
	if vars["NSELF_ROUTING_STRATEGY"] != "latency" {
		t.Errorf("NSELF_ROUTING_STRATEGY: got %q", vars["NSELF_ROUTING_STRATEGY"])
	}
}
