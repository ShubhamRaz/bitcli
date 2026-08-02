// Package database owns SQLite connection setup, migrations, and concrete repositories.
package database

import (
	"context"
	"database/sql"
	"time"
)

// DownloadRecord stores resumable download state for one artifact.
type DownloadRecord struct {
	ID          string
	RepoID      string
	Revision    string
	Filename    string
	TargetPath  string
	PartialPath string
	BytesDone   int64
	BytesTotal  int64
	ETag        string
	State       string
	Error       string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// DownloadRepository stores resumable download state in SQLite.
type DownloadRepository struct {
	db *DB
}

// NewDownloadRepository creates a SQLite-backed download repository.
func NewDownloadRepository(db *DB) *DownloadRepository {
	return &DownloadRepository{db: db}
}

// Upsert stores the latest download state.
func (r *DownloadRepository) Upsert(ctx context.Context, rec DownloadRecord) error {
	now := time.Now().UTC()
	if rec.CreatedAt.IsZero() {
		rec.CreatedAt = now
	}
	rec.UpdatedAt = now
	_, err := r.db.SQL.ExecContext(ctx, `INSERT INTO downloads (
		id, repo_id, revision, filename, target_path, partial_path, bytes_done, bytes_total,
		etag, state, error, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id) DO UPDATE SET
		bytes_done=excluded.bytes_done,
		bytes_total=excluded.bytes_total,
		etag=excluded.etag,
		state=excluded.state,
		error=excluded.error,
		updated_at=excluded.updated_at`,
		rec.ID, rec.RepoID, rec.Revision, rec.Filename, rec.TargetPath, rec.PartialPath,
		rec.BytesDone, rec.BytesTotal, rec.ETag, rec.State, rec.Error,
		rec.CreatedAt.Format(time.RFC3339Nano), rec.UpdatedAt.Format(time.RFC3339Nano))
	return err
}

// ByArtifact returns the current download state for an artifact, if present.
func (r *DownloadRepository) ByArtifact(ctx context.Context, repoID, revision, filename string) (DownloadRecord, bool, error) {
	row := r.db.SQL.QueryRowContext(ctx, `SELECT id, repo_id, revision, filename, target_path, partial_path,
		bytes_done, bytes_total, etag, state, error, created_at, updated_at
		FROM downloads WHERE repo_id = ? AND revision = ? AND filename = ? LIMIT 1`, repoID, revision, filename)

	var rec DownloadRecord
	var created, updated string
	if err := row.Scan(&rec.ID, &rec.RepoID, &rec.Revision, &rec.Filename, &rec.TargetPath, &rec.PartialPath,
		&rec.BytesDone, &rec.BytesTotal, &rec.ETag, &rec.State, &rec.Error, &created, &updated); err != nil {
		if err == sql.ErrNoRows {
			return DownloadRecord{}, false, nil
		}
		return DownloadRecord{}, false, err
	}
	rec.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	rec.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return rec, true, nil
}
