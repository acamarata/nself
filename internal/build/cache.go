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

// buildProfileFile records the service profile that produced the last build.
//
// NeedsRebuild compares .env against docker-compose.yml by mtime and checks the
// CLI version, but the profile is neither of those: it arrives as a flag. So
// `nself build --profile ops` on a project last built as "app" found the cache
// fresh, regenerated nothing, exited 0, and left every app service in the
// compose file. The operator asked for an ops server and kept redis, minio and
// mailpit, with nothing in the output saying so.
const buildProfileFile = ".nself/build-profile"

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
// ProfileChanged reports whether the requested service profile differs from
// the one recorded by the last build. An unreadable or missing record counts as
// changed: the safe answer is to rebuild, since a stale compose file for the
// wrong profile is the failure this exists to prevent.
//
// Kept separate from NeedsRebuild so the three existing callers that do not
// know the profile keep working unchanged.
func ProfileChanged(workdir, profile string) bool {
	if profile == "" {
		profile = "app"
	}
	recorded, err := os.ReadFile(filepath.Join(workdir, buildProfileFile))
	if err != nil {
		return true
	}
	return strings.TrimSpace(string(recorded)) != profile
}

// RecordProfile stores the profile that produced this build.
func RecordProfile(workdir, profile string) error {
	if profile == "" {
		profile = "app"
	}
	return os.WriteFile(filepath.Join(workdir, buildProfileFile), []byte(profile), 0644)
}

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
