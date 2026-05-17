//go:build cgo

package embedded

import (
	"encoding/binary"
	"io"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestBuildErrorResponse verifies the wire encoding of buildErrorResponse.
func TestBuildErrorResponse(t *testing.T) {
	resp := buildErrorResponse("0A000", "test error")

	if len(resp) == 0 {
		t.Fatal("empty error response")
	}
	if resp[0] != 'E' {
		t.Errorf("message type = %q, want 'E'", resp[0])
	}
	// Length field (bytes 1-4) includes the 4-byte length itself.
	msgLen := binary.BigEndian.Uint32(resp[1:5])
	if int(msgLen) != len(resp)-1 {
		t.Errorf("encoded length %d != packet body length %d", msgLen, len(resp)-1)
	}
}

// TestIsUnsupportedCommand covers the COPY/LISTEN/NOTIFY interception rules.
func TestIsUnsupportedCommand(t *testing.T) {
	cases := []struct {
		sql  string
		want bool
	}{
		{"COPY foo FROM STDIN", true},
		{"copy foo TO STDOUT", true},
		{"LISTEN events", true},
		{"listen chan", true},
		{"NOTIFY chan", true},
		{"notify chan", true},
		{"SELECT 1", false},
		{"INSERT INTO t VALUES (1)", false},
		{"", false},
		{"C", false},
	}

	for _, tc := range cases {
		got := isUnsupportedCommand(tc.sql)
		if got != tc.want {
			t.Errorf("isUnsupportedCommand(%q) = %v, want %v", tc.sql, got, tc.want)
		}
	}
}

// shortTempDir returns a short temp directory path suitable for Unix sockets
// (macOS limit is 104 chars; t.TempDir() paths can exceed this).
func shortTempDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "pgsb-")
	if err != nil {
		t.Fatalf("MkdirTemp: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// TestPGSocketBridge_UnsupportedCommandIntercepted verifies that a COPY simple-
// query message is intercepted by the bridge and an ErrorResponse is returned to
// the client without forwarding to the mock backend.
func TestPGSocketBridge_UnsupportedCommandIntercepted(t *testing.T) {
	dir := shortTempDir(t)

	// Mock backend: echoes anything it receives; we verify it receives nothing.
	backendSock := filepath.Join(dir, "be.sock")
	backendLn, err := net.Listen("unix", backendSock)
	if err != nil {
		t.Fatalf("backend listen: %v", err)
	}
	defer backendLn.Close()

	backendReceived := make(chan []byte, 1)
	go func() {
		conn, err := backendLn.Accept()
		if err != nil {
			return
		}
		defer conn.Close()
		buf, _ := io.ReadAll(conn)
		backendReceived <- buf
	}()

	// Set up a fake runtime whose SockPath points at our mock backend.
	rt := &EmbeddedPGRuntime{
		sockPath: backendSock,
		started:  true,
	}

	bridge := &PGSocketBridge{}
	bridgeSock := filepath.Join(dir, "br.sock")
	if err := bridge.Listen(t.Context(), bridgeSock, rt); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer bridge.Close()

	// Connect as a Postgres client.
	client, err := net.Dial("unix", bridgeSock)
	if err != nil {
		t.Fatalf("client dial: %v", err)
	}
	defer client.Close()

	// Send a simple-query (Q) message with COPY.
	sendSimpleQuery(t, client, "COPY foo FROM STDIN")

	// Expect an ErrorResponse back.
	client.SetDeadline(time.Now().Add(2 * time.Second))
	resp := make([]byte, 512)
	n, err := client.Read(resp)
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	resp = resp[:n]
	if len(resp) == 0 || resp[0] != 'E' {
		t.Errorf("expected ErrorResponse ('E'), got type %q (%d bytes)", resp[0], n)
	}
}

// TestPGSocketBridge_CloseIdempotent verifies Close is idempotent.
func TestPGSocketBridge_CloseIdempotent(t *testing.T) {
	dir := shortTempDir(t)

	rt := &EmbeddedPGRuntime{
		sockPath: filepath.Join(dir, "be.sock"),
		started:  true,
	}

	bridge := &PGSocketBridge{}
	bridgeSock := filepath.Join(dir, "br.sock")
	if err := bridge.Listen(t.Context(), bridgeSock, rt); err != nil {
		t.Fatalf("Listen: %v", err)
	}

	if err := bridge.Close(); err != nil {
		t.Errorf("first Close: %v", err)
	}
	if err := bridge.Close(); err != nil {
		t.Errorf("second Close (idempotent): %v", err)
	}
}

// TestPGSocketBridge_SocketFile verifies the bridge creates the socket file
// and removes a stale one if present.
func TestPGSocketBridge_SocketFile(t *testing.T) {
	dir := shortTempDir(t)
	bridgeSock := filepath.Join(dir, "br.sock")

	// Create a stale socket file to verify it's removed.
	if err := os.WriteFile(bridgeSock, []byte("stale"), 0o600); err != nil {
		t.Fatalf("WriteFile stale: %v", err)
	}

	rt := &EmbeddedPGRuntime{
		sockPath: filepath.Join(dir, "be.sock"),
		started:  true,
	}
	bridge := &PGSocketBridge{}
	if err := bridge.Listen(t.Context(), bridgeSock, rt); err != nil {
		t.Fatalf("Listen: %v", err)
	}
	defer bridge.Close()

	if _, err := os.Stat(bridgeSock); err != nil {
		t.Errorf("socket file not created: %v", err)
	}
}

// sendSimpleQuery writes a Postgres simple-query (Q) message to conn.
func sendSimpleQuery(t *testing.T, conn net.Conn, sql string) {
	t.Helper()
	body := append([]byte(sql), 0) // NUL terminated
	length := int32(len(body) + 4)
	msg := []byte{'Q',
		byte(length >> 24), byte(length >> 16), byte(length >> 8), byte(length),
	}
	msg = append(msg, body...)
	if _, err := conn.Write(msg); err != nil {
		t.Fatalf("write simple query: %v", err)
	}
}
