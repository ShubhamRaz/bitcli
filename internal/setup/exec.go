// Package setup provides dependency detection, portable tool installation,
// and shell environment activation for the BitCLI self-contained setup flow.
package setup

import (
	"context"
	"os/exec"
)

// newCommand is a thin wrapper so tests can intercept exec calls.
// All internal exec calls in this package go through here.
func newCommand(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}
