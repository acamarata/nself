package deprecation_test

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"

	"github.com/nself-org/cli/sdk/go/internal/deprecation"
)

func TestMark_EmitsWarn(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx := deprecation.WithLogger(context.Background(), logger)

	deprecation.Mark(ctx, "OldHandler", "NewHandler", "v1.0.9", "https://docs.nself.org/plugins/deprecation")

	out := buf.String()
	if !strings.Contains(out, "deprecated") {
		t.Errorf("expected 'deprecated' in log output, got: %q", out)
	}
	if !strings.Contains(out, "OldHandler") {
		t.Errorf("expected handler name in log output, got: %q", out)
	}
	if !strings.Contains(out, "NewHandler") {
		t.Errorf("expected replacement in log output, got: %q", out)
	}
}

func TestMark_HandlerStillExecutes(t *testing.T) {
	// Mark must not panic or block — handler must be able to continue normally.
	executed := false
	ctx := context.Background()

	deprecation.Mark(ctx, "OldHandler", "NewHandler", "v1.0.9", "https://docs.nself.org")
	executed = true

	if !executed {
		t.Error("code after Mark() must be reachable — Mark must not panic or block")
	}
}

func TestMark_LogFormat(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelWarn}))
	ctx := deprecation.WithLogger(context.Background(), logger)

	deprecation.Mark(ctx, "Legacy", "Modern", "v1.0.9", "https://docs.nself.org")

	out := buf.String()
	// JSON handler should emit structured fields
	if !strings.Contains(out, `"handler"`) {
		t.Errorf("structured log must include handler field, got: %q", out)
	}
	if !strings.Contains(out, `"since"`) {
		t.Errorf("structured log must include since field, got: %q", out)
	}
}
