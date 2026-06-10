package compose

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// AdminImagePath is the Docker Hub image name for the nSelf Admin GUI service.
// It uses the Docker Hub nself/ namespace (NOT GitHub Container Registry).
// Intentionally referencing nself/nself-admin (Docker Hub) — never github.com/nself-org/ paths.
const AdminImagePath = "nself/nself-admin"

// DefaultImageVersions maps service name to pinned image:tag.
// Update with each nSelf release.
var DefaultImageVersions = map[string]string{
	"postgres":    "pgvector/pgvector:pg16",
	"hasura":      "hasura/graphql-engine:v2.44.0",
	"auth":        "nhost/hasura-auth:0.36.0",
	"nginx":       "nginx:1.25-alpine",
	"redis":       "redis:7.2-alpine",
	"minio":       "minio/minio:RELEASE.2024-01-16T16-07-38Z",
	"functions":   "nhost/functions:0.3.7",
	"mailpit":     "axllent/mailpit:v1.15",
	"meilisearch": "getmeili/meilisearch:v1.6",
	"typesense":   "typesense/typesense:0.25.2",
	"admin":       AdminImagePath + ":latest", // intentionally latest — our own image
	"mlflow":      "ghcr.io/mlflow/mlflow:v2.10.0",
}

// ImageDigests maps service name to sha256 digest for image pinning.
// When a digest is available, ResolveImage appends @sha256:... to the tag.
// Populated by LoadImageDigests from the project config directory.
var ImageDigests = map[string]string{}

// DigestConfigFile is the filename where image digests are stored.
const DigestConfigFile = ".nself-image-digests.json"

// ResolveImage returns the image tag for a service, optionally with a sha256
// digest suffix when available.
//
// Precedence: a non-empty caller-supplied image (built from env/config such as
// POSTGRES_VERSION / HASURA_VERSION) ALWAYS wins. The DefaultImageVersions pin
// applies only when the caller supplies no image. Pins overriding configured
// versions force-migrated existing deployments on upgrade (P1 EOP staging
// incident 2026-06-10: postgres:16-alpine volume recreated as pgvector pg16).
func ResolveImage(service, image string) string {
	resolved := image
	if resolved == "" {
		if pinned, ok := DefaultImageVersions[service]; ok {
			resolved = pinned
		}
	}
	// Append digest if available for this service.
	if digest, ok := ImageDigests[service]; ok && digest != "" {
		resolved = resolved + "@sha256:" + digest
	}
	return resolved
}

// LoadImageDigests reads digest pins from the project config directory.
// Missing file is not an error (digests are opt-in via `nself update images`).
func LoadImageDigests(projectDir string) error {
	path := filepath.Join(projectDir, DigestConfigFile)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("reading image digests: %w", err)
	}
	var digests map[string]string
	if err := json.Unmarshal(data, &digests); err != nil {
		return fmt.Errorf("parsing image digests: %w", err)
	}
	ImageDigests = digests
	return nil
}

// SaveImageDigests writes digest pins to the project config directory.
func SaveImageDigests(projectDir string, digests map[string]string) error {
	path := filepath.Join(projectDir, DigestConfigFile)
	data, err := json.MarshalIndent(digests, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling image digests: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("writing image digests: %w", err)
	}
	return nil
}
