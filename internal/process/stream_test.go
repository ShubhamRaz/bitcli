// Package process tests stream diagnostic detection.
package process

import "testing"

func TestLooksLikeDiagnostic_ErrorPrefix(t *testing.T) {
	if !LooksLikeDiagnostic("Error: something failed") {
		t.Fatal("expected 'Error:' to be a diagnostic")
	}
}

func TestLooksLikeDiagnostic_WarningPrefix(t *testing.T) {
	if !LooksLikeDiagnostic("Warning: deprecated option") {
		t.Fatal("expected 'Warning:' to be a diagnostic")
	}
}

func TestLooksLikeDiagnostic_Traceback(t *testing.T) {
	if !LooksLikeDiagnostic("Traceback (most recent call last):") {
		t.Fatal("expected 'Traceback' to be a diagnostic")
	}
}

func TestLooksLikeDiagnostic_Exception(t *testing.T) {
	if !LooksLikeDiagnostic("RuntimeException: out of memory") {
		t.Fatal("expected 'Exception' to be a diagnostic")
	}
}

func TestLooksLikeDiagnostic_NormalOutput(t *testing.T) {
	if LooksLikeDiagnostic("the quick brown fox") {
		t.Fatal("normal text should not be classified as diagnostic")
	}
}

func TestLooksLikeDiagnostic_EmptyString(t *testing.T) {
	if LooksLikeDiagnostic("") {
		t.Fatal("empty string should not be classified as diagnostic")
	}
}

func TestLooksLikeDiagnostic_CaseInsensitive(t *testing.T) {
	// 'ERROR' uppercase should still be detected
	if !LooksLikeDiagnostic("ERROR: fatal") {
		t.Fatal("uppercase ERROR should be classified as diagnostic")
	}
}

func TestLooksLikeDiagnostic_LeadingWhitespace(t *testing.T) {
	if !LooksLikeDiagnostic("  warning: close enough") {
		t.Fatal("leading whitespace should not prevent diagnostic detection")
	}
}
