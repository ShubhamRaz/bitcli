// Package downloader tests the retry delay helper.
package downloader

import (
	"testing"
	"time"
)

func TestRetryDelay_FirstAttempt(t *testing.T) {
	d := RetryDelay(1)
	if d != 1*time.Second {
		t.Fatalf("attempt 1: expected 1s, got %v", d)
	}
}

func TestRetryDelay_SecondAttempt(t *testing.T) {
	d := RetryDelay(2)
	if d != 4*time.Second {
		t.Fatalf("attempt 2: expected 4s, got %v", d)
	}
}

func TestRetryDelay_IsCappedAt15s(t *testing.T) {
	d := RetryDelay(100)
	if d != 15*time.Second {
		t.Fatalf("large attempt should cap at 15s, got %v", d)
	}
}

func TestRetryDelay_ZeroAttemptClamped(t *testing.T) {
	// Attempt 0 is clamped to 1, so delay should equal attempt 1.
	d := RetryDelay(0)
	if d != 1*time.Second {
		t.Fatalf("attempt 0: expected 1s (clamped), got %v", d)
	}
}
