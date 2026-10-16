// Package process owns safe external process execution for backend adapters.
package process

import (
	"context"
	"time"
)

// RestartPolicy controls bounded process restart behavior.
type RestartPolicy struct {
	MaxRestarts int
	Delay       time.Duration
}

// Supervisor reruns a function under a conservative restart policy.
type Supervisor struct {
	Policy RestartPolicy
}

// Run invokes fn and retries when it fails until the policy is exhausted.
func (s Supervisor) Run(ctx context.Context, fn func(context.Context) error) error {
	var err error
	for attempt := 0; attempt <= s.Policy.MaxRestarts; attempt++ {
		if err = fn(ctx); err == nil {
			return nil
		}
		if attempt == s.Policy.MaxRestarts {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(s.Policy.Delay):
		}
	}
	return err
}

