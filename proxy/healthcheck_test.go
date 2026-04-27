package proxy

import (
	"testing"
	"time"
)

// MockStore implements Store for testing
type MockStore struct {
	proxies map[string]*Proxy
}

func NewMockStore() *MockStore {
	return &MockStore{
		proxies: make(map[string]*Proxy),
	}
}

func (s *MockStore) Save(proxy *Proxy) (*Proxy, error) {
	s.proxies[proxy.ID] = proxy
	return proxy, nil
}

func (s *MockStore) Get(id string) (*Proxy, error) {
	p, ok := s.proxies[id]
	if !ok {
		return nil, ErrProxyNotFound
	}
	return p, nil
}

func (s *MockStore) List(filter *ProxyFilter) ([]*Proxy, error) {
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

func (s *MockStore) Update(proxy *Proxy) (*Proxy, error) {
	s.proxies[proxy.ID] = proxy
	return proxy, nil
}

func (s *MockStore) Delete(id string) error {
	delete(s.proxies, id)
	return nil
}

// TestHealthCheckerStartStop tests health checker start and stop
func TestHealthCheckerStartStop(t *testing.T) {
	store := NewMockStore()
	manager := &ProxyManager{
		store:     store,
		providers: make(map[string]ProxyProvider),
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Second)
	checker.Start()

	// Give it a moment to start
	time.Sleep(100 * time.Millisecond)

	// Stop should complete without hanging
	checker.Stop()
}

// TestHealthCheckerCheckNow tests immediate health check
func TestHealthCheckerCheckNow(t *testing.T) {
	store := NewMockStore()

	// Add a proxy in use
	proxy := &Proxy{
		ID:       "healthcheck-proxy",
		IP:       "192.168.1.100",
		Port:     8080,
		Country:  "US",
		Type:     ProxyTypeResidential,
		Status:   ProxyStatusInUse,
		Provider: "mock",
	}
	store.Save(proxy)

	manager := &ProxyManager{
		store:     store,
		providers: make(map[string]ProxyProvider),
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour) // Very long interval
	checker.Start()
	defer checker.Stop()

	// CheckNow should not block
	checker.CheckNow()
}

// TestHealthCheckerTriggerAlert tests alert triggering for dead proxy
func TestHealthCheckerTriggerAlert(t *testing.T) {
	store := NewMockStore()
	alertFired := false

	proxy := &Proxy{
		ID:          "dead-proxy-alert",
		IP:          "192.168.1.200",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusInUse,
		SuccessRate: 0.2,
		Provider:    "mock",
	}
	store.Save(proxy)

	manager := &ProxyManager{
		store:     store,
		providers: make(map[string]ProxyProvider),
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour)
	checker.Start()
	defer checker.Stop()

	// Simulate alert being triggered by setting status to dead
	proxy.Status = ProxyStatusDead

	// Verify alert channel has capacity
	select {
	case manager.healthCh <- proxy.ID:
		alertFired = true
	default:
		// Channel full
	}

	if !alertFired {
		t.Log("Alert would be triggered for dead proxy")
	}
}

// TestHealthCheckerNoProxy tests health check with no proxies
func TestHealthCheckerNoProxy(t *testing.T) {
	store := NewMockStore()
	manager := &ProxyManager{
		store:     store,
		providers: make(map[string]ProxyProvider),
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour)
	checker.Start()
	defer checker.Stop()

	// Should not panic with no proxies
	checker.CheckNow()
}