// Package region implements multi-region awareness for nSelf deployments (B49).
//
// All code paths in this package are gated by the feature flag
// "multi_region_enabled". When the flag is off (default), all traffic
// continues to route to the primary region as before — zero user impact.
//
// Status: PLANNED — deferred. Requires UD-12 minor release approval.
// Dependencies: B46 (multi-tenant controller), B47/B48 (blue-green deploy).
package region

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"
)

// FeatureFlag is the flag key that gates all multi-region behavior.
const FeatureFlag = "multi_region_enabled"

// RoutingStrategy controls how reads are distributed across regions.
type RoutingStrategy string

const (
	// RoutingLatency routes reads to the nearest region based on latency.
	RoutingLatency RoutingStrategy = "latency"
	// RoutingRoundRobin distributes reads evenly across regions.
	RoutingRoundRobin RoutingStrategy = "round-robin"
	// RoutingPrimaryOnly forces all traffic to the primary region.
	RoutingPrimaryOnly RoutingStrategy = "primary-only"
)

// Region describes a single nSelf deployment region.
type Region struct {
	ID       string    `json:"id"`
	Primary  bool      `json:"primary"`
	PgURL    string    `json:"pg_url"`
	RedisURL string    `json:"redis_url,omitempty"`
	AddedAt  time.Time `json:"added_at"`
}

// Config holds the persisted multi-region configuration.
type Config struct {
	Regions         []Region        `json:"regions"`
	RoutingStrategy RoutingStrategy `json:"routing_strategy"`
}

// configPath returns the path to the persisted region config file.
func configPath() string {
	if p := os.Getenv("NSELF_REGION_CONFIG"); p != "" {
		return p
	}
	home, _ := os.UserHomeDir()
	return fmt.Sprintf("%s/.nself/regions.json", home)
}

// Load reads the region configuration from disk.
// Returns an empty Config with primary-only routing if the file does not exist.
func Load() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &Config{RoutingStrategy: RoutingPrimaryOnly}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("region: read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("region: parse config: %w", err)
	}
	return &cfg, nil
}

// Save writes the region configuration to disk (mode 0600).
func (c *Config) Save() error {
	path := configPath()
	if err := os.MkdirAll(dirOf(path), 0700); err != nil {
		return fmt.Errorf("region: mkdir: %w", err)
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("region: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("region: write config: %w", err)
	}
	return nil
}

// Add appends a new region.
// Returns an error if a region with the same ID already exists.
func (c *Config) Add(r Region) error {
	for _, existing := range c.Regions {
		if existing.ID == r.ID {
			return fmt.Errorf("region: region %q already exists", r.ID)
		}
	}
	r.AddedAt = time.Now().UTC()
	c.Regions = append(c.Regions, r)
	return nil
}

// Promote makes the named region the new primary. The previous primary
// is demoted to a replica. Returns an error if the region is not found.
func (c *Config) Promote(id string) error {
	found := false
	for i := range c.Regions {
		if c.Regions[i].ID == id {
			c.Regions[i].Primary = true
			found = true
		} else {
			c.Regions[i].Primary = false
		}
	}
	if !found {
		return fmt.Errorf("region: region %q not found", id)
	}
	return nil
}

// Primary returns the primary region. Returns nil if none is marked primary.
func (c *Config) Primary() *Region {
	for i := range c.Regions {
		if c.Regions[i].Primary {
			return &c.Regions[i]
		}
	}
	return nil
}

// ReplicaIDs returns a comma-separated list of non-primary region IDs,
// matching the NSELF_REPLICA_REGIONS env var format.
func (c *Config) ReplicaIDs() string {
	var ids []string
	for _, r := range c.Regions {
		if !r.Primary {
			ids = append(ids, r.ID)
		}
	}
	return strings.Join(ids, ",")
}

// EnvVars returns the environment variables that should be set for
// region-aware routing. Used by nself build to inject into docker-compose.
func (c *Config) EnvVars() map[string]string {
	vars := map[string]string{
		"NSELF_ROUTING_STRATEGY": string(c.RoutingStrategy),
	}
	if p := c.Primary(); p != nil {
		vars["NSELF_REGION"] = p.ID
		vars["NSELF_PRIMARY_REGION"] = p.ID
	}
	if replicas := c.ReplicaIDs(); replicas != "" {
		vars["NSELF_REPLICA_REGIONS"] = replicas
	}
	return vars
}

func dirOf(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return "."
	}
	return path[:idx]
}
