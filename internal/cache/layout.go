// Package cache owns BitCLI's on-disk model cache layout and safety checks.
package cache

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/utils"
)

// Layout calculates safe paths under the configured cache root.
type Layout struct {
	ModelRoot    string
	DownloadRoot string
	BackendRoot string
}

// ModelDir returns the directory for a model artifact.
func (l Layout) ModelDir(artifact model.Artifact) string {
	return filepath.Join(l.ModelRoot, utils.SanitizeModelPathSegment(artifact.RepoID), artifact.Revision)
}

// ModelFile returns the target path for a model artifact file.
func (l Layout) ModelFile(artifact model.Artifact) string {
	return filepath.Join(l.ModelDir(artifact), artifact.Filename)
}

// PartialFile returns the resumable partial path for an artifact.
func (l Layout) PartialFile(artifact model.Artifact) string {
	name := utils.SanitizeModelPathSegment(artifact.RepoID + "-" + artifact.Revision + "-" + artifact.Filename)
	return filepath.Join(l.DownloadRoot, name+".partial")
}

// Ensure creates cache directories.
func (l Layout) Ensure() error {
	for _, dir := range []string{l.ModelRoot, l.DownloadRoot, l.BackendRoot} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	return nil
}

// IsInsideModelRoot verifies that a path is safely contained by the model root.
func (l Layout) IsInsideModelRoot(path string) bool {
	root, err := filepath.Abs(l.ModelRoot)
	if err != nil {
		return false
	}
	target, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	rel, err := filepath.Rel(root, target)
	if err != nil {
		return false
	}
	return rel == "." || (!strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel))
}

