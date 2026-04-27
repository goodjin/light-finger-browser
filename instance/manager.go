package instance

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InstanceManager manages browser instances.
type InstanceManager interface {
	Create(ctx context.Context, cfg *InstanceConfig) (*BrowserInstance, error)
	Destroy(ctx context.Context, id string) error
	Get(ctx context.Context, id string) (*BrowserInstance, error)
	List(ctx context.Context, filter *InstanceFilter) ([]*BrowserInstance, error)
	GetCDPClient(ctx context.Context, id string) (CDPClientInterface, error)
	CloseCDPClient(id string) error
}

// instanceManager implements InstanceManager.
type instanceManager struct {
	store      Store
	processMgr *ProcessManager
	cdpClients sync.Map // map[string]CDPClientInterface
}

// NewInstanceManager creates a new instance manager.
func NewInstanceManager(store Store, processMgr *ProcessManager) InstanceManager {
	return &instanceManager{
		store:      store,
		processMgr: processMgr,
	}
}

// Create creates a new browser instance.
func (m *instanceManager) Create(ctx context.Context, cfg *InstanceConfig) (*BrowserInstance, error) {
	// Check concurrent limit
	count, err := m.store.Count(&InstanceFilter{Status: StatusPtr(StatusRunning)})
	if err != nil {
		return nil, fmt.Errorf("failed to count instances: %w", err)
	}

	if count >= MaxInstancesPerServer {
		return nil, ErrInstanceLimitReached
	}

	// Start process
	instance, err := m.processMgr.Start(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to start process: %w", err)
	}

	// Save to store
	saved, err := m.store.Save(instance)
	if err != nil {
		// Try to stop the process if save fails
		m.processMgr.Stop(ctx, instance.ID)
		return nil, fmt.Errorf("failed to save instance: %w", err)
	}

	return saved, nil
}

// Destroy stops and removes a browser instance.
func (m *instanceManager) Destroy(ctx context.Context, id string) error {
	// Get instance first
	instance, err := m.store.Get(id)
	if err != nil {
		return fmt.Errorf("failed to get instance: %w", err)
	}

	// Stop the process
	if err := m.processMgr.Stop(ctx, id); err != nil {
		return fmt.Errorf("failed to stop process: %w", err)
	}

	// Remove CDP client from cache
	m.cdpClients.Delete(id)

	// Delete from store
	if err := m.store.Delete(id); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}

	_ = instance // instance no longer needed
	return nil
}

// Get retrieves an instance by ID.
func (m *instanceManager) Get(ctx context.Context, id string) (*BrowserInstance, error) {
	instance, err := m.store.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	// Update last active time - work on a copy to avoid modifying the stored object
	copy := *instance
	copy.LastActiveAt = time.Now()
	if err := m.store.Update(&copy); err != nil {
		// Log but don't fail
		_ = err
	}

	return &copy, nil
}

// List returns instances matching the filter.
func (m *instanceManager) List(ctx context.Context, filter *InstanceFilter) ([]*BrowserInstance, error) {
	instances, err := m.store.List(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}
	return instances, nil
}

// GetCDPClient returns a CDP client for the instance.
func (m *instanceManager) GetCDPClient(ctx context.Context, id string) (CDPClientInterface, error) {
	// Check cache first
	if client, ok := m.cdpClients.Load(id); ok {
		return client.(CDPClientInterface), nil
	}

	// Get instance
	instance, err := m.store.Get(id)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if instance.Status != StatusRunning {
		return nil, ErrInstanceNotRunning
	}

	// Establish new connection
	conn, _, err := DefaultDialer.DialContext(ctx, "tcp", instance.CDPEndpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial CDP endpoint: %w", err)
	}

	client := NewCDPClient(conn)
	m.cdpClients.Store(id, client)

	return client, nil
}

// CloseCDPClient closes and removes a CDP client from cache.
func (m *instanceManager) CloseCDPClient(id string) error {
	if client, ok := m.cdpClients.LoadAndDelete(id); ok {
		return client.(CDPClientInterface).Close()
	}
	return nil
}