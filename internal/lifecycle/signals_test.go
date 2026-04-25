package lifecycle

import (
	"os"
	"syscall"
	"testing"
)

// TestSignalMessage_SIGINT verifies that SIGINT maps to the interrupted message.
func TestSignalMessage_SIGINT(t *testing.T) {
	msg := SignalMessage(os.Interrupt)
	want := "Interrupted. Cleaning up..."
	if msg != want {
		t.Errorf("SIGINT message: got %q, want %q", msg, want)
	}
}

// TestSignalMessage_SIGTERM verifies that SIGTERM maps to the terminating message.
func TestSignalMessage_SIGTERM(t *testing.T) {
	msg := SignalMessage(syscall.SIGTERM)
	want := "Terminating. Cleaning up..."
	if msg != want {
		t.Errorf("SIGTERM message: got %q, want %q", msg, want)
	}
}

// TestSignalMessage_Unknown verifies that an unknown signal returns the generic
// "Signal received" message.
func TestSignalMessage_Unknown(t *testing.T) {
	msg := SignalMessage(syscall.SIGUSR1)
	want := "Signal received. Cleaning up..."
	if msg != want {
		t.Errorf("unknown signal message: got %q, want %q", msg, want)
	}
}

// TestSignalMessage_TableDriven covers all documented signal cases.
func TestSignalMessage_TableDriven(t *testing.T) {
	cases := []struct {
		sig  os.Signal
		want string
	}{
		{os.Interrupt, "Interrupted. Cleaning up..."},
		{syscall.SIGTERM, "Terminating. Cleaning up..."},
		{syscall.SIGUSR1, "Signal received. Cleaning up..."},
		{syscall.SIGUSR2, "Signal received. Cleaning up..."},
		{syscall.SIGHUP, "Signal received. Cleaning up..."},
	}
	for _, tc := range cases {
		t.Run(tc.sig.String(), func(t *testing.T) {
			got := SignalMessage(tc.sig)
			if got != tc.want {
				t.Errorf("SignalMessage(%v): got %q, want %q", tc.sig, got, tc.want)
			}
		})
	}
}
