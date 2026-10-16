// Package model defines BitCLI's model catalog, metadata, and persistence contracts.
package model

import (
	"context"
	"fmt"

	"github.com/bitcli/bitcli/internal/utils"
)

// Service coordinates model identity and persisted model metadata.
type Service struct {
	repo     Repository
	resolver Resolver
}

// NewService creates a model service from a repository and resolver.
func NewService(repo Repository, resolver Resolver) *Service {
	return &Service{repo: repo, resolver: resolver}
}

// Resolve returns a concrete artifact for the requested model.
func (s *Service) Resolve(id string) (Artifact, error) {
	return s.resolver.Resolve(id)
}

// Local returns a local model by user ID or canonical ID.
func (s *Service) Local(ctx context.Context, id string) (Model, error) {
	if m, err := s.repo.GetByUserID(ctx, id); err == nil {
		return m, nil
	}
	artifact, err := s.Resolve(id)
	if err != nil {
		return Model{}, err
	}
	if m, err := s.repo.GetByCanonicalID(ctx, artifact.CanonicalID); err == nil {
		return m, nil
	}
	return Model{}, utils.NewError(utils.CodeModelNotFound, fmt.Sprintf("model %s is not installed", id))
}

// List returns all local models.
func (s *Service) List(ctx context.Context) ([]Model, error) {
	return s.repo.List(ctx)
}

// Files returns the artifact files recorded for an installed model.
func (s *Service) Files(ctx context.Context, modelID string) ([]File, error) {
	return s.repo.Files(ctx, modelID)
}

// Save persists a local model and its files.
func (s *Service) Save(ctx context.Context, m Model, files []File) error {
	return s.repo.Upsert(ctx, m, files)
}

// Remove deletes model metadata after the caller has safely removed files.
func (s *Service) Remove(ctx context.Context, id string) (Model, error) {
	m, err := s.Local(ctx, id)
	if err != nil {
		return Model{}, err
	}
	if err := s.repo.MarkDeleting(ctx, m.ID); err != nil {
		return Model{}, err
	}
	if err := s.repo.Delete(ctx, m.ID); err != nil {
		return Model{}, err
	}
	return m, nil
}
