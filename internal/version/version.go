package version

// Version, Commit, and BuildDate are set via ldflags at build time.
//
//	go build -ldflags "-X nself/internal/version.Version=1.0.0 -X nself/internal/version.Commit=abc123 -X nself/internal/version.BuildDate=2026-01-01"
var (
	Version   string = "1.1.4"
	Commit    string = "unknown"
	BuildDate string = "unknown"
)

// GetVersion returns the build version string.
func GetVersion() string {
	return Version
}

// GetCommit returns the git commit hash.
func GetCommit() string {
	return Commit
}

// GetBuildDate returns the build date.
func GetBuildDate() string {
	return BuildDate
}
