package proxy

import (
	"testing"
)

// TestStoreUpdateNotFound tests updating non-existent proxy
func TestStoreUpdateNotFound(t *testing.T) {
	store := NewPostgresStore()

	proxy := &Proxy{
		ID:     "update-nonexistent",
		IP:     "192.168.1.1",
		Port:   8080,
		Country: "US",
		Type:   ProxyTypeResidential,
		Status: ProxyStatusAvailable,
	}

	_, err := store.Update(proxy)
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}
}

// TestStoreDeleteNotFound tests deleting non-existent proxy
func TestStoreDeleteNotFound(t *testing.T) {
	store := NewPostgresStore()

	err := store.Delete("non-existent")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound, got: %v", err)
	}
}

// TestStoreListWithAllFilters tests listing with all filters
func TestStoreListWithAllFilters(t *testing.T) {
	store := NewPostgresStore()

	proxy := &Proxy{
		ID:       "filter-test",
		IP:       "192.168.1.100",
		Port:     8080,
		Country:  "US",
		Type:     ProxyTypeResidential,
		Status:   ProxyStatusAvailable,
		BindID:   "instance-1",
		Provider: "mock",
	}
	store.Save(proxy)

	country := "US"
	ptype := ProxyTypeResidential
	status := ProxyStatusAvailable
	bindID := "instance-1"
	provider := "mock"

	proxies, err := store.List(&ProxyFilter{
		Country:  &country,
		Type:     &ptype,
		Status:   &status,
		BindID:   &bindID,
		Provider: &provider,
	})
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(proxies) != 1 {
		t.Errorf("Expected 1 proxy, got %d", len(proxies))
	}
}

// TestStoreListNoFilter tests listing with nil filter
func TestStoreListNoFilter(t *testing.T) {
	store := NewPostgresStore()

	proxies := []*Proxy{
		{ID: "proxy-1", Country: "US", Type: ProxyTypeResidential, Status: ProxyStatusAvailable},
		{ID: "proxy-2", Country: "UK", Type: ProxyTypeDatacenter, Status: ProxyStatusInUse},
	}
	for _, p := range proxies {
		store.Save(p)
	}

	result, err := store.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected 2 proxies, got %d", len(result))
	}
}

// TestStoreSaveWithoutID tests saving proxy without ID
func TestStoreSaveWithoutID(t *testing.T) {
	store := NewPostgresStore()

	proxy := &Proxy{
		IP:      "192.168.1.1",
		Port:    8080,
		Country: "US",
		Type:    ProxyTypeResidential,
		Status:  ProxyStatusAvailable,
	}

	_, err := store.Save(proxy)
	if err == nil {
		t.Error("Expected error when saving proxy without ID")
	}
}

// TestStoreUpdateSuccess tests successful update
func TestStoreUpdateSuccess(t *testing.T) {
	store := NewPostgresStore()

	proxy := &Proxy{
		ID:     "update-test",
		IP:     "192.168.1.1",
		Port:   8080,
		Country: "US",
		Type:   ProxyTypeResidential,
		Status: ProxyStatusAvailable,
	}
	store.Save(proxy)

	proxy.Status = ProxyStatusInUse
	proxy.Latency = 100

	updated, err := store.Update(proxy)
	if err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	if updated.Status != ProxyStatusInUse {
		t.Errorf("Expected status 'in_use', got '%s'", updated.Status)
	}

	if updated.Latency != 100 {
		t.Errorf("Expected latency 100, got %d", updated.Latency)
	}
}

// TestStoreDeleteSuccess tests successful delete
func TestStoreDeleteSuccess(t *testing.T) {
	store := NewPostgresStore()

	proxy := &Proxy{
		ID:     "delete-test",
		IP:     "192.168.1.1",
		Port:   8080,
		Country: "US",
		Type:   ProxyTypeResidential,
		Status: ProxyStatusAvailable,
	}
	store.Save(proxy)

	err := store.Delete("delete-test")
	if err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify it's deleted
	_, err = store.Get("delete-test")
	if err != ErrProxyNotFound {
		t.Errorf("Expected ErrProxyNotFound after delete, got: %v", err)
	}
}

// TestStoreGetSuccess tests successful get
func TestStoreGetSuccess(t *testing.T) {
	store := NewPostgresStore()

	proxy := &Proxy{
		ID:     "get-test",
		IP:     "192.168.1.1",
		Port:   8080,
		Country: "US",
		Type:   ProxyTypeResidential,
		Status: ProxyStatusAvailable,
	}
	store.Save(proxy)

	retrieved, err := store.Get("get-test")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.ID != "get-test" {
		t.Errorf("Expected ID 'get-test', got '%s'", retrieved.ID)
	}
}

// TestStoreListEmpty tests listing from empty store
func TestStoreListEmpty(t *testing.T) {
	store := NewPostgresStore()

	result, err := store.List(nil)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 proxies, got %d", len(result))
	}
}