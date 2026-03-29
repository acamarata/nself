package config

import (
	"fmt"
	"log/slog"
	"os"
)

// ParseRemoteSchemas parses REMOTE_SCHEMA_1 through REMOTE_SCHEMA_10 environment
// variables into RemoteSchema structs. Each remote schema is defined by a set of
// per-index env vars:
//
//	REMOTE_SCHEMA_N_NAME     — required (skip if empty)
//	REMOTE_SCHEMA_N_URL      — must be http:// or https:// if set; hard error otherwise
//	REMOTE_SCHEMA_N_HEADERS  — format: key:val,key:val
//
// The Name field is sanitized via SanitizeName: path separators, spaces, and
// special characters are stripped. If the result is empty the entry is rejected
// and skipped to prevent nginx or Hasura config injection.
// The URL field is validated via validateRemoteSchemaURL: only http/https schemes
// are accepted to prevent javascript: and other unsafe scheme injection.
func parseRemoteSchemas() ([]RemoteSchema, error) {
	var schemas []RemoteSchema
	for i := 1; i <= 10; i++ {
		prefix := fmt.Sprintf("REMOTE_SCHEMA_%d_", i)
		name := os.Getenv(prefix + "NAME")
		if name == "" {
			continue // No schema at this index
		}

		sanitized, err := SanitizeName(name)
		if err != nil {
			slog.Warn("remote schema name rejected — contains invalid characters; skipping",
				"index", i,
				"env_var", prefix+"NAME",
				"raw_name", name,
				"reason", err.Error(),
			)
			continue
		}

		rawURL := os.Getenv(prefix + "URL")
		if err := validateRemoteSchemaURL(rawURL); err != nil {
			return nil, fmt.Errorf("REMOTE_SCHEMA_%d_URL: %w", i, err)
		}

		schema := RemoteSchema{
			Index:   i,
			Name:    sanitized,
			URL:     rawURL,
			Headers: os.Getenv(prefix + "HEADERS"),
		}
		schemas = append(schemas, schema)
	}
	return schemas, nil
}
