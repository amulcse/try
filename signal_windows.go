//go:build windows

package main

// Windows doesn't support SIGWINCH, so this is a no-op
func setupResizeHandler(callback func()) {
	// No-op on Windows - terminal resize handling not supported
}
