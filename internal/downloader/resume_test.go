// Package downloader tests resumable download helpers.
package downloader

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExistingPartialSize_Missing(t *testing.T) {
	size := ExistingPartialSize(filepath.Join(t.TempDir(), "nonexistent.partial"))
	if size != 0 {
		t.Fatalf("expected 0 for missing file, got %d", size)
	}
}

func TestExistingPartialSize_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "model.partial")
	content := []byte("partial content")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write partial file: %v", err)
	}
	size := ExistingPartialSize(path)
	if size != int64(len(content)) {
		t.Fatalf("expected %d bytes, got %d", len(content), size)
	}
}

func TestExistingPartialSize_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.partial")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatalf("write empty partial file: %v", err)
	}
	size := ExistingPartialSize(path)
	if size != 0 {
		t.Fatalf("expected 0 for empty file, got %d", size)
	}
}

func TestExistingPartialSize_Directory(t *testing.T) {
	// A directory should return 0 (IsDir guard).
	size := ExistingPartialSize(t.TempDir())
	if size != 0 {
		t.Fatalf("expected 0 for directory, got %d", size)
	}
}
