// Package database owns SQLite connection setup, migrations, and concrete repositories.
package database

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/bitcli/bitcli/internal/model"
	"github.com/bitcli/bitcli/internal/utils"
)

// ModelRepository is the SQLite implementation of model.Repository.
type ModelRepository struct {
	db *DB
}

// NewModelRepository creates a SQLite-backed model repository.
func NewModelRepository(db *DB) *ModelRepository {
	return &ModelRepository{db: db}
}

// Upsert stores model metadata and associated files in a single transaction.
func (r *ModelRepository) Upsert(ctx context.Context, m model.Model, files []model.File) error {
	now := time.Now().UTC()
	if m.CreatedAt.IsZero() {
		m.CreatedAt = now
	}
	m.UpdatedAt = now
	return r.db.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `INSERT INTO models (
			id, user_id, canonical_id, backend, repo_id, revision, quantization, family, parameters,
			context_length, path, state, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(canonical_id) DO UPDATE SET
			user_id=excluded.user_id,
			backend=excluded.backend,
			repo_id=excluded.repo_id,
			revision=excluded.revision,
			quantization=excluded.quantization,
			family=excluded.family,
			parameters=excluded.parameters,
			context_length=excluded.context_length,
			path=excluded.path,
			state=excluded.state,
			updated_at=excluded.updated_at`,
			m.ID, m.UserID, m.CanonicalID, m.Backend, m.RepoID, m.Revision, m.Quantization,
			m.Family, m.Parameters, m.ContextLength, m.Path, string(m.State),
			m.CreatedAt.Format(time.RFC3339Nano), m.UpdatedAt.Format(time.RFC3339Nano))
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `DELETE FROM model_files WHERE model_id = ?`, m.ID); err != nil {
			return err
		}
		for _, f := range files {
			if f.CreatedAt.IsZero() {
				f.CreatedAt = now
			}
			f.UpdatedAt = now
			if _, err := tx.ExecContext(ctx, `INSERT INTO model_files (
				id, model_id, path, filename, size_bytes, sha256, etag, state, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				f.ID, m.ID, f.Path, f.Filename, f.SizeBytes, f.SHA256, f.ETag, string(f.State),
				f.CreatedAt.Format(time.RFC3339Nano), f.UpdatedAt.Format(time.RFC3339Nano)); err != nil {
				return err
			}
		}
		return nil
	})
}

// GetByUserID loads a model by its friendly user-facing ID.
func (r *ModelRepository) GetByUserID(ctx context.Context, userID string) (model.Model, error) {
	return r.get(ctx, `SELECT id, user_id, canonical_id, backend, repo_id, revision, quantization, family,
		parameters, context_length, path, state, created_at, updated_at FROM models
		WHERE user_id = ? AND state != ? LIMIT 1`, userID, string(model.StatePendingDelete))
}

// GetByCanonicalID loads a model by its concrete canonical artifact ID.
func (r *ModelRepository) GetByCanonicalID(ctx context.Context, canonicalID string) (model.Model, error) {
	return r.get(ctx, `SELECT id, user_id, canonical_id, backend, repo_id, revision, quantization, family,
		parameters, context_length, path, state, created_at, updated_at FROM models
		WHERE canonical_id = ? AND state != ? LIMIT 1`, canonicalID, string(model.StatePendingDelete))
}

// List returns all non-deleted local models.
func (r *ModelRepository) List(ctx context.Context) ([]model.Model, error) {
	rows, err := r.db.SQL.QueryContext(ctx, `SELECT id, user_id, canonical_id, backend, repo_id, revision,
		quantization, family, parameters, context_length, path, state, created_at, updated_at
		FROM models WHERE state != ? ORDER BY updated_at DESC`, string(model.StatePendingDelete))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var models []model.Model
	for rows.Next() {
		m, err := scanModel(rows)
		if err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

// MarkDeleting protects remove operations from partially completed filesystem deletes.
func (r *ModelRepository) MarkDeleting(ctx context.Context, id string) error {
	res, err := r.db.SQL.ExecContext(ctx, `UPDATE models SET state = ?, updated_at = ? WHERE id = ?`,
		string(model.StatePendingDelete), time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return utils.NewError(utils.CodeModelNotFound, "model is not installed")
	}
	return nil
}

// Delete removes model metadata. Files should already have been safely removed by cache.Service.
func (r *ModelRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.SQL.ExecContext(ctx, `DELETE FROM models WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return utils.NewError(utils.CodeModelNotFound, "model is not installed")
	}
	return nil
}

func (r *ModelRepository) get(ctx context.Context, query string, args ...any) (model.Model, error) {
	row := r.db.SQL.QueryRowContext(ctx, query, args...)
	m, err := scanModel(row)
	if errors.Is(err, sql.ErrNoRows) {
		return model.Model{}, utils.NewError(utils.CodeModelNotFound, "model is not installed")
	}
	return m, err
}

type modelScanner interface {
	Scan(dest ...any) error
}

func scanModel(scanner modelScanner) (model.Model, error) {
	var m model.Model
	var state, created, updated string
	if err := scanner.Scan(
		&m.ID, &m.UserID, &m.CanonicalID, &m.Backend, &m.RepoID, &m.Revision, &m.Quantization,
		&m.Family, &m.Parameters, &m.ContextLength, &m.Path, &state, &created, &updated,
	); err != nil {
		return model.Model{}, err
	}
	m.State = model.State(state)
	m.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	m.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return m, nil
}

