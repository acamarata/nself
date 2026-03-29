package build

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/nself-org/cli/internal/version"
)

// buildVersionFile is the path relative to workdir where the CLI version
// that produced the last build is recorded.
const buildVersionFile = ".nself/build-version"

// NeedsRebuild reports whether the build outputs are stale and a rebuild
// is required. It returns true when any of the following hold:
//
//   - .env or docker-compose.yml does not exist
//   - .env is newer than docker-compose.yml
//   - The CLI version that produced the last build differs from the running version
//   - The build-version file is missing or unreadable
//
// It returns false only when docker-compose.yml is at least as new as .env
// and the recorded build version matches the current CLI version.
func NeedsRebuild(workdir string) (bool, error) {
	envPath := filepath.Join(workdir, ".env")
	composePath := filepath.Join(workdir, "docker-compose.yml")
	versionPath := filepath.Join(workdir, buildVersionFile)

	envInfo, err := os.Stat(envPath)
	if err != nil {
		// .env missing means we cannot compare; rebuild needed.
		return true, nil
	}

	composeInfo, err := os.Stat(composePath)
	if err != nil {
		// docker-compose.yml missing; definitely need a build.
		return true, nil
	}

	// If .env is newer than docker-compose.yml, config has changed.
	if envInfo.ModTime().After(composeInfo.ModTime()) {
		return true, nil
	}

	// Check CLI version that produced the last build.
	data, err := os.ReadFile(versionPath)
	if err != nil {
		// build-version file missing or unreadable; rebuild.
		return true, nil
	}

	recorded := strings.TrimSpace(string(data))
	if recorded != version.GetVersion() {
		return true, nil
	}

	return false, nil
}
