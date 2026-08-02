// Package utils tests shared helper utilities.
package utils

import (
	"strings"
	"testing"
)

func TestNewIDHasPrefix(t *testing.T) {
	id := NewID("dl")
	if !strings.HasPrefix(id, "dl_") {
		t.Fatalf("expected prefix 'dl_', got %q", id)
	}
}

func TestNewIDIsRandom(t *testing.T) {
	a := NewID("x")
	b := NewID("x")
	if a == b {
		t.Fatal("expected two different IDs, got the same")
	}
}

func TestNewIDTrimsTrailingUnderscore(t *testing.T) {
	// prefix already has trailing underscore — should not double-up
	id := NewID("msg_")
	if strings.HasPrefix(id, "msg__") {
		t.Fatalf("double underscore in ID: %q", id)
	}
}

func TestSanitizeModelPathSegment_SlashReplaced(t *testing.T) {
	out := SanitizeModelPathSegment("microsoft/BitNet-b1.58-2B-4T")
	if strings.Contains(out, "/") {
		t.Fatalf("path separator not removed: %q", out)
	}
}

func TestSanitizeModelPathSegment_ColonReplaced(t *testing.T) {
	out := SanitizeModelPathSegment("model:v1")
	if strings.Contains(out, ":") {
		t.Fatalf("colon not removed: %q", out)
	}
}

func TestSanitizeModelPathSegment_BackslashReplaced(t *testing.T) {
	out := SanitizeModelPathSegment(`C:\Users\bitnet`)
	if strings.Contains(out, `\`) {
		t.Fatalf("backslash not removed: %q", out)
	}
}

func TestSanitizeModelPathSegment_TrimSpace(t *testing.T) {
	out := SanitizeModelPathSegment("  model  ")
	if strings.HasPrefix(out, " ") || strings.HasSuffix(out, " ") {
		t.Fatalf("leading/trailing space not trimmed: %q", out)
	}
}
