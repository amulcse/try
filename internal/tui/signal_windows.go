//go:build windows

package tui

// SetupResizeHandler is a no-op on Windows as SIGWINCH is not supported
func SetupResizeHandler(callback func()) {
	// No-op on Windows - terminal resize handling not supported
}
