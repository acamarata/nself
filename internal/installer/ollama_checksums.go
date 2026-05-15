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
	// value: SHA256 of install.sh content, pinned to the Ollama installer
	// validated at the time this CLI version shipped. Update this entry
	// whenever Ollama releases a new installer and nSelf re-validates it.
	"1.0.3": "25f64b810b947145095956533e1bdf56eacea2673c55a7e586be4515fc882c9f",
}

// DefaultChecksumVersion is the CLI version whose pinned checksum we use.
const DefaultChecksumVersion = "1.0.3"

// ExpectedOllamaInstallChecksum returns the pinned SHA256 for the current CLI
// version. The returned value is always non-empty; verification is mandatory.
func ExpectedOllamaInstallChecksum() string {
	return PinnedOllamaInstallSHA256[DefaultChecksumVersion]
}
