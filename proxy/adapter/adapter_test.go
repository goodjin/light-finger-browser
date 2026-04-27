package adapter

import (
	"context"
	"errors"
	"testing"

	"github.com/tmos/facebook/internal/proxy"
)

// MockProxyProvider implements ProxyProvider for testing
type MockProxyProvider struct {
	GetProxyFunc    func(ctx context.Context, country string, proxyType proxy.ProxyType) (*proxy.Proxy, error)
	ReleaseProxyFunc func(ctx context.Context, id string) error
	CheckProxyFunc   func(ctx context.Context, p *proxy.Proxy) (bool, int, error)
}

func (m *MockProxyProvider) GetProxy(ctx context.Context, country string, proxyType proxy.ProxyType) (*proxy.Proxy, error) {
	if m.GetProxyFunc != nil {
		return m.GetProxyFunc(ctx, country, proxyType)
	}
	return &proxy.Proxy{
		ID:       "mock-proxy-1",
		IP:       "192.168.1.1",
		Port:     8080,
		Country:  country,
		Type:     proxyType,
		Status:   proxy.ProxyStatusAvailable,
		Provider: "mock",
	}, nil
}

func (m *MockProxyProvider) ReleaseProxy(ctx context.Context, id string) error {
	if m.ReleaseProxyFunc != nil {
		return m.ReleaseProxyFunc(ctx, id)
	}
	return nil
}

func (m *MockProxyProvider) CheckProxy(ctx context.Context, p *proxy.Proxy) (bool, int, error) {
	if m.CheckProxyFunc != nil {
		return m.CheckProxyFunc(ctx, p)
	}
	return true, 100, nil
}

// TestBrightDataAdapterGetProxy tests BrightData GetProxy
func TestBrightDataAdapterGetProxy(t *testing.T) {
	adapter := NewBrightDataAdapter("test-api-key", "test-zone")

	ctx := context.Background()
	p, err := adapter.GetProxy(ctx, "US", proxy.ProxyTypeResidential)
	if err != nil {
		t.Fatalf("GetProxy failed: %v", err)
	}

	if p.IP != "zproxy.lum-superproxy.io" {
		t.Errorf("Expected IP 'zproxy.lum-superproxy.io', got '%s'", p.IP)
	}

	if p.Port != 22225 {
		t.Errorf("Expected port 22225, got %d", p.Port)
	}

	if p.Type != proxy.ProxyTypeResidential {
		t.Errorf("Expected type 'residential', got '%s'", p.Type)
	}

	if p.Status != proxy.ProxyStatusAvailable {
		t.Errorf("Expected status 'available', got '%s'", p.Status)
	}

	if p.Provider != "brightdata" {
		t.Errorf("Expected provider 'brightdata', got '%s'", p.Provider)
	}
}

// TestBrightDataAdapterReleaseProxy tests BrightData ReleaseProxy
func TestBrightDataAdapterReleaseProxy(t *testing.T) {
	adapter := NewBrightDataAdapter("test-api-key", "test-zone")

	ctx := context.Background()
	err := adapter.ReleaseProxy(ctx, "test-proxy-id")
	if err != nil {
		t.Errorf("ReleaseProxy failed: %v", err)
	}
}

// TestOxylabsAdapterGetProxy tests Oxylabs GetProxy
func TestOxylabsAdapterGetProxy(t *testing.T) {
	adapter := NewOxylabsAdapter("test-user", "test-pass")

	ctx := context.Background()
	p, err := adapter.GetProxy(ctx, "UK", proxy.ProxyTypeDatacenter)
	if err != nil {
		t.Fatalf("GetProxy failed: %v", err)
	}

	if p.IP != "pr.oxylabs.io" {
		t.Errorf("Expected IP 'pr.oxylabs.io', got '%s'", p.IP)
	}

	if p.Port != 7777 {
		t.Errorf("Expected port 7777, got %d", p.Port)
	}

	if p.Type != proxy.ProxyTypeDatacenter {
		t.Errorf("Expected type 'datacenter', got '%s'", p.Type)
	}

	if p.Status != proxy.ProxyStatusAvailable {
		t.Errorf("Expected status 'available', got '%s'", p.Status)
	}

	if p.Provider != "oxylabs" {
		t.Errorf("Expected provider 'oxylabs', got '%s'", p.Provider)
	}
}

// TestOxylabsAdapterReleaseProxy tests Oxylabs ReleaseProxy
func TestOxylabsAdapterReleaseProxy(t *testing.T) {
	adapter := NewOxylabsAdapter("test-user", "test-pass")

	ctx := context.Background()
	err := adapter.ReleaseProxy(ctx, "test-proxy-id")
	if err != nil {
		t.Errorf("ReleaseProxy failed: %v", err)
	}
}

// TestMockProxyProviderGetProxyError tests mock provider error handling
func TestMockProxyProviderGetProxyError(t *testing.T) {
	mockProvider := &MockProxyProvider{
		GetProxyFunc: func(ctx context.Context, country string, proxyType proxy.ProxyType) (*proxy.Proxy, error) {
			return nil, errors.New("provider error")
		},
	}

	ctx := context.Background()
	_, err := mockProvider.GetProxy(ctx, "US", proxy.ProxyTypeResidential)
	if err == nil {
		t.Error("Expected error, got nil")
	}
}

// TestMockProxyProviderCheckProxyError tests mock provider check error
func TestMockProxyProviderCheckProxyError(t *testing.T) {
	mockProvider := &MockProxyProvider{
		CheckProxyFunc: func(ctx context.Context, p *proxy.Proxy) (bool, int, error) {
			return false, 0, errors.New("connection failed")
		},
	}

	ctx := context.Background()
	success, latency, err := mockProvider.CheckProxy(ctx, &proxy.Proxy{})
	if err == nil {
		t.Error("Expected error, got nil")
	}
	if success != false {
		t.Errorf("Expected success=false, got %v", success)
	}
	if latency != 0 {
		t.Errorf("Expected latency=0, got %d", latency)
	}
}

// TestMockAdapterGetProxy tests MockAdapter GetProxy
func TestMockAdapterGetProxy(t *testing.T) {
	adapter := NewMockAdapter()

	ctx := context.Background()
	p, err := adapter.GetProxy(ctx, "US", proxy.ProxyTypeResidential)
	if err != nil {
		t.Fatalf("GetProxy failed: %v", err)
	}

	if p.IP == "" {
		t.Error("Expected non-empty IP")
	}

	if p.Type != proxy.ProxyTypeResidential {
		t.Errorf("Expected type 'residential', got '%s'", p.Type)
	}

	if p.Status != proxy.ProxyStatusAvailable {
		t.Errorf("Expected status 'available', got '%s'", p.Status)
	}

	if p.Provider != "mock" {
		t.Errorf("Expected provider 'mock', got '%s'", p.Provider)
	}
}

// TestMockAdapterGetProxyMultiple tests MockAdapter GetProxy multiple times
func TestMockAdapterGetProxyMultiple(t *testing.T) {
	adapter := NewMockAdapter()

	ctx := context.Background()

	p1, err := adapter.GetProxy(ctx, "US", proxy.ProxyTypeResidential)
	if err != nil {
		t.Fatalf("GetProxy 1 failed: %v", err)
	}

	p2, err := adapter.GetProxy(ctx, "UK", proxy.ProxyTypeDatacenter)
	if err != nil {
		t.Fatalf("GetProxy 2 failed: %v", err)
	}

	if p1.ID == p2.ID {
		t.Error("Expected different proxy IDs")
	}
}

// TestMockAdapterReleaseProxy tests MockAdapter ReleaseProxy
func TestMockAdapterReleaseProxy(t *testing.T) {
	adapter := NewMockAdapter()

	ctx := context.Background()
	err := adapter.ReleaseProxy(ctx, "test-id")
	if err != nil {
		t.Errorf("ReleaseProxy failed: %v", err)
	}
}

// TestMockAdapterCheckProxy tests MockAdapter CheckProxy
func TestMockAdapterCheckProxy(t *testing.T) {
	adapter := NewMockAdapter()

	ctx := context.Background()
	p := &proxy.Proxy{
		ID:    "test-proxy",
		IP:    "192.168.1.1",
		Port:  8080,
		Username: "user",
		Password: "pass",
	}

	success, latency, err := adapter.CheckProxy(ctx, p)
	if err != nil {
		t.Fatalf("CheckProxy failed: %v", err)
	}

	if !success {
		t.Error("Expected success=true")
	}

	if latency <= 0 {
		t.Errorf("Expected positive latency, got %d", latency)
	}
}

// TestFailingMockAdapterGetProxy tests FailingMockAdapter GetProxy
func TestFailingMockAdapterGetProxy(t *testing.T) {
	adapter := NewFailingMockAdapter()

	ctx := context.Background()
	_, err := adapter.GetProxy(ctx, "US", proxy.ProxyTypeResidential)
	if err == nil {
		t.Error("Expected error from FailingMockAdapter")
	}
}

// TestFailingMockAdapterCheckProxy tests FailingMockAdapter CheckProxy
func TestFailingMockAdapterCheckProxy(t *testing.T) {
	adapter := NewFailingMockAdapter()

	ctx := context.Background()
	p := &proxy.Proxy{ID: "test"}

	success, _, err := adapter.CheckProxy(ctx, p)
	if err == nil {
		t.Error("Expected error from FailingMockAdapter.CheckProxy")
	}

	if success {
		t.Error("Expected success=false")
	}
}

// TestMockAdapterReleaseProxyWithExistingProxy tests MockAdapter ReleaseProxy with existing proxy
func TestMockAdapterReleaseProxyWithExistingProxy(t *testing.T) {
	adapter := NewMockAdapter()

	ctx := context.Background()

	// First get a proxy
	p, err := adapter.GetProxy(ctx, "US", proxy.ProxyTypeResidential)
	if err != nil {
		t.Fatalf("GetProxy failed: %v", err)
	}

	// Release should work without error even though it doesn't do much in mock
	err = adapter.ReleaseProxy(ctx, p.ID)
	if err != nil {
		t.Errorf("ReleaseProxy failed: %v", err)
	}
}

// TestMockAdapterReleaseProxyNonExistent tests MockAdapter ReleaseProxy with non-existent proxy
func TestMockAdapterReleaseProxyNonExistent(t *testing.T) {
	adapter := NewMockAdapter()

	ctx := context.Background()

	// Release non-existent proxy should not error
	err := adapter.ReleaseProxy(ctx, "non-existent-id")
	if err != nil {
		t.Errorf("ReleaseProxy for non-existent proxy failed: %v", err)
	}
}