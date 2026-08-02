// Package model defines BitCLI's model catalog, metadata, and persistence contracts.
package model

import "context"

// Repository stores model and file metadata.
type Repository interface {
	Upsert(ctx context.Context, m Model, files []File) error
	GetByUserID(ctx context.Context, userID string) (Model, error)
	GetByCanonicalID(ctx context.Context, canonicalID string) (Model, error)
	List(ctx context.Context) ([]Model, error)
	MarkDeleting(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
}

