package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// withMockExec swaps the lookPath and runCommand seams for the duration of a
// test, restoring originals on cleanup.
func withMockExec(t *testing.T, look func(string) (string, error), run func(context.Context, string, ...string) (string, error)) {
	t.Helper()
	origLook := lookPath
	origRun := runCommand
	lookPath = look
	runCommand = run
	t.Cleanup(func() {
		lookPath = origLook
		runCommand = origRun
	})
}

// TestDetect_DockerStandard simulates a host with only docker on PATH running
// the community edition. Detection should pick docker, populate version, and
// leave IsEnterprise false.
func TestDetect_DockerStandard(t *testing.T) {
	withMockExec(t,
		func(name string) (string, error) {
			if name == "docker" {
				return "/usr/local/bin/docker", nil
			}
			return "", errors.New("not found")
		},
		func(ctx context.Context, name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "version" {
				return "24.0.6\n", nil
			}
			if len(args) > 0 && args[0] == "info" {
				return "Server Version: 24.0.6\nPlan: Community\n", nil
			}
			return "", errors.New("unexpected call")
		},
	)

	rt, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if rt.Name != NameDocker {
		t.Errorf("Name: got %q, want %q", rt.Name, NameDocker)
	}
	if rt.Binary != "/usr/local/bin/docker" {
		t.Errorf("Binary: got %q", rt.Binary)
	}
	if rt.Version != "24.0.6" {
		t.Errorf("Version: got %q, want 24.0.6", rt.Version)
	}
	if rt.IsEnterprise {
		t.Errorf("IsEnterprise: got true, want false for community edition")
	}
	if w := rt.EnterpriseWarning(); w != "" {
		t.Errorf("EnterpriseWarning: got %q, want empty", w)
	}
}

// TestDetect_DockerEnterprise simulates Docker Desktop with the enterprise
// plan. Detection should set IsEnterprise=true and EnterpriseWarning should
// return a non-empty advisory string.
func TestDetect_DockerEnterprise(t *testing.T) {
	withMockExec(t,
		func(name string) (string, error) {
			if name == "docker" {
				return "/Applications/Docker.app/Contents/Resources/bin/docker", nil
			}
			return "", errors.New("not found")
		},
		func(ctx context.Context, name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "version" {
				return "25.0.1\n", nil
			}
			if len(args) > 0 && args[0] == "info" {
				return "Server Version: 25.0.1\nPlan: Enterprise\n", nil
			}
			return "", errors.New("unexpected call")
		},
	)

	rt, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if rt.Name != NameDocker {
		t.Errorf("Name: got %q, want docker", rt.Name)
	}
	if !rt.IsEnterprise {
		t.Errorf("IsEnterprise: got false, want true")
	}
	w := rt.EnterpriseWarning()
	if w == "" {
		t.Fatal("EnterpriseWarning: got empty, want advisory text")
	}
	if !strings.Contains(strings.ToLower(w), "enterprise") {
		t.Errorf("EnterpriseWarning should mention enterprise: %q", w)
	}
}

// TestDetect_OrbStack simulates a host with OrbStack installed (orb on PATH,
// no docker). Detection should fall through docker and pick orb.
func TestDetect_OrbStack(t *testing.T) {
	withMockExec(t,
		func(name string) (string, error) {
			if name == "orb" {
				return "/opt/homebrew/bin/orb", nil
			}
			return "", errors.New("not found")
		},
		func(ctx context.Context, name string, args ...string) (string, error) {
			if len(args) > 0 && args[0] == "version" {
				return "1.7.2\n", nil
			}
			return "", errors.New("unexpected call")
		},
	)

	rt, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if rt.Name != NameOrbStack {
		t.Errorf("Name: got %q, want %q", rt.Name, NameOrbStack)
	}
	if rt.Binary != "/opt/homebrew/bin/orb" {
		t.Errorf("Binary: got %q", rt.Binary)
	}
	if rt.Version != "1.7.2" {
		t.Errorf("Version: got %q", rt.Version)
	}
	// OrbStack must never trigger the enterprise warning (Docker-specific).
	if rt.IsEnterprise {
		t.Errorf("IsEnterprise: got true, expected false for OrbStack")
	}
}

// TestDetect_Podman simulates a host with podman as the only runtime
// (typical Linux environment without Docker installed). Detection should fall
// through docker and orb to pick podman, falling back to --version parsing
// when the structured `version --format` form is not supported.
func TestDetect_Podman(t *testing.T) {
	withMockExec(t,
		func(name string) (string, error) {
			if name == "podman" {
				return "/usr/bin/podman", nil
			}
			return "", errors.New("not found")
		},
		func(ctx context.Context, name string, args ...string) (string, error) {
			if len(args) >= 2 && args[0] == "version" && args[1] == "--format" {
				// Simulate older Podman that doesn't support --format.
				return "", errors.New("unknown flag --format")
			}
			if len(args) > 0 && args[0] == "--version" {
				return "podman version 4.6.2\n", nil
			}
			return "", errors.New("unexpected call")
		},
	)

	rt, err := Detect()
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if rt.Name != NamePodman {
		t.Errorf("Name: got %q, want %q", rt.Name, NamePodman)
	}
	if rt.Version != "4.6.2" {
		t.Errorf("Version: got %q, want 4.6.2", rt.Version)
	}
	if rt.IsEnterprise {
		t.Errorf("IsEnterprise: got true, expected false for Podman")
	}
}

// TestDetect_NoRuntime verifies that ErrNoRuntime is returned when no
// supported runtime is on PATH.
func TestDetect_NoRuntime(t *testing.T) {
	withMockExec(t,
		func(name string) (string, error) {
			return "", errors.New("not found")
		},
		func(ctx context.Context, name string, args ...string) (string, error) {
			return "", errors.New("should not be called")
		},
	)

	rt, err := Detect()
	if err == nil {
		t.Fatalf("Detect: got nil error, want ErrNoRuntime; runtime=%+v", rt)
	}
	if !errors.Is(err, ErrNoRuntime) {
		t.Errorf("Detect: got %v, want ErrNoRuntime", err)
	}
}

// TestParseVersionLine_TableDriven covers the version-line parser which is
// shared across runtimes that don't support `version --format`.
func TestParseVersionLine_TableDriven(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"Docker version 24.0.6, build ed223bc", "24.0.6"},
		{"podman version 4.6.2", "4.6.2"},
		{"  Docker version 25.0.1  \n", "25.0.1"},
		{"unrelated output", ""},
		{"", ""},
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			got := parseVersionLine(tc.in)
			if got != tc.want {
				t.Errorf("parseVersionLine(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

// TestRuntime_String covers the String() formatter used by --runtime output.
func TestRuntime_String(t *testing.T) {
	var nilRT *Runtime
	if got := nilRT.String(); got == "" {
		t.Error("nil String: got empty, want fallback message")
	}
	rt := &Runtime{Name: NameDocker, Binary: "/usr/local/bin/docker", Version: "24.0.6"}
	got := rt.String()
	if !strings.Contains(got, "docker") || !strings.Contains(got, "24.0.6") {
		t.Errorf("String: got %q, missing name or version", got)
	}
	rtNoVer := &Runtime{Name: NamePodman, Binary: "/usr/bin/podman"}
	if got := rtNoVer.String(); !strings.Contains(got, "unknown") {
		t.Errorf("String with no version: got %q, want 'unknown' fallback", got)
	}
}
