package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/nself-org/cli/internal/errs"
)

// ParseManifest reads a plugin.json file at the given path and returns a
// validated PluginManifest. It returns an error if the file cannot be read,
// is not valid JSON, or is missing any required field (name, version,
// description, category, license).
func parseManifest(path string) (*PluginManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading plugin manifest: %w", err)
	}

	var m PluginManifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parsing plugin manifest: %w", err)
	}

	if err := validateRequiredFields(&m); err != nil {
		return nil, err
	}

	if err := validateManifest(&m); err != nil {
		return nil, err
	}

	return &m, nil
}

// validateRequiredFields checks that all required manifest fields are present.
func validateRequiredFields(m *PluginManifest) error {
	var missing []string

	if m.Name == "" {
		missing = append(missing, "name")
	}
	if m.Version == "" {
		missing = append(missing, "version")
	}
	if m.Description == "" {
		missing = append(missing, "description")
	}
	if m.Category == "" {
		missing = append(missing, "category")
	}
	if m.License == "" {
		missing = append(missing, "license")
	}

	if len(missing) > 0 {
		return fmt.Errorf("plugin manifest missing required fields %v: %w", missing, errs.ErrPluginManifest)
	}

	return nil
}

var (
	namePattern    = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
	versionPattern = regexp.MustCompile(`^v?[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?(\+[a-zA-Z0-9.]+)?$`)
)

// validPluginStatuses enumerates the 6 canonical lifecycle values (S58-T01).
// Absent status defaults to "stable" (backwards compatible — CR-A).
var validPluginStatuses = map[string]bool{
	"experimental": true,
	"planned":      true,
	"beta":         true,
	"stable":       true,
	"deprecated":   true,
	"eol":          true,
}

// validLanguages lists allowed values for the Language field.
var validLanguages = map[string]bool{
	"go":         true,
	"rust":       true,
	"typescript": true,
	"python":     true,
	"bash":       true,
}

// validateManifest performs strict validation of manifest field values beyond
// simple presence checks. It validates name format, semver version, port
// range, table name prefixes, and language enum.
func validateManifest(m *PluginManifest) error {
	if !namePattern.MatchString(m.Name) {
		return fmt.Errorf("plugin name must be lowercase with hyphens only: %w", errs.ErrPluginManifest)
	}

	if !versionPattern.MatchString(m.Version) {
		return fmt.Errorf("version must be semver format (X.Y.Z, optional pre-release -suffix and build +suffix): %w", errs.ErrPluginManifest)
	}

	if m.Port > 0 && m.Port >= 65536 {
		return fmt.Errorf("port must be between 1 and 65535: %w", errs.ErrPluginManifest)
	}

	for _, table := range m.Tables {
		if !strings.HasPrefix(table, "np_") {
			return fmt.Errorf("table names must start with np_ prefix: %w", errs.ErrPluginManifest)
		}
	}

	if m.Language != "" && !validLanguages[m.Language] {
		return fmt.Errorf("language must be one of: go, rust, typescript, python, bash: %w", errs.ErrPluginManifest)
	}

	// Validate status field (S58-T01): 6 valid values; absent defaults to stable.
	if m.PublishStatus != "" && !validPluginStatuses[m.PublishStatus] {
		validList := "experimental, planned, beta, stable, deprecated, eol"
		return fmt.Errorf("invalid plugin status %q: must be one of: %s: %w", m.PublishStatus, validList, errs.ErrPluginManifest)
	}

	// Validate deprecation block (S58-T02): required when status == "deprecated".
	if m.PublishStatus == "deprecated" {
		if m.Deprecation == nil {
			return fmt.Errorf("status \"deprecated\" requires a deprecation block with announcedDate, eolDate, and migrationGuide: %w", errs.ErrPluginManifest)
		}
		if m.Deprecation.AnnouncedDate == "" {
			return fmt.Errorf("deprecation.announcedDate is required: %w", errs.ErrPluginManifest)
		}
		if m.Deprecation.EOLDate == "" {
			return fmt.Errorf("deprecation.eolDate is required: %w", errs.ErrPluginManifest)
		}
		if m.Deprecation.MigrationGuide == "" {
			return fmt.Errorf("deprecation.migrationGuide is required: %w", errs.ErrPluginManifest)
		}
	}

	// Validate consumes/provides entries: each must be a valid plugin name.
	for _, dep := range m.Consumes {
		if !namePattern.MatchString(dep) {
			return fmt.Errorf("consumes entry %q is not a valid plugin name (lowercase-with-hyphens): %w", dep, errs.ErrPluginManifest)
		}
	}
	for _, dep := range m.Provides {
		if !namePattern.MatchString(dep) {
			return fmt.Errorf("provides entry %q is not a valid plugin name (lowercase-with-hyphens): %w", dep, errs.ErrPluginManifest)
		}
	}

	return nil
}
