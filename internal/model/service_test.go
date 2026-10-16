// Package model tests BitCLI model service logic.
package model

import (
	"context"
	"errors"
	"testing"
)

// fakeRepo is a minimal in-memory Repository for testing.
type fakeRepo struct {
	byUserID      map[string]Model
	byCanonicalID map[string]Model
}

func (r *fakeRepo) Upsert(_ context.Context, m Model, _ []File) error {
	if r.byUserID == nil {
		r.byUserID = map[string]Model{}
	}
	if r.byCanonicalID == nil {
		r.byCanonicalID = map[string]Model{}
	}
	r.byUserID[m.UserID] = m
	r.byCanonicalID[m.CanonicalID] = m
	return nil
}

func (r *fakeRepo) GetByUserID(_ context.Context, id string) (Model, error) {
	m, ok := r.byUserID[id]
	if !ok {
		return Model{}, errors.New("not found")
	}
	return m, nil
}

func (r *fakeRepo) GetByCanonicalID(_ context.Context, id string) (Model, error) {
	m, ok := r.byCanonicalID[id]
	if !ok {
		return Model{}, errors.New("not found")
	}
	return m, nil
}

func (r *fakeRepo) List(_ context.Context) ([]Model, error) {
	out := make([]Model, 0, len(r.byUserID))
	for _, m := range r.byUserID {
		out = append(out, m)
	}
	return out, nil
}

func (r *fakeRepo) MarkDeleting(_ context.Context, _ string) error { return nil }

func (r *fakeRepo) Delete(_ context.Context, id string) error {
	for uid, m := range r.byUserID {
		if m.ID == id {
			delete(r.byUserID, uid)
			delete(r.byCanonicalID, m.CanonicalID)
			return nil
		}
	}
	return errors.New("not found")
}

func newTestService() (*Service, *fakeRepo) {
	repo := &fakeRepo{
		byUserID:      map[string]Model{},
		byCanonicalID: map[string]Model{},
	}
	resolver := NewResolver(DefaultCatalog())
	return NewService(repo, resolver), repo
}

func TestServiceSaveAndLocal(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	m := Model{
		ID:          "test-id",
		UserID:      "microsoft/BitNet-b1.58-2B-4T",
		CanonicalID: "microsoft/bitnet-b1.58-2B-4T-gguf",
		Backend:     "bitnet",
		State:       StateReady,
	}
	_ = repo.Upsert(ctx, m, nil)

	got, err := svc.Local(ctx, "microsoft/BitNet-b1.58-2B-4T")
	if err != nil {
		t.Fatalf("Local returned error: %v", err)
	}
	if got.ID != "test-id" {
		t.Fatalf("unexpected model ID: %s", got.ID)
	}
}

func TestServiceLocalFallsBackToCanonical(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	m := Model{
		ID:          "test-id",
		UserID:      "some-user-id",
		CanonicalID: "microsoft/bitnet-b1.58-2B-4T-gguf",
		Backend:     "bitnet",
		State:       StateReady,
	}
	_ = repo.Upsert(ctx, m, nil)

	// Look up by the official user ID — it should resolve via canonical ID.
	got, err := svc.Local(ctx, "microsoft/BitNet-b1.58-2B-4T")
	if err != nil {
		t.Fatalf("Local (canonical fallback) returned error: %v", err)
	}
	if got.ID != "test-id" {
		t.Fatalf("unexpected model ID via canonical fallback: %s", got.ID)
	}
}

func TestServiceList(t *testing.T) {
	svc, repo := newTestService()
	ctx := context.Background()

	_ = repo.Upsert(ctx, Model{ID: "a", UserID: "model-a", CanonicalID: "canon-a", State: StateReady}, nil)
	_ = repo.Upsert(ctx, Model{ID: "b", UserID: "model-b", CanonicalID: "canon-b", State: StateReady}, nil)

	models, err := svc.List(ctx)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
}
