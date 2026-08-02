// Package runtime coordinates backend-neutral model execution.
package runtime

import "time"

// Session records a running backend interaction.
type Session struct {
	ID        string
	ModelID   string
	Backend   string
	StartedAt time.Time
}

