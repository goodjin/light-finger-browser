package adapter

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/tmos/facebook/internal/proxy"
)

// MockAdapter implements ProxyProvider for testing
type MockAdapter struct {
	proxyIDCounter int64
	proxies       map[string]*proxy.Proxy
	alwaysFail    bool
	latency       int
}

// NewMockAdapter creates a new MockAdapter
func NewMockAdapter() *MockAdapter {
	return &MockAdapter{
		proxies: make(map[string]*proxy.Proxy),
		latency: 50,
	}
}

// NewFailingMockAdapter creates a mock adapter that always fails
func NewFailingMockAdapter() *MockAdapter {
	return &MockAdapter{
		proxies:    make(map[string]*proxy.Proxy),
		alwaysFail: true,
		latency:    50,
	}
}

// GetProxy retrieves a proxy from the mock pool
func (m *MockAdapter) GetProxy(ctx context.Context, country string, proxyType proxy.ProxyType) (*proxy.Proxy, error) {
	if m.alwaysFail {
		return nil, errors.New("mock adapter: always fails")
	}

	id := atomic.AddInt64(&m.proxyIDCounter, 1)
	proxyID := fmt.Sprintf("mock-proxy-%d", id)

	p := &proxy.Proxy{
		ID:       proxyID,
		IP:       fmt.Sprintf("192.168.%d.%d", id/256, id%256),
		Port:     8080,
		Country:  country,
		Type:     proxyType,
		Status:   proxy.ProxyStatusAvailable,
		Provider: "mock",
	}

	m.proxies[proxyID] = p
	return p, nil
}

// ReleaseProxy releases a proxy back to the mock pool
func (m *MockAdapter) ReleaseProxy(ctx context.Context, id string) error {
	if p, ok := m.proxies[id]; ok {
		p.Status = proxy.ProxyStatusAvailable
	}
	return nil
}

// CheckProxy tests if a proxy is working
// In mock adapter, this always returns success with configurable latency
func (m *MockAdapter) CheckProxy(ctx context.Context, p *proxy.Proxy) (bool, int, error) {
	if m.alwaysFail {
		return false, 0, errors.New("mock check: always fails")
	}
	return true, m.latency, nil
}