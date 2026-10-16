// Package process owns safe external process execution for backend adapters.
//go:build !windows

package process

import "os"

// InterruptSignal returns the platform interrupt signal for graceful shutdown.
func InterruptSignal() os.Signal {
	return os.Interrupt
}

