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
func (s *Service) RemoveModel(m model.Model) error {
	if m.Path == "" {
		return nil
	}
	dir := filepath.Dir(m.Path)
	if !s.layout.IsInsideModelRoot(dir) {
		return utils.NewError(utils.CodeInvalidInput, "refusing to delete a model outside the BitCLI cache")
	}
	if err := os.RemoveAll(dir); err != nil {
		return err
	}
	return nil
}

