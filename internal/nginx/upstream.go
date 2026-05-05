// Package nginx provides Nginx configuration generation and management for the nSelf CLI.
// This file implements a thin wrapper for blue/green upstream weight management (T06).
//
// Traffic splitting is achieved via Nginx weighted upstream blocks written to
// nginx/conf.d/bluegreen-upstream.conf and reloaded atomically with nginx -s reload.
// This keeps the nSelf-First doctrine: no direct Docker API calls, no hand-edited
// generated files — all routing goes through the nself CLI surface.
package nginx

import (
	"fmt"
	"strings"
)

// UpstreamConfig holds the parameters for generating a blue/green upstream block.
type UpstreamConfig struct {
	// UpstreamName is the nginx upstream block name. Default: "nself_api".
	UpstreamName string

	// BlueHost is the blue server address. Default: "127.0.0.1".
	BlueHost string

	// BluePort is the blue server port. Default: 8080.
	BluePort int

	// GreenHost is the green server address. Default: "127.0.0.1".
	GreenHost string

	// GreenPort is the green server port. Default: 8180.
	GreenPort int
}

// DefaultUpstreamConfig returns an UpstreamConfig with production defaults.
// Blue at 127.0.0.1:8080, green at 127.0.0.1:8180 (port offset +100).
func DefaultUpstreamConfig() UpstreamConfig {
	return UpstreamConfig{
		UpstreamName: "nself_api",
		BlueHost:     "127.0.0.1",
		BluePort:     8080,
		GreenHost:    "127.0.0.1",
		GreenPort:    8180,
	}
}

// GenerateWeightedUpstream generates an nginx upstream block for blue/green traffic splitting.
// canaryPercent (0-100) controls how much traffic routes to green:
//   - 0:   all traffic to blue (initial state / after rollback)
//   - 1-99: weighted split (canary)
//   - 100: all traffic to green (post-promote)
//
// Inactive servers are included as "backup" to avoid nginx config errors.
func GenerateWeightedUpstream(cfg UpstreamConfig, canaryPercent int) (string, error) {
	if canaryPercent < 0 || canaryPercent > 100 {
		return "", fmt.Errorf("canaryPercent must be 0-100, got %d", canaryPercent)
	}

	name := cfg.UpstreamName
	if name == "" {
		name = "nself_api"
	}

	blueWeight := 100 - canaryPercent
	greenWeight := canaryPercent

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("upstream %s {\n", name))

	if blueWeight > 0 {
		sb.WriteString(fmt.Sprintf("    server %s:%d weight=%d;   # blue\n", cfg.BlueHost, cfg.BluePort, blueWeight))
	} else {
		sb.WriteString(fmt.Sprintf("    server %s:%d weight=0 backup;   # blue (idle)\n", cfg.BlueHost, cfg.BluePort))
	}

	if greenWeight > 0 {
		sb.WriteString(fmt.Sprintf("    server %s:%d weight=%d;   # green (%d%% canary)\n", cfg.GreenHost, cfg.GreenPort, greenWeight, canaryPercent))
	} else {
		sb.WriteString(fmt.Sprintf("    server %s:%d weight=0 backup;   # green (idle)\n", cfg.GreenHost, cfg.GreenPort))
	}

	sb.WriteString("}\n")
	return sb.String(), nil
}
