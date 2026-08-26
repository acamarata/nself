package access

// Purpose: orchestrates Grant, Revoke, and List against a Transport —
// reading the current authorized_keys, deciding what changes, backing up
// before any mutation, writing the result, and recording an audit line.
// Inputs: a Transport and a GrantRequest / RevokeRequest.
// Outputs: GrantResult / RevokeResult / ListResult, each carrying the
// fingerprint(s) involved for on-screen verification.
// Constraints: Grant is idempotent (re-granting the same key for the same
// user is a no-op); Revoke refuses to remove the last remaining key on the
// host without --force (ErrLastKey); --dry-run never calls Transport.Backup
// or Transport.Write.

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ErrLastKey is returned by Revoke when removing the requested key would
// leave the host with zero authorized keys — the lockout failure mode this
// command exists to prevent.
var ErrLastKey = errors.New("refusing to remove the last remaining authorized key: this would lock out all SSH access to the host; pass --force to override")

// GrantRequest describes one `nself access grant` invocation.
type GrantRequest struct {
	User    string
	Key     PublicKey
	Sudo    bool
	Docker  bool
	Expires *time.Time
	DryRun  bool
}

// GrantResult reports what Grant did or would do.
type GrantResult struct {
	AlreadyGranted bool
	Fingerprint    string
	BackupPath     string
	Diff           string
}

// Grant adds or updates req.User's managed key on t. Re-granting the exact
// same key with the same --sudo/--docker/--expires for the same user is a
// no-op (AlreadyGranted=true) — it never duplicates a line. Granting a
// different key, or different metadata, for an existing user replaces that
// user's single managed line in place.
func Grant(ctx context.Context, t Transport, req GrantRequest) (GrantResult, error) {
	fp, err := req.Key.Fingerprint()
	if err != nil {
		return GrantResult{}, fmt.Errorf("invalid key: %w", err)
	}

	current, err := t.Read(ctx)
	if err != nil {
		return GrantResult{}, fmt.Errorf("read authorized_keys from %s: %w", t.Describe(), err)
	}
	file := parseAuthFile(current)

	existing, hadExisting := file.get(req.User)
	if hadExisting {
		existingFP, _ := existing.Key.Fingerprint()
		if existingFP == fp && existing.Sudo == req.Sudo && existing.Docker == req.Docker &&
			sameExpiry(existing.Expires, req.Expires) {
			return GrantResult{AlreadyGranted: true, Fingerprint: fp}, nil
		}
	}

	entry := Entry{
		User:    req.User,
		Key:     req.Key,
		Sudo:    req.Sudo,
		Docker:  req.Docker,
		Expires: req.Expires,
		Granted: time.Now(),
	}

	var diff strings.Builder
	if hadExisting {
		fmt.Fprintf(&diff, "- %s\n", existing.render())
	}
	fmt.Fprintf(&diff, "+ %s\n", entry.render())

	if req.DryRun {
		return GrantResult{Fingerprint: fp, Diff: diff.String()}, nil
	}

	file.upsert(entry)

	backupPath, err := t.Backup(ctx)
	if err != nil {
		return GrantResult{}, fmt.Errorf("backup authorized_keys on %s: %w", t.Describe(), err)
	}
	if err := t.Write(ctx, file.render()); err != nil {
		return GrantResult{}, fmt.Errorf("write authorized_keys on %s: %w", t.Describe(), err)
	}

	auditErr := writeAudit(auditRecord{
		Action: "grant", Host: t.Describe(), User: req.User,
		Fingerprint: fp, Sudo: req.Sudo, Docker: req.Docker, Expires: req.Expires,
	})

	result := GrantResult{Fingerprint: fp, BackupPath: backupPath, Diff: diff.String()}
	if auditErr != nil {
		return result, fmt.Errorf("granted, but failed to write audit log: %w", auditErr)
	}
	return result, nil
}

// RevokeRequest describes one `nself access revoke` invocation.
type RevokeRequest struct {
	User   string
	Force  bool
	DryRun bool
}

// RevokeResult reports what Revoke did or would do.
type RevokeResult struct {
	Fingerprint string
	BackupPath  string
	Diff        string
}

// Revoke removes req.User's managed key from t. It refuses to proceed
// (ErrLastKey) when the target host would be left with zero authorized keys,
// unless req.Force is set.
func Revoke(ctx context.Context, t Transport, req RevokeRequest) (RevokeResult, error) {
	current, err := t.Read(ctx)
	if err != nil {
		return RevokeResult{}, fmt.Errorf("read authorized_keys from %s: %w", t.Describe(), err)
	}
	file := parseAuthFile(current)

	existing, ok := file.get(req.User)
	if !ok {
		return RevokeResult{}, fmt.Errorf("no nself-managed key found for user %q on %s", req.User, t.Describe())
	}
	fp := existing.Fingerprint()

	if file.totalKeyLines() <= 1 && !req.Force {
		return RevokeResult{}, ErrLastKey
	}

	diff := fmt.Sprintf("- %s\n", existing.render())
	if req.DryRun {
		return RevokeResult{Fingerprint: fp, Diff: diff}, nil
	}

	file.remove(req.User)

	backupPath, err := t.Backup(ctx)
	if err != nil {
		return RevokeResult{}, fmt.Errorf("backup authorized_keys on %s: %w", t.Describe(), err)
	}
	if err := t.Write(ctx, file.render()); err != nil {
		return RevokeResult{}, fmt.Errorf("write authorized_keys on %s: %w", t.Describe(), err)
	}

	auditErr := writeAudit(auditRecord{Action: "revoke", Host: t.Describe(), User: req.User, Fingerprint: fp})

	result := RevokeResult{Fingerprint: fp, BackupPath: backupPath, Diff: diff}
	if auditErr != nil {
		return result, fmt.Errorf("revoked, but failed to write audit log: %w", auditErr)
	}
	return result, nil
}

// ListResult reports every nself-managed entry found, plus a count of
// foreign (non-nself-managed) key lines sharing the file.
type ListResult struct {
	Entries      []Entry
	ForeignCount int
}

// List reads t's authorized_keys and returns every nself-managed entry,
// sorted by user label.
func List(ctx context.Context, t Transport) (ListResult, error) {
	current, err := t.Read(ctx)
	if err != nil {
		return ListResult{}, fmt.Errorf("read authorized_keys from %s: %w", t.Describe(), err)
	}
	file := parseAuthFile(current)

	entries := file.entries()
	sort.Slice(entries, func(i, j int) bool { return entries[i].User < entries[j].User })

	return ListResult{
		Entries:      entries,
		ForeignCount: file.totalKeyLines() - len(entries),
	}, nil
}

// sameExpiry reports whether a and b represent the same expiry, treating two
// nil pointers as equal.
func sameExpiry(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}
