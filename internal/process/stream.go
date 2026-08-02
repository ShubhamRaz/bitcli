// Package process owns safe external process execution for backend adapters.
package process

import "strings"

// LooksLikeDiagnostic reports whether a backend line is likely diagnostic output.
func LooksLikeDiagnostic(text string) bool {
	lower := strings.ToLower(strings.TrimSpace(text))
	return strings.HasPrefix(lower, "error") ||
		strings.HasPrefix(lower, "warning") ||
		strings.Contains(lower, "traceback") ||
		strings.Contains(lower, "exception")
}

