package access

import (
	"strings"
	"testing"
)

func TestParseAuthFile_PreservesForeignLines(t *testing.T) {
	content := "# a comment\n\nssh-rsa AAAAB3NzaC1yc2E foreign-key\n"
	f := parseAuthFile([]byte(content))
	if len(f.managed) != 0 {
		t.Errorf("expected no managed entries, got %d", len(f.managed))
	}
	if got := string(f.render()); got != content {
		t.Errorf("render() = %q, want unchanged %q", got, content)
	}
}

func TestParseAuthFile_MalformedTagTreatedAsForeign(t *testing.T) {
	// A line that starts the nself-access: tag but has no "user=" key must not
	// be treated as managed — a corrupt tag must never block the rest of the
	// file from being read.
	line := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIMHXHuK8L4SFSmmpHWBnzPFAcJGYHjABCulfo5ZbKvum nself-access:sudo=true"
	f := parseAuthFile([]byte(line + "\n"))
	if len(f.managed) != 0 {
		t.Errorf("expected the malformed tag line to be treated as foreign, got %d managed", len(f.managed))
	}
	if f.totalKeyLines() != 1 {
		t.Errorf("totalKeyLines() = %d, want 1 (still counts as a key)", f.totalKeyLines())
	}
}

func TestAuthFile_UpsertThenRemove(t *testing.T) {
	f := parseAuthFile(nil)
	e := Entry{User: "alice"}
	akey, _ := ParsePublicKey(aliceKeyLine)
	e.Key = akey

	f.upsert(e)
	if _, ok := f.get("alice"); !ok {
		t.Fatal("expected alice to be present after upsert")
	}
	if f.totalKeyLines() != 1 {
		t.Errorf("totalKeyLines() = %d, want 1", f.totalKeyLines())
	}

	if !f.remove("alice") {
		t.Fatal("remove() = false, want true")
	}
	if _, ok := f.get("alice"); ok {
		t.Error("expected alice to be gone after remove")
	}
	if f.totalKeyLines() != 0 {
		t.Errorf("totalKeyLines() = %d, want 0", f.totalKeyLines())
	}
}

func TestAuthFile_RemoveReindexesLaterEntries(t *testing.T) {
	f := parseAuthFile(nil)
	akey, _ := ParsePublicKey(aliceKeyLine)
	bkey, _ := ParsePublicKey(bobKeyLine)
	f.upsert(Entry{User: "alice", Key: akey})
	f.upsert(Entry{User: "bob", Key: bkey})

	if !f.remove("alice") {
		t.Fatal("remove(alice) = false")
	}
	bob, ok := f.get("bob")
	if !ok {
		t.Fatal("bob missing after removing alice")
	}
	if bob.User != "bob" {
		t.Errorf("bob.User = %q after reindex", bob.User)
	}
}

func TestAuthFile_RemoveUnknownUser(t *testing.T) {
	f := parseAuthFile(nil)
	if f.remove("nobody") {
		t.Error("remove() of an unknown user should return false")
	}
}

func TestEntry_RenderRoundTrip(t *testing.T) {
	akey, _ := ParsePublicKey(aliceKeyLine)
	e := Entry{User: "alice", Key: akey, Sudo: true, Docker: false}
	line := e.render()
	if !strings.Contains(line, "user=alice") || !strings.Contains(line, "sudo=true") || !strings.Contains(line, "docker=false") {
		t.Errorf("render() missing expected fields: %s", line)
	}

	parsed, ok := parseEntry(line)
	if !ok {
		t.Fatalf("parseEntry() failed to parse its own render() output: %s", line)
	}
	if parsed.User != "alice" || !parsed.Sudo || parsed.Docker {
		t.Errorf("round-tripped entry = %+v", parsed)
	}
}
