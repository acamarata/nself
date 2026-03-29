package confirm

import (
	"io"
	"strings"
	"testing"
)

func TestConfirmDestruction_CorrectInput(t *testing.T) {
	r := strings.NewReader("myproject\n")
	w := io.Discard
	err := ConfirmDestruction("myproject", r, w)
	if err != nil {
		t.Errorf("expected nil, got %v", err)
	}
}

func TestConfirmDestruction_WrongInput(t *testing.T) {
	r := strings.NewReader("wrongname\n")
	w := io.Discard
	err := ConfirmDestruction("myproject", r, w)
	if err != ErrDestructionCanceled {
		t.Errorf("expected ErrDestructionCanceled, got %v", err)
	}
}

func TestConfirmDestruction_EOF(t *testing.T) {
	// Empty reader simulates EOF / Ctrl+C
	r := strings.NewReader("")
	w := io.Discard
	err := ConfirmDestruction("myproject", r, w)
	if err != ErrDestructionCanceled {
		t.Errorf("expected ErrDestructionCanceled on EOF, got %v", err)
	}
}
