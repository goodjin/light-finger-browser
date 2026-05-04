package sqlite

import (
	"database/sql"
	"time"

	"github.com/tmos/fingerbrower/proxy"
)

type ProxyStore struct {
	db *DB
}

func NewProxyStore(db *DB) *ProxyStore {
	return &ProxyStore{db: db}
}

func (s *ProxyStore) Save(p *proxy.Proxy) (*proxy.Proxy, error) {
	query := `INSERT INTO proxies
		(id, ip, port, country, city, type, username, password, status, bind_id, bound_at, last_check_at, success_rate, latency, provider, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	p.CreatedAt = now
	p.BoundAt = now
	p.LastCheckAt = now

	_, err := s.db.Exec(query,
		p.ID,
		p.IP,
		p.Port,
		p.Country,
		p.City,
		p.Type,
		nullIfEmpty(p.Username),
		nullIfEmpty(p.Password),
		p.Status,
		nullIfEmpty(p.BindID),
		p.BoundAt,
		p.LastCheckAt,
		p.SuccessRate,
		p.Latency,
		p.Provider,
		p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (s *ProxyStore) Update(p *proxy.Proxy) error {
	query := `UPDATE proxies SET
		ip = ?, port = ?, country = ?, city = ?, type = ?, username = ?, password = ?, status = ?, bind_id = ?, bound_at = ?, last_check_at = ?, success_rate = ?, latency = ?, provider = ?
		WHERE id = ?`

	p.LastCheckAt = time.Now()

	_, err := s.db.Exec(query,
		p.IP,
		p.Port,
		p.Country,
		p.City,
		p.Type,
		nullIfEmpty(p.Username),
		nullIfEmpty(p.Password),
		p.Status,
		nullIfEmpty(p.BindID),
		p.BoundAt,
		p.LastCheckAt,
		p.SuccessRate,
		p.Latency,
		p.Provider,
		p.ID,
	)
	return err
}

func (s *ProxyStore) Get(id string) (*proxy.Proxy, error) {
	query := `SELECT id, ip, port, country, city, type, username, password, status, bind_id, bound_at, last_check_at, success_rate, latency, provider, created_at
		FROM proxies WHERE id = ?`
	row := s.db.QueryRow(query, id)

	var p proxy.Proxy
	var username sql.NullString
	var password sql.NullString
	var bindID sql.NullString

	err := row.Scan(
		&p.ID,
		&p.IP,
		&p.Port,
		&p.Country,
		&p.City,
		&p.Type,
		&username,
		&password,
		&p.Status,
		&bindID,
		&p.BoundAt,
		&p.LastCheckAt,
		&p.SuccessRate,
		&p.Latency,
		&p.Provider,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Username = username.String
	p.Password = password.String
	p.BindID = bindID.String
	return &p, nil
}

func (s *ProxyStore) List() ([]*proxy.Proxy, error) {
	query := `SELECT id, ip, port, country, city, type, username, password, status, bind_id, bound_at, last_check_at, success_rate, latency, provider, created_at
		FROM proxies ORDER BY created_at DESC`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []*proxy.Proxy
	for rows.Next() {
		p, err := scanProxyFromRows(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, p)
	}
	return proxies, rows.Err()
}

func (s *ProxyStore) Delete(id string) error {
	_, err := s.db.Exec("DELETE FROM proxies WHERE id = ?", id)
	return err
}

func scanProxyFromRows(rows *sql.Rows) (*proxy.Proxy, error) {
	var p proxy.Proxy
	var username sql.NullString
	var password sql.NullString
	var bindID sql.NullString

	err := rows.Scan(
		&p.ID,
		&p.IP,
		&p.Port,
		&p.Country,
		&p.City,
		&p.Type,
		&username,
		&password,
		&p.Status,
		&bindID,
		&p.BoundAt,
		&p.LastCheckAt,
		&p.SuccessRate,
		&p.Latency,
		&p.Provider,
		&p.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	p.Username = username.String
	p.Password = password.String
	p.BindID = bindID.String
	return &p, nil
}
