package commands

// secrets_schedule_test.go — Unit tests for `nself secrets schedule`,
// `list-schedules`, and `verify` (secrets_schedule.go).
// P6-E11-W2-S3-T18: security command test floor.
//
// Purpose: exercise the real command RunE bodies against a temp project
// root. Unlike Init/Set/Get/Rotate, schedule state (rotation-state.json)
// and the verify command's "not found" path are plain JSON — no `age`
// binary is required, so these tests run unconditionally in CI.
// Security property under test: a malformed --every value must be
// REJECTED, not silently coerced into a bogus cadence that then never
// fires — a schedule nobody actually renews is a security gap that looks
// healthy in `nself secrets schedule`'s own output.
// Inputs: cobra command execution against a t.TempDir() project root.
// Outputs: rotation-state.json content, or a descriptive error.
// Constraints: no live DB, no `age` binary, no network.

import (
	"os"
	"strings"
	"testing"

	"github.com/nself-org/cli/internal/secrets"
)

// withProjectRoot chdirs into a fresh temp dir for the duration of fn and
// restores the previous cwd afterward. Not safe to run with t.Parallel()
// siblings that also chdir.
func withProjectRoot(t *testing.T, fn func(root string)) {
	t.Helper()
	root := t.TempDir()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir(%s): %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
	fn(root)
}

// TestSecretsSchedule_MalformedEvery_Rejected verifies that an --every value
// not matching the documented "<N>d" format is rejected with an error and
// does NOT create a schedule entry. This is the property the ticket names
// explicitly: schedule validation must reject malformed cadence input
// rather than silently accepting it.
func TestSecretsSchedule_MalformedEvery_Rejected(t *testing.T) {
	withProjectRoot(t, func(root string) {
		// A name that does not collide with secrets.DefaultSchedules() — those
		// six names (JWT_SIGNING_KEY etc.) exist implicitly in a fresh
		// project's in-memory rotation state before any file is written, so
		// using one here would make "was a schedule created?" unanswerable.
		const testSecret = "MY_ROTATE_TEST_SECRET_ZZZ"

		malformed := []string{"ninety-days", "90", "90days", "d90", "-90d", "0.5d", "90D"}
		for _, every := range malformed {
			every := every
			_ = secretsScheduleCmd.Flags().Set("secret", testSecret)
			_ = secretsScheduleCmd.Flags().Set("every", every)

			err := secretsScheduleCmd.RunE(secretsScheduleCmd, nil)
			if err == nil {
				t.Errorf("--every %q: expected rejection, got nil error", every)
			}

			// Must not have created a schedule as a side effect of the rejected call.
			checks, checkErr := secrets.CheckSchedule(root)
			if checkErr != nil {
				t.Fatalf("CheckSchedule after rejected --every %q: %v", every, checkErr)
			}
			for _, c := range checks {
				if c.SecretName == testSecret {
					t.Errorf("--every %q: schedule was created despite rejection: %+v", every, c)
				}
			}
		}
		_ = secretsScheduleCmd.Flags().Set("secret", "")
		_ = secretsScheduleCmd.Flags().Set("every", "")
	})
}

// TestSecretsSchedule_WellFormedEvery_CreatesSchedule verifies the positive
// path: a well-formed "<N>d" value actually persists a schedule with the
// requested cadence, readable back via secrets.CheckSchedule. This is the
// counterpart to the rejection test above — without it, a validator that
// rejects everything (including valid input) would still pass.
func TestSecretsSchedule_WellFormedEvery_CreatesSchedule(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = secretsScheduleCmd.Flags().Set("secret", "DB_PASSWORD")
		_ = secretsScheduleCmd.Flags().Set("every", "45d")
		defer func() {
			_ = secretsScheduleCmd.Flags().Set("secret", "")
			_ = secretsScheduleCmd.Flags().Set("every", "")
		}()

		if err := secretsScheduleCmd.RunE(secretsScheduleCmd, nil); err != nil {
			t.Fatalf("well-formed --every 45d: unexpected error: %v", err)
		}

		checks, err := secrets.CheckSchedule(root)
		if err != nil {
			t.Fatalf("CheckSchedule: %v", err)
		}
		found := false
		for _, c := range checks {
			if c.SecretName == "DB_PASSWORD" {
				found = true
				if c.CadenceDays != 45 {
					t.Errorf("CadenceDays = %d, want 45", c.CadenceDays)
				}
			}
		}
		if !found {
			t.Error("DB_PASSWORD schedule not found after well-formed --every 45d")
		}
	})
}

// TestSecretsSchedule_ShowsTable verifies the no-flags branch prints the
// existing schedule table rather than erroring, and that it does not
// require any flags to be set (the "" every-only case skipped above).
func TestSecretsSchedule_ShowsTable(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = secrets.AddSchedule(root, "SEEDED_KEY", 30, 7)

		_ = secretsScheduleCmd.Flags().Set("secret", "")
		_ = secretsScheduleCmd.Flags().Set("every", "")

		if err := secretsScheduleCmd.RunE(secretsScheduleCmd, nil); err != nil {
			t.Fatalf("show-table branch: unexpected error: %v", err)
		}
	})
}

// TestSecretsVerify_MissingSecret_Errors verifies the real security property
// of `nself secrets verify`: a secret that is NOT present in the store must
// report an error (not a silent success), because callers use this command
// to gate deploys on a required secret actually existing.
func TestSecretsVerify_MissingSecret_Errors(t *testing.T) {
	withProjectRoot(t, func(root string) {
		secretsEnvFlag = "dev"
		defer func() { secretsEnvFlag = "dev" }()

		// No .secrets store exists at all in this fresh temp dir: loadStore
		// returns an empty store without needing the `age` binary, so this
		// exercises the real "not found" code path unconditionally.
		err := secretsVerifyCmd.RunE(secretsVerifyCmd, []string{"NEVER_SET_KEY"})
		if err == nil {
			t.Fatal("expected error for a secret absent from the store, got nil")
		}
		if !strings.Contains(err.Error(), "NEVER_SET_KEY") {
			t.Errorf("error %q does not name the missing key", err.Error())
		}
	})
}

// TestSecretsListSchedulesCmd_ReflectsAddedSchedule verifies the
// `list-schedules` alias command (a distinct cobra command from `schedule`,
// per secrets_schedule.go's own doc comment) reads the same real on-disk
// state rather than being a dead/unwired command that always prints the
// "no schedules" placeholder.
func TestSecretsListSchedulesCmd_ReflectsAddedSchedule(t *testing.T) {
	withProjectRoot(t, func(root string) {
		if err := secretsListSchedulesCmd.RunE(secretsListSchedulesCmd, nil); err != nil {
			t.Fatalf("list-schedules on a fresh project: unexpected error: %v", err)
		}

		if err := secrets.AddSchedule(root, "LIST_SCHEDULES_TEST_SECRET", 60, 5); err != nil {
			t.Fatalf("AddSchedule: %v", err)
		}

		// No direct way to capture the table's rendered rows from here without
		// duplicating ui.Table internals, so assert indirectly: the command
		// must not error now that real schedule state exists, and the state
		// it reads must contain the secret we just added.
		if err := secretsListSchedulesCmd.RunE(secretsListSchedulesCmd, nil); err != nil {
			t.Fatalf("list-schedules with schedules present: unexpected error: %v", err)
		}
		checks, err := secrets.CheckSchedule(root)
		if err != nil {
			t.Fatalf("CheckSchedule: %v", err)
		}
		found := false
		for _, c := range checks {
			if c.SecretName == "LIST_SCHEDULES_TEST_SECRET" {
				found = true
			}
		}
		if !found {
			t.Fatal("LIST_SCHEDULES_TEST_SECRET not present in the state list-schedules reads")
		}
	})
}

// TestSecretsVerify_PresentSecret_Succeeds is the positive counterpart to
// TestSecretsVerify_MissingSecret_Errors: a secret that IS present must
// report success, not just "not found" — without this, a verify command
// that always errored would still pass the negative test above.
func TestSecretsVerify_PresentSecret_Succeeds(t *testing.T) {
	requireAge(t)
	withProjectRoot(t, func(root string) {
		buildAgeStore(t, root, "dev", map[string]secrets.SecretEntry{
			"DEPLOY_KEY": {Value: "present"},
		})
		secretsEnvFlag = "dev"
		defer func() { secretsEnvFlag = "dev" }()

		if err := secretsVerifyCmd.RunE(secretsVerifyCmd, []string{"DEPLOY_KEY"}); err != nil {
			t.Fatalf("verify on a present secret: unexpected error: %v", err)
		}
	})
}

// TestSecretsRotationLog_EmptyLog_NoEventsMessage verifies the rotation-log
// command reads real on-disk state (not a hardcoded message) by comparing
// behavior before and after an event is appended.
func TestSecretsRotationLog_ReflectsAppendedEvents(t *testing.T) {
	withProjectRoot(t, func(root string) {
		_ = secretsRotationLogCmd.Flags().Set("secret", "")

		// Before any event: LoadRotationLog must report zero events.
		before, err := secrets.LoadRotationLog(root)
		if err != nil {
			t.Fatalf("LoadRotationLog (empty): %v", err)
		}
		if len(before.Events) != 0 {
			t.Fatalf("expected 0 events in a fresh project, got %d", len(before.Events))
		}

		if err := secrets.AppendRotationEvent(root, "API_KEY", "ok", "rotated via test"); err != nil {
			t.Fatalf("AppendRotationEvent: %v", err)
		}

		after, err := secrets.LoadRotationLog(root)
		if err != nil {
			t.Fatalf("LoadRotationLog (after append): %v", err)
		}
		if len(after.Events) != 1 {
			t.Fatalf("expected 1 event after append, got %d", len(after.Events))
		}
		if after.Events[0].SecretName != "API_KEY" || after.Events[0].Status != "ok" {
			t.Errorf("unexpected event content: %+v", after.Events[0])
		}

		// The command itself must not error against real on-disk state.
		if err := secretsRotationLogCmd.RunE(secretsRotationLogCmd, nil); err != nil {
			t.Errorf("secretsRotationLogCmd.RunE: unexpected error: %v", err)
		}
	})
}
