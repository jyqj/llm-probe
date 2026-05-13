package persist

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
	path string
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create data dir: %w", err)
	}

	dsn := path + "?_pragma=journal_mode(wal)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(on)"
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(1)

	db := &DB{conn: conn, path: path}
	if err := db.Migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

func (db *DB) Close() error {
	if db == nil || db.conn == nil {
		return nil
	}
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	if db == nil {
		return nil
	}
	return db.conn
}

// Migrate applies the schema idempotently using CREATE IF NOT EXISTS.
// During development, delete data/detector.db to apply schema changes.
// Add a versioned migration system before production deployment.
func (db *DB) Migrate() error {
	_, err := db.conn.Exec(schemaSQL)
	return err
}
