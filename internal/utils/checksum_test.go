// Package utils tests shared helper utilities.
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256File_KnownContent(t *testing.T) {
	content := []byte("hello bitcli")
	dir := t.TempDir()
	path := filepath.Join(dir, "test.bin")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	got, err := SHA256File(path)
	if err != nil {
		t.Fatalf("SHA256File returned error: %v", err)
	}

	// Compute expected independently.
	h := sha256.Sum256(content)
	want := hex.EncodeToString(h[:])

	if got != want {
		t.Fatalf("hash mismatch: got %s, want %s", got, want)
	}
}

func TestSHA256File_MissingFile(t *testing.T) {
	_, err := SHA256File(filepath.Join(t.TempDir(), "nonexistent"))
	if err == nil {
		t.Fatal("expected error for missing file, got nil")
	}
}

func TestSHA256File_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.bin")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	got, err := SHA256File(path)
	if err != nil {
		t.Fatalf("SHA256File returned error: %v", err)
	}
	// SHA256 of empty input is well-known
	h := sha256.Sum256(nil)
	want := hex.EncodeToString(h[:])
	if got != want {
		t.Fatalf("hash mismatch for empty file: got %s, want %s", got, want)
	}
}
