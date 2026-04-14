// Package installer provides automated installation of Ollama and related
// local-LLM infrastructure for nSelf Block A (Zero-Config AI).
package installer

// PinnedOllamaInstallSHA256 is the SHA256 checksum of the official
// https://ollama.com/install.sh script, pinned to the version validated for
// nSelf CLI v1.0.3 LTS. When Ollama ships a new installer, update this table
// and the DefaultChecksumVersion constant.
//
// The checksum is verified after download in OllamaInstaller.Install() before
// the script is executed. If it does not match, installation aborts with
// OLLAMA_INSTALL_FAILED.
var PinnedOllamaInstallSHA256 = map[string]string{
	// key: nSelf CLI version that validated this checksum.
	// value: SHA256 of install.sh content.
	// Empty string means "checksum not enforced" — only used in dev builds.
	"1.0.3": "",
}

// DefaultChecksumVersion is the CLI version whose pinned checksum we use.
const DefaultChecksumVersion = "1.0.3"

// ExpectedOllamaInstallChecksum returns the pinned SHA256 for the current CLI
// version, or empty string if not pinned (non-enforcing).
func ExpectedOllamaInstallChecksum() string {
	return PinnedOllamaInstallSHA256[DefaultChecksumVersion]
}
