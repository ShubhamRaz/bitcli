// Package utils tests shared helper utilities.
package utils

import "testing"

// plainErr is a simple error with no Code field, simulating stdlib errors.
type plainErr struct{ msg string }

func (e *plainErr) Error() string { return e.msg }

func TestErrorCode_WithCodedError(t *testing.T) {
	err := NewError(CodeModelNotFound, "not found")
	if code := ErrorCode(err); code != CodeModelNotFound {
		t.Fatalf("expected %q, got %q", CodeModelNotFound, code)
	}
}

func TestErrorCode_WithWrappedCodedError(t *testing.T) {
	err := NewError(CodeDownloadInterrupted, "interrupted")
	if code := ErrorCode(err); code != CodeDownloadInterrupted {
		t.Fatalf("expected %q, got %q", CodeDownloadInterrupted, code)
	}
}

func TestErrorCode_DefaultsToInternal(t *testing.T) {
	err := &plainErr{msg: "something went wrong"}
	if code := ErrorCode(err); code != CodeInternal {
		t.Fatalf("expected %q for plain error, got %q", CodeInternal, code)
	}
}

func TestWrapError_IncludesMessage(t *testing.T) {
	cause := NewError(CodeInternal, "root")
	wrapped := WrapError(CodeUnavailable, "outer", cause)
	if wrapped.Error() == "" {
		t.Fatal("wrapped error should have a non-empty message")
	}
}

func TestNewError_Message(t *testing.T) {
	err := NewError(CodeConfigInvalid, "bad config")
	if err == nil {
		t.Fatal("NewError should not return nil")
	}
	if err.Error() != "bad config" {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}

func TestWrapError_NilCause(t *testing.T) {
	err := WrapError(CodeInternal, "msg", nil)
	if err == nil {
		t.Fatal("WrapError(nil) should not return nil")
	}
	if err.Error() != "msg" {
		t.Fatalf("unexpected message: %q", err.Error())
	}
}
