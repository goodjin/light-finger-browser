package proxy

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"
)

// ErrNoAvailableProxy indicates no proxy is available in the pool
var ErrNoAvailableProxy = errors.New("no available proxy in pool")

// ErrProxyNotFound indicates the proxy was not found
var ErrProxyNotFound = errors.New("proxy not found")

// ErrProxyAlreadyBound indicates the proxy is already bound to another instance
var ErrProxyAlreadyBound = errors.New("proxy already bound to another instance")

// ProxyManager manages the proxy pool
type ProxyManager struct {
	store     Store
	providers map[string]ProxyProvider
	healthCh  chan string
	mu        sync.RWMutex
}

// NewProxyManager creates a new ProxyManager
func NewProxyManager(store Store, providers map[string]ProxyProvider) *ProxyManager {
	return &ProxyManager{
		store:     store,
		providers: providers,
		healthCh:  make(chan string, 100),
	}
}

// Acquire retrieves an available proxy from the pool
func (m *ProxyManager) Acquire(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 1. Find available IPs
	candidates, err := m.store.List(&ProxyFilter{
		Country: &country,
		Type:    &proxyType,
		Status:  ProxyStatusPtr(ProxyStatusAvailable),
	})
	if err != nil {
		return nil, err
	}

	// 2. Select optimal IP based on success rate and latency
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].SuccessRate != candidates[j].SuccessRate {
			return candidates[i].SuccessRate > candidates[j].SuccessRate
		}
		return candidates[i].Latency < candidates[j].Latency
	})

	if len(candidates) > 0 {
		proxy := candidates[0]
		proxy.Status = ProxyStatusInUse
		return m.store.Update(proxy)
	}

	// 3. Pool empty, fetch from provider
	for name, provider := range m.providers {
		newProxy, err := provider.GetProxy(ctx, country, proxyType)
		if err == nil {
			newProxy.Provider = name
			return m.store.Save(newProxy)
		}
	}

	return nil, ErrNoAvailableProxy
}

// Release releases a proxy back to the pool
func (m *ProxyManager) Release(ctx context.Context, proxyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxy, err := m.store.Get(proxyID)
	if err != nil {
		return err
	}

	proxy.Status = ProxyStatusAvailable
	proxy.BindID = ""
	proxy.BoundAt = time.Time{}

	_, err = m.store.Update(proxy)
	return err
}

// Bind binds a proxy to an instance
func (m *ProxyManager) Bind(ctx context.Context, proxyID string, instanceID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxy, err := m.store.Get(proxyID)
	if err != nil {
		return err
	}

	if proxy.BindID != "" && proxy.BindID != instanceID {
		return ErrProxyAlreadyBound
	}

	proxy.BindID = instanceID
	proxy.Status = ProxyStatusInUse
	proxy.BoundAt = time.Now()

	_, err = m.store.Update(proxy)
	return err
}

// Unbind unbinds a proxy from its current instance
func (m *ProxyManager) Unbind(ctx context.Context, proxyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	proxy, err := m.store.Get(proxyID)
	if err != nil {
		return err
	}

	proxy.BindID = ""
	proxy.Status = ProxyStatusAvailable
	proxy.BoundAt = time.Time{}

	_, err = m.store.Update(proxy)
	return err
}

// Get retrieves a proxy by ID
func (m *ProxyManager) Get(ctx context.Context, proxyID string) (*Proxy, error) {
	return m.store.Get(proxyID)
}

// List retrieves proxies based on filter criteria
func (m *ProxyManager) List(ctx context.Context, filter *ProxyFilter) ([]*Proxy, error) {
	return m.store.List(filter)
}

// HealthCheck performs a health check on a specific proxy
func (m *ProxyManager) HealthCheck(ctx context.Context, proxyID string) error {
	proxy, err := m.store.Get(proxyID)
	if err != nil {
		return err
	}

	provider, ok := m.providers[proxy.Provider]
	if !ok {
		return errors.New("unknown provider")
	}

	proxy.Status = ProxyStatusChecking
	m.store.Update(proxy)

	success, latency, err := provider.CheckProxy(ctx, proxy)
	if err != nil {
		proxy.SuccessRate = (proxy.SuccessRate * 9) / 10
		if proxy.SuccessRate < 0.3 {
			proxy.Status = ProxyStatusDead
		} else {
			proxy.Status = ProxyStatusAvailable
		}
	} else {
		if success {
			proxy.Status = ProxyStatusAvailable
			proxy.SuccessRate = (proxy.SuccessRate*9 + 1) / 10
			proxy.Latency = latency
		} else {
			proxy.SuccessRate = (proxy.SuccessRate * 9) / 10
			if proxy.SuccessRate < 0.3 {
				proxy.Status = ProxyStatusDead
			} else {
				proxy.Status = ProxyStatusAvailable
			}
		}
	}

	proxy.LastCheckAt = time.Now()
	_, err = m.store.Update(proxy)
	return err
}

// RefreshPool refreshes the proxy pool from providers
func (m *ProxyManager) RefreshPool(ctx context.Context) error {
	for name, provider := range m.providers {
		newProxy, err := provider.GetProxy(ctx, "US", ProxyTypeResidential)
		if err != nil {
			continue
		}
		newProxy.Provider = name
		_, err = m.store.Save(newProxy)
		if err != nil {
			continue
		}
	}
	return nil
}

// GetHealthChannel returns the health check notification channel
func (m *ProxyManager) GetHealthChannel() chan string {
	return m.healthCh
}