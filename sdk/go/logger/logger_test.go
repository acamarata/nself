package logger

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"
)

func TestNewJSON(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{Plugin: "ai", Version: "1.0.0", Writer: &buf, Level: slog.LevelInfo})
	log.Info("hello")
	out := buf.String()
	if !strings.Contains(out, `"plugin":"ai"`) {
		t.Errorf("expected plugin=ai in output, got: %s", out)
	}
	if !strings.Contains(out, `"version":"1.0.0"`) {
		t.Errorf("expected version=1.0.0 in output, got: %s", out)
	}
	if !strings.Contains(out, `"msg":"hello"`) {
		t.Errorf("expected msg=hello, got: %s", out)
	}
}

func TestNewText(t *testing.T) {
	var buf bytes.Buffer
	log := New(Options{Plugin: "ai", Format: "text", Writer: &buf, Level: slog.LevelDebug})
	log.Debug("x")
	if !strings.Contains(buf.String(), "plugin=ai") {
		t.Errorf("expected plugin=ai in text output, got: %s", buf.String())
	}
}

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug":   slog.LevelDebug,
		"DEBUG":   slog.LevelDebug,
		"info":    slog.LevelInfo,
		"":        slog.LevelInfo,
		"warn":    slog.LevelWarn,
		"warning": slog.LevelWarn,
		"error":   slog.LevelError,
		"err":     slog.LevelError,
		"weird":   slog.LevelInfo,
	}
	for in, want := range cases {
		got := ParseLevel(in)
		if got != want {
			t.Errorf("ParseLevel(%q)=%v, want %v", in, got, want)
		}
	}
}
