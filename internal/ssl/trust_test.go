package ssl

import (
	"os"
	"testing"
)

func TestIsCAInstalled_NonexistentPath(t *testing.T) {
	installed, err := IsCAInstalled("/tmp/nonexistent-mkcert-ca-99999.pem")
	if err != nil {
		t.Fatalf("expected no error for nonexistent path, got: %v", err)
	}
	if installed {
		t.Error("expected IsCAInstalled to return false for nonexistent CA path")
	}
}

func TestIsCAInstalled_EmptyFile(t *testing.T) {
	// Create a temporary file that exists but is empty/invalid.
	f, err := os.CreateTemp("", "mkcert-ca-test-*.pem")
	if err != nil {
		t.Fatalf("creating temp file: %v", err)
	}
	defer os.Remove(f.Name())
	f.Close()

	// Should return false (not trusted), not an error.
	installed, err := IsCAInstalled(f.Name())
	if err != nil {
		t.Fatalf("unexpected error for empty CA file: %v", err)
	}
	// An empty file is not in the trust store.
	if installed {
		t.Error("expected IsCAInstalled to return false for an untrusted CA file")
	}
}

func TestMkcertCACertPath_Format(t *testing.T) {
	if !mkcertAvailable() {
		t.Skip("mkcert not available")
	}

	caPath, err := MkcertCACertPath()
	if err != nil {
		t.Fatalf("MkcertCACertPath: %v", err)
	}
	if caPath == "" {
		t.Error("MkcertCACertPath returned empty string")
	}
	// Path must end with rootCA.pem.
	if len(caPath) < 10 || caPath[len(caPath)-10:] != "rootCA.pem" {
		t.Errorf("expected path to end with rootCA.pem, got: %s", caPath)
	}
}

func TestManualInstallCommand_NotEmpty(t *testing.T) {
	cmd := ManualInstallCommand("/tmp/rootCA.pem")
	if cmd == "" {
		t.Error("ManualInstallCommand returned empty string")
	}
}
