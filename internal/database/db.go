// Package database owns SQLite connection setup, migrations, and concrete repositories.
package database

import (
	"context"
	"database/sql"

	_ "modernc.org/sqlite"
)

// DB wraps the SQLite handle used by BitCLI repositories.
type DB struct {
	SQL *sql.DB
}

// Open opens SQLite and applies migrations.
func Open(ctx context.Context, path string) (*DB, error) {
	// Per-connection PRAGMAs (busy_timeout, foreign_keys) must be part of
	// the DSN: database/sql recycles connections, and a freshly opened
	// replacement connection would otherwise silently lose them.
	dsn := path + "?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.ExecContext(ctx, "PRAGMA journal_mode=WAL;"); err != nil {
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
