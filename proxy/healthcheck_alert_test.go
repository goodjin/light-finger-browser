package proxy

import (
	"context"
	"testing"
	"time"
)

// MockStoreWithAlert implements Store and tracks alerts
type MockStoreWithAlert struct {
	proxies       map[string]*Proxy
	alertReceived string
}

func NewMockStoreWithAlert() *MockStoreWithAlert {
	return &MockStoreWithAlert{
		proxies: make(map[string]*Proxy),
	}
}

func (s *MockStoreWithAlert) Save(proxy *Proxy) (*Proxy, error) {
	proxy.CreatedAt = time.Now()
	proxy.LastCheckAt = time.Now()
	s.proxies[proxy.ID] = proxy
	return proxy, nil
}

func (s *MockStoreWithAlert) Get(id string) (*Proxy, error) {
	p, ok := s.proxies[id]
	if !ok {
		return nil, ErrProxyNotFound
	}
	return p, nil
}

func (s *MockStoreWithAlert) List(filter *ProxyFilter) ([]*Proxy, error) {
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

func (s *MockStoreWithAlert) Update(proxy *Proxy) (*Proxy, error) {
	s.proxies[proxy.ID] = proxy
	return proxy, nil
}

func (s *MockStoreWithAlert) Delete(id string) error {
	delete(s.proxies, id)
	return nil
}

// TestTriggerAlert tests that alert is triggered for dead proxy
func TestTriggerAlert(t *testing.T) {
	store := NewMockStoreWithAlert()
	alertReceived := ""

	proxy := &Proxy{
		ID:          "alert-test-proxy",
		IP:          "192.168.1.250",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusInUse,
		SuccessRate: 0.25, // Low enough to be marked dead
		Provider:    "mock",
	}
	store.Save(proxy)

	// Create a mock provider that returns failure
	mockProvider := &MockProviderForAlert{
		CheckProxyFunc: func(ctx context.Context, p *Proxy) (bool, int, error) {
			return false, 0, nil // Fail but no error
		},
	}

	manager := &ProxyManager{
		store:     store,
		providers: map[string]ProxyProvider{"mock": mockProvider},
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour)
	checker.Start()
	defer checker.Stop()

	// Manually call checkProxy which should trigger alert
	checker.checkProxy("alert-test-proxy")

	// Give time for alert to be processed
	time.Sleep(100 * time.Millisecond)

	// Check if alert was received
	select {
	case alertID := <-manager.healthCh:
		alertReceived = alertID
	default:
		// No alert received yet
	}

	// Verify proxy is marked as dead
	updatedProxy, _ := store.Get("alert-test-proxy")
	if updatedProxy.Status != ProxyStatusDead {
		t.Errorf("Expected proxy status 'dead', got '%s'", updatedProxy.Status)
	}

	t.Logf("Alert received for proxy: %s", alertReceived)
}

// MockProviderForAlert is a mock provider for alert testing
type MockProviderForAlert struct {
	GetProxyFunc     func(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error)
	ReleaseProxyFunc func(ctx context.Context, id string) error
	CheckProxyFunc   func(ctx context.Context, p *Proxy) (bool, int, error)
}

func (m *MockProviderForAlert) GetProxy(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error) {
	if m.GetProxyFunc != nil {
		return m.GetProxyFunc(ctx, country, proxyType)
	}
	return &Proxy{
		ID:       "mock-proxy",
		IP:       "192.168.1.1",
		Port:     8080,
		Country:  country,
		Type:     proxyType,
		Status:   ProxyStatusAvailable,
		Provider: "mock",
	}, nil
}

func (m *MockProviderForAlert) ReleaseProxy(ctx context.Context, id string) error {
	if m.ReleaseProxyFunc != nil {
		return m.ReleaseProxyFunc(ctx, id)
	}
	return nil
}

func (m *MockProviderForAlert) CheckProxy(ctx context.Context, p *Proxy) (bool, int, error) {
	if m.CheckProxyFunc != nil {
		return m.CheckProxyFunc(ctx, p)
	}
	return true, 100, nil
}

// TestCheckProxyWithHealthCheckError tests checkProxy when HealthCheck returns error
func TestCheckProxyWithHealthCheckError(t *testing.T) {
	store := NewMockStoreWithAlert()

	proxy := &Proxy{
		ID:          "hc-error-proxy",
		IP:          "192.168.1.251",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusInUse,
		SuccessRate: 0.5,
		Provider:    "unknown", // Unknown provider will cause error
	}
	store.Save(proxy)

	manager := &ProxyManager{
		store:     store,
		providers: make(map[string]ProxyProvider), // No providers
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour)
	checker.Start()
	defer checker.Stop()

	// Should not panic when HealthCheck returns error
	checker.checkProxy("hc-error-proxy")

	t.Log("checkProxy handled HealthCheck error gracefully")
}

// TestCheckProxyProxyNotFound tests checkProxy when proxy is not found
func TestCheckProxyProxyNotFound(t *testing.T) {
	store := NewMockStoreWithAlert()

	manager := &ProxyManager{
		store:     store,
		providers: make(map[string]ProxyProvider),
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour)
	checker.Start()
	defer checker.Stop()

	// Should not panic when proxy is not found
	checker.checkProxy("non-existent-proxy")

	t.Log("checkProxy handled not found gracefully")
}