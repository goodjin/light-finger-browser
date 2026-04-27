package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
)

type InstanceStore struct {
	db *DB
}

func NewInstanceStore(db *DB) *InstanceStore {
	return &InstanceStore{db: db}
}

func (s *InstanceStore) Save(inst *instance.BrowserInstance) (*instance.BrowserInstance, error) {
	fpJSON, _ := json.Marshal(inst.Fingerprint)

	query := `INSERT INTO browser_instances
		(id, status, fingerprint_json, proxy_id, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, started_at, last_active_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	inst.CreatedAt = now
	inst.LastActiveAt = now

	_, err := s.db.Exec(query,
		inst.ID, inst.Status, fpJSON, inst.ProxyID, inst.AccountID,
		inst.CDPEndpoint, inst.PID, inst.Port, inst.UserDataDir, inst.Group,
		inst.StartedAt, inst.LastActiveAt, inst.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func (s *InstanceStore) Get(id string) (*instance.BrowserInstance, error) {
	query := `SELECT id, status, fingerprint_json, proxy_id, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, started_at, last_active_at, created_at
		FROM browser_instances WHERE id = ?`

	row := s.db.QueryRow(query, id)
	return s.scanInstance(row)
}

func (s *InstanceStore) List(filter *instance.InstanceFilter) ([]*instance.BrowserInstance, error) {
	query := `SELECT id, status, fingerprint_json, proxy_id, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, started_at, last_active_at, created_at
		FROM browser_instances WHERE 1=1`
	args := []interface{}{}

	if filter != nil {
		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, *filter.Status)
		}
		if filter.Group != "" {
			query += " AND group_name = ?"
			args = append(args, filter.Group)
		}
		if filter.ProxyID != "" {
			query += " AND proxy_id = ?"
			args = append(args, filter.ProxyID)
		}
		if filter.AccountID != "" {
			query += " AND account_id = ?"
			args = append(args, filter.AccountID)
		}
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var instances []*instance.BrowserInstance
	for rows.Next() {
		inst, err := s.scanInstanceFromRows(rows)
		if err != nil {
			return nil, err
		}
		instances = append(instances, inst)
	}
	return instances, nil
}

func (s *InstanceStore) Update(inst *instance.BrowserInstance) error {
	fpJSON, _ := json.Marshal(inst.Fingerprint)
	inst.LastActiveAt = time.Now()

	query := `UPDATE browser_instances SET
		status = ?, fingerprint_json = ?, proxy_id = ?, account_id = ?,
		cdp_endpoint = ?, pid = ?, port = ?, user_data_dir = ?, group_name = ?,
		started_at = ?, last_active_at = ?
		WHERE id = ?`

	_, err := s.db.Exec(query,
		inst.Status, fpJSON, inst.ProxyID, inst.AccountID,
		inst.CDPEndpoint, inst.PID, inst.Port, inst.UserDataDir, inst.Group,
		inst.StartedAt, inst.LastActiveAt, inst.ID,
	)
	return err
}

func (s *InstanceStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM browser_instances WHERE id = ?", id)
	return err
}

func (s *InstanceStore) Count(filter *instance.InstanceFilter) (int, error) {
	query := `SELECT COUNT(*) FROM browser_instances WHERE 1=1`
	args := []interface{}{}

	if filter != nil {
		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, *filter.Status)
		}
		if filter.Group != "" {
			query += " AND group_name = ?"
			args = append(args, filter.Group)
		}
	}

	var count int
	err := s.db.QueryRow(query, args...).Scan(&count)
	return count, err
}

func (s *InstanceStore) scanInstance(row *sql.Row) (*instance.BrowserInstance, error) {
	var inst instance.BrowserInstance
	var fpJSON sql.NullString

	err := row.Scan(
		&inst.ID, &inst.Status, &fpJSON, &inst.ProxyID, &inst.AccountID,
		&inst.CDPEndpoint, &inst.PID, &inst.Port, &inst.UserDataDir, &inst.Group,
		&inst.StartedAt, &inst.LastActiveAt, &inst.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if fpJSON.Valid {
		var fp fingerprint.Fingerprint
		json.Unmarshal([]byte(fpJSON.String), &fp)
		inst.Fingerprint = &fp
	}

	return &inst, nil
}

func (s *InstanceStore) scanInstanceFromRows(rows *sql.Rows) (*instance.BrowserInstance, error) {
	var inst instance.BrowserInstance
	var fpJSON sql.NullString

	err := rows.Scan(
		&inst.ID, &inst.Status, &fpJSON, &inst.ProxyID, &inst.AccountID,
		&inst.CDPEndpoint, &inst.PID, &inst.Port, &inst.UserDataDir, &inst.Group,
		&inst.StartedAt, &inst.LastActiveAt, &inst.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	if fpJSON.Valid {
		var fp fingerprint.Fingerprint
		json.Unmarshal([]byte(fpJSON.String), &fp)
		inst.Fingerprint = &fp
	}

	return &inst, nil
}
