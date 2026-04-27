package proxy

import (
	"context"
	"errors"
	"testing"
)

// MockProvider implements ProxyProvider for testing
type MockProvider struct {
	GetProxyFunc     func(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error)
	ReleaseProxyFunc func(ctx context.Context, id string) error
	CheckProxyFunc   func(ctx context.Context, p *Proxy) (bool, int, error)
}

func (m *MockProvider) GetProxy(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error) {
	if m.GetProxyFunc != nil {
		return m.GetProxyFunc(ctx, country, proxyType)
	}
	return &Proxy{
		ID:       "mock-proxy-1",
		IP:       "192.168.1.1",
		Port:     8080,
		Country:  country,
		Type:     proxyType,
		Status:   ProxyStatusAvailable,
		Provider: "mock",
	}, nil
}

func (m *MockProvider) ReleaseProxy(ctx context.Context, id string) error {
	if m.ReleaseProxyFunc != nil {
		return m.ReleaseProxyFunc(ctx, id)
	}
	return nil
}

func (m *MockProvider) CheckProxy(ctx context.Context, p *Proxy) (bool, int, error) {
	if m.CheckProxyFunc != nil {
		return m.CheckProxyFunc(ctx, p)
	}
	return true, 100, nil
}

// TestAcquireSuccess tests successful proxy acquisition
func TestAcquireSuccess(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	// Save a proxy directly to the store first
	proxy := &Proxy{
		ID:          "test-proxy-1",
		IP:          "192.168.1.1",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusAvailable,
		SuccessRate: 0.9,
		Latency:     100,
	}
	_, err := store.Save(proxy)
	if err != nil {
		t.Fatalf("Failed to save proxy: %v", err)
	}

	// Acquire should return the available proxy
	acquired, err := manager.Acquire(ctx, "US", ProxyTypeResidential)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if acquired.ID != "test-proxy-1" {
		t.Errorf("Expected proxy ID 'test-proxy-1', got '%s'", acquired.ID)
	}

	if acquired.Status != ProxyStatusInUse {
		t.Errorf("Expected status 'in_use', got '%s'", acquired.Status)
	}
}

// TestAcquirePoolExhausted tests error when pool is exhausted
func TestAcquirePoolExhausted(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{
			GetProxyFunc: func(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error) {
				return nil, errors.New("provider exhausted")
			},
		},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	_, err := manager.Acquire(ctx, "US", ProxyTypeResidential)
	if err != ErrNoAvailableProxy {
		t.Errorf("Expected ErrNoAvailableProxy, got: %v", err)
	}
}

// TestReleaseSuccess tests successful proxy release
func TestReleaseSuccess(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ctx := context.Background()

	// Save a proxy in use
	proxy := &Proxy{
		ID:       "test-proxy-release",
		IP:       "192.168.1.2",
		Port:     8080,
		Country:  "US",
		Type:     ProxyTypeResidential,
		Status:   ProxyStatusInUse,
		BindID:   "instance-1",
	}
	store.Save(proxy)

	// Release should make it available again
	err := manager.Release(ctx, "test-proxy-release")
	if err != nil {
		t.Fatalf("Release failed: %v", err)
	}

	released, _ := store.Get("test-proxy-release")
	if released.Status != ProxyStatusAvailable {
		t.Errorf("Expected status 'available', got '%s'", released.Status)
	}

	if released.BindID != "" {
		t.Errorf("Expected empty BindID, got '%s'", released.BindID)
	}
}

// TestBindUnbind tests proxy binding and unbinding
func TestBindUnbind(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ctx := context.Background()

	// Save a proxy
	proxy := &Proxy{
		ID:      "test-proxy-bind",
		IP:      "192.168.1.3",
		Port:    8080,
		Country: "US",
		Type:    ProxyTypeResidential,
		Status:  ProxyStatusAvailable,
	}
	store.Save(proxy)

	// Bind to instance
	err := manager.Bind(ctx, "test-proxy-bind", "instance-1")
	if err != nil {
		t.Fatalf("Bind failed: %v", err)
	}

	bound, _ := store.Get("test-proxy-bind")
	if bound.BindID != "instance-1" {
		t.Errorf("Expected BindID 'instance-1', got '%s'", bound.BindID)
	}

	// Unbind
	err = manager.Unbind(ctx, "test-proxy-bind")
	if err != nil {
		t.Fatalf("Unbind failed: %v", err)
	}

	unbound, _ := store.Get("test-proxy-bind")
	if unbound.BindID != "" {
		t.Errorf("Expected empty BindID after Unbind, got '%s'", unbound.BindID)
	}

	if unbound.Status != ProxyStatusAvailable {
		t.Errorf("Expected status 'available' after Unbind, got '%s'", unbound.Status)
	}
}

// TestBindAlreadyBound tests binding to an already bound proxy
func TestBindAlreadyBound(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ctx := context.Background()

	// Save a proxy already bound to another instance
	proxy := &Proxy{
		ID:      "test-proxy-bound",
		IP:      "192.168.1.4",
		Port:    8080,
		Country: "US",
		Type:    ProxyTypeResidential,
		Status:  ProxyStatusInUse,
		BindID:  "instance-1",
	}
	store.Save(proxy)

	// Try to bind to a different instance
	err := manager.Bind(ctx, "test-proxy-bound", "instance-2")
	if err != ErrProxyAlreadyBound {
		t.Errorf("Expected ErrProxyAlreadyBound, got: %v", err)
	}
}

// TestHealthCheckDeadProxy tests health check marking proxy as dead
func TestHealthCheckDeadProxy(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{
			CheckProxyFunc: func(ctx context.Context, p *Proxy) (bool, int, error) {
				return false, 0, errors.New("connection failed")
			},
		},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	// Save a proxy with low success rate
	proxy := &Proxy{
		ID:          "test-proxy-health",
		IP:          "192.168.1.5",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusInUse,
		SuccessRate: 0.29, // Below threshold
		Provider:    "mock",
	}
	store.Save(proxy)

	// Health check should mark it as dead
	err := manager.HealthCheck(ctx, "test-proxy-health")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	checked, _ := store.Get("test-proxy-health")
	if checked.Status != ProxyStatusDead {
		t.Errorf("Expected status 'dead', got '%s'", checked.Status)
	}
}

// TestHealthCheckRecovery tests health check recovering proxy
func TestHealthCheckRecovery(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{
			CheckProxyFunc: func(ctx context.Context, p *Proxy) (bool, int, error) {
				return true, 50, nil
			},
		},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	// Save a proxy
	proxy := &Proxy{
		ID:          "test-proxy-recovery",
		IP:          "192.168.1.6",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusChecking,
		SuccessRate: 0.8,
		Provider:    "mock",
	}
	store.Save(proxy)

	// Health check should recover it
	err := manager.HealthCheck(ctx, "test-proxy-recovery")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	checked, _ := store.Get("test-proxy-recovery")
	if checked.Status != ProxyStatusAvailable {
		t.Errorf("Expected status 'available', got '%s'", checked.Status)
	}

	if checked.Latency != 50 {
		t.Errorf("Expected latency 50, got %d", checked.Latency)
	}
}

// TestListWithFilter tests listing proxies with filters
func TestListWithFilter(t *testing.T) {
	store := NewPostgresStore()

	// Save multiple proxies
	proxies := []*Proxy{
		{ID: "proxy-1", Country: "US", Type: ProxyTypeResidential, Status: ProxyStatusAvailable},
		{ID: "proxy-2", Country: "US", Type: ProxyTypeDatacenter, Status: ProxyStatusInUse},
		{ID: "proxy-3", Country: "UK", Type: ProxyTypeResidential, Status: ProxyStatusAvailable},
	}
	for _, p := range proxies {
		store.Save(p)
	}

	// Filter by country
	usCountry := "US"
	usProxies, err := store.List(&ProxyFilter{Country: &usCountry})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(usProxies) != 2 {
		t.Errorf("Expected 2 US proxies, got %d", len(usProxies))
	}

	// Filter by status
	availableProxies, err := store.List(&ProxyFilter{Status: ProxyStatusPtr(ProxyStatusAvailable)})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(availableProxies) != 2 {
		t.Errorf("Expected 2 available proxies, got %d", len(availableProxies))
	}
}

// TestStoreNotFound tests store operations on non-existent proxy
func TestStoreNotFound(t *testing.T) {
	store := NewPostgresStore()

	_, err := store.Get("non-existent")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}

	err = store.Delete("non-existent")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}
}

// TestConcurrentAccess tests concurrent access to store
func TestConcurrentAccess(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ctx := context.Background()

	proxy := &Proxy{
		ID:      "concurrent-proxy",
		IP:      "192.168.1.100",
		Port:    8080,
		Country: "US",
		Type:    ProxyTypeResidential,
		Status:  ProxyStatusAvailable,
	}
	store.Save(proxy)

	done := make(chan bool)

	// Concurrent acquires
	for i := 0; i < 5; i++ {
		go func() {
			manager.Acquire(ctx, "US", ProxyTypeResidential)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}
}