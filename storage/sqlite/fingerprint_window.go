package sqlite

import (
	"database/sql"
	"time"
)

// FingerprintWindowRecord represents a fingerprint window record in the database
type FingerprintWindowRecord struct {
	ID                 string         `json:"id"`
	Country            string         `json:"country"`
	Seed               string         `json:"seed"`
	ContextID          string         `json:"context_id"`
	InstanceID         string         `json:"instance_id"`
	Status             string         `json:"status"`
	CreatedAt          string         `json:"created_at"`  // RFC3339 format
	LastActiveAt       string         `json:"last_active_at"` // RFC3339 format
	ClosedAt           sql.NullTime   `json:"closed_at,omitempty"`
	WindowType         string         `json:"window_type"` // "window" or "tab"
	ParentWindowID     string         `json:"parent_window_id,omitempty"` // for tabs, the parent window ID
	Title              string         `json:"title,omitempty"`
	URL                string         `json:"url,omitempty"`
}

// FingerprintWindowStore handles fingerprint window persistence to database
type FingerprintWindowStore struct {
	db *DB
}

// NewFingerprintWindowStore creates a new FingerprintWindowStore
func NewFingerprintWindowStore(db *DB) *FingerprintWindowStore {
	return &FingerprintWindowStore{db: db}
}

// Save creates or updates a fingerprint window record
func (s *FingerprintWindowStore) Save(w *FingerprintWindowRecord) error {
	query := `INSERT INTO fingerprint_windows 
		(id, country, seed, context_id, instance_id, status, created_at, last_active_at, closed_at, window_type, parent_window_id, title, url)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			country = excluded.country,
			seed = excluded.seed,
			context_id = excluded.context_id,
			instance_id = excluded.instance_id,
			status = excluded.status,
			last_active_at = excluded.last_active_at,
			closed_at = excluded.closed_at,
			title = excluded.title,
			url = excluded.url`

	now := time.Now()
	createdAt := w.CreatedAt
	if createdAt == "" {
		createdAt = now.Format(time.RFC3339)
	}
	lastActiveAt := w.LastActiveAt
	if lastActiveAt == "" {
		lastActiveAt = now.Format(time.RFC3339)
	}

	_, err := s.db.Exec(query,
		w.ID, w.Country, w.Seed, w.ContextID, w.InstanceID, w.Status,
		createdAt, lastActiveAt, w.ClosedAt, w.WindowType, w.ParentWindowID, w.Title, w.URL,
	)
	return err
}

// Get retrieves a fingerprint window by ID
func (s *FingerprintWindowStore) Get(id string) (*FingerprintWindowRecord, error) {
	query := `SELECT id, country, seed, context_id, instance_id, status, created_at, last_active_at, closed_at, window_type, parent_window_id, title, url
		FROM fingerprint_windows WHERE id = ?`

	row := s.db.QueryRow(query, id)
	return s.scanWindow(row)
}

// GetByContextID retrieves a fingerprint window by context ID
func (s *FingerprintWindowStore) GetByContextID(contextID string) (*FingerprintWindowRecord, error) {
	query := `SELECT id, country, seed, context_id, instance_id, status, created_at, last_active_at, closed_at, window_type, parent_window_id, title, url
		FROM fingerprint_windows WHERE context_id = ? AND closed_at IS NULL LIMIT 1`

	row := s.db.QueryRow(query, contextID)
	return s.scanWindow(row)
}

// List returns all fingerprint windows, optionally filtered
func (s *FingerprintWindowStore) List(instanceID string, includeClosed bool) ([]*FingerprintWindowRecord, error) {
	query := `SELECT id, country, seed, context_id, instance_id, status, created_at, last_active_at, closed_at, window_type, parent_window_id, title, url
		FROM fingerprint_windows WHERE 1=1`
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

	var windows []*FingerprintWindowRecord
	for rows.Next() {
		w, err := s.scanWindowFromRows(rows)
		if err != nil {
			return nil, err
		}
		windows = append(windows, w)
	}
	return windows, rows.Err()
}

// ListOpen returns all open fingerprint windows
func (s *FingerprintWindowStore) ListOpen() ([]*FingerprintWindowRecord, error) {
	return s.List("", false)
}

// ListOpenByInstance returns all open fingerprint windows for a specific instance
func (s *FingerprintWindowStore) ListOpenByInstance(instanceID string) ([]*FingerprintWindowRecord, error) {
	return s.List(instanceID, false)
}

// UpdateStatus updates the status of a fingerprint window
func (s *FingerprintWindowStore) UpdateStatus(id string, status string) error {
	query := `UPDATE fingerprint_windows SET status = ?, last_active_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, status, time.Now().Format(time.RFC3339), id)
	return err
}

// UpdateURL updates the URL and last_active_at for a fingerprint window
func (s *FingerprintWindowStore) UpdateURL(id string, url string) error {
	query := `UPDATE fingerprint_windows SET url = ?, last_active_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, url, time.Now().Format(time.RFC3339), id)
	return err
}

// UpdateTitle updates the title for a fingerprint window
func (s *FingerprintWindowStore) UpdateTitle(id string, title string) error {
	query := `UPDATE fingerprint_windows SET title = ?, last_active_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, title, time.Now().Format(time.RFC3339), id)
	return err
}

// UpdateContextID updates the context ID for a fingerprint window
func (s *FingerprintWindowStore) UpdateContextID(id string, contextID string) error {
	query := `UPDATE fingerprint_windows SET context_id = ?, last_active_at = ? WHERE id = ?`
	_, err := s.db.Exec(query, contextID, time.Now().Format(time.RFC3339), id)
	return err
}

// UpdateClosedAt marks a fingerprint window as closed
func (s *FingerprintWindowStore) UpdateClosedAt(id string, closedAt time.Time) error {
	query := `UPDATE fingerprint_windows SET closed_at = ?, last_active_at = ?, status = ? WHERE id = ?`
	now := time.Now().Format(time.RFC3339)
	_, err := s.db.Exec(query, closedAt.Format(time.RFC3339), now, "closed", id)
	return err
}

// Delete removes a fingerprint window record
func (s *FingerprintWindowStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM fingerprint_windows WHERE id = ?", id)
	return err
}

// CountByInstance returns the count of open windows for an instance
func (s *FingerprintWindowStore) CountByInstance(instanceID string) (int, error) {
	query := `SELECT COUNT(*) FROM fingerprint_windows WHERE instance_id = ? AND closed_at IS NULL`
	var count int
	err := s.db.QueryRow(query, instanceID).Scan(&count)
	return count, err
}

// GetActiveContextIDs returns all active context IDs for an instance
func (s *FingerprintWindowStore) GetActiveContextIDs(instanceID string) ([]string, error) {
	query := `SELECT context_id FROM fingerprint_windows WHERE instance_id = ? AND closed_at IS NULL AND context_id != ''`
	rows, err := s.db.Query(query, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contextIDs []string
	for rows.Next() {
		var contextID string
		if err := rows.Scan(&contextID); err != nil {
			return nil, err
		}
		if contextID != "" {
			contextIDs = append(contextIDs, contextID)
		}
	}
	return contextIDs, rows.Err()
}

func (s *FingerprintWindowStore) scanWindow(row *sql.Row) (*FingerprintWindowRecord, error) {
	var w FingerprintWindowRecord
	var country, seed, contextID, instanceID, status, windowType, parentWindowID, title, url sql.NullString
	var createdAt, lastActiveAt, closedAt sql.NullString

	err := row.Scan(
		&w.ID, &country, &seed, &contextID, &instanceID, &status,
		&createdAt, &lastActiveAt, &closedAt, &windowType, &parentWindowID, &title, &url,
	)
	if err != nil {
		return nil, err
	}

	if country.Valid {
		w.Country = country.String
	}
	if seed.Valid {
		w.Seed = seed.String
	}
	if contextID.Valid {
		w.ContextID = contextID.String
	}
	if instanceID.Valid {
		w.InstanceID = instanceID.String
	}
	if status.Valid {
		w.Status = status.String
	}
	if createdAt.Valid {
		w.CreatedAt = createdAt.String
	}
	if lastActiveAt.Valid {
		w.LastActiveAt = lastActiveAt.String
	}
	if closedAt.Valid {
		parsedTime, err := time.Parse(time.RFC3339, closedAt.String)
		if err == nil {
			w.ClosedAt = sql.NullTime{Time: parsedTime, Valid: true}
		}
	}
	if windowType.Valid {
		w.WindowType = windowType.String
	}
	if parentWindowID.Valid {
		w.ParentWindowID = parentWindowID.String
	}
	if title.Valid {
		w.Title = title.String
	}
	if url.Valid {
		w.URL = url.String
	}

	return &w, nil
}

func (s *FingerprintWindowStore) scanWindowFromRows(rows *sql.Rows) (*FingerprintWindowRecord, error) {
	var w FingerprintWindowRecord
	var country, seed, contextID, instanceID, status, windowType, parentWindowID, title, url sql.NullString
	var createdAt, lastActiveAt, closedAt sql.NullString

	err := rows.Scan(
		&w.ID, &country, &seed, &contextID, &instanceID, &status,
		&createdAt, &lastActiveAt, &closedAt, &windowType, &parentWindowID, &title, &url,
	)
	if err != nil {
		return nil, err
	}

	if country.Valid {
		w.Country = country.String
	}
	if seed.Valid {
		w.Seed = seed.String
	}
	if contextID.Valid {
		w.ContextID = contextID.String
	}
	if instanceID.Valid {
		w.InstanceID = instanceID.String
	}
	if status.Valid {
		w.Status = status.String
	}
	if createdAt.Valid {
		w.CreatedAt = createdAt.String
	}
	if lastActiveAt.Valid {
		w.LastActiveAt = lastActiveAt.String
	}
	if closedAt.Valid {
		parsedTime, err := time.Parse(time.RFC3339, closedAt.String)
		if err == nil {
			w.ClosedAt = sql.NullTime{Time: parsedTime, Valid: true}
		}
	}
	if windowType.Valid {
		w.WindowType = windowType.String
	}
	if parentWindowID.Valid {
		w.ParentWindowID = parentWindowID.String
	}
	if title.Valid {
		w.Title = title.String
	}
	if url.Valid {
		w.URL = url.String
	}

	return &w, nil
}
