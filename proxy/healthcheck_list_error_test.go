package proxy

import (
	"errors"
	"testing"
	"time"
)

// MockStoreWithListError implements Store that returns error on List
type MockStoreWithListError struct {
	MockStore
	shouldError bool
}

func (s *MockStoreWithListError) List(filter *ProxyFilter) ([]*Proxy, error) {
	if s.shouldError {
		return nil, errors.New("list error")
	}
	return s.MockStore.List(filter)
}

// TestCheckAllProxiesWithError tests checkAllProxies when List returns error
func TestCheckAllProxiesWithError(t *testing.T) {
	mockStore := &MockStore{
		proxies: make(map[string]*Proxy),
	}

	// Create a proper mock that fails on List
	failingStore := &MockStoreWithListError{
		MockStore:  *mockStore,
		shouldError: true,
	}

	manager := &ProxyManager{
		store:     failingStore,
		providers: make(map[string]ProxyProvider),
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour)
	checker.Start()
	defer checker.Stop()

	// checkAllProxies should handle the error gracefully
	checker.checkAllProxies()

	t.Log("checkAllProxies handled List error gracefully")
}

// TestCheckAllProxiesEmptyList tests checkAllProxies when no proxies in use
func TestCheckAllProxiesEmptyList(t *testing.T) {
	mockStore := &MockStore{
		proxies: make(map[string]*Proxy),
	}

	manager := &ProxyManager{
		store:     mockStore,
		providers: make(map[string]ProxyProvider),
		healthCh:  make(chan string, 10),
	}

	checker := NewHealthChecker(manager, 1*time.Hour)
	checker.Start()
	defer checker.Stop()

	// No proxies in use, checkAllProxies should not panic
	checker.checkAllProxies()

	t.Log("checkAllProxies handled empty list gracefully")
}