// Package utils contains shared helpers that are intentionally small and dependency-light.
package utils

import (
	"crypto/rand"
	"encoding/hex"
	"strings"
)

// NewID returns a compact random identifier with a stable prefix.
func NewID(prefix string) string {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return prefix + "_unknown"
	}
	return strings.TrimSuffix(prefix, "_") + "_" + hex.EncodeToString(b[:])
}

// SanitizeModelPathSegment keeps a model path segment filesystem-safe.
func SanitizeModelPathSegment(s string) string {
	s = strings.TrimSpace(s)
	replacer := strings.NewReplacer("\\", "-", "/", "-", ":", "-", "..", "-")
	return replacer.Replace(s)
}

