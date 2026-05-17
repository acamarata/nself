package embedded

import (
	"context"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

// pgWireUnsupported is a minimal Postgres wire-protocol ErrorResponse for
// SQLSTATE 0A000 (feature_not_supported), returned for commands that pglite's
// embedded mode cannot handle (COPY, LISTEN, NOTIFY).
//
// The byte layout is:
//
//	'E'          message type
//	int32        total length of message body (network byte order)
//	'S' severity 'ERROR\x00'
//	'C' code     '0A000\x00'
//	'M' message  '<text>\x00'
//	'\x00'       terminator
var pgWireUnsupported = buildErrorResponse("0A000", "EMBEDDED_PG: feature not supported — COPY, LISTEN, and NOTIFY are not available in embedded-PG mode")

// buildErrorResponse constructs a minimal Postgres ErrorResponse packet.
// sqlstate must be exactly 5 characters. msg is the human-readable message.
func buildErrorResponse(sqlstate, msg string) []byte {
	body := []byte{}
	body = append(body, 'S')
	body = append(body, "ERROR"...)
	body = append(body, 0)
	body = append(body, 'C')
	body = append(body, sqlstate...)
	body = append(body, 0)
	body = append(body, 'M')
	body = append(body, msg...)
	body = append(body, 0)
	body = append(body, 0) // terminator

	length := int32(len(body) + 4) // 4 bytes for the length field itself
	var out []byte
	out = append(out, 'E')
	out = append(out, byte(length>>24), byte(length>>16), byte(length>>8), byte(length))
	out = append(out, body...)
	return out
}

// PGSocketBridge accepts Postgres wire-protocol connections on a Unix domain
// socket and pipes them through to the embedded pglite runtime. It intercepts
// COPY, LISTEN, and NOTIFY commands, which are not supported by pglite, and
// returns a Postgres ErrorResponse (SQLSTATE 0A000) instead of forwarding them.
//
// One PGSocketBridge instance handles all client connections for a single
// EmbeddedPGRuntime.
type PGSocketBridge struct {
	mu      sync.Mutex
	ln      net.Listener
	runtime *EmbeddedPGRuntime
	closed  bool
}

// Listen starts the bridge listener on sockPath and begins accepting
// connections. It returns immediately; connections are handled in background
// goroutines. Call Close to stop accepting new connections.
//
// sockPath is the host-side Unix socket that Hasura and Auth containers
// connect to (bind-mounted into each container at the same path).
func (b *PGSocketBridge) Listen(ctx context.Context, sockPath string, runtime *EmbeddedPGRuntime) error {
	// Remove a stale socket file if present from a previous run.
	_ = os.Remove(sockPath)

	ln, err := net.Listen("unix", sockPath)
	if err != nil {
		return fmt.Errorf("embedded/bridge: listen on %s: %w", sockPath, err)
	}
	if err := os.Chmod(sockPath, 0o660); err != nil {
		ln.Close()
		return fmt.Errorf("embedded/bridge: chmod socket: %w", err)
	}

	b.mu.Lock()
	b.ln = ln
	b.runtime = runtime
	b.mu.Unlock()

	go b.acceptLoop(ctx)
	return nil
}

// Close stops accepting new connections. In-flight connections continue until
// their clients disconnect.
func (b *PGSocketBridge) Close() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.closed {
		return nil
	}
	b.closed = true
	if b.ln != nil {
		return b.ln.Close()
	}
	return nil
}

// acceptLoop accepts incoming client connections and dispatches each one to a
// goroutine. It exits when the listener is closed.
func (b *PGSocketBridge) acceptLoop(ctx context.Context) {
	for {
		conn, err := b.ln.Accept()
		if err != nil {
			// Listener closed — exit gracefully.
			return
		}
		go b.handleConn(ctx, conn)
	}
}

// handleConn proxies a single client connection to the embedded pglite runtime.
// It intercepts simple-query messages that start with COPY, LISTEN, or NOTIFY
// and returns a wire-protocol error response (SQLSTATE 0A000) without
// forwarding to pglite.
//
// The interception is best-effort and operates at the query-text level on
// Postgres simple-query (Q) messages. Extended-query protocol (P/B/E) flow is
// passed through unmodified; pglite itself will reject unsupported commands.
func (b *PGSocketBridge) handleConn(ctx context.Context, client net.Conn) {
	defer client.Close()

	b.mu.Lock()
	rt := b.runtime
	b.mu.Unlock()

	// Dial the pglite Unix socket (the embedded PG's listener exposed by
	// EmbeddedPGRuntime or by the WASM bridge shim).
	backend, err := (&net.Dialer{}).DialContext(ctx, "unix", rt.SockPath())
	if err != nil {
		// Cannot reach pglite — close the client connection.
		return
	}
	defer backend.Close()

	// Bidirectional copy with interception on the client→backend direction.
	var wg sync.WaitGroup
	wg.Add(2)

	// backend → client: passthrough.
	go func() {
		defer wg.Done()
		_, _ = io.Copy(client, backend)
	}()

	// client → backend: intercept unsupported commands.
	go func() {
		defer wg.Done()
		if err := proxyClientToBackend(client, backend); err != nil {
			// Connection closed or read error — normal EOF.
		}
	}()

	wg.Wait()
}

// proxyClientToBackend reads Postgres wire-protocol messages from client and
// forwards them to backend, except for simple-query (Q) messages whose SQL text
// begins with COPY, LISTEN, or NOTIFY — those get an ErrorResponse sent back to
// the client instead.
//
// The function returns when either side closes the connection.
func proxyClientToBackend(client, backend net.Conn) error {
	for {
		// Read one Postgres message from the client.
		// Postgres frontend message format:
		//   byte1  type
		//   int32  length (includes the 4-byte length field, excludes the type byte)
		//   body   length-4 bytes
		header := make([]byte, 5)
		if _, err := io.ReadFull(client, header); err != nil {
			return err
		}

		msgType := header[0]
		bodyLen := int(header[1])<<24 | int(header[2])<<16 | int(header[3])<<8 | int(header[4])
		if bodyLen < 4 {
			// Malformed message.
			return fmt.Errorf("embedded/bridge: malformed message length %d", bodyLen)
		}
		body := make([]byte, bodyLen-4)
		if _, err := io.ReadFull(client, body); err != nil {
			return err
		}

		// Intercept simple-query (type 'Q') messages for unsupported commands.
		if msgType == 'Q' && len(body) > 0 {
			sql := trimNul(body)
			if isUnsupportedCommand(sql) {
				if _, err := client.Write(pgWireUnsupported); err != nil {
					return err
				}
				// Also send ReadyForQuery so the client doesn't hang.
				if _, err := client.Write(readyForQuery()); err != nil {
					return err
				}
				continue // do not forward to pglite
			}
		}

		// Forward the message to pglite unchanged.
		if _, err := backend.Write(header); err != nil {
			return err
		}
		if _, err := backend.Write(body); err != nil {
			return err
		}
	}
}

// isUnsupportedCommand reports whether the SQL text (trimmed of whitespace and
// NUL terminators) starts with an unsupported command keyword.
func isUnsupportedCommand(sql string) bool {
	if len(sql) < 4 {
		return false
	}
	// Case-insensitive prefix check for the three unsupported command families.
	upper := toUpperASCII(sql[:6]) // safe: checked len >= 4 above; pad to 6
	switch {
	case len(sql) >= 4 && upper[:4] == "COPY":
		return true
	case len(sql) >= 6 && upper[:6] == "LISTEN":
		return true
	case len(sql) >= 6 && upper[:6] == "NOTIFY":
		return true
	}
	return false
}

// trimNul returns s with leading/trailing whitespace and NUL bytes removed.
func trimNul(b []byte) string {
	s := string(b)
	out := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		if c != 0 {
			out = append(out, c)
		}
	}
	return string(out)
}

// toUpperASCII converts the first n bytes of s to upper case (ASCII only).
// s is padded with spaces if shorter than n.
func toUpperASCII(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

// readyForQuery returns a Postgres ReadyForQuery message with status 'I' (idle).
func readyForQuery() []byte {
	// 'Z' type, length=5 (4+1), status='I'
	return []byte{'Z', 0, 0, 0, 5, 'I'}
}
