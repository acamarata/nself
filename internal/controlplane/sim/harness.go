// Package sim provides a multi-server Docker sshd simulation harness for
// integration-testing the control-plane pipeline without touching live hosts.
//
// The harness spins up N Docker containers (each running sshd) on an isolated
// bridge network, injects an in-memory ED25519 key-pair, and exposes a Fleet
// value whose servers can be passed directly to controlplane.Resolve /
// controlplane.Run.
//
// All containers are torn down on Close, even if the test panics, so the host
// environment is never dirtied.
//
// Build tag: integration — tests in this package are skipped unless
// INTEGRATION=1 is set in the environment.
package sim

import (
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Role constants mirror controlplane.ServerRole but kept local to avoid
// importing the parent package (the harness must not import impl under test).
const (
	RoleApp           = "app"
	RoleLB            = "lb"
	RoleObservability = "observability"
)

// ContainerSpec describes one simulated host in the fleet.
type ContainerSpec struct {
	Name string
	Role string
	// Primary marks the server as the primary app server (non-zero exit on skip).
	Primary bool
}

// SimServer is one running container with its connection details.
type SimServer struct {
	ContainerSpec
	// Host is the container's address reachable from the test host.
	Host string
	// Port is the sshd port mapped to a random high port on the host.
	Port int
	// ContainerID is the Docker container ID for cleanup.
	ContainerID string
}

// Fleet is the full simulation environment returned by Start.
type Fleet struct {
	// Servers is the ordered list of simulated hosts.
	Servers []SimServer
	// PrivateKeyPath is the path to the ED25519 private key that all
	// containers accept. Set this env var value as the SSH key for probers.
	PrivateKeyPath string
	// EnvVarName is the environment variable that must be set to PrivateKeyPath
	// so the control-plane prober can find the key.
	EnvVarName string
	// NetworkName is the Docker bridge network created for this fleet.
	NetworkName string
	// cleanup holds teardown operations in reverse-creation order.
	cleanup []func()
}

// Close tears down all containers and the bridge network.
func (f *Fleet) Close() {
	for i := len(f.cleanup) - 1; i >= 0; i-- {
		f.cleanup[i]()
	}
}

// FleetConfig controls how many servers of each role are started.
type FleetConfig struct {
	// AppCount is the number of app-role containers (≥1).
	AppCount int
	// HasLB controls whether a load-balancer container is included.
	HasLB bool
	// HasObservability controls whether an observability container is added.
	HasObservability bool
	// EnvVarName is the SSH key env var name injected into the fleet.
	// Defaults to "NSELF_SSH_KEY_SIM".
	EnvVarName string
}

// DefaultFleetConfig returns the 4-server configuration used by CP-T19:
// 1 lb + 1 observability + 2 app.
func DefaultFleetConfig() FleetConfig {
	return FleetConfig{
		AppCount:         2,
		HasLB:            true,
		HasObservability: true,
		EnvVarName:       "NSELF_SSH_KEY_SIM",
	}
}

// Start launches the simulation fleet. It is called from integration tests
// gated by INTEGRATION=1. The caller must call fleet.Close() or defer it.
//
// Start generates an ephemeral ED25519 key-pair, writes the private key to a
// temp file (mode 0600), and configures each sshd container to accept that key
// as the only authorised identity.
func Start(t *testing.T, cfg FleetConfig) *Fleet {
	t.Helper()

	if cfg.EnvVarName == "" {
		cfg.EnvVarName = "NSELF_SSH_KEY_SIM"
	}

	// Require Docker to be available.
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not in PATH — skipping integration test")
	}

	f := &Fleet{EnvVarName: cfg.EnvVarName}

	// 1. Generate ephemeral ED25519 key-pair.
	pubKeyPEM, keyPath := generateKeyPair(t)
	f.PrivateKeyPath = keyPath
	f.cleanup = append(f.cleanup, func() { _ = os.Remove(keyPath) })

	// 2. Create isolated bridge network.
	networkName := fmt.Sprintf("nself-sim-%d", time.Now().UnixNano())
	f.NetworkName = networkName
	runDockerCmd(t, "network", "create", "--driver", "bridge", networkName)
	f.cleanup = append(f.cleanup, func() {
		runDockerCmdBest("network", "rm", networkName)
	})

	// 3. Spin up containers per spec.
	specs := buildSpecs(cfg)
	for _, spec := range specs {
		srv := startContainer(t, spec, networkName, pubKeyPEM)
		f.Servers = append(f.Servers, srv)
		cid := srv.ContainerID
		f.cleanup = append(f.cleanup, func() {
			runDockerCmdBest("rm", "-f", cid)
		})
	}

	// 4. Set the key env var so probe-based tests can pick it up.
	t.Setenv(cfg.EnvVarName, keyPath)

	return f
}
