//go:build !cgo

package commands

import (
	"context"
	"fmt"
)

// startEmbeddedPGRuntime is a stub for non-CGO builds. Embedded PostgreSQL
// requires wasmtime CGO bindings and cannot run when CGO_ENABLED=0.
func startEmbeddedPGRuntime(_ context.Context, _, _ string) (cleanup func(), bridgeSockPath string, err error) {
	return func() {}, "", fmt.Errorf("embedded PG requires CGO support; rebuild with CGO_ENABLED=1")
}
