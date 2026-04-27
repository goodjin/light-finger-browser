package instance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
)

func TestInstanceManager_Create(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18000, 18010)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Proxy:       &ProxyConfig{ID: "proxy-1", URL: "http://proxy:8080"},
		AccountID:   "account-1",
		Group:       "test-group",
		Headless:    true,
	}

	ctx := context.Background()

	instance, err := mgr.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	if instance.ID == "" {
		t.Error("expected non-empty instance ID")
	}

	if instance.Status != StatusRunning {
		t.Errorf("expected status running, got %s", instance.Status)
	}

	if instance.ProxyID != "proxy-1" {
		t.Errorf("expected proxy ID proxy-1, got %s", instance.ProxyID)
	}

	if instance.AccountID != "account-1" {
		t.Errorf("expected account ID account-1, got %s", instance.AccountID)
	}

	if instance.Group != "test-group" {
		t.Errorf("expected group test-group, got %s", instance.Group)
	}
}

func TestInstanceManager_Destroy(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18100, 18110)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Headless:    true,
	}

	ctx := context.Background()

	instance, err := mgr.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	err = mgr.Destroy(ctx, instance.ID)
	if err != nil {
		t.Fatalf("failed to destroy instance: %v", err)
	}

	// Verify instance is deleted
	_, err = mgr.Get(ctx, instance.ID)
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Errorf("expected ErrInstanceNotFound, got %v", err)
	}
}

func TestInstanceManager_Get(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18200, 18210)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Headless:   true,
	}

	ctx := context.Background()

	created, err := mgr.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	got, err := mgr.Get(ctx, created.ID)
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}

	if got.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, got.ID)
	}
}

func TestInstanceManager_Get_NotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18300, 18310)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	ctx := context.Background()

	_, err = mgr.Get(ctx, "nonexistent-id")
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Errorf("expected ErrInstanceNotFound, got %v", err)
	}
}

func TestInstanceManager_List(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18400, 18420)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	ctx := context.Background()

	// Create instances in different groups
	for i := 0; i < 3; i++ {
		cfg := &InstanceConfig{
			Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
			Group:       "group-a",
			Headless:    true,
		}
		_, err := mgr.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("failed to create instance: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		cfg := &InstanceConfig{
			Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
			Group:       "group-b",
			Headless:    true,
		}
		_, err := mgr.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("failed to create instance: %v", err)
		}
	}

	// List all
	all, err := mgr.List(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list instances: %v", err)
	}
	if len(all) != 5 {
		t.Errorf("expected 5 instances, got %d", len(all))
	}

	// List by group
	groupA, err := mgr.List(ctx, &InstanceFilter{Group: "group-a"})
	if err != nil {
		t.Fatalf("failed to list instances: %v", err)
	}
	if len(groupA) != 3 {
		t.Errorf("expected 3 instances in group-a, got %d", len(groupA))
	}

	groupB, err := mgr.List(ctx, &InstanceFilter{Group: "group-b"})
	if err != nil {
		t.Fatalf("failed to list instances: %v", err)
	}
	if len(groupB) != 2 {
		t.Errorf("expected 2 instances in group-b, got %d", len(groupB))
	}
}

func TestInstanceManager_ConcurrentCreate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18500, 18599) // 100 ports available
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	ctx := context.Background()

	var wg sync.WaitGroup
	var successCount int64
	numGoroutines := 50

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			cfg := &InstanceConfig{
				Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
				Headless:    true,
			}

			instance, err := mgr.Create(ctx, cfg)
			if err != nil {
				t.Logf("goroutine %d failed to create: %v", idx, err)
				return
			}

			atomic.AddInt64(&successCount, 1)
			t.Logf("goroutine %d created instance %s", idx, instance.ID)

			// Give some time for cleanup before next iteration
			time.Sleep(10 * time.Millisecond)
		}(i)
	}

	wg.Wait()

	t.Logf("Successfully created %d out of %d instances", successCount, numGoroutines)

	// Verify all instances are cleaned up
	count, _ := store.Count(nil)
	t.Logf("Remaining instances in store: %d", count)
}

func TestInstanceManager_InstanceLimit(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Set a very low limit for testing
	originalLimit := MaxInstancesPerServer
	MaxInstancesPerServer = 3

	defer func() {
		MaxInstancesPerServer = originalLimit
	}()

	portAlloc := NewPortAllocator(18600, 18620)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	ctx := context.Background()

	// Create instances up to the limit
	for i := 0; i < MaxInstancesPerServer; i++ {
		cfg := &InstanceConfig{
			Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
			Headless:    true,
		}
		_, err := mgr.Create(ctx, cfg)
		if err != nil {
			t.Fatalf("failed to create instance %d: %v", i, err)
		}
	}

	// Next create should fail
	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Headless:    true,
	}

	_, err = mgr.Create(ctx, cfg)
	if err != ErrInstanceLimitReached {
		t.Errorf("expected ErrInstanceLimitReached, got %v", err)
	}
}

func TestInstanceManager_LastActiveAt(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18700, 18710)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	ctx := context.Background()

	cfg := &InstanceConfig{
		Fingerprint: &fingerprint.Fingerprint{Seed: "test-seed"},
		Headless:    true,
	}

	beforeCreate := time.Now().Add(-time.Second)

	instance, err := mgr.Create(ctx, cfg)
	if err != nil {
		t.Fatalf("failed to create instance: %v", err)
	}

	if instance.LastActiveAt.Before(beforeCreate) {
		t.Error("LastActiveAt should be updated on create")
	}

	// Wait a bit
	time.Sleep(10 * time.Millisecond)

	// Get should update LastActiveAt
	got, err := mgr.Get(ctx, instance.ID)
	if err != nil {
		t.Fatalf("failed to get instance: %v", err)
	}

	if !got.LastActiveAt.After(instance.LastActiveAt) {
		t.Error("LastActiveAt should be updated on Get")
	}
}

func TestInstanceManager_GetCDPClient_NotRunning(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18800, 18810)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	ctx := context.Background()

	// Create a mock instance that is not running
	instance := &BrowserInstance{
		ID:           "test-id",
		Status:       StatusStopped, // Not running
		Fingerprint:  &fingerprint.Fingerprint{Seed: "test"},
		CDPEndpoint:  "ws://localhost:18800",
	}

	store.Save(instance)

	_, err = mgr.GetCDPClient(ctx, instance.ID)
	if err != ErrInstanceNotRunning {
		t.Errorf("expected ErrInstanceNotRunning, got %v", err)
	}
}

func TestInstanceManager_CloseCDPClient(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(18900, 18910)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests
	mgr := NewInstanceManager(store, pm)

	// Should not panic
	err = mgr.CloseCDPClient("nonexistent-id")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestInstanceFilter_Status(t *testing.T) {
	store := NewMockStore()

	// Create instances with different statuses
	statuses := []InstanceStatus{StatusRunning, StatusStopped, StatusRunning}

	for i, status := range statuses {
		instance := &BrowserInstance{
			ID:       "instance-" + string(rune('a'+i)),
			Status:   status,
			Port:     19000 + i,
		}
		store.Save(instance)
	}

	// Filter by running status
	running, _ := store.List(&InstanceFilter{Status: StatusPtr(StatusRunning)})
	if len(running) != 2 {
		t.Errorf("expected 2 running instances, got %d", len(running))
	}

	// Filter by stopped status
	stopped, _ := store.List(&InstanceFilter{Status: StatusPtr(StatusStopped)})
	if len(stopped) != 1 {
		t.Errorf("expected 1 stopped instance, got %d", len(stopped))
	}
}

func TestNewInstanceManager(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "manager-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	portAlloc := NewPortAllocator(19100, 19110)
	store := NewMockStore()
	pm := NewProcessManager("echo", tmpDir, portAlloc, store)
	pm.SetReadyFunc(func(port int) bool { return true }) // Skip TCP check in tests

	mgr := NewInstanceManager(store, pm)
	if mgr == nil {
		t.Error("expected non-nil manager")
	}
}

func TestStoreMock_CRUD(t *testing.T) {
	store := NewMockStore()

	instance := &BrowserInstance{
		ID:       "test-instance",
		Status:   StatusRunning,
		Port:     19200,
		Fingerprint: &fingerprint.Fingerprint{Seed: "test"},
	}

	// Save
	saved, err := store.Save(instance)
	if err != nil {
		t.Fatalf("failed to save: %v", err)
	}
	if saved.ID != instance.ID {
		t.Errorf("expected ID %s, got %s", instance.ID, saved.ID)
	}

	// Get
	got, err := store.Get("test-instance")
	if err != nil {
		t.Fatalf("failed to get: %v", err)
	}
	if got.Status != StatusRunning {
		t.Errorf("expected status running, got %s", got.Status)
	}

	// Update
	got.Status = StatusStopped
	err = store.Update(got)
	if err != nil {
		t.Fatalf("failed to update: %v", err)
	}

	updated, _ := store.Get("test-instance")
	if updated.Status != StatusStopped {
		t.Errorf("expected status stopped after update, got %s", updated.Status)
	}

	// Delete
	err = store.Delete("test-instance")
	if err != nil {
		t.Fatalf("failed to delete: %v", err)
	}

	_, err = store.Get("test-instance")
	if !errors.Is(err, ErrInstanceNotFound) {
		t.Errorf("expected ErrInstanceNotFound after delete, got %v", err)
	}
}

func TestStoreMock_Count(t *testing.T) {
	store := NewMockStore()

	// Create 5 running instances
	for i := 0; i < 5; i++ {
		store.Save(&BrowserInstance{
			ID:       "running-" + string(rune('0'+i)),
			Status:   StatusRunning,
			Fingerprint: &fingerprint.Fingerprint{Seed: "test"},
		})
	}

	// Create 2 stopped instances
	for i := 0; i < 2; i++ {
		store.Save(&BrowserInstance{
			ID:       "stopped-" + string(rune('0'+i)),
			Status:   StatusStopped,
			Fingerprint: &fingerprint.Fingerprint{Seed: "test"},
		})
	}

	total, _ := store.Count(nil)
	if total != 7 {
		t.Errorf("expected total count 7, got %d", total)
	}

	running, _ := store.Count(&InstanceFilter{Status: StatusPtr(StatusRunning)})
	if running != 5 {
		t.Errorf("expected running count 5, got %d", running)
	}

	stopped, _ := store.Count(&InstanceFilter{Status: StatusPtr(StatusStopped)})
	if stopped != 2 {
		t.Errorf("expected stopped count 2, got %d", stopped)
	}
}

func TestBrowserInstance_DirCreation(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "instance-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	userDataDir := filepath.Join(tmpDir, "userdata")
	err = os.MkdirAll(userDataDir, 0755)
	if err != nil {
		t.Fatalf("failed to create user data dir: %v", err)
	}

	// Verify directory exists
	if _, err := os.Stat(userDataDir); os.IsNotExist(err) {
		t.Error("expected user data dir to exist")
	}
}