package access

// Purpose: the seam between authorized_keys business logic (manager.go,
// file.go, entry.go) and wherever the file actually lives. Production code
// talks to a real host over SSH (SSHTransport, transport_ssh.go); tests talk
// to a plain file (LocalFileTransport, transport_local.go) so grant/revoke/
// list are exercised without ever opening a network connection — the CLI
// test suite must never SSH to a real host, staging or production included.
// Inputs: none (interface only).
// Outputs: none (interface only).
// Constraints: an implementation must never log or return private key
// material — it only ever reads and writes the public authorized_keys file.

import "context"

// Transport reads, backs up, and writes one remote (or fixture) authorized_keys
// file.
type Transport interface {
	// Describe returns a short human-readable label for error messages and
	// audit lines, e.g. "root@5.75.235.42" or a fixture's file path.
	Describe() string

	// Read returns the current authorized_keys content, or (nil, nil) if the
	// file does not exist yet — a fresh host with no managed keys is not an
	// error.
	Read(ctx context.Context) ([]byte, error)

	// Backup copies the current file to a timestamped sibling before any
	// mutation and returns its path. If the file does not exist yet, Backup
	// is a no-op that returns "".
	Backup(ctx context.Context) (string, error)

	// Write replaces the authorized_keys content and ensures the result is
	// readable only by its owner (0600), matching the CLI's env-file
	// permission rule.
	Write(ctx context.Context, content []byte) error
}
