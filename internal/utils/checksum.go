// Package utils contains shared helpers that are intentionally small and dependency-light.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
)

// SHA256File calculates the SHA256 hash for a file without loading it all into memory.
func SHA256File(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

