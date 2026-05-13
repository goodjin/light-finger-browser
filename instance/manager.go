package instance

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
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
	GetBrowserCDPClient(ctx context.Context, id string) (CDPClientInterface, error)
	CloseCDPClient(id string) error
}

// instanceManager implements InstanceManager.
type instanceManager struct {
	store      Store
	processMgr *ProcessManager
	cdpClients sync.Map // map[string]CDPClientInterface
	browserClients sync.Map // map[string]CDPClientInterface - for browser-level commands
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

	// Parse the CDP endpoint to get host:port
	wsURL := instance.CDPEndpoint
	if !strings.HasPrefix(wsURL, "ws://") {
		wsURL = "ws://" + wsURL
	}

	// Query /json to get the correct WebSocket URL for this target
	jsonURL := strings.Replace(wsURL, "ws://", "http://", 1) + "/json"

	resp, err := http.Get(jsonURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query CDP targets: %w", err)
	}
	defer resp.Body.Close()

	// Decode JSON response - it's an array of targets
	var targets []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("failed to decode CDP targets: %w", err)
	}

	// Find the target that matches our instance ID
	targetID := id[:8] // first 8 chars of instance ID
	var targetWSURL string
	for _, t := range targets {
		title, _ := t["title"].(string)
		url, _ := t["url"].(string)
		if strings.Contains(title, targetID) || strings.Contains(url, targetID) {
			if wsu, ok := t["webSocketDebuggerUrl"].(string); ok {
				targetWSURL = wsu
				break
			}
		}
	}

	if targetWSURL == "" {
		return nil, fmt.Errorf("could not find CDP target for instance %s", id)
	}

	// Connect to the target's WebSocket endpoint
	conn, _, err := DefaultDialer.DialContext(ctx, "tcp", targetWSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial CDP target endpoint: %w", err)
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

// GetBrowserCDPClient returns a CDP client connected to the browser-level endpoint.
// This is needed for commands like CreateBrowserContext, CreateTargetWithContext,
// and CloseBrowserContext which must be sent to the browser-level endpoint.
func (m *instanceManager) GetBrowserCDPClient(ctx context.Context, id string) (CDPClientInterface, error) {
	// Check cache first
	if client, ok := m.browserClients.Load(id); ok {
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

	// Parse the CDP endpoint to get host:port
	wsURL := instance.CDPEndpoint
	if !strings.HasPrefix(wsURL, "ws://") {
		wsURL = "ws://" + wsURL
	}

	// Query /json/version to get the correct browser-level WebSocket URL
	// This returns a URL like ws://host:port/devtools/browser/<id>
	versionURL := strings.Replace(wsURL, "ws://", "http://", 1) + "/json/version"
	resp, err := http.Get(versionURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query CDP version: %w", err)
	}
	defer resp.Body.Close()

	var versionResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&versionResp); err != nil {
		return nil, fmt.Errorf("failed to decode CDP version response: %w", err)
	}

	browserWSURL, ok := versionResp["webSocketDebuggerUrl"].(string)
	if !ok || browserWSURL == "" {
		return nil, fmt.Errorf("browser WebSocket URL not found in version response")
	}

	log.Printf("[GetBrowserCDPClient] Browser-level WebSocket URL for instance %s: %s", id, browserWSURL)

	// Connect to the browser-level WebSocket endpoint
	conn, _, err := DefaultDialer.DialContext(ctx, "tcp", browserWSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial CDP browser endpoint: %w", err)
	}

	client := NewCDPClient(conn)

	// Validate the connection is still alive by sending a simple command
	// This catches cases where the browser process crashed but the WebSocket is still connected
	if !client.IsConnected(ctx) {
		client.Close()
		return nil, fmt.Errorf("CDP connection is not alive: browser process may have crashed")
	}

	log.Printf("[GetBrowserCDPClient] Successfully connected and validated browser-level CDP for instance %s", id)

	m.browserClients.Store(id, client)

	return client, nil
}

// CloseBrowserCDPClient closes and removes a browser-level CDP client from cache.
func (m *instanceManager) CloseBrowserCDPClient(id string) error {
	if client, ok := m.browserClients.LoadAndDelete(id); ok {
		return client.(CDPClientInterface).Close()
	}
	return nil
}