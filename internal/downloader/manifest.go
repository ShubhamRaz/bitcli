// Package downloader handles Hugging Face Hub transfers and local verification.
package downloader

import "github.com/bitcli/bitcli/internal/model"

// Manifest describes the concrete file BitCLI must download for a model.
type Manifest struct {
	Artifact model.Artifact
	URL      string
	ETag     string
	Size     int64
}

// DownloadState describes persisted transfer progress.
type DownloadState string

const (
	DownloadStateRunning DownloadState = "running"
	DownloadStateReady   DownloadState = "ready"
	DownloadStateFailed  DownloadState = "failed"
)

