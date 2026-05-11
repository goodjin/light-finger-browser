package sqlite

import (
	"database/sql"
	"time"
)

// TabRecord represents a tab record in the database
type TabRecord struct {
	ID                 string
	ContextID          string
	InstanceID         string
	FingerprintSeed    string
	FingerprintCountry string
	URL                string
	Title              string
	CreatedAt          string // stored as TEXT in format RFC3339
	LastActiveAt       string // stored as TEXT in format RFC3339
	ClosedAt           sql.NullTime
}

// TabStore handles tab persistence to database
type TabStore struct {
	db *DB
}

// NewTabStore creates a new TabStore
func NewTabStore(db *DB) *TabStore {
	return &TabStore{db: db}
}

// Save creates or updates a tab record
func (s *TabStore) Save(tab *TabRecord) error {
	query := `INSERT INTO browser_tabs 
		(id, context_id, instance_id, fingerprint_seed, fingerprint_country, url, title, created_at, last_active_at, closed_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			context_id = excluded.context_id,
			instance_id = excluded.instance_id,
			fingerprint_seed = excluded.fingerprint_seed,
			fingerprint_country = excluded.fingerprint_country,
			url = excluded.url,
			title = excluded.title,
			last_active_at = excluded.last_active_at,
			closed_at = excluded.closed_at`

	// Ensure timestamps are in RFC3339 format
	createdAt := tab.CreatedAt
	if createdAt == "" {
		createdAt = time.Now().Format(time.RFC3339)
	}
	lastActiveAt := tab.LastActiveAt
	if lastActiveAt == "" {
		lastActiveAt = time.Now().Format(time.RFC3339)
	}

	_, err := s.db.Exec(query,
		tab.ID, tab.ContextID, tab.InstanceID, tab.FingerprintSeed, tab.FingerprintCountry,
		tab.URL, tab.Title, createdAt, lastActiveAt, tab.ClosedAt,
	)
	return err
}

// Get retrieves a tab by ID
func (s *TabStore) Get(id string) (*TabRecord, error) {
	query := `SELECT id, context_id, instance_id, fingerprint_seed, fingerprint_country, url, title, created_at, last_active_at, closed_at
		FROM browser_tabs WHERE id = ?`

	row := s.db.QueryRow(query, id)
	return s.scanTab(row)
}

// List returns all tabs, optionally filtering by instance or closed status
func (s *TabStore) List(instanceID string, includeClosed bool) ([]*TabRecord, error) {
	query := `SELECT id, context_id, instance_id, fingerprint_seed, fingerprint_country, url, title, created_at, last_active_at, closed_at
		FROM browser_tabs WHERE 1=1`
	args := []interface{}{}

	if instanceID != "" {
		query += " AND instance_id = ?"
		args = append(args, instanceID)
	}

	if !includeClosed {
		query += " AND closed_at IS NULL"
	}

	query += " ORDER BY last_active_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tabs []*TabRecord
	for rows.Next() {
		tab, err := s.scanTabFromRows(rows)
		if err != nil {
			return nil, err
		}
		tabs = append(tabs, tab)
	}
	return tabs, rows.Err()
}

// ListOpenByInstance returns all open tabs for a specific instance
func (s *TabStore) ListOpenByInstance(instanceID string) ([]*TabRecord, error) {
	return s.List(instanceID, false)
}

// ListAllOpen returns all open tabs
func (s *TabStore) ListAllOpen() ([]*TabRecord, error) {
	return s.List("", false)
}

// UpdateClosedAt marks a tab as closed
func (s *TabStore) UpdateClosedAt(id string, closedAt time.Time) error {
	query := `UPDATE browser_tabs SET closed_at = ?, last_active_at = ? WHERE id = ?`
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(query, closedAt.Format(time.RFC3339), now, id)
	return err
}

// UpdateURL updates the URL and last_active_at for a tab
func (s *TabStore) UpdateURL(id string, url string) error {
	query := `UPDATE browser_tabs SET url = ?, last_active_at = ? WHERE id = ?`
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(query, url, now, id)
	return err
}

// UpdateTitle updates the title for a tab
func (s *TabStore) UpdateTitle(id string, title string) error {
	query := `UPDATE browser_tabs SET title = ?, last_active_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, title, time.Now(), id)
	return err
}

// Delete removes a tab record
func (s *TabStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM browser_tabs WHERE id = ?", id)
	return err
}

func (s *TabStore) scanTab(row *sql.Row) (*TabRecord, error) {
	var tab TabRecord
	var contextID, instanceID, fingerprintSeed, fingerprintCountry, url, title sql.NullString
	var createdAt, lastActiveAt, closedAt sql.NullString

	err := row.Scan(
		&tab.ID, &contextID, &instanceID, &fingerprintSeed, &fingerprintCountry,
		&url, &title, &createdAt, &lastActiveAt, &closedAt,
	)
	if err != nil {
		return nil, err
	}

	if contextID.Valid {
		tab.ContextID = contextID.String
	}
	if instanceID.Valid {
		tab.InstanceID = instanceID.String
	}
	if fingerprintSeed.Valid {
		tab.FingerprintSeed = fingerprintSeed.String
	}
	if fingerprintCountry.Valid {
		tab.FingerprintCountry = fingerprintCountry.String
	}
	if url.Valid {
		tab.URL = url.String
	}
	if title.Valid {
		tab.Title = title.String
	}
	if createdAt.Valid {
		tab.CreatedAt = createdAt.String
	}
	if lastActiveAt.Valid {
		tab.LastActiveAt = lastActiveAt.String
	}
	if closedAt.Valid {
		parsedTime, err := time.Parse(time.RFC3339, closedAt.String)
		if err == nil {
			tab.ClosedAt = sql.NullTime{Time: parsedTime, Valid: true}
		}
	}

	return &tab, nil
}

func (s *TabStore) scanTabFromRows(rows *sql.Rows) (*TabRecord, error) {
	var tab TabRecord
	var contextID, instanceID, fingerprintSeed, fingerprintCountry, url, title sql.NullString
	var createdAt, lastActiveAt, closedAt sql.NullString

	err := rows.Scan(
		&tab.ID, &contextID, &instanceID, &fingerprintSeed, &fingerprintCountry,
		&url, &title, &createdAt, &lastActiveAt, &closedAt,
	)
	if err != nil {
		return nil, err
	}

	if contextID.Valid {
		tab.ContextID = contextID.String
	}
	if instanceID.Valid {
		tab.InstanceID = instanceID.String
	}
	if fingerprintSeed.Valid {
		tab.FingerprintSeed = fingerprintSeed.String
	}
	if fingerprintCountry.Valid {
		tab.FingerprintCountry = fingerprintCountry.String
	}
	if url.Valid {
		tab.URL = url.String
	}
	if title.Valid {
		tab.Title = title.String
	}
	if createdAt.Valid {
		tab.CreatedAt = createdAt.String
	}
	if lastActiveAt.Valid {
		tab.LastActiveAt = lastActiveAt.String
	}
	if closedAt.Valid {
		parsedTime, err := time.Parse(time.RFC3339, closedAt.String)
		if err == nil {
			tab.ClosedAt = sql.NullTime{Time: parsedTime, Valid: true}
		}
	}

	return &tab, nil
}
