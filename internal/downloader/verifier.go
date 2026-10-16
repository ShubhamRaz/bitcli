// Package downloader handles Hugging Face Hub transfers and local verification.
package downloader

import (
	"fmt"

	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/utils"
)

// VerifyFile checks artifact checksums when a known SHA256 is available.
func VerifyFile(path string, artifact model.Artifact) (string, error) {
	sum, err := utils.SHA256File(path)
	if err != nil {
		return "", err
	}
	if artifact.SHA256 != "" && sum != artifact.SHA256 {
		return sum, utils.NewError(utils.CodeChecksumMismatch, fmt.Sprintf("checksum mismatch for %s", artifact.Filename))
	}
	return sum, nil
}

