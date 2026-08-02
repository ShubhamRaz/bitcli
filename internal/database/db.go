// Package database owns SQLite connection setup, migrations, and concrete repositories.
package database

import (
	"context"
	"database/sql"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite handle used by BitCLI repositories.
type DB struct {
	SQL *sql.DB
}

// Open opens SQLite and applies migrations.
func Open(ctx context.Context, path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetConnMaxLifetime(30 * time.Minute)
	if _, err := sqlDB.ExecContext(ctx, "PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;"); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	db := &DB{SQL: sqlDB}
	if err := db.Migrate(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

// Close releases the SQLite connection.
func (db *DB) Close() error {
	if db == nil || db.SQL == nil {
		return nil
	}
	return db.SQL.Close()
}

