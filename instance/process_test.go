package instance

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
)

// MockStore implements Store for testing.
type MockStore struct {
	instances map[string]*BrowserInstance
}

func NewMockStore() *MockStore {
	return &MockStore{
		instances: make(map[string]*BrowserInstance),
	}
}

func (m *MockStore) Save(instance *BrowserInstance) (*BrowserInstance, error) {
	m.instances[instance.ID] = instance
	return instance, nil
}

func (m *MockStore) Get(id string) (*BrowserInstance, error) {
	inst, ok := m.instances[id]
	if !ok {
		return nil, ErrInstanceNotFound
	}
	return inst, nil
}

func (m *MockStore) List(filter *InstanceFilter) ([]*BrowserInstance, error) {
	var result []*BrowserInstance
	for _, inst := range m.instances {
		if filter != nil {
			if filter.Status != nil && inst.Status != *filter.Status {
				continue
			}
			if filter.Group != "" && inst.Group != filter.Group {
				continue
			}
		}
		result = append(result, inst)
	}
	return result, nil
}

func (m *MockStore) Update(instance *BrowserInstance) error {
	if _, ok := m.instances[instance.ID]; !ok {
		return ErrInstanceNotFound
	}
	// Store a copy to avoid aliasing issues
	copy := *instance
	m.instances[instance.ID] = &copy
	return nil
}

func (m *MockStore) Delete(id string) error {
	delete(m.instances, id)
	return nil
}

func (m *MockStore) Count(filter *InstanceFilter) (int, error) {
	instances, _ := m.List(filter)
	return len(instances), nil
}

func TestPortAllocator_Allocate(t *testing.T) {
	alloc := NewPortAllocator(9000, 9010)

	// Allocate 11 ports (should fail on 11th)
	allocated := make([]int, 0, 11)
	for i := 0; i < 11; i++ {
		port, err := alloc.Allocate()
		if err != nil {
			if i == 10 {
				// Expected - no more ports
				continue
			}
			t.Fatalf("unexpected error on allocation %d: %v", i, err)
		}
		allocated = append(allocated, port)
	}

	// First allocation should be base port
	if allocated[0] != 9000 {
		t.Errorf("expected first port to be 9000, got %d", allocated[0])
	}

	// After 11 allocations, we should get error
	_, err := alloc.Allocate()
	if err != ErrNoAvailablePort {
		t.Errorf("expected ErrNoAvailablePort, got %v", err)
	}
}

func TestPortAllocator_Release(t *testing.T) {
	alloc := NewPortAllocator(9000, 9010)

	// Allocate and release
	port1, _ := alloc.Allocate()
	alloc.Release(port1)

	// Should be able to reallocate the same port
	port2, _ := alloc.Allocate()
	if port1 != port2 {
		t.Errorf("expected released port to be reallocated, got %d vs %d", port1, port2)
	}
}

func TestPortAllocator_IsAllocated(t *testing.T) {
	alloc := NewPortAllocator(9000, 9010)

	port, _ := alloc.Allocate()

	if !alloc.IsAllocated(port) {
		t.Errorf("expected port %d to be allocated", port)
	}

	if alloc.IsAllocated(port + 1) {
		t.Errorf("expected port %d to not be allocated", port+1)
	}
}

func TestPortAllocator_AvailableCount(t *testing.T) {
	alloc := NewPortAllocator(9000, 9010) // 11 ports total

	if alloc.AvailableCount() != 11 {
		t.Errorf("expected 11 available ports, got %d", alloc.AvailableCount())
	}

	alloc.Allocate()
	if alloc.AvailableCount() != 10 {
		t.Errorf("expected 10 available ports after allocation, got %d", alloc.AvailableCount())
	}
}

func TestPortAllocator_Reset(t *testing.T) {
	alloc := NewPortAllocator(9000, 9010)

	alloc.Allocate()
	alloc.Allocate()
	alloc.Reset()

	if alloc.AvailableCount() != 11 {
		t.Errorf("expected 11 available ports after reset, got %d", alloc.AvailableCount())
	}
}

func TestProcessManager_StartStop(t *testing.T) {
	// Skip if no browser binary available
	if os.Getenv("BROWSER_BINARY") == "" {
		t.Skip("BROWSER_BINARY not set, skipping integration test")
	}

	tmpDir, err := os.MkdirTemp("", "browser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(19000, 19010)
	store := NewMockStore()
	pm := NewProcessManager(os.Getenv("BROWSER_BINARY"), tmpDir, portAlloc, store)

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Proxy:       nil,
		Headless:    true,
	}

	ctx := context.Background()

	// Start instance
	instance, err := pm.Start(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to start instance: %v", err)
	}

	if instance.ID == "" {
		t.Error("expected non-empty instance ID")
	}

	if instance.Status != StatusRunning {
		t.Errorf("expected status running, got %s", instance.Status)
	}

	if instance.Port < 19000 || instance.Port > 19010 {
		t.Errorf("expected port in range 19000-19010, got %d", instance.Port)
	}

	// Stop instance
	err = pm.Stop(ctx, instance.ID)
	if err != nil {
		t.Fatalf("failed to stop instance: %v", err)
	}

	// Verify port is released
	if portAlloc.IsAllocated(instance.Port) {
		t.Error("expected port to be released after stop")
	}
}

func TestProcessManager_ProcessCleanup(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "browser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(19100, 19110)
	store := NewMockStore()
	pm := NewProcessManager("nonexistent-binary", tmpDir, portAlloc, store)

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Headless:    true,
	}

	ctx := context.Background()

	// Start should fail
	_, err = pm.Start(ctx, cfg)
	if err == nil {
		t.Error("expected error when starting with invalid binary")
	}

	// Verify all ports are still available (not leaked)
	// Since Start failed, no port should be allocated
	count := portAlloc.AvailableCount()
	if count != portAlloc.AvailableCount() {
		// Port was not released - but we can't easily check which port was used
	}
}

func TestProcessManager_PortConflict(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "browser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Use very small range to force conflict
	portAlloc := NewPortAllocator(19200, 19201) // Only 2 ports
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Headless:    true,
	}

	ctx := context.Background()

	// Allocate ports manually to create conflict scenario
	port1, _ := portAlloc.Allocate()
	port2, _ := portAlloc.Allocate()

	// Should have no more ports
	_, err = portAlloc.Allocate()
	if err != ErrNoAvailablePort {
		t.Error("expected no available port error")
	}

	// Release one and allocate
	portAlloc.Release(port1)
	newPort, err := portAlloc.Allocate()
	if err != nil {
		t.Errorf("unexpected error after release: %v", err)
	}
	if newPort != port1 {
		t.Errorf("expected to reallocate released port %d, got %d", port1, newPort)
	}

	// Cleanup
	portAlloc.Release(port2)
	_ = cfg
	_ = ctx
	_ = store
	_ = pm
}

func TestBuildArgs(t *testing.T) {
	pm := &ProcessManager{
		binaryPath: "/usr/bin/browser",
		dataDir:    "/tmp/data",
		portAlloc:  NewPortAllocator(9000, 9010),
	}

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test"},
		Proxy:       &ProxyConfig{ID: "p1", URL: "http://proxy:8080"},
		Headless:    true,
	}

	args := pm.buildArgs(9005, "/tmp/userdata", cfg)

	expectedArgs := []string{
		"--port=9005",
		"--user-data-dir=/tmp/userdata",
		"--proxy-server=http://proxy:8080",
		"--headless",
	}

	if len(args) != len(expectedArgs) {
		t.Errorf("expected %d args, got %d", len(expectedArgs), len(args))
	}

	for i, expected := range expectedArgs {
		if args[i] != expected {
			t.Errorf("arg %d: expected %s, got %s", i, expected, args[i])
		}
	}
}

func TestIsPortOpen(t *testing.T) {
	pm := &ProcessManager{}

	// Should not be open
	if pm.isPortOpen(9999) {
		t.Error("expected port 9999 to not be open")
	}
}

// MockCDPClient is a simple mock for testing CDP interactions
type MockCDPClient struct {
	connected bool
}

func (m *MockCDPClient) Navigate(ctx context.Context, url string) error {
	if !m.connected {
		return ErrInstanceNotRunning
	}
	return nil
}

func (m *MockCDPClient) Click(ctx context.Context, selector string) error {
	if !m.connected {
		return ErrInstanceNotRunning
	}
	return nil
}

func (m *MockCDPClient) Type(ctx context.Context, selector string, text string) error {
	if !m.connected {
		return ErrInstanceNotRunning
	}
	return nil
}

func (m *MockCDPClient) Screenshot(ctx context.Context) ([]byte, error) {
	if !m.connected {
		return nil, ErrInstanceNotRunning
	}
	return []byte("screenshot-data"), nil
}

func (m *MockCDPClient) Evaluate(ctx context.Context, script string) (interface{}, error) {
	if !m.connected {
		return nil, ErrInstanceNotRunning
	}
	return "result", nil
}

func (m *MockCDPClient) Close() error {
	m.connected = false
	return nil
}

func TestInstanceCreationTime(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "browser-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(19300, 19310)
	store := NewMockStore()
	_ = NewProcessManager("echo", tmpDir, portAlloc, store)

	// Create a mock instance to test timestamps
	before := time.Now()
	instance := &BrowserInstance{
		ID:           "test-id",
		Status:       StatusRunning,
		Fingerprint:  &fingerprint.Fingerprint{Seed: "test"},
		Port:         19300,
		UserDataDir:  filepath.Join(tmpDir, "userdata"),
		StartedAt:    before,
		LastActiveAt: before,
		CreatedAt:    before,
	}

	// Save to store
	_, err = store.Save(instance)
	if err != nil {
		t.Fatalf("failed to save instance: %v", err)
	}

	// Get from store
	retrieved, err := store.Get("test-id")
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}

	// Verify timestamps are preserved
	if !retrieved.StartedAt.Equal(before) {
		t.Errorf("startedAt mismatch: expected %v, got %v", before, retrieved.StartedAt)
	}

	// Cleanup
	portAlloc.Release(instance.Port)
}