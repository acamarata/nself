package compose

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
	"admin":       "github.com/nself-org/cli/nself-admin:latest", // intentionally latest — our own image
	"mlflow":      "ghcr.io/mlflow/mlflow:v2.10.0",
}

// ResolveImage returns the pinned image tag for a service.
// Falls back to the image string as-is if not in DefaultImageVersions.
func ResolveImage(service, image string) string {
	if pinned, ok := DefaultImageVersions[service]; ok {
		return pinned
	}
	return image
}
