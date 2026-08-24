package sim

// Purpose: container-spec building, key-pair generation, and low-level shell/docker helpers backing the Fleet simulation harness in harness.go.
// Inputs: FleetConfig (for buildSpecs) and *testing.T for the docker-exec helpers.
// Outputs: ContainerSpec lists, an in-memory ED25519 key pair, and docker container lifecycle side effects.
// Constraints: split out of harness.go as a pure move (CLI-R12); no behavior change. See harness.go's package doc for the full harness contract.

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// buildSpecs returns ContainerSpec list: observability first, then app(s), then lb.
// This ordering matches the topology-aware pipeline expectation (obs→app→lb).
func buildSpecs(cfg FleetConfig) []ContainerSpec {
	var specs []ContainerSpec
	if cfg.HasObservability {
		specs = append(specs, ContainerSpec{Name: "sim-obs-01", Role: RoleObservability})
	}
	for i := 0; i < cfg.AppCount; i++ {
		primary := i == 0
		specs = append(specs, ContainerSpec{
			Name:    fmt.Sprintf("sim-app-%02d", i+1),
			Role:    RoleApp,
			Primary: primary,
		})
	}
	if cfg.HasLB {
		specs = append(specs, ContainerSpec{Name: "sim-lb-01", Role: RoleLB})
	}
	return specs
}

// startContainer launches a single sshd container and waits for it to accept
// connections. It returns when the container is reachable via SSH.
//
// The lscr.io/linuxserver/openssh-server image listens on port 2222 (not 22).
// We publish that port to a random host port via "-p 2222".
func startContainer(t *testing.T, spec ContainerSpec, network, authorizedKey string) SimServer {
	t.Helper()

	containerID := strings.TrimSpace(runDockerCmdOutput(t,
		"run", "-d",
		"--network", network,
		"--name", spec.Name,
		"-e", fmt.Sprintf("PUBLIC_KEY=%s", strings.TrimSpace(authorizedKey)),
		"-e", "USER_NAME=nself",
		"-e", "SUDO_ACCESS=false",
		"-e", "PASSWORD_ACCESS=false",
		"-p", "2222",
		"lscr.io/linuxserver/openssh-server:latest",
	))

	// Discover the mapped host port for the container's port 2222.
	portLine := strings.TrimSpace(runDockerCmdOutput(t,
		"port", containerID, "2222/tcp",
	))
	if portLine == "" {
		// Fallback to port 22 for generic sshd images.
		portLine = strings.TrimSpace(runDockerCmdOutput(t,
			"port", containerID, "22/tcp",
		))
	}
	hostPort := extractPort(portLine)

	// Wait until sshd accepts TCP connections (up to 60s — image takes time to start).
	addr := fmt.Sprintf("127.0.0.1:%d", hostPort)
	waitForPort(t, addr, 60*time.Second)

	return SimServer{
		ContainerSpec: spec,
		Host:          "127.0.0.1",
		Port:          hostPort,
		ContainerID:   containerID,
	}
}

// generateKeyPair creates an ED25519 key pair, writes the private key to a
// temp file (0600), and returns the authorized_keys-format public key string
// and the private key file path.
func generateKeyPair(t *testing.T) (authorizedKey string, keyPath string) {
	t.Helper()

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("sim: generate key pair: %v", err)
	}

	// Marshal private key to OpenSSH PEM using crypto/x509.
	privPEM, err := marshalED25519Private(priv)
	if err != nil {
		t.Fatalf("sim: marshal private key: %v", err)
	}

	f, err := os.CreateTemp(t.TempDir(), "sim-key-*")
	if err != nil {
		t.Fatalf("sim: create key file: %v", err)
	}
	defer f.Close()

	if err := f.Chmod(0600); err != nil {
		t.Fatalf("sim: chmod key file: %v", err)
	}
	if _, err := f.Write(privPEM); err != nil {
		t.Fatalf("sim: write key file: %v", err)
	}

	// Build authorized_keys line: "ssh-ed25519 <base64> sim-test"
	// We use the raw public key bytes encoded as OpenSSH wire format.
	authorizedKey = buildAuthorizedKey(pub)

	return authorizedKey, f.Name()
}

// marshalED25519Private encodes an ed25519 private key as a PKCS8 PEM block
// using the standard library only (no x/crypto dependency).
func marshalED25519Private(priv ed25519.PrivateKey) ([]byte, error) {
	b, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("sim: marshal pkcs8: %w", err)
	}
	block := &pem.Block{Type: "PRIVATE KEY", Bytes: b}
	return pem.EncodeToMemory(block), nil
}

// buildAuthorizedKey formats an ed25519.PublicKey as an authorized_keys line.
// Format: "ssh-ed25519 <base64-openssh-wire> sim-test"
//
// The OpenSSH wire format for ed25519 is:
//
//	uint32(len("ssh-ed25519")) || "ssh-ed25519" || uint32(32) || raw-pub-bytes
//
// Encoded as standard base64 (RFC 4648).
func buildAuthorizedKey(pub ed25519.PublicKey) string {
	// Encode as OpenSSH wire format manually (no x/crypto).
	const keyType = "ssh-ed25519"
	raw := []byte(pub)

	// Encode the key type as a length-prefixed string.
	typeBytes := append(uint32BE(len(keyType)), []byte(keyType)...)
	// Encode the key bytes as a length-prefixed string.
	keyBytes := append(uint32BE(len(raw)), raw...)

	wire := append(typeBytes, keyBytes...)

	import64 := base64Encode(wire)
	return keyType + " " + import64 + " sim-test"
}

// uint32BE encodes n as a 4-byte big-endian slice.
func uint32BE(n int) []byte {
	return []byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}
}

// base64Encode encodes data using standard base64 (RFC 4648, no line wrapping).
func base64Encode(data []byte) string {
	const alphabet = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	n := len(data)
	result := make([]byte, 0, ((n+2)/3)*4)
	for i := 0; i < n; i += 3 {
		var b0, b1, b2 byte
		b0 = data[i]
		if i+1 < n {
			b1 = data[i+1]
		}
		if i+2 < n {
			b2 = data[i+2]
		}
		result = append(result,
			alphabet[b0>>2],
			alphabet[((b0&0x3)<<4)|(b1>>4)],
			alphabet[((b1&0xf)<<2)|(b2>>6)],
			alphabet[b2&0x3f],
		)
	}
	// Apply padding.
	switch n % 3 {
	case 1:
		result[len(result)-2] = '='
		result[len(result)-1] = '='
	case 2:
		result[len(result)-1] = '='
	}
	return string(result)
}

// waitForPort polls addr until a TCP connection succeeds or deadline passes.
func waitForPort(t *testing.T, addr string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 500*time.Millisecond)
		if err == nil {
			conn.Close()
			return
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("sim: timed out waiting for %s to accept connections", addr)
}

// extractPort pulls the host port from docker port output like "0.0.0.0:54321".
func extractPort(line string) int {
	if line == "" {
		return 2222
	}
	parts := strings.Split(line, ":")
	if len(parts) < 2 {
		return 2222
	}
	var p int
	fmt.Sscanf(parts[len(parts)-1], "%d", &p)
	return p
}

// runDockerCmd runs a docker subcommand, failing the test on error.
func runDockerCmd(t *testing.T, args ...string) {
	t.Helper()
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// runDockerCmdOutput runs a docker subcommand and returns stdout.
func runDockerCmdOutput(t *testing.T, args ...string) string {
	t.Helper()
	out, err := exec.Command("docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return string(out)
}

// runDockerCmdBest runs a docker subcommand ignoring errors (used for cleanup).
func runDockerCmdBest(args ...string) {
	_ = exec.Command("docker", args...).Run()
}

// TempFile returns a path inside t.TempDir() for storing auxiliary files.
func TempFile(t *testing.T, name string) string {
	t.Helper()
	return filepath.Join(t.TempDir(), name)
}

// SSHHostPort returns the SSH connection address for a SimServer as "host:port".
func (s SimServer) SSHHostPort() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// unused suppresses the context import warning — context is imported by tests.
var _ = context.Background
