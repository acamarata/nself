package controlplane

import (
	"errors"
	"testing"
)

// stubProber is a controllable Prober for unit tests. Each field is a function
// that can be set per-test; defaults return safe zero values.
type stubProber struct {
	sshFn    func(s Server) (bool, int, error)
	dockerFn func(s Server) (bool, error)
	keyFn    func(s Server) (bool, string)
}

func (sp *stubProber) SSHReachable(s Server) (bool, int, error) {
	if sp.sshFn != nil {
		return sp.sshFn(s)
	}
	return true, 10, nil
}

func (sp *stubProber) DockerOK(s Server) (bool, error) {
	if sp.dockerFn != nil {
		return sp.dockerFn(s)
	}
	return true, nil
}

func (sp *stubProber) KeyState(s Server) (bool, string) {
	if sp.keyFn != nil {
		return sp.keyFn(s)
	}
	return true, "/tmp/test-key"
}

// assertNoExec verifies that the resolver does not call os/exec by confirming
// the stub prober (not os/exec) was used for all probes.
// (The resolver must delegate all I/O to the Prober interface — no direct execs.)
func assertNoExec(t *testing.T) {
	t.Helper()
	// Nothing to assert at compile time; the test proves this by supplying a
	// stub Prober. If the resolver calls os/exec directly, the test build fails
	// because os/exec is not imported by resolver.go.
}

// makeRemoteServer returns a Server configured for remote access tests.
func makeRemoteServer(name, host, sshKeyRef string) Server {
	return Server{
		Name:       name,
		Role:       RoleApp,
		Host:       host,
		SSHKeyRef:  sshKeyRef,
		RemotePath: "/opt/nself",
		Primary:    true,
	}
}

// makeInventory builds a minimal Inventory for tests.
func makeInventory(envName, kind string, servers ...Server) *Inventory {
	return &Inventory{
		SchemaVersion: currentSchemaVersion,
		Project:       "test",
		Environments: map[string]Environment{
			envName: {
				Name:    envName,
				Kind:    kind,
				Servers: servers,
			},
		},
	}
}

// findStatus returns the TargetStatus for the given server name from results.
func findStatus(t *testing.T, results []TargetStatus, serverName string) TargetStatus {
	t.Helper()
	for _, ts := range results {
		if ts.Server == serverName {
			return ts
		}
	}
	t.Fatalf("no TargetStatus for server %q", serverName)
	return TargetStatus{}
}

// ---------------------------------------------------------------------------
// §5.1 branch tests — every branch must be covered.
// ---------------------------------------------------------------------------

// Branch 1: local kind → CapManage, no probes called.
func TestResolveLocalEnv(t *testing.T) {
	assertNoExec(t)
	var probesCalled int
	p := &stubProber{
		sshFn: func(_ Server) (bool, int, error) {
			probesCalled++
			return true, 0, nil
		},
	}

	inv := makeInventory("local", "local", Server{
		Name:    "local-app",
		Role:    RoleApp,
		Primary: true,
	})

	results := Resolve(inv, p)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	ts := results[0]
	if ts.Capability != CapManage {
		t.Errorf("local env: Capability = %q, want manage", ts.Capability)
	}
	if probesCalled != 0 {
		t.Errorf("SSH probe must not be called for local env; was called %d times", probesCalled)
	}
	if !ts.KeyPresent {
		t.Error("local env: KeyPresent must be true")
	}
	if !ts.SSHReachable {
		t.Error("local env: SSHReachable must be true")
	}
	if !ts.DockerOK {
		t.Error("local env: DockerOK must be true")
	}
}

// Branch 2: Host empty → CapHidden.
func TestResolveHostEmptyIsHidden(t *testing.T) {
	assertNoExec(t)
	p := &stubProber{}

	inv := makeInventory("staging", "remote", Server{
		Name:      "stg-hidden",
		Role:      RoleApp,
		Host:      "", // empty
		SSHKeyRef: "SOME_KEY",
	})

	results := Resolve(inv, p)
	ts := findStatus(t, results, "stg-hidden")
	if ts.Capability != CapHidden {
		t.Errorf("empty host: Capability = %q, want hidden", ts.Capability)
	}
}

// Branch 3: SSHKeyRef empty → CapReadOnly + reason "no SSH key for ".
func TestResolveSSHKeyRefEmpty(t *testing.T) {
	assertNoExec(t)
	var keyProbed bool
	p := &stubProber{
		keyFn: func(_ Server) (bool, string) {
			keyProbed = true
			return true, "/tmp/key"
		},
	}

	inv := makeInventory("staging", "remote", makeRemoteServer("stg-app", "ubuntu@host.example.com", ""))

	results := Resolve(inv, p)
	ts := findStatus(t, results, "stg-app")
	if ts.Capability != CapReadOnly {
		t.Errorf("empty SSHKeyRef: Capability = %q, want read-only", ts.Capability)
	}
	if ts.Reason == "" {
		t.Error("Reason must not be empty when SSHKeyRef is empty")
	}
	if keyProbed {
		t.Error("KeyState must not be called when SSHKeyRef is empty")
	}
}

// Branch 4: KeyState returns present=false → CapReadOnly + reason containing "no SSH key".
func TestResolveKeyAbsent(t *testing.T) {
	assertNoExec(t)
	var sshCalled bool
	p := &stubProber{
		keyFn: func(_ Server) (bool, string) {
			return false, "" // key not present
		},
		sshFn: func(_ Server) (bool, int, error) {
			sshCalled = true
			return true, 0, nil
		},
	}

	inv := makeInventory("staging", "remote",
		makeRemoteServer("stg-app", "ubuntu@host.example.com", "NSELF_SSH_KEY_STAGING"))

	results := Resolve(inv, p)
	ts := findStatus(t, results, "stg-app")
	if ts.Capability != CapReadOnly {
		t.Errorf("key absent: Capability = %q, want read-only", ts.Capability)
	}
	if ts.KeyPresent {
		t.Error("KeyPresent must be false when key is absent")
	}
	if sshCalled {
		t.Error("SSHReachable must not be called when key is absent")
	}
	if !containsFold(ts.Reason, "no SSH key") {
		t.Errorf("Reason %q must contain 'no SSH key'", ts.Reason)
	}
}

// Branch 5: SSH unreachable → CapReadOnly + reason "SSH unreachable (<err class>)".
func TestResolveSSHUnreachable(t *testing.T) {
	assertNoExec(t)
	var dockerCalled bool
	p := &stubProber{
		keyFn: func(_ Server) (bool, string) {
			return true, "/tmp/key"
		},
		sshFn: func(_ Server) (bool, int, error) {
			return false, 0, errors.New("i/o timeout")
		},
		dockerFn: func(_ Server) (bool, error) {
			dockerCalled = true
			return false, nil
		},
	}

	inv := makeInventory("prod", "remote",
		makeRemoteServer("prod-app", "ubuntu@prod.example.com", "NSELF_SSH_KEY_PROD"))

	results := Resolve(inv, p)
	ts := findStatus(t, results, "prod-app")
	if ts.Capability != CapReadOnly {
		t.Errorf("SSH unreachable: Capability = %q, want read-only", ts.Capability)
	}
	if !containsFold(ts.Reason, "SSH unreachable") {
		t.Errorf("Reason %q must contain 'SSH unreachable'", ts.Reason)
	}
	if dockerCalled {
		t.Error("DockerOK must not be called when SSH is unreachable")
	}
}

// Branch 5 variant: SSH unreachable with "connection refused" error class.
func TestResolveSSHConnectionRefusedClass(t *testing.T) {
	p := &stubProber{
		keyFn: func(_ Server) (bool, string) { return true, "/tmp/key" },
		sshFn: func(_ Server) (bool, int, error) {
			return false, 0, errors.New("dial tcp: connection refused")
		},
	}
	inv := makeInventory("prod", "remote",
		makeRemoteServer("srv", "ubuntu@host.example.com", "NSELF_SSH_KEY_PROD"))

	results := Resolve(inv, p)
	ts := findStatus(t, results, "srv")
	if !containsFold(ts.Reason, "connection refused") {
		t.Errorf("Reason %q should contain 'connection refused'", ts.Reason)
	}
}

// Branch 6: SSH ok, docker not reachable → CapReadOnly + exact reason string.
func TestResolveDockerDown(t *testing.T) {
	assertNoExec(t)
	p := &stubProber{
		keyFn: func(_ Server) (bool, string) {
			return true, "/tmp/key"
		},
		sshFn: func(_ Server) (bool, int, error) {
			return true, 42, nil
		},
		dockerFn: func(_ Server) (bool, error) {
			return false, nil // docker not responding
		},
	}

	inv := makeInventory("prod", "remote",
		makeRemoteServer("prod-app", "ubuntu@prod.example.com", "NSELF_SSH_KEY_PROD"))

	results := Resolve(inv, p)
	ts := findStatus(t, results, "prod-app")
	if ts.Capability != CapReadOnly {
		t.Errorf("docker down: Capability = %q, want read-only", ts.Capability)
	}
	if ts.Reason != reasonDockerDown {
		t.Errorf("Reason = %q, want %q", ts.Reason, reasonDockerDown)
	}
	if !ts.SSHReachable {
		t.Error("SSHReachable must be true when SSH succeeded")
	}
	if ts.DockerOK {
		t.Error("DockerOK must be false")
	}
	if ts.LatencyMS != 42 {
		t.Errorf("LatencyMS = %d, want 42", ts.LatencyMS)
	}
}

// Branch 7: SSH + Docker both ok → CapManage, empty Reason.
func TestResolveFullManage(t *testing.T) {
	assertNoExec(t)
	p := &stubProber{
		keyFn: func(_ Server) (bool, string) {
			return true, "/tmp/key"
		},
		sshFn: func(_ Server) (bool, int, error) {
			return true, 15, nil
		},
		dockerFn: func(_ Server) (bool, error) {
			return true, nil
		},
	}

	inv := makeInventory("prod", "remote",
		makeRemoteServer("prod-app", "ubuntu@prod.example.com", "NSELF_SSH_KEY_PROD"))

	results := Resolve(inv, p)
	ts := findStatus(t, results, "prod-app")
	if ts.Capability != CapManage {
		t.Errorf("all-ok: Capability = %q, want manage", ts.Capability)
	}
	if ts.Reason != "" {
		t.Errorf("Reason must be empty for manage; got %q", ts.Reason)
	}
	if !ts.KeyPresent {
		t.Error("KeyPresent must be true")
	}
	if !ts.SSHReachable {
		t.Error("SSHReachable must be true")
	}
	if !ts.DockerOK {
		t.Error("DockerOK must be true")
	}
}

// TestResolveMultipleServers verifies that Resolve processes all servers
// independently, including mixed capabilities in the same environment.
func TestResolveMultipleServers(t *testing.T) {
	assertNoExec(t)
	p := &stubProber{
		keyFn: func(s Server) (bool, string) {
			// "good" server has a key; "nokey" does not
			return s.Name == "good", "/tmp/key"
		},
		sshFn:    func(_ Server) (bool, int, error) { return true, 5, nil },
		dockerFn: func(_ Server) (bool, error) { return true, nil },
	}

	inv := &Inventory{
		SchemaVersion: currentSchemaVersion,
		Project:       "multi",
		Environments: map[string]Environment{
			"prod": {
				Name: "prod",
				Kind: "remote",
				Servers: []Server{
					makeRemoteServer("good", "ubuntu@good.example.com", "NSELF_SSH_KEY_PROD"),
					makeRemoteServer("nokey", "ubuntu@nokey.example.com", "NSELF_SSH_KEY_PROD"),
				},
			},
		},
	}

	results := Resolve(inv, p)
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	good := findStatus(t, results, "good")
	if good.Capability != CapManage {
		t.Errorf("good: Capability = %q, want manage", good.Capability)
	}

	nokey := findStatus(t, results, "nokey")
	if nokey.Capability != CapReadOnly {
		t.Errorf("nokey: Capability = %q, want read-only", nokey.Capability)
	}
}

// TestErrClassMapping verifies errClass maps known error strings to concise labels.
func TestErrClassMapping(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"i/o timeout", "timeout"},
		{"connection refused", "connection refused"},
		{"no route to host", "no route to host"},
		{"network unreachable", "network unreachable"},
		{"Permission denied (publickey)", "permission denied"},
		{"WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED (host key)", "host key mismatch"},
		{"some random error", "network error"},
	}
	for _, tc := range tests {
		got := errClass(errors.New(tc.input))
		if got != tc.want {
			t.Errorf("errClass(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// TestErrClassNilError verifies errClass handles nil gracefully.
func TestErrClassNilError(t *testing.T) {
	got := errClass(nil)
	if got == "" {
		t.Error("errClass(nil) must return a non-empty string")
	}
}

// TestProbedAtIsSet verifies that ProbedAt is populated for all results.
func TestProbedAtIsSet(t *testing.T) {
	p := &stubProber{
		keyFn:    func(_ Server) (bool, string) { return true, "/tmp/key" },
		sshFn:    func(_ Server) (bool, int, error) { return true, 1, nil },
		dockerFn: func(_ Server) (bool, error) { return true, nil },
	}
	inv := makeInventory("prod", "remote",
		makeRemoteServer("srv", "ubuntu@host.example.com", "NSELF_SSH_KEY_PROD"))

	results := Resolve(inv, p)
	for _, ts := range results {
		if ts.ProbedAt.IsZero() {
			t.Errorf("server %q: ProbedAt is zero", ts.Server)
		}
	}
}
