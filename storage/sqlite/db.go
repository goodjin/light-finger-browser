package sqlite

import (
	"database/sql"
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
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			return err
		}
	}

	log.Println("Database migrations completed")
	return nil
}
