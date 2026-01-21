//go:build !windows

package tui

import (
	"os"
	"os/signal"
	"syscall"
)

// SetupResizeHandler sets up a handler for terminal resize events (Unix only)
func SetupResizeHandler(callback func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			callback()
		}
	}()
}
