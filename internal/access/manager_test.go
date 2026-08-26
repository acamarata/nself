package access

// Tests drive Grant/Revoke/List exclusively through LocalFileTransport
// against a temp-dir fixture standing in for a remote authorized_keys file —
// SSHTransport is never exercised here or anywhere else in this suite, so
// running these tests never opens a network connection to any host.

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newFixture(t *testing.T) (*LocalFileTransport, string) {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "authorized_keys")
	restore := SetAuditLogPathForTest(filepath.Join(dir, "audit.log"))
	t.Cleanup(restore)
	return NewLocalFileTransport(path), path
}

func aliceKey(t *testing.T) PublicKey {
	t.Helper()
	k, err := ParsePublicKey(aliceKeyLine)
	if err != nil {
		t.Fatalf("parse alice key: %v", err)
	}
	return k
}

func bobKey(t *testing.T) PublicKey {
	t.Helper()
	k, err := ParsePublicKey(bobKeyLine)
	if err != nil {
		t.Fatalf("parse bob key: %v", err)
	}
	return k
}

func TestGrant_NewUser(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	res, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)})
	if err != nil {
		t.Fatalf("Grant: %v", err)
	}
	if res.AlreadyGranted {
		t.Error("first grant must not be AlreadyGranted")
	}
	if res.Fingerprint != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", res.Fingerprint, aliceFP)
	}
	if res.BackupPath != "" {
		t.Errorf("BackupPath = %q, want empty (nothing existed to back up)", res.BackupPath)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authorized_keys: %v", err)
	}
	if !strings.Contains(string(data), "nself-access:user=alice") {
		t.Errorf("authorized_keys does not contain alice's managed entry:\n%s", data)
	}
}

func TestGrant_IdempotentReGrant(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("first grant: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after first grant: %v", err)
	}

	res, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)})
	if err != nil {
		t.Fatalf("re-grant: %v", err)
	}
	if !res.AlreadyGranted {
		t.Error("re-granting the identical key must report AlreadyGranted")
	}
	if res.Fingerprint != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", res.Fingerprint, aliceFP)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after re-grant: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("re-grant must not change the file:\nbefore:\n%s\nafter:\n%s", before, after)
	}

	// Never a duplicate line: exactly one line carries alice's tag.
	count := strings.Count(string(after), "nself-access:user=alice")
	if count != 1 {
		t.Errorf("expected exactly 1 managed line for alice, found %d", count)
	}
}

func TestGrant_DifferentKeyReplacesEntry(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("first grant: %v", err)
	}
	res, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: bobKey(t)})
	if err != nil {
		t.Fatalf("second grant: %v", err)
	}
	if res.AlreadyGranted {
		t.Error("granting a different key for the same user must not be AlreadyGranted")
	}
	if res.Fingerprint != bobFP {
		t.Errorf("Fingerprint = %q, want %q", res.Fingerprint, bobFP)
	}
	if res.BackupPath == "" {
		t.Error("expected a backup path before replacing an existing key")
	}
	if _, err := os.Stat(res.BackupPath); err != nil {
		t.Errorf("backup file does not exist at %s: %v", res.BackupPath, err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authorized_keys: %v", err)
	}
	if strings.Count(string(data), "nself-access:user=alice") != 1 {
		t.Errorf("expected exactly one managed line for alice after replacement:\n%s", data)
	}
	if !strings.Contains(string(data), bobKeyLine[:40]) {
		t.Errorf("expected bob's key material in the file after replacement:\n%s", data)
	}
}

func TestGrant_DryRunMutatesNothing(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	res, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t), DryRun: true})
	if err != nil {
		t.Fatalf("Grant dry-run: %v", err)
	}
	if res.Fingerprint != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", res.Fingerprint, aliceFP)
	}
	if !strings.Contains(res.Diff, "+ ") {
		t.Errorf("dry-run diff should show an addition, got: %q", res.Diff)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("dry-run must not create authorized_keys, stat err = %v", err)
	}
}

func TestGrant_RecordsSudoDockerExpiry(t *testing.T) {
	tp, _ := newFixture(t)
	ctx := context.Background()
	expires := time.Date(2026, 12, 31, 0, 0, 0, 0, time.UTC)

	if _, err := Grant(ctx, tp, GrantRequest{
		User: "alice", Key: aliceKey(t), Sudo: true, Docker: true, Expires: &expires,
	}); err != nil {
		t.Fatalf("Grant: %v", err)
	}

	list, err := List(ctx, tp)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(list.Entries))
	}
	e := list.Entries[0]
	if !e.Sudo || !e.Docker {
		t.Errorf("Sudo=%v Docker=%v, want both true", e.Sudo, e.Docker)
	}
	if e.Expires == nil || !e.Expires.Equal(expires) {
		t.Errorf("Expires = %v, want %v", e.Expires, expires)
	}
}

func TestRevoke_RemovesEntry(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	// Grant two users so revoking one never trips the lockout guard.
	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("grant alice: %v", err)
	}
	if _, err := Grant(ctx, tp, GrantRequest{User: "bob", Key: bobKey(t)}); err != nil {
		t.Fatalf("grant bob: %v", err)
	}

	res, err := Revoke(ctx, tp, RevokeRequest{User: "alice"})
	if err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if res.Fingerprint != aliceFP {
		t.Errorf("Fingerprint = %q, want %q", res.Fingerprint, aliceFP)
	}
	if res.BackupPath == "" {
		t.Error("expected a backup path before revoking")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authorized_keys: %v", err)
	}
	if strings.Contains(string(data), "user=alice") {
		t.Errorf("alice's entry should be gone:\n%s", data)
	}
	if !strings.Contains(string(data), "user=bob") {
		t.Errorf("bob's entry should remain:\n%s", data)
	}
}

func TestRevoke_UnknownUser(t *testing.T) {
	tp, _ := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("grant: %v", err)
	}
	if _, err := Revoke(ctx, tp, RevokeRequest{User: "nobody"}); err == nil {
		t.Fatal("expected error revoking an unknown user")
	}
}

func TestRevoke_LastKeyRefusedWithoutForce(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("grant: %v", err)
	}

	_, err := Revoke(ctx, tp, RevokeRequest{User: "alice"})
	if err == nil {
		t.Fatal("expected ErrLastKey when revoking the only remaining key")
	}
	if !errors.Is(err, ErrLastKey) {
		t.Errorf("err = %v, want ErrLastKey", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authorized_keys: %v", err)
	}
	if !strings.Contains(string(data), "user=alice") {
		t.Error("refused revoke must leave the key in place")
	}
}

func TestRevoke_LastKeyAllowedWithForce(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("grant: %v", err)
	}

	if _, err := Revoke(ctx, tp, RevokeRequest{User: "alice", Force: true}); err != nil {
		t.Fatalf("Revoke with --force: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read authorized_keys: %v", err)
	}
	if strings.Contains(string(data), "user=alice") {
		t.Error("forced revoke should remove the last key")
	}
}

func TestRevoke_DryRunMutatesNothing(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("grant alice: %v", err)
	}
	if _, err := Grant(ctx, tp, GrantRequest{User: "bob", Key: bobKey(t)}); err != nil {
		t.Fatalf("grant bob: %v", err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	res, err := Revoke(ctx, tp, RevokeRequest{User: "alice", DryRun: true})
	if err != nil {
		t.Fatalf("Revoke dry-run: %v", err)
	}
	if !strings.Contains(res.Diff, "- ") {
		t.Errorf("dry-run diff should show a removal, got: %q", res.Diff)
	}

	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read after dry-run: %v", err)
	}
	if string(before) != string(after) {
		t.Error("dry-run revoke must not change the file")
	}
}

func TestList_ReturnsSortedEntriesAndForeignCount(t *testing.T) {
	tp, path := newFixture(t)
	ctx := context.Background()

	// Seed a foreign (non-nself-managed) key directly, as if it were the
	// original key the host shipped with.
	foreign := "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC7... original-root-key\n"
	if err := os.WriteFile(path, []byte(foreign), 0o600); err != nil {
		t.Fatalf("seed foreign key: %v", err)
	}

	if _, err := Grant(ctx, tp, GrantRequest{User: "bob", Key: bobKey(t)}); err != nil {
		t.Fatalf("grant bob: %v", err)
	}
	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("grant alice: %v", err)
	}

	list, err := List(ctx, tp)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list.Entries) != 2 {
		t.Fatalf("expected 2 managed entries, got %d", len(list.Entries))
	}
	if list.Entries[0].User != "alice" || list.Entries[1].User != "bob" {
		t.Errorf("entries not sorted by user: %v, %v", list.Entries[0].User, list.Entries[1].User)
	}
	if list.ForeignCount != 1 {
		t.Errorf("ForeignCount = %d, want 1", list.ForeignCount)
	}
}

func TestList_EmptyFile(t *testing.T) {
	tp, _ := newFixture(t)
	list, err := List(context.Background(), tp)
	if err != nil {
		t.Fatalf("List on missing file: %v", err)
	}
	if len(list.Entries) != 0 || list.ForeignCount != 0 {
		t.Errorf("expected an empty result, got %+v", list)
	}
}

func TestGrant_AuditLogWritten(t *testing.T) {
	tp, _ := newFixture(t)
	ctx := context.Background()

	if _, err := Grant(ctx, tp, GrantRequest{User: "alice", Key: aliceKey(t)}); err != nil {
		t.Fatalf("Grant: %v", err)
	}

	data, err := os.ReadFile(currentAuditLogPath())
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}
	line := string(data)
	if !strings.Contains(line, "action=grant") || !strings.Contains(line, "user=alice") ||
		!strings.Contains(line, aliceFP) {
		t.Errorf("audit log missing expected fields: %s", line)
	}
	if strings.Contains(line, "PRIVATE") {
		t.Error("audit log must never contain private key material")
	}
}
