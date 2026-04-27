package proxy

import (
	"context"
	"errors"
	"testing"
)

// TestAcquireFromProvider tests acquiring when pool has no available proxy
func TestAcquireFromProvider(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{
			GetProxyFunc: func(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error) {
				return &Proxy{
					ID:       "new-proxy-from-provider",
					IP:       "10.0.0.1",
					Port:     8080,
					Country:  country,
					Type:     proxyType,
					Status:   ProxyStatusAvailable,
					Provider: "mock",
				}, nil
			},
		},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	// Pool is empty, should get from provider
	acquired, err := manager.Acquire(ctx, "US", ProxyTypeResidential)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if acquired.ID != "new-proxy-from-provider" {
		t.Errorf("Expected proxy ID 'new-proxy-from-provider', got '%s'", acquired.ID)
	}

	if acquired.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got '%s'", acquired.Provider)
	}
}

// TestAcquireSelectsBestProxy tests that Acquire selects optimal proxy
func TestAcquireSelectsBestProxy(t *testing.T) {
	store := NewPostgresStore()

	// Save multiple proxies with different success rates and latency
	proxies := []*Proxy{
		{ID: "slow-bad", IP: "192.168.1.1", Country: "US", Type: ProxyTypeResidential, Status: ProxyStatusAvailable, SuccessRate: 0.5, Latency: 500},
		{ID: "fast-good", IP: "192.168.1.2", Country: "US", Type: ProxyTypeResidential, Status: ProxyStatusAvailable, SuccessRate: 0.9, Latency: 50},
		{ID: "fast-medium", IP: "192.168.1.3", Country: "US", Type: ProxyTypeResidential, Status: ProxyStatusAvailable, SuccessRate: 0.7, Latency: 80},
	}
	for _, p := range proxies {
		store.Save(p)
	}

	providers := map[string]ProxyProvider{
		"mock": &MockProvider{},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	acquired, err := manager.Acquire(ctx, "US", ProxyTypeResidential)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	// Should select the proxy with highest success rate (fast-good)
	if acquired.ID != "fast-good" {
		t.Errorf("Expected proxy ID 'fast-good', got '%s'", acquired.ID)
	}
}

// TestAcquireCountryFilter tests acquiring with country filter
func TestAcquireCountryFilter(t *testing.T) {
	store := NewPostgresStore()

	// Save proxies from different countries
	proxies := []*Proxy{
		{ID: "us-proxy", IP: "192.168.1.1", Country: "US", Type: ProxyTypeResidential, Status: ProxyStatusAvailable},
		{ID: "uk-proxy", IP: "192.168.1.2", Country: "UK", Type: ProxyTypeResidential, Status: ProxyStatusAvailable},
	}
	for _, p := range proxies {
		store.Save(p)
	}

	providers := map[string]ProxyProvider{
		"mock": &MockProvider{},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	// Request UK proxy
	acquired, err := manager.Acquire(ctx, "UK", ProxyTypeResidential)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if acquired.ID != "uk-proxy" {
		t.Errorf("Expected proxy ID 'uk-proxy', got '%s'", acquired.ID)
	}
}

// TestAcquireTypeFilter tests acquiring with type filter
func TestAcquireTypeFilter(t *testing.T) {
	store := NewPostgresStore()

	proxies := []*Proxy{
		{ID: "res-proxy", IP: "192.168.1.1", Country: "US", Type: ProxyTypeResidential, Status: ProxyStatusAvailable},
		{ID: "dc-proxy", IP: "192.168.1.2", Country: "US", Type: ProxyTypeDatacenter, Status: ProxyStatusAvailable},
	}
	for _, p := range proxies {
		store.Save(p)
	}

	providers := map[string]ProxyProvider{
		"mock": &MockProvider{},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	// Request datacenter proxy
	acquired, err := manager.Acquire(ctx, "US", ProxyTypeDatacenter)
	if err != nil {
		t.Fatalf("Acquire failed: %v", err)
	}

	if acquired.ID != "dc-proxy" {
		t.Errorf("Expected proxy ID 'dc-proxy', got '%s'", acquired.ID)
	}
}

// TestBindSameInstance tests binding to same instance succeeds
func TestBindSameInstance(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ctx := context.Background()

	proxy := &Proxy{
		ID:      "bind-same",
		IP:      "192.168.1.50",
		Port:    8080,
		Country: "US",
		Type:    ProxyTypeResidential,
		Status:  ProxyStatusInUse,
		BindID:  "instance-1",
	}
	store.Save(proxy)

	// Binding to same instance should succeed
	err := manager.Bind(ctx, "bind-same", "instance-1")
	if err != nil {
		t.Errorf("Bind to same instance failed: %v", err)
	}
}

// TestUnbindNotBound tests unbinding a proxy that is not bound
func TestUnbindNotBound(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ctx := context.Background()

	proxy := &Proxy{
		ID:      "unbind-not-bound",
		IP:      "192.168.1.51",
		Port:    8080,
		Country: "US",
		Type:    ProxyTypeResidential,
		Status:  ProxyStatusAvailable,
		BindID:  "",
	}
	store.Save(proxy)

	// Unbinding should still work
	err := manager.Unbind(ctx, "unbind-not-bound")
	if err != nil {
		t.Fatalf("Unbind failed: %v", err)
	}
}

// TestHealthCheckSuccessRateIncrease tests success rate increases on successful check
func TestHealthCheckSuccessRateIncrease(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{
			CheckProxyFunc: func(ctx context.Context, p *Proxy) (bool, int, error) {
				return true, 30, nil
			},
		},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	proxy := &Proxy{
		ID:          "hc-increase",
		IP:          "192.168.1.60",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusChecking,
		SuccessRate: 0.8,
		Provider:    "mock",
	}
	store.Save(proxy)

	err := manager.HealthCheck(ctx, "hc-increase")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	checked, _ := store.Get("hc-increase")
	// Success rate should increase: (0.8*9 + 1) / 10 = 0.82
	expectedRate := (0.8*9 + 1) / 10
	if checked.SuccessRate != expectedRate {
		t.Errorf("Expected success rate %f, got %f", expectedRate, checked.SuccessRate)
	}
}

// TestHealthCheckSuccessRateDecrease tests success rate decreases on failed check
func TestHealthCheckSuccessRateDecrease(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{
			CheckProxyFunc: func(ctx context.Context, p *Proxy) (bool, int, error) {
				return false, 0, errors.New("failed")
			},
		},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	proxy := &Proxy{
		ID:          "hc-decrease",
		IP:          "192.168.1.61",
		Port:        8080,
		Country:     "US",
		Type:        ProxyTypeResidential,
		Status:      ProxyStatusChecking,
		SuccessRate: 0.5,
		Provider:    "mock",
	}
	store.Save(proxy)

	err := manager.HealthCheck(ctx, "hc-decrease")
	if err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}

	checked, _ := store.Get("hc-decrease")
	// Success rate should decrease: (0.5 * 9) / 10 = 0.45
	expectedRate := (0.5 * 9) / 10
	if checked.SuccessRate != expectedRate {
		t.Errorf("Expected success rate %f, got %f", expectedRate, checked.SuccessRate)
	}
}

// TestHealthCheckUnknownProvider tests health check with unknown provider
func TestHealthCheckUnknownProvider(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{} // No providers
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	proxy := &Proxy{
		ID:       "unknown-provider",
		IP:       "192.168.1.62",
		Port:     8080,
		Country:  "US",
		Type:     ProxyTypeResidential,
		Status:   ProxyStatusChecking,
		Provider: "unknown",
	}
	store.Save(proxy)

	err := manager.HealthCheck(ctx, "unknown-provider")
	if err == nil {
		t.Error("Expected error for unknown provider, got nil")
	}
}

// TestGetHealthChannel tests health channel creation
func TestGetHealthChannel(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ch := manager.GetHealthChannel()
	if ch == nil {
		t.Error("GetHealthChannel returned nil")
	}
}

// TestRefreshPool tests pool refresh
func TestRefreshPool(t *testing.T) {
	store := NewPostgresStore()
	providers := map[string]ProxyProvider{
		"mock": &MockProvider{
			GetProxyFunc: func(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error) {
				return &Proxy{
					ID:       "refreshed-proxy",
					IP:       "10.0.0.100",
					Port:     8080,
					Country:  "US",
					Type:     ProxyTypeResidential,
					Status:   ProxyStatusAvailable,
					Provider: "mock",
				}, nil
			},
		},
	}
	manager := NewProxyManager(store, providers)

	ctx := context.Background()

	err := manager.RefreshPool(ctx)
	if err != nil {
		t.Fatalf("RefreshPool failed: %v", err)
	}

	// Verify proxy was added
	proxies, _ := store.List(nil)
	if len(proxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(proxies))
	}
}

// TestProxyNotFound tests operations on non-existent proxy
func TestProxyNotFound(t *testing.T) {
	store := NewPostgresStore()
	manager := NewProxyManager(store, nil)

	ctx := context.Background()

	_, err := manager.Get(ctx, "non-existent")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}

	err = manager.Bind(ctx, "non-existent", "instance-1")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}

	err = manager.Release(ctx, "non-existent")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}

	err = manager.Unbind(ctx, "non-existent")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}
}