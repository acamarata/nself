package build

import (
	"os"
	"path/filepath"
	"testing"
)

// A profile switch changes which services are emitted but touches neither
// .env's mtime nor the CLI version, so NeedsRebuild cannot see it. Before this
// check, `nself build --profile ops` on a project last built as "app" found the
// cache fresh, regenerated nothing, exited 0, and left redis/minio/mailpit in
// the compose file of a server explicitly asked to exclude them.
func TestProfileChanged_DetectsSwitch(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".nself"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := RecordProfile(dir, "app"); err != nil {
		t.Fatal(err)
	}
	if ProfileChanged(dir, "app") {
		t.Error("same profile must not force a rebuild")
	}
	if !ProfileChanged(dir, "ops") {
		t.Error("switching app -> ops must force a rebuild")
	}
}

// An empty profile is the default, "app". Treating "" and "app" as different
// would rebuild on every invocation that omits the flag.
func TestProfileChanged_EmptyMeansApp(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".nself"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := RecordProfile(dir, ""); err != nil {
		t.Fatal(err)
	}
	if ProfileChanged(dir, "app") {
		t.Error(`recorded "" then asked for "app" — same thing, should not rebuild`)
	}
	if ProfileChanged(dir, "") {
		t.Error(`recorded "" then asked for "" — should not rebuild`)
	}
}

// No record means we cannot know what produced the existing compose file. The
// safe answer is to rebuild: a stale compose for the wrong profile is the exact
// failure this guards against.
func TestProfileChanged_MissingRecordRebuilds(t *testing.T) {
	if !ProfileChanged(t.TempDir(), "app") {
		t.Error("missing profile record must force a rebuild")
	}
}
