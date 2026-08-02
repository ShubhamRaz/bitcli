// Package database owns SQLite connection setup, migrations, and concrete repositories.
package database

import "context"

// Migrate creates or upgrades BitCLI's local SQLite schema.
func (db *DB) Migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS models (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			canonical_id TEXT NOT NULL UNIQUE,
			backend TEXT NOT NULL,
			repo_id TEXT NOT NULL,
			revision TEXT NOT NULL,
			quantization TEXT NOT NULL,
			family TEXT NOT NULL,
			parameters TEXT NOT NULL,
			context_length INTEGER NOT NULL,
			path TEXT NOT NULL,
			state TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_models_user_id ON models(user_id);`,
		`CREATE TABLE IF NOT EXISTS model_files (
			id TEXT PRIMARY KEY,
			model_id TEXT NOT NULL REFERENCES models(id) ON DELETE CASCADE,
			path TEXT NOT NULL,
			filename TEXT NOT NULL,
			size_bytes INTEGER NOT NULL,
			sha256 TEXT NOT NULL,
			etag TEXT NOT NULL,
			state TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS downloads (
			id TEXT PRIMARY KEY,
			repo_id TEXT NOT NULL,
			revision TEXT NOT NULL,
			filename TEXT NOT NULL,
			target_path TEXT NOT NULL,
			partial_path TEXT NOT NULL,
			bytes_done INTEGER NOT NULL,
			bytes_total INTEGER NOT NULL,
			etag TEXT NOT NULL,
			state TEXT NOT NULL,
			error TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS backends (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL UNIQUE,
			path TEXT NOT NULL,
			version TEXT NOT NULL,
			revision TEXT NOT NULL,
			state TEXT NOT NULL,
			last_checked_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS runs (
			id TEXT PRIMARY KEY,
			model_id TEXT NOT NULL,
			backend TEXT NOT NULL,
			started_at TEXT NOT NULL,
			ended_at TEXT NOT NULL,
			exit_code INTEGER NOT NULL,
			error TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id TEXT PRIMARY KEY,
			title TEXT NOT NULL,
			model_id TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			token_count INTEGER NOT NULL,
			created_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);`,
		`INSERT OR IGNORE INTO schema_migrations(version, applied_at) VALUES (1, datetime('now'));`,
	}
	for _, stmt := range statements {
		if _, err := db.SQL.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	return nil
}

