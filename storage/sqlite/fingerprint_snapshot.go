package sqlite

import (
	"database/sql"
	"time"
)

type FingerprintSnapshot struct {
	ID         string
	AccountID  string
	InstanceID string
	ProxyID    string
	Snapshot   string
	CreatedAt  time.Time
}

type FingerprintSnapshotStore struct {
	db *DB
}

func NewFingerprintSnapshotStore(db *DB) *FingerprintSnapshotStore {
	return &FingerprintSnapshotStore{db: db}
}

func (s *FingerprintSnapshotStore) Save(snapshot *FingerprintSnapshot) (*FingerprintSnapshot, error) {
	query := `INSERT INTO fingerprint_snapshots
		(id, account_id, instance_id, proxy_id, snapshot_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`

	snapshot.CreatedAt = time.Now()

	_, err := s.db.Exec(query,
		snapshot.ID,
		nullIfEmpty(snapshot.AccountID),
		nullIfEmpty(snapshot.InstanceID),
		nullIfEmpty(snapshot.ProxyID),
		snapshot.Snapshot,
		snapshot.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (s *FingerprintSnapshotStore) GetLatestByAccount(accountID string) (*FingerprintSnapshot, error) {
	query := `SELECT id, account_id, instance_id, proxy_id, snapshot_json, created_at
		FROM fingerprint_snapshots WHERE account_id = ? ORDER BY created_at DESC LIMIT 1`
	row := s.db.QueryRow(query, accountID)
	return scanSnapshot(row)
}

func scanSnapshot(row *sql.Row) (*FingerprintSnapshot, error) {
	var snapshot FingerprintSnapshot
	var accountID sql.NullString
	var instanceID sql.NullString
	var proxyID sql.NullString

	err := row.Scan(
		&snapshot.ID,
		&accountID,
		&instanceID,
		&proxyID,
		&snapshot.Snapshot,
		&snapshot.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	snapshot.AccountID = accountID.String
	snapshot.InstanceID = instanceID.String
	snapshot.ProxyID = proxyID.String
	return &snapshot, nil
}
