package utils

import (
	"flowday/internal/logger"
	"runtime/debug"
)

// Go runs the given function in a goroutine with panic recovery.
// Use this instead of 'go func()' for background tasks to prevent server crashes.
func Go(fn func()) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Log.Errorf("Recovered from panic in background goroutine: %v\nStack trace:\n%s", r, debug.Stack())
			}
		}()
		fn()
	}()
}
