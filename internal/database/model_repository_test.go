// Package database tests the SQLite model repository.
package database

import (
	"context"
	"testing"
	"time"

	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/utils"
)

func sampleModel() model.Model {
	now := time.Now().UTC()
	return model.Model{
		ID:            utils.NewID("model"),
		UserID:        "microsoft/BitNet-b1.58-2B-4T",
		CanonicalID:   "microsoft/bitnet-b1.58-2B-4T-gguf",
		Backend:       "bitnet",
		RepoID:        "microsoft/bitnet-b1.58-2B-4T-gguf",
		Revision:      "main",
		Quantization:  "i2_s",
		Family:        "BitNet b1.58",
		Parameters:    "2.4B",
		ContextLength: 4096,
		Path:          "/tmp/model.gguf",
		State:         model.StateReady,
		CreatedAt:     now,
		UpdatedAt:     now,
	}
}

func sampleFile(modelID, path string) model.File {
	now := time.Now().UTC()
	return model.File{
		ID:        utils.NewID("file"),
		ModelID:   modelID,
		Path:      path,
		Filename:  "ggml-model-i2_s.gguf",
		SizeBytes: 1190000000,
		SHA256:    "",
		State:     model.StateReady,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func TestModelRepository_UpsertAndGetByUserID(t *testing.T) {
	db := openTestDB(t)
	repo := NewModelRepository(db)
	ctx := context.Background()

	m := sampleModel()
	f := sampleFile(m.ID, m.Path)

	if err := repo.Upsert(ctx, m, []model.File{f}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByUserID(ctx, m.UserID)
	if err != nil {
		t.Fatalf("GetByUserID: %v", err)
	}
	if got.ID != m.ID {
		t.Fatalf("model ID mismatch: got %s, want %s", got.ID, m.ID)
	}
	if got.Backend != "bitnet" {
		t.Fatalf("unexpected backend: %s", got.Backend)
	}
}

func TestModelRepository_GetByCanonicalID(t *testing.T) {
	db := openTestDB(t)
	repo := NewModelRepository(db)
	ctx := context.Background()

	m := sampleModel()
	if err := repo.Upsert(ctx, m, nil); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err := repo.GetByCanonicalID(ctx, m.CanonicalID)
	if err != nil {
		t.Fatalf("GetByCanonicalID: %v", err)
	}
	if got.ID != m.ID {
		t.Fatalf("canonical ID lookup returned wrong model")
	}
}

func TestModelRepository_List(t *testing.T) {
	db := openTestDB(t)
	repo := NewModelRepository(db)
	ctx := context.Background()

	m := sampleModel()
	if err := repo.Upsert(ctx, m, nil); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	models, err := repo.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
}

func TestModelRepository_UpsertIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	repo := NewModelRepository(db)
	ctx := context.Background()

	m := sampleModel()
	if err := repo.Upsert(ctx, m, nil); err != nil {
		t.Fatalf("first Upsert: %v", err)
	}
	// Update path and upsert again — should not return UNIQUE violation.
	m.Path = "/tmp/updated.gguf"
	if err := repo.Upsert(ctx, m, nil); err != nil {
		t.Fatalf("second Upsert: %v", err)
	}
	got, _ := repo.GetByCanonicalID(ctx, m.CanonicalID)
	if got.Path != "/tmp/updated.gguf" {
		t.Fatalf("expected updated path, got %q", got.Path)
	}
}

func TestModelRepository_MarkDeletingAndDelete(t *testing.T) {
	db := openTestDB(t)
	repo := NewModelRepository(db)
	ctx := context.Background()

	m := sampleModel()
	if err := repo.Upsert(ctx, m, nil); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := repo.MarkDeleting(ctx, m.ID); err != nil {
		t.Fatalf("MarkDeleting: %v", err)
	}

	// After MarkDeleting, List should exclude the model.
	models, _ := repo.List(ctx)
	if len(models) != 0 {
		t.Fatalf("expected 0 models after MarkDeleting, got %d", len(models))
	}

	if err := repo.Delete(ctx, m.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
}

func TestModelRepository_GetByUserID_NotFound(t *testing.T) {
	db := openTestDB(t)
	repo := NewModelRepository(db)
	_, err := repo.GetByUserID(context.Background(), "nonexistent/model")
	if err == nil {
		t.Fatal("expected error for nonexistent model, got nil")
	}
}
