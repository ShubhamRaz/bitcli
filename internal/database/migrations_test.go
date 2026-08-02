// Package database tests that Migrate creates the required SQLite schema.
package database

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func openTestDB(t *testing.T) *DB {
	t.Helper()
	sqlDB, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open in-memory sqlite: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	if _, err := sqlDB.ExecContext(context.Background(),
		"PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
		t.Fatalf("pragma: %v", err)
	}
	db := &DB{SQL: sqlDB}
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	t.Cleanup(func() { _ = sqlDB.Close() })
	return db
}

func tableExists(t *testing.T, db *DB, name string) bool {
	t.Helper()
	row := db.SQL.QueryRowContext(context.Background(),
		`SELECT count(*) FROM sqlite_master WHERE type='table' AND name=?`, name)
	var n int
	if err := row.Scan(&n); err != nil {
		t.Fatalf("check table %s: %v", name, err)
	}
	return n > 0
}

func TestMigrateCreatesModelsTable(t *testing.T) {
	db := openTestDB(t)
	if !tableExists(t, db, "models") {
		t.Fatal("models table should exist after migration")
	}
}

func TestMigrateCreatesModelFilesTable(t *testing.T) {
	db := openTestDB(t)
	if !tableExists(t, db, "model_files") {
		t.Fatal("model_files table should exist after migration")
	}
}

func TestMigrateCreatesDownloadsTable(t *testing.T) {
	db := openTestDB(t)
	if !tableExists(t, db, "downloads") {
		t.Fatal("downloads table should exist after migration")
	}
}

func TestMigrateCreatesChatSessionsTable(t *testing.T) {
	db := openTestDB(t)
	if !tableExists(t, db, "chat_sessions") {
		t.Fatal("chat_sessions table should exist after migration")
	}
}

func TestMigrateCreatesChatMessagesTable(t *testing.T) {
	db := openTestDB(t)
	if !tableExists(t, db, "chat_messages") {
		t.Fatal("chat_messages table should exist after migration")
	}
}

func TestMigrateCreatesBackendsTable(t *testing.T) {
	db := openTestDB(t)
	if !tableExists(t, db, "backends") {
		t.Fatal("backends table should exist after migration")
	}
}

func TestMigrateCreatesSettingsTable(t *testing.T) {
	db := openTestDB(t)
	if !tableExists(t, db, "settings") {
		t.Fatal("settings table should exist after migration")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	db := openTestDB(t)
	// Running migration twice should not fail (CREATE TABLE IF NOT EXISTS).
	if err := db.Migrate(context.Background()); err != nil {
		t.Fatalf("second migration run returned error: %v", err)
	}
}
