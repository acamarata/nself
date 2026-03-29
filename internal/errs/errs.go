// Package errs defines sentinel errors for the nself CLI.
//
// Named "errs" (not "errors") to avoid shadowing the Go standard library.
// Other packages import as: nself/internal/errs
// Usage: errs.ErrDockerNotRunning
package errs

import "errors"

var (
	// Docker
	ErrDockerNotRunning   = errors.New("docker daemon is not running")
	ErrDockerNotInstalled = errors.New("docker not found in PATH")
	ErrComposeNotFound    = errors.New("docker-compose.yml not found — run 'nself build' first")
	ErrPortConflict       = errors.New("port already in use")

	// Config
	ErrWeakPassword       = errors.New("password does not meet minimum length")
	ErrInsecurePassword   = errors.New("password matches insecure pattern")
	ErrInvalidProjectName = errors.New("invalid project name format")
	ErrInvalidCORS        = errors.New("CORS wildcards not allowed in production")

	// Health
	ErrServiceUnhealthy = errors.New("service health check failed")
	ErrHealthTimeout    = errors.New("health check timed out")
	ErrServiceNotFound  = errors.New("service not found")

	// Plugin
	ErrInvalidLicenseKey        = errors.New("invalid license key format")
	ErrLicenseTierTooLow        = errors.New("license tier does not include this plugin")
	ErrLicenseExpired           = errors.New("license key is expired")
	ErrLicenseNetworkUnavailable = errors.New("cannot validate license: network unavailable and no valid cache")
	ErrPluginNotFound           = errors.New("plugin not found in registry")
	ErrPluginManifest           = errors.New("invalid plugin manifest")
	ErrCircularDependency       = errors.New("circular plugin dependency detected")

	// SSL
	ErrMkcertNotFound      = errors.New("mkcert not installed — falling back to OpenSSL")
	ErrSSLGenerationFailed = errors.New("SSL certificate generation failed")

	// Nginx
	ErrDuplicateRoute = errors.New("duplicate nginx route detected")

	// Domain / Port
	ErrInvalidDomain = errors.New("invalid domain name")
	ErrInvalidPort   = errors.New("invalid port number")

	// Database
	ErrDatabaseNotRunning = errors.New("database is not running")
	ErrMigrationFailed    = errors.New("database migration failed")
	ErrBackupFailed       = errors.New("database backup failed")
)
