package instance

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
)

// Store defines the interface for instance data persistence.
type Store interface {
	Save(instance *BrowserInstance) (*BrowserInstance, error)
	Get(id string) (*BrowserInstance, error)
	List(filter *InstanceFilter) ([]*BrowserInstance, error)
	Update(instance *BrowserInstance) error
	Delete(id string) error
	Count(filter *InstanceFilter) (int, error)
}

// PostgresStore implements Store using PostgreSQL.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates a new PostgresStore.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// Save inserts a new instance into the database.
func (s *PostgresStore) Save(instance *BrowserInstance) (*BrowserInstance, error) {
	fingerprintJSON, err := json.Marshal(instance.Fingerprint)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal fingerprint: %w", err)
	}

	query := `
		INSERT INTO browser_instances (id, status, fingerprint_json, proxy_id, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, started_at, last_active_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
		RETURNING created_at
	`

	err = s.db.QueryRow(
		query,
		instance.ID,
		instance.Status,
		fingerprintJSON,
		instance.ProxyID,
		instance.AccountID,
		instance.CDPEndpoint,
		instance.PID,
		instance.Port,
		instance.UserDataDir,
		instance.Group,
		instance.StartedAt,
		instance.LastActiveAt,
		instance.CreatedAt,
	).Scan(&instance.CreatedAt)

	if err != nil {
		return nil, fmt.Errorf("failed to save instance: %w", err)
	}

	return instance, nil
}

// Get retrieves an instance by ID.
func (s *PostgresStore) Get(id string) (*BrowserInstance, error) {
	query := `
		SELECT id, status, fingerprint_json, proxy_id, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, started_at, last_active_at, created_at
		FROM browser_instances
		WHERE id = $1
	`

	var instance BrowserInstance
	var fingerprintJSON []byte

	err := s.db.QueryRow(query, id).Scan(
		&instance.ID,
		&instance.Status,
		&fingerprintJSON,
		&instance.ProxyID,
		&instance.AccountID,
		&instance.CDPEndpoint,
		&instance.PID,
		&instance.Port,
		&instance.UserDataDir,
		&instance.Group,
		&instance.StartedAt,
		&instance.LastActiveAt,
		&instance.CreatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, ErrInstanceNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if err := json.Unmarshal(fingerprintJSON, &instance.Fingerprint); err != nil {
		return nil, fmt.Errorf("failed to unmarshal fingerprint: %w", err)
	}

	return &instance, nil
}

// List returns instances matching the filter.
func (s *PostgresStore) List(filter *InstanceFilter) ([]*BrowserInstance, error) {
	query := `
		SELECT id, status, fingerprint_json, proxy_id, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, started_at, last_active_at, created_at
		FROM browser_instances
		WHERE 1=1
	`

	var args []interface{}
	argIdx := 1

	if filter != nil {
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.Group != "" {
			query += fmt.Sprintf(" AND group_name = $%d", argIdx)
			args = append(args, filter.Group)
			argIdx++
		}
		if filter.ProxyID != "" {
			query += fmt.Sprintf(" AND proxy_id = $%d", argIdx)
			args = append(args, filter.ProxyID)
			argIdx++
		}
		if filter.AccountID != "" {
			query += fmt.Sprintf(" AND account_id = $%d", argIdx)
			args = append(args, filter.AccountID)
			argIdx++
		}
	}

	query += " ORDER BY created_at DESC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	defer rows.Close()

	var instances []*BrowserInstance
	for rows.Next() {
		var instance BrowserInstance
		var fingerprintJSON []byte

		err := rows.Scan(
			&instance.ID,
			&instance.Status,
			&fingerprintJSON,
			&instance.ProxyID,
			&instance.AccountID,
			&instance.CDPEndpoint,
			&instance.PID,
			&instance.Port,
			&instance.UserDataDir,
			&instance.Group,
			&instance.StartedAt,
			&instance.LastActiveAt,
			&instance.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan instance: %w", err)
		}

		if err := json.Unmarshal(fingerprintJSON, &instance.Fingerprint); err != nil {
			return nil, fmt.Errorf("failed to unmarshal fingerprint: %w", err)
		}

		instances = append(instances, &instance)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("row iteration error: %w", err)
	}

	return instances, nil
}

// Update updates an existing instance.
func (s *PostgresStore) Update(instance *BrowserInstance) error {
	fingerprintJSON, err := json.Marshal(instance.Fingerprint)
	if err != nil {
		return fmt.Errorf("failed to marshal fingerprint: %w", err)
	}

	query := `
		UPDATE browser_instances
		SET status = $2, fingerprint_json = $3, proxy_id = $4, account_id = $5, cdp_endpoint = $6, pid = $7, port = $8, user_data_dir = $9, group_name = $10, started_at = $11, last_active_at = $12
		WHERE id = $1
	`

	result, err := s.db.Exec(
		query,
		instance.ID,
		instance.Status,
		fingerprintJSON,
		instance.ProxyID,
		instance.AccountID,
		instance.CDPEndpoint,
		instance.PID,
		instance.Port,
		instance.UserDataDir,
		instance.Group,
		instance.StartedAt,
		instance.LastActiveAt,
	)

	if err != nil {
		return fmt.Errorf("failed to update instance: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrInstanceNotFound
	}

	return nil
}

// Delete removes an instance by ID.
func (s *PostgresStore) Delete(id string) error {
	query := `DELETE FROM browser_instances WHERE id = $1`

	result, err := s.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return ErrInstanceNotFound
	}

	return nil
}

// Count returns the number of instances matching the filter.
func (s *PostgresStore) Count(filter *InstanceFilter) (int, error) {
	query := `SELECT COUNT(*) FROM browser_instances WHERE 1=1`

	var args []interface{}
	argIdx := 1

	if filter != nil {
		if filter.Status != nil {
			query += fmt.Sprintf(" AND status = $%d", argIdx)
			args = append(args, *filter.Status)
			argIdx++
		}
		if filter.Group != "" {
			query += fmt.Sprintf(" AND group_name = $%d", argIdx)
			args = append(args, filter.Group)
			argIdx++
		}
		if filter.ProxyID != "" {
			query += fmt.Sprintf(" AND proxy_id = $%d", argIdx)
			args = append(args, filter.ProxyID)
			argIdx++
		}
		if filter.AccountID != "" {
			query += fmt.Sprintf(" AND account_id = $%d", argIdx)
			args = append(args, filter.AccountID)
		}
	}

	query = strings.Replace(query, "AND 1=1", "", 1)

	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count instances: %w", err)
	}

	return count, nil
}