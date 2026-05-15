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
	if _, err := db.conn.Exec(schemaSQL); err != nil {
		return err
	}
	for _, col := range []struct{ table, col, def string }{
		{"intelligence_history", "effort", "TEXT DEFAULT ''"},
		{"intelligence_history", "thinking_mode", "TEXT DEFAULT ''"},
	} {
		db.conn.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", col.table, col.col, col.def))
	}

	// health_states PK changed from (target_id, model) to (target_id, model, check_type).
	// Detect old schema and recreate — health states are ephemeral.
	var hasCheckType bool
	rows, err := db.conn.Query("PRAGMA table_info(health_states)")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int
			var name, typ string
			var notNull, pk int
			var dflt sql.NullString
			rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk)
			if name == "check_type" {
				hasCheckType = true
			}
		}
	}
	if !hasCheckType {
		db.conn.Exec("DROP TABLE IF EXISTS health_states")
		db.conn.Exec(`CREATE TABLE IF NOT EXISTS health_states (
			target_id TEXT NOT NULL,
			model TEXT NOT NULL,
			check_type TEXT NOT NULL DEFAULT 'channel',
			status TEXT,
			score REAL,
			grade TEXT,
			last_check TEXT,
			last_change TEXT,
			consec_fails INTEGER,
			consec_ok INTEGER,
			payload_json TEXT NOT NULL,
			PRIMARY KEY(target_id, model, check_type)
		)`)
	}

	return nil
}
