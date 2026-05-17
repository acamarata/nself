//go:build cgo

package commands

import (
	"context"
	"fmt"

	"github.com/nself-org/cli/internal/embedded"
)

// startEmbeddedPGRuntime boots the embedded pglite/wasmtime PostgreSQL runtime
// and the socket bridge that proxies Hasura's wire-protocol traffic.
//
// It returns a cleanup function (always non-nil) that stops the runtime and
// closes the bridge, the Unix socket path for the bridge, and any error.
// On error the cleanup function is a no-op and may be called safely.
func startEmbeddedPGRuntime(ctx context.Context, runtimeDir, wasmPath string) (cleanup func(), bridgeSockPath string, err error) {
	noop := func() {}

	rt, runtimeErr := embedded.NewEmbeddedPGRuntime(runtimeDir, wasmPath)
	if runtimeErr != nil {
		return noop, "", fmt.Errorf("embedded-pg: new runtime: %w", runtimeErr)
	}

	if bootErr := rt.Start(ctx); bootErr != nil {
		return noop, "", fmt.Errorf("embedded-pg: start runtime: %w", bootErr)
	}

	bridge := &embedded.PGSocketBridge{}
	sockPath := rt.SockPath() + ".bridge"
	if bridgeErr := bridge.Listen(ctx, sockPath, rt); bridgeErr != nil {
		_ = rt.Stop()
		return noop, "", fmt.Errorf("embedded-pg: start bridge: %w", bridgeErr)
	}

	cleanup = func() {
		_ = bridge.Close()
		_ = rt.Stop()
	}
	return cleanup, sockPath, nil
}
