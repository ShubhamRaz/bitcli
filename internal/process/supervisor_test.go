// Package process tests the process restart supervisor.
package process

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestSupervisorSucceedsOnFirstTry(t *testing.T) {
	sv := Supervisor{Policy: RestartPolicy{MaxRestarts: 2, Delay: time.Millisecond}}
	calls := 0
	err := sv.Run(context.Background(), func(_ context.Context) error {
		calls++
		return nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 call, got %d", calls)
	}
}

func TestSupervisorRetriesOnError(t *testing.T) {
	sv := Supervisor{Policy: RestartPolicy{MaxRestarts: 2, Delay: time.Millisecond}}
	calls := 0
	err := sv.Run(context.Background(), func(_ context.Context) error {
		calls++
		return errors.New("transient failure")
	})
	if err == nil {
		t.Fatal("expected an error after exhausting retries")
	}
	// Called once initially + 2 retries = 3 total
	if calls != 3 {
		t.Fatalf("expected 3 calls (1 + 2 retries), got %d", calls)
	}
}

func TestSupervisorSucceedsOnSecondTry(t *testing.T) {
	sv := Supervisor{Policy: RestartPolicy{MaxRestarts: 3, Delay: time.Millisecond}}
	calls := 0
	err := sv.Run(context.Background(), func(_ context.Context) error {
		calls++
		if calls < 2 {
			return errors.New("not yet")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("expected success on second try, got %v", err)
	}
	if calls != 2 {
		t.Fatalf("expected 2 calls, got %d", calls)
	}
}

func TestSupervisorRespectsContextCancellation(t *testing.T) {
	sv := Supervisor{Policy: RestartPolicy{MaxRestarts: 10, Delay: 100 * time.Millisecond}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	err := sv.Run(ctx, func(_ context.Context) error {
		return errors.New("fail")
	})
	if err == nil {
		t.Fatal("expected error after context cancellation")
	}
}
