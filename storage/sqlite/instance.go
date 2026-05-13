package sqlite

import (
	"database/sql"
	"encoding/json"
	"log"
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
		(id, name, status, fingerprint_json, proxy_id, proxy_url, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, headless, started_at, last_active_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	inst.CreatedAt = now
	inst.LastActiveAt = now

	_, err := s.db.Exec(query,
		inst.ID, inst.Name, inst.Status, fpJSON, inst.ProxyID, inst.ProxyURL, inst.AccountID,
		inst.CDPEndpoint, inst.PID, inst.Port, inst.UserDataDir, inst.Group, boolToInt(inst.Headless),
		inst.StartedAt, inst.LastActiveAt, inst.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return inst, nil
}

func (s *InstanceStore) Get(id string) (*instance.BrowserInstance, error) {
	query := `SELECT id, name, status, fingerprint_json, proxy_id, proxy_url, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, headless, started_at, last_active_at, created_at
		FROM browser_instances WHERE id = ?`

	row := s.db.QueryRow(query, id)
	return s.scanInstance(row)
}

func (s *InstanceStore) List(filter *instance.InstanceFilter) ([]*instance.BrowserInstance, error) {
	log.Println("[DB List] START - about to query database")
	log.Printf("[DB List] DB stats: %+v", s.db.Stats())
	query := `SELECT id, name, status, fingerprint_json, proxy_id, proxy_url, account_id, cdp_endpoint, pid, port, user_data_dir, group_name, headless, started_at, last_active_at, created_at
		FROM browser_instances WHERE 1=1`
	args := []interface{}{}

	if filter != nil {
		if filter.Status != nil {
			query += " AND status = ?"
			args = append(args, *filter.Status)
			log.Printf("[DB List] Filter by status: %s", *filter.Status)
		}
		if filter.Group != "" {
			query += " AND group_name = ?"
			args = append(args, filter.Group)
			log.Printf("[DB List] Filter by group: %s", filter.Group)
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

	log.Printf("[DB List] Executing query: %s", query)
	log.Printf("[DB List] Args: %v", args)
	log.Println("[DB List] About to call s.db.Query...")

	rows, err := s.db.Query(query, args...)
	if err != nil {
		log.Printf("[DB List] Query error: %v", err)
		return nil, err
	}
	log.Println("[DB List] Query returned successfully")
	defer rows.Close()
	log.Println("[DB List] Query executed, scanning rows...")

	var instances []*instance.BrowserInstance
	for rows.Next() {
		inst, err := s.scanInstanceFromRows(rows)
		if err != nil {
			log.Printf("[DB List] Scan error: %v", err)
			return nil, err
		}
		instances = append(instances, inst)
	}
	log.Printf("[DB List] Done, found %d instances", len(instances))
	return instances, nil
}

func (s *InstanceStore) Update(inst *instance.BrowserInstance) error {
	fpJSON, _ := json.Marshal(inst.Fingerprint)
	inst.LastActiveAt = time.Now()

	query := `UPDATE browser_instances SET
		name = ?, status = ?, fingerprint_json = ?, proxy_id = ?, proxy_url = ?, account_id = ?,
		cdp_endpoint = ?, pid = ?, port = ?, user_data_dir = ?, group_name = ?, headless = ?,
		started_at = ?, last_active_at = ?
		WHERE id = ?`

	_, err := s.db.Exec(query,
		inst.Name, inst.Status, fpJSON, inst.ProxyID, inst.ProxyURL, inst.AccountID,
		inst.CDPEndpoint, inst.PID, inst.Port, inst.UserDataDir, inst.Group, boolToInt(inst.Headless),
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
	var name sql.NullString
	var proxyID sql.NullString
	var proxyURL sql.NullString
	var accountID sql.NullString
	var cdpEndpoint sql.NullString
	var userDataDir sql.NullString
	var group sql.NullString
	var headless int
	var fpJSON sql.NullString

	err := row.Scan(
		&inst.ID, &name, &inst.Status, &fpJSON, &proxyID, &proxyURL, &accountID,
		&cdpEndpoint, &inst.PID, &inst.Port, &userDataDir, &group, &headless,
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

	if name.Valid {
		inst.Name = name.String
	}
	if proxyID.Valid {
		inst.ProxyID = proxyID.String
	}
	if proxyURL.Valid {
		inst.ProxyURL = proxyURL.String
	}
	if accountID.Valid {
		inst.AccountID = accountID.String
	}
	if cdpEndpoint.Valid {
		inst.CDPEndpoint = cdpEndpoint.String
	}
	if userDataDir.Valid {
		inst.UserDataDir = userDataDir.String
	}
	if group.Valid {
		inst.Group = group.String
	}
	inst.Headless = headless == 1

	return &inst, nil
}

func (s *InstanceStore) scanInstanceFromRows(rows *sql.Rows) (*instance.BrowserInstance, error) {
	var inst instance.BrowserInstance
	var name sql.NullString
	var proxyID sql.NullString
	var proxyURL sql.NullString
	var accountID sql.NullString
	var cdpEndpoint sql.NullString
	var userDataDir sql.NullString
	var group sql.NullString
	var headless int
	var fpJSON sql.NullString

	err := rows.Scan(
		&inst.ID, &name, &inst.Status, &fpJSON, &proxyID, &proxyURL, &accountID,
		&cdpEndpoint, &inst.PID, &inst.Port, &userDataDir, &group, &headless,
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

	if name.Valid {
		inst.Name = name.String
	}
	if proxyID.Valid {
		inst.ProxyID = proxyID.String
	}
	if proxyURL.Valid {
		inst.ProxyURL = proxyURL.String
	}
	if accountID.Valid {
		inst.AccountID = accountID.String
	}
	if cdpEndpoint.Valid {
		inst.CDPEndpoint = cdpEndpoint.String
	}
	if userDataDir.Valid {
		inst.UserDataDir = userDataDir.String
	}
	if group.Valid {
		inst.Group = group.String
	}
	inst.Headless = headless == 1

	return &inst, nil
}
