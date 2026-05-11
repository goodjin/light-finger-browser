package sqlite

import (
	"database/sql"
	"time"
)

// AccessLogRecord represents an access log entry in the database
type AccessLogRecord struct {
	ID         string
	TabID      string
	URL        string
	Title      string
	VisitedAt  string // stored as TEXT in format RFC3339
	DurationMs int64
}

// AccessLogStore handles access log persistence to database
type AccessLogStore struct {
	db *DB
}

// NewAccessLogStore creates a new AccessLogStore
func NewAccessLogStore(db *DB) *AccessLogStore {
	return &AccessLogStore{db: db}
}

// Save creates a new access log record
func (s *AccessLogStore) Save(log *AccessLogRecord) error {
	query := `INSERT INTO access_logs (id, tab_id, url, title, visited_at, duration_ms)
		VALUES (?, ?, ?, ?, ?, ?)`

	visitedAt := log.VisitedAt
	if visitedAt == "" {
		visitedAt = time.Now().Format(time.RFC3339)
	}

	_, err := s.db.Exec(query,
		log.ID, log.TabID, log.URL, log.Title, visitedAt, log.DurationMs,
	)
	return err
}

// Get retrieves an access log by ID
func (s *AccessLogStore) Get(id string) (*AccessLogRecord, error) {
	query := `SELECT id, tab_id, url, title, visited_at, duration_ms
		FROM access_logs WHERE id = ?`

	row := s.db.QueryRow(query, id)
	return s.scanAccessLog(row)
}

// AccessLogFilter contains filter options for querying access logs
type AccessLogFilter struct {
	TabID     string // Filter by tab ID (empty means all tabs)
	StartTime string // Filter logs after this time (RFC3339 format, empty means no start limit)
	EndTime   string // Filter logs before this time (RFC3339 format, empty means no end limit)
}

// List returns all access logs, optionally filtering by tabID
func (s *AccessLogStore) List(tabID string) ([]*AccessLogRecord, error) {
	return s.ListWithFilter(&AccessLogFilter{TabID: tabID})
}

// ListWithFilter returns access logs with advanced filtering options
func (s *AccessLogStore) ListWithFilter(filter *AccessLogFilter) ([]*AccessLogRecord, error) {
	query := `SELECT id, tab_id, url, title, visited_at, duration_ms
		FROM access_logs WHERE 1=1`
	args := []interface{}{}

	if filter != nil {
		if filter.TabID != "" {
			query += " AND tab_id = ?"
			args = append(args, filter.TabID)
		}
		if filter.StartTime != "" {
			query += " AND visited_at >= ?"
			args = append(args, filter.StartTime)
		}
		if filter.EndTime != "" {
			query += " AND visited_at <= ?"
			args = append(args, filter.EndTime)
		}
	}

	query += " ORDER BY visited_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*AccessLogRecord
	for rows.Next() {
		log, err := s.scanAccessLogFromRows(rows)
		if err != nil {
			return nil, err
		}
		logs = append(logs, log)
	}
	return logs, rows.Err()
}

// ListByTab returns all access logs for a specific tab
func (s *AccessLogStore) ListByTab(tabID string) ([]*AccessLogRecord, error) {
	return s.List(tabID)
}

// ListAll returns all access logs
func (s *AccessLogStore) ListAll() ([]*AccessLogRecord, error) {
	return s.List("")
}

// Delete removes an access log record
func (s *AccessLogStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM access_logs WHERE id = ?", id)
	return err
}

// DeleteByTab removes all access logs for a specific tab
func (s *AccessLogStore) DeleteByTab(tabID string) error {
	_, err := s.db.Exec("DELETE FROM access_logs WHERE tab_id = ?", tabID)
	return err
}

func (s *AccessLogStore) scanAccessLog(row *sql.Row) (*AccessLogRecord, error) {
	var log AccessLogRecord
	var tabID, url, title, visitedAt sql.NullString
	var durationMs sql.NullInt64

	err := row.Scan(&log.ID, &tabID, &url, &title, &visitedAt, &durationMs)
	if err != nil {
		return nil, err
	}

	if tabID.Valid {
		log.TabID = tabID.String
	}
	if url.Valid {
		log.URL = url.String
	}
	if title.Valid {
		log.Title = title.String
	}
	if visitedAt.Valid {
		log.VisitedAt = visitedAt.String
	}
	if durationMs.Valid {
		log.DurationMs = durationMs.Int64
	}

	return &log, nil
}

func (s *AccessLogStore) scanAccessLogFromRows(rows *sql.Rows) (*AccessLogRecord, error) {
	var log AccessLogRecord
	var tabID, url, title, visitedAt sql.NullString
	var durationMs sql.NullInt64

	err := rows.Scan(&log.ID, &tabID, &url, &title, &visitedAt, &durationMs)
	if err != nil {
		return nil, err
	}

	if tabID.Valid {
		log.TabID = tabID.String
	}
	if url.Valid {
		log.URL = url.String
	}
	if title.Valid {
		log.Title = title.String
	}
	if visitedAt.Valid {
		log.VisitedAt = visitedAt.String
	}
	if durationMs.Valid {
		log.DurationMs = durationMs.Int64
	}

	return &log, nil
}
