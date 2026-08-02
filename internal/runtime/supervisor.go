// Package runtime coordinates backend-neutral model execution.
package runtime

import (
	"context"
	"time"

	"github.com/bitcli/bitcli/internal/process"
)

// DefaultSupervisor returns a conservative restart supervisor for server processes.
func DefaultSupervisor() process.Supervisor {
	return process.Supervisor{
		Policy: process.RestartPolicy{
			MaxRestarts: 2,
			Delay:       2 * time.Second,
		},
	}
}

// KeepContextUsed documents that runtime supervisors are context-aware.
func KeepContextUsed(ctx context.Context) context.Context {
	return ctx
}

