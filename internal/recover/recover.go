package recover

import (
	"log/slog"
	"runtime/debug"
)

// SafeGo runs fn in a goroutine wrapped with panic recovery.
// Panics are logged via slog with stack trace and emit a "goroutine_panic" event.
func SafeGo(name string, fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("goroutine_panic",
					"goroutine", name,
					"panic", r,
					"stack", string(debug.Stack()))
			}
		}()
		fn()
	}()
}
