// Package cache owns BitCLI's on-disk model cache layout and safety checks.
package cache

import (
	"os"
	"path/filepath"

	"github.com/bitcli/bitcli/internal/config"
	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/utils"
)

// Service performs safe cache filesystem operations.
type Service struct {
	layout Layout
}

// NewService creates a cache service from configured paths.
func NewService(paths config.Paths) *Service {
	return &Service{layout: Layout{
		ModelRoot:    paths.ModelDir,
		DownloadRoot: paths.DownloadDir,
		BackendRoot:  paths.BackendDir,
	}}
}

// Layout returns the immutable cache layout.
func (s *Service) Layout() Layout {
	return s.layout
}

// Ensure prepares cache directories.
func (s *Service) Ensure() error {
	return s.layout.Ensure()
}

// RemoveModel deletes the cached files for a model after verifying cache containment.
// Siblings lists other installed models: when another model shares the same
// artifact file (e.g. the base and -chat variants resolve to one GGUF), the
// shared file is kept so the surviving variant stays usable.
func (s *Service) RemoveModel(m model.Model, siblings []model.Model) error {
	if m.Path == "" {
		return nil
	}
	dir := filepath.Dir(m.Path)
	if !s.layout.IsInsideModelRoot(dir) {
		return utils.NewError(utils.CodeInvalidInput, "refusing to delete a model outside the BitCLI cache")
	}
	for _, other := range siblings {
		if other.ID != m.ID && other.Path == m.Path {
			// The artifact file is still referenced by another
			// installed model; only drop this model's metadata.
			return nil
		}
	}
	if err := os.Remove(m.Path); err != nil && !os.IsNotExist(err) {
		return err
	}
	// Drop any preparation markers left next to the artifact.
	markers, _ := filepath.Glob(filepath.Join(dir, ".bitcli-prepared-*"))
	for _, marker := range markers {
		_ = os.Remove(marker)
	}
	// Remove the directory only when it is now empty; ignore failures.
	_ = os.Remove(dir)
	return nil
}
