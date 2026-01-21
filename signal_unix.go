//go:build !windows

package main

import (
	"os"
	"os/signal"
	"syscall"
)

func setupResizeHandler(callback func()) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	go func() {
		for range sigCh {
			callback()
		}
	}()
}
