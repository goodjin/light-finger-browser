package sqlite

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type DB struct {
	*sql.DB
}

func NewDB(path string) (*DB, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	// Enable WAL mode for better concurrent access
	_, err = db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		log.Printf("Warning: failed to enable WAL mode: %v", err)
	}

	// Set busy timeout to 5 seconds to prevent indefinite waiting
	_, err = db.Exec("PRAGMA busy_timeout=5000")
	if err != nil {
		log.Printf("Warning: failed to set busy_timeout: %v", err)
	}

	return &DB{db}, nil
}

func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) Migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS browser_instances (
			id TEXT PRIMARY KEY,
			status TEXT NOT NULL DEFAULT 'pending',
			fingerprint_json TEXT,
			proxy_id TEXT,
			account_id TEXT,
			cdp_endpoint TEXT,
			pid INTEGER,
			port INTEGER,
			user_data_dir TEXT,
			group_name TEXT,
			started_at DATETIME,
			last_active_at DATETIME,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS browser_tabs (
			id TEXT PRIMARY KEY,
			context_id TEXT NOT NULL,
			instance_id TEXT,
			fingerprint_seed TEXT,
			fingerprint_country TEXT,
			url TEXT,
			title TEXT,
			created_at TEXT,
			last_active_at TEXT,
			closed_at TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS access_logs (
			id TEXT PRIMARY KEY,
			tab_id TEXT NOT NULL,
			url TEXT NOT NULL,
			title TEXT,
			visited_at TEXT,
			duration_ms INTEGER,
			FOREIGN KEY (tab_id) REFERENCES browser_tabs(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_access_logs_tab_id ON access_logs(tab_id)`,
		`CREATE TABLE IF NOT EXISTS tiktok_accounts (
			id TEXT PRIMARY KEY,
			username TEXT,
			email TEXT UNIQUE NOT NULL,
			email_password TEXT,
			phone_id TEXT,
			phone_number TEXT,
			status TEXT NOT NULL DEFAULT 'pending',
			account_level INTEGER DEFAULT 0,
			account_group TEXT,
			instance_id TEXT,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS proxies (
			id TEXT PRIMARY KEY,
			ip TEXT NOT NULL,
			port INTEGER NOT NULL,
			country TEXT NOT NULL,
			city TEXT,
			type TEXT NOT NULL DEFAULT 'residential',
			username TEXT,
			password TEXT,
			status TEXT NOT NULL DEFAULT 'available',
			bind_id TEXT,
			bound_at DATETIME,
			last_check_at DATETIME,
			success_rate REAL DEFAULT 1.0,
			latency INTEGER DEFAULT 0,
			provider TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS fingerprint_snapshots (
			id TEXT PRIMARY KEY,
			account_id TEXT,
			instance_id TEXT,
			proxy_id TEXT,
			snapshot_json TEXT NOT NULL,
			created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS fingerprint_windows (
			id TEXT PRIMARY KEY,
			country TEXT NOT NULL,
			seed TEXT NOT NULL,
			context_id TEXT NOT NULL DEFAULT '',
			instance_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'active',
			created_at TEXT NOT NULL,
			last_active_at TEXT NOT NULL,
			closed_at TEXT,
			window_type TEXT NOT NULL DEFAULT 'window',
			parent_window_id TEXT NOT NULL DEFAULT '',
			title TEXT NOT NULL DEFAULT '',
			url TEXT NOT NULL DEFAULT ''
		)`,
		`CREATE INDEX IF NOT EXISTS idx_fingerprint_windows_instance ON fingerprint_windows(instance_id)`,
		`CREATE INDEX IF NOT EXISTS idx_fingerprint_windows_context ON fingerprint_windows(context_id)`,
		`CREATE INDEX IF NOT EXISTS idx_fingerprint_windows_status ON fingerprint_windows(status)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}

	accountColumns := map[string]string{
		"proxy_id":            "proxy_id TEXT",
		"proxy_url":           "proxy_url TEXT",
		"fingerprint_seed":    "fingerprint_seed TEXT",
		"fingerprint_country": "fingerprint_country TEXT",
		"instance_name":       "instance_name TEXT",
		"headless":            "headless INTEGER NOT NULL DEFAULT 0",
		"pending_restart":     "pending_restart INTEGER NOT NULL DEFAULT 0",
	}
	for name, definition := range accountColumns {
		if err := ensureColumn(db.DB, "tiktok_accounts", name, definition); err != nil {
			return err
		}
	}

	instanceColumns := map[string]string{
		"name":      "name TEXT",
		"proxy_url": "proxy_url TEXT",
		"headless":  "headless INTEGER NOT NULL DEFAULT 0",
	}
	for name, definition := range instanceColumns {
		if err := ensureColumn(db.DB, "browser_instances", name, definition); err != nil {
			return err
		}
	}

	tabColumns := map[string]string{
		"fingerprint_country": "fingerprint_country TEXT",
	}
	for name, definition := range tabColumns {
		if err := ensureColumn(db.DB, "browser_tabs", name, definition); err != nil {
			return err
		}
	}

	log.Println("Database migrations completed")
	return nil
}

func ensureColumn(db *sql.DB, table string, column string, definition string) error {
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name string
		var ctype string
		var notnull int
		var dflt sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s", table, definition))
	return err
}
