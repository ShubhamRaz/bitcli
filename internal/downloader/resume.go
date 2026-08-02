// Package downloader handles Hugging Face Hub transfers and local verification.
package downloader

import "os"

// ExistingPartialSize returns how many bytes can be resumed from a partial file.
func ExistingPartialSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
}

