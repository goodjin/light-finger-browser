package proxy

import (
	"errors"
	"sync"
	"time"
)

// Store defines the interface for proxy data persistence
type Store interface {
	Save(proxy *Proxy) (*Proxy, error)
	Get(id string) (*Proxy, error)
	List(filter *ProxyFilter) ([]*Proxy, error)
	Update(proxy *Proxy) (*Proxy, error)
	Delete(id string) error
}

// PostgresStore implements Store using PostgreSQL
type PostgresStore struct {
	// In production, this would be a *sql.DB connection
	mu    sync.RWMutex
	proxies map[string]*Proxy
}

// NewPostgresStore creates a new PostgresStore
func NewPostgresStore() *PostgresStore {
	return &PostgresStore{
		proxies: make(map[string]*Proxy),
	}
}

// Save saves a new proxy to the store
func (s *PostgresStore) Save(proxy *Proxy) (*Proxy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if proxy.ID == "" {
		return nil, errors.New("proxy ID is required")
	}

	proxy.CreatedAt = time.Now()
	proxy.LastCheckAt = time.Now()

	s.proxies[proxy.ID] = proxy
	return proxy, nil
}

// Get retrieves a proxy by ID
func (s *PostgresStore) Get(id string) (*Proxy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	proxy, ok := s.proxies[id]
	if !ok {
		return nil, ErrProxyNotFound
	}

	return proxy, nil
}

// List retrieves proxies based on filter criteria
func (s *PostgresStore) List(filter *ProxyFilter) ([]*Proxy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Proxy

	for _, p := range s.proxies {
		if filter != nil {
			if filter.ID != nil && p.ID != *filter.ID {
				continue
			}
			if filter.Country != nil && p.Country != *filter.Country {
				continue
			}
			if filter.Type != nil && p.Type != *filter.Type {
				continue
			}
			if filter.Status != nil && p.Status != *filter.Status {
				continue
			}
			if filter.BindID != nil && p.BindID != *filter.BindID {
				continue
			}
			if filter.Provider != nil && p.Provider != *filter.Provider {
				continue
			}
		}
		result = append(result, p)
	}

	return result, nil
}

// Update updates an existing proxy
func (s *PostgresStore) Update(proxy *Proxy) (*Proxy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.proxies[proxy.ID]; !ok {
		return nil, ErrProxyNotFound
	}

	s.proxies[proxy.ID] = proxy
	return proxy, nil
}

// Delete removes a proxy from the store
func (s *PostgresStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.proxies[id]; !ok {
		return ErrProxyNotFound
	}

	delete(s.proxies, id)
	return nil
}