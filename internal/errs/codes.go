package errs

// CodeEntry defines a registered error code with default guidance fields.
type CodeEntry struct {
	// Category groups related errors (e.g., "docker", "config", "plugin").
	Category string

	// Summary is a short description of this error class.
	Summary string

	// DefaultWhy is the default root-cause explanation.
	DefaultWhy string

	// DefaultFix is the default remediation steps.
	DefaultFix string

	// DocsPath is the docs URL path fragment.
	DocsPath string
}

// Registry maps error codes (E001..E999) to their metadata.
// New codes must be added here before use. Keep codes sequential within each category.
var Registry = map[string]CodeEntry{
	// ── Docker (E001-E049) ───────────────────────────────────────────────────

	"E001": {
		Category:   "docker",
		Summary:    "Docker not installed",
		DefaultWhy: "The docker binary was not found in PATH.",
		DefaultFix: "Install Docker: https://docs.docker.com/get-docker/",
		DocsPath:   "reference/error-codes#e001",
	},
	"E002": {
		Category:   "docker",
		Summary:    "Docker daemon not running",
		DefaultWhy: "The Docker daemon is not responding to commands.",
		DefaultFix: "Start Docker Desktop or run: sudo systemctl start docker",
		DocsPath:   "reference/error-codes#e002",
	},
	"E003": {
		Category:   "docker",
		Summary:    "Docker Compose not available",
		DefaultWhy: "docker compose v2 plugin is not installed.",
		DefaultFix: "Update Docker Desktop or install the compose plugin manually.",
		DocsPath:   "reference/error-codes#e003",
	},
	"E004": {
		Category:   "docker",
		Summary:    "docker-compose.yml not found",
		DefaultWhy: "No docker-compose.yml exists in the project directory.",
		DefaultFix: "Run 'nself build' to generate the compose file.",
		DocsPath:   "reference/error-codes#e004",
	},
	"E005": {
		Category:   "docker",
		Summary:    "Port conflict",
		DefaultWhy: "A required port is already in use by another process.",
		DefaultFix: "Run 'nself doctor' to identify the conflict, then stop the conflicting process or change the port in .env.",
		DocsPath:   "reference/error-codes#e005",
	},

	// ── Config (E050-E099) ───────────────────────────────────────────────────

	"E050": {
		Category:   "config",
		Summary:    "No .env file found",
		DefaultWhy: "No .env or .env.dev file exists in the project directory.",
		DefaultFix: "Run 'nself init' to generate a configuration file.",
		DocsPath:   "reference/error-codes#e050",
	},
	"E051": {
		Category:   "config",
		Summary:    "Invalid config key",
		DefaultWhy: "The configuration key contains invalid characters.",
		DefaultFix: "Config keys must contain only A-Z, 0-9, and underscores.",
		DocsPath:   "reference/error-codes#e051",
	},
	"E052": {
		Category:   "config",
		Summary:    "Config validation failed",
		DefaultWhy: "One or more configuration values are invalid.",
		DefaultFix: "Run 'nself config validate' to see all issues, then fix the reported values.",
		DocsPath:   "reference/error-codes#e052",
	},
	"E053": {
		Category:   "config",
		Summary:    "Weak password detected",
		DefaultWhy: "A password field does not meet minimum length or contains insecure patterns.",
		DefaultFix: "Use a strong, randomly generated password of at least 16 characters.",
		DocsPath:   "reference/error-codes#e053",
	},
	"E054": {
		Category:   "config",
		Summary:    "Invalid project name",
		DefaultWhy: "Project name contains characters not allowed in Docker and DNS contexts.",
		DefaultFix: "Use only lowercase letters, numbers, and hyphens. Must start with a letter.",
		DocsPath:   "reference/error-codes#e054",
	},
	"E055": {
		Category:   "config",
		Summary:    "Duplicate route detected",
		DefaultWhy: "Two or more services are configured with the same nginx route.",
		DefaultFix: "Check ROUTE values in .env and ensure each service has a unique route.",
		DocsPath:   "reference/error-codes#e055",
	},
	"E056": {
		Category:   "config",
		Summary:    "Unknown config key",
		DefaultWhy: "The key is not recognized by nself.",
		DefaultFix: "Check for typos. Run 'nself config list' to see all valid keys.",
		DocsPath:   "reference/error-codes#e056",
	},

	// ── Plugin / License (E100-E149) ─────────────────────────────────────────

	"E100": {
		Category:   "plugin",
		Summary:    "Plugin not found",
		DefaultWhy: "The requested plugin does not exist in the registry.",
		DefaultFix: "Run 'nself plugin list' to see available plugins.",
		DocsPath:   "reference/error-codes#e100",
	},
	"E101": {
		Category:   "plugin",
		Summary:    "Invalid license key",
		DefaultWhy: "The license key format is invalid.",
		DefaultFix: "License keys start with 'nself_pro_' followed by 32+ characters. Check your key.",
		DocsPath:   "reference/error-codes#e101",
	},
	"E102": {
		Category:   "plugin",
		Summary:    "License tier insufficient",
		DefaultWhy: "Your license tier does not include this plugin.",
		DefaultFix: "Upgrade your plan at https://nself.org/pricing",
		DocsPath:   "reference/error-codes#e102",
	},
	"E103": {
		Category:   "plugin",
		Summary:    "License expired",
		DefaultWhy: "The license key has expired.",
		DefaultFix: "Renew your license at https://nself.org/account",
		DocsPath:   "reference/error-codes#e103",
	},
	"E104": {
		Category:   "plugin",
		Summary:    "License validation failed (network)",
		DefaultWhy: "Cannot reach the license server and no valid local cache exists.",
		DefaultFix: "Check your internet connection. Previously validated licenses work offline for 7 days.",
		DocsPath:   "reference/error-codes#e104",
	},
	"E105": {
		Category:   "plugin",
		Summary:    "Circular plugin dependency",
		DefaultWhy: "Plugin dependency graph contains a cycle.",
		DefaultFix: "Report this as a bug at https://github.com/nself-org/cli/issues",
		DocsPath:   "reference/error-codes#e105",
	},

	// ── SSL / Network (E150-E199) ────────────────────────────────────────────

	"E150": {
		Category:   "ssl",
		Summary:    "mkcert not installed",
		DefaultWhy: "mkcert is not installed; falling back to OpenSSL self-signed certs.",
		DefaultFix: "Install mkcert: brew install mkcert (macOS) or see https://github.com/FiloSottile/mkcert",
		DocsPath:   "reference/error-codes#e150",
	},
	"E151": {
		Category:   "ssl",
		Summary:    "SSL certificate generation failed",
		DefaultWhy: "Could not generate SSL certificates for the configured domain.",
		DefaultFix: "Check domain configuration and ensure openssl is available.",
		DocsPath:   "reference/error-codes#e151",
	},

	// ── Database (E200-E249) ─────────────────────────────────────────────────

	"E200": {
		Category:   "database",
		Summary:    "Database not running",
		DefaultWhy: "PostgreSQL container is not running or not accepting connections.",
		DefaultFix: "Run 'nself start' to start all services, or 'nself doctor' to diagnose.",
		DocsPath:   "reference/error-codes#e200",
	},
	"E201": {
		Category:   "database",
		Summary:    "Migration failed",
		DefaultWhy: "A database migration could not be applied.",
		DefaultFix: "Check the migration SQL for errors. Run 'nself db migrate --status' to see pending migrations.",
		DocsPath:   "reference/error-codes#e201",
	},
	"E202": {
		Category:   "database",
		Summary:    "Backup failed",
		DefaultWhy: "Database backup operation failed.",
		DefaultFix: "Ensure sufficient disk space and that the database is running.",
		DocsPath:   "reference/error-codes#e202",
	},

	// ── Health (E250-E299) ───────────────────────────────────────────────────

	"E250": {
		Category:   "health",
		Summary:    "Service unhealthy",
		DefaultWhy: "A service health check returned an unhealthy status.",
		DefaultFix: "Run 'nself doctor --verbose' for detailed diagnostics.",
		DocsPath:   "reference/error-codes#e250",
	},
	"E251": {
		Category:   "health",
		Summary:    "Health check timeout",
		DefaultWhy: "The health check did not complete within the timeout period.",
		DefaultFix: "The service may be starting slowly. Wait and retry, or check logs with 'nself logs'.",
		DocsPath:   "reference/error-codes#e251",
	},

	// ── Init / Project (E300-E349) ───────────────────────────────────────────

	"E300": {
		Category:   "init",
		Summary:    "Project already initialized",
		DefaultWhy: "A .env file already exists in this directory.",
		DefaultFix: "Use 'nself config set' to modify existing config, or delete .env to reinitialize.",
		DocsPath:   "reference/error-codes#e300",
	},
	"E301": {
		Category:   "init",
		Summary:    "Source directory detected",
		DefaultWhy: "You are running nself inside the CLI source repository.",
		DefaultFix: "Change to your project directory first: cd /path/to/your/project",
		DocsPath:   "reference/error-codes#e301",
	},

	// ── Domain (E350-E399) ───────────────────────────────────────────────────

	"E350": {
		Category:   "domain",
		Summary:    "Invalid domain name",
		DefaultWhy: "The configured domain name is not valid.",
		DefaultFix: "Use a valid domain like 'example.com' or 'localhost'.",
		DocsPath:   "reference/error-codes#e350",
	},
	"E351": {
		Category:   "domain",
		Summary:    "Invalid port number",
		DefaultWhy: "Port number is outside the valid range (1-65535).",
		DefaultFix: "Use a port number between 1024 and 65535 for non-privileged ports.",
		DocsPath:   "reference/error-codes#e351",
	},
}

// Categories returns all unique category names from the registry, sorted.
func Categories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, entry := range Registry {
		if !seen[entry.Category] {
			seen[entry.Category] = true
			cats = append(cats, entry.Category)
		}
	}
	return cats
}
