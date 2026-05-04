package sqlite

import (
	"database/sql"
	"time"
)

type Account struct {
	ID                 string
	Username           string
	Email              string
	Status             string
	AccountLevel       int
	Group              string
	InstanceID         string
	InstanceName       string
	ProxyID            string
	ProxyURL           string
	FingerprintSeed    string
	FingerprintCountry string
	Headless           bool
	PendingRestart     bool
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

type AccountStore struct {
	db *DB
}

func NewAccountStore(db *DB) *AccountStore {
	return &AccountStore{db: db}
}

func (s *AccountStore) Save(account *Account) (*Account, error) {
	query := `INSERT INTO tiktok_accounts
		(id, username, email, status, account_level, account_group, instance_id, instance_name, proxy_id, proxy_url, fingerprint_seed, fingerprint_country, headless, pending_restart, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	account.CreatedAt = now
	account.UpdatedAt = now

	_, err := s.db.Exec(query,
		account.ID,
		nullIfEmpty(account.Username),
		account.Email,
		account.Status,
		account.AccountLevel,
		nullIfEmpty(account.Group),
		nullIfEmpty(account.InstanceID),
		nullIfEmpty(account.InstanceName),
		nullIfEmpty(account.ProxyID),
		nullIfEmpty(account.ProxyURL),
		nullIfEmpty(account.FingerprintSeed),
		nullIfEmpty(account.FingerprintCountry),
		boolToInt(account.Headless),
		boolToInt(account.PendingRestart),
		account.CreatedAt,
		account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return account, nil
}

func (s *AccountStore) Update(account *Account) error {
	query := `UPDATE tiktok_accounts SET
		username = ?, email = ?, status = ?, account_level = ?, account_group = ?, instance_id = ?, instance_name = ?,
		proxy_id = ?, proxy_url = ?, fingerprint_seed = ?, fingerprint_country = ?, headless = ?, pending_restart = ?, updated_at = ?
		WHERE id = ?`

	account.UpdatedAt = time.Now()

	_, err := s.db.Exec(query,
		nullIfEmpty(account.Username),
		account.Email,
		account.Status,
		account.AccountLevel,
		nullIfEmpty(account.Group),
		nullIfEmpty(account.InstanceID),
		nullIfEmpty(account.InstanceName),
		nullIfEmpty(account.ProxyID),
		nullIfEmpty(account.ProxyURL),
		nullIfEmpty(account.FingerprintSeed),
		nullIfEmpty(account.FingerprintCountry),
		boolToInt(account.Headless),
		boolToInt(account.PendingRestart),
		account.UpdatedAt,
		account.ID,
	)
	return err
}

func (s *AccountStore) Get(id string) (*Account, error) {
	query := `SELECT id, username, email, status, account_level, account_group, instance_id, instance_name, proxy_id, proxy_url, fingerprint_seed, fingerprint_country, headless, pending_restart, created_at, updated_at
		FROM tiktok_accounts WHERE id = ?`
	row := s.db.QueryRow(query, id)
	return scanAccount(row)
}

func (s *AccountStore) List() ([]*Account, error) {
	query := `SELECT id, username, email, status, account_level, account_group, instance_id, instance_name, proxy_id, proxy_url, fingerprint_seed, fingerprint_country, headless, pending_restart, created_at, updated_at
		FROM tiktok_accounts ORDER BY created_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var accounts []*Account
	for rows.Next() {
		account, err := scanAccountFromRows(rows)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, account)
	}
	return accounts, rows.Err()
}

func (s *AccountStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM tiktok_accounts WHERE id = ?", id)
	return err
}

func scanAccount(row *sql.Row) (*Account, error) {
	var account Account
	var username sql.NullString
	var group sql.NullString
	var instanceID sql.NullString
	var instanceName sql.NullString
	var proxyID sql.NullString
	var proxyURL sql.NullString
	var fpSeed sql.NullString
	var fpCountry sql.NullString
	var headless int
	var pending int

	err := row.Scan(
		&account.ID,
		&username,
		&account.Email,
		&account.Status,
		&account.AccountLevel,
		&group,
		&instanceID,
		&instanceName,
		&proxyID,
		&proxyURL,
		&fpSeed,
		&fpCountry,
		&headless,
		&pending,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	account.Username = username.String
	account.Group = group.String
	account.InstanceID = instanceID.String
	account.InstanceName = instanceName.String
	account.ProxyID = proxyID.String
	account.ProxyURL = proxyURL.String
	account.FingerprintSeed = fpSeed.String
	account.FingerprintCountry = fpCountry.String
	account.Headless = headless == 1
	account.PendingRestart = pending == 1

	return &account, nil
}

func scanAccountFromRows(rows *sql.Rows) (*Account, error) {
	var account Account
	var username sql.NullString
	var group sql.NullString
	var instanceID sql.NullString
	var instanceName sql.NullString
	var proxyID sql.NullString
	var proxyURL sql.NullString
	var fpSeed sql.NullString
	var fpCountry sql.NullString
	var headless int
	var pending int

	err := rows.Scan(
		&account.ID,
		&username,
		&account.Email,
		&account.Status,
		&account.AccountLevel,
		&group,
		&instanceID,
		&instanceName,
		&proxyID,
		&proxyURL,
		&fpSeed,
		&fpCountry,
		&headless,
		&pending,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	account.Username = username.String
	account.Group = group.String
	account.InstanceID = instanceID.String
	account.InstanceName = instanceName.String
	account.ProxyID = proxyID.String
	account.ProxyURL = proxyURL.String
	account.FingerprintSeed = fpSeed.String
	account.FingerprintCountry = fpCountry.String
	account.Headless = headless == 1
	account.PendingRestart = pending == 1

	return &account, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

func nullIfEmpty(value string) interface{} {
	if value == "" {
		return nil
	}
	return value
}
