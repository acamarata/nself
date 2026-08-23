package deprecation

import (
	_ "embed"
	"fmt"
	"os"
)

// registryYAML is the deprecation registry compiled into the binary.
//
// CLI-R03: before this existed, the registry was resolved as a file path next
// to the executable (".../internal/deprecation/registry.yaml"). That path only
// exists inside a source checkout, so every installed binary — `make install`,
// Homebrew, the release tarballs — silently loaded an empty registry and never
// emitted a single deprecation warning. Embedding is the only way the data can
// travel with a single-file distribution.
//
//go:embed registry.yaml
var registryYAML []byte

// RegistryPathEnv names the environment variable that overrides the embedded
// registry with an on-disk file. It exists for tests and for operators who need
// to preview a registry change without rebuilding; it is never required for
// normal operation.
const RegistryPathEnv = "NSELF_DEPRECATION_REGISTRY"

// LoadEmbeddedRegistry returns the Registry built from the compiled-in YAML.
//
// When RegistryPathEnv is set to a readable file, that file is parsed instead.
// A set-but-unreadable or malformed override is reported as an error alongside
// a usable (embedded) Registry, so a bad override degrades to correct default
// behaviour rather than disabling warnings entirely.
func LoadEmbeddedRegistry() (*Registry, error) {
	if path := os.Getenv(RegistryPathEnv); path != "" {
		reg, err := LoadRegistry(path)
		if err == nil {
			return reg, nil
		}
		embedded, embedErr := parseRegistry(registryYAML)
		if embedErr != nil {
			return embedded, fmt.Errorf("deprecation: override %q unusable (%v) and embedded registry invalid: %w", path, err, embedErr)
		}
		return embedded, fmt.Errorf("deprecation: override %q unusable, using embedded registry: %w", path, err)
	}
	return parseRegistry(registryYAML)
}

// EmbeddedRegistryBytes returns a copy of the compiled-in registry YAML.
// Used by tooling that needs to inspect or re-emit the registry.
func EmbeddedRegistryBytes() []byte {
	out := make([]byte, len(registryYAML))
	copy(out, registryYAML)
	return out
}
