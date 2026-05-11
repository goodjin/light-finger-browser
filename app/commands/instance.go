package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

type InstanceService struct {
	manager       browserRuntimeManager
	store         *sqlite.InstanceStore
	cdpClients    sync.Map // instanceID -> CDPClient
	targetURLs    sync.Map // instanceID -> string (cached webSocketDebuggerUrl)
	contextStores sync.Map // instanceID -> *ContextStore
	configSvc    *ConfigService // config service for singleton mode
	singleton    *instance.BrowserInstance // singleton instance
	singletonMu  sync.RWMutex
}

func NewInstanceService(db *sqlite.DB) *InstanceService {
	manager := newBrowserRuntimeManager(db)
	store := sqlite.NewInstanceStore(db)
	return &InstanceService{
		manager:    manager,
		store:      store,
		configSvc:  NewConfigService(),
	}
}

// GetOrCreateSingletonInstance gets the singleton instance, creating one if it doesn't exist.
// This implements SI-001: Application startup auto-creates instance
func (s *InstanceService) GetOrCreateSingletonInstance(ctx context.Context) (*instance.BrowserInstance, error) {
	s.singletonMu.RLock()
	if s.singleton != nil && s.singleton.Status == instance.StatusRunning {
		s.singletonMu.RUnlock()
		return s.singleton, nil
	}
	s.singletonMu.RUnlock()

	s.singletonMu.Lock()
	defer s.singletonMu.Unlock()

	// Double-check after acquiring write lock
	if s.singleton != nil && s.singleton.Status == instance.StatusRunning {
		return s.singleton, nil
	}

	// Get port from config
	port, err := s.configSvc.GetInstancePort()
	if err != nil {
		log.Printf("[GetOrCreateSingletonInstance] Failed to get port from config, using default: %v", err)
		port = DefaultInstancePort
	}

	// Get headless mode from config
	headless, _ := s.configSvc.GetHeadless()

	// Create new instance config
	cfg := &instance.InstanceConfig{
		Name:      "Default Instance",
		Headless: headless,
	}

	// Try to create the instance
	inst, err := s.manager.Start(ctx, cfg)
	if err != nil {
		// If the port is in use, try with auto-assigned port
		log.Printf("[GetOrCreateSingletonInstance] Failed to start with port %d, trying auto-assigned port: %v", port, err)
		cfgNoPort := &instance.InstanceConfig{
			Name:      "Default Instance",
			Headless: headless,
		}
		inst, err = s.manager.Start(ctx, cfgNoPort)
		if err != nil {
			return nil, fmt.Errorf("failed to create singleton instance: %w", err)
		}
	}

	s.singleton = inst
	log.Printf("[GetOrCreateSingletonInstance] Created singleton instance: %s on port %d", inst.ID, inst.Port)
	return inst, nil
}

// GetSingletonInstance returns the singleton instance without creating it.
func (s *InstanceService) GetSingletonInstance(ctx context.Context) (*instance.BrowserInstance, error) {
	s.singletonMu.RLock()
	defer s.singletonMu.RUnlock()

	if s.singleton == nil {
		// Try to load from database
		instances, err := s.store.List(&instance.InstanceFilter{Status: instance.StatusPtr(instance.StatusRunning)})
		if err != nil {
			return nil, err
		}
		if len(instances) > 0 {
			s.singleton = instances[0]
		}
	}

	return s.singleton, nil
}

// IsSingletonRunning checks if a singleton instance is currently running.
func (s *InstanceService) IsSingletonRunning() bool {
	s.singletonMu.RLock()
	defer s.singletonMu.RUnlock()
	return s.singleton != nil && s.singleton.Status == instance.StatusRunning
}

// GetSingletonInstanceID returns the ID of the singleton instance.
func (s *InstanceService) GetSingletonInstanceID() string {
	s.singletonMu.RLock()
	defer s.singletonMu.RUnlock()
	if s.singleton != nil {
		return s.singleton.ID
	}
	return ""
}

// StopSingleton stops the singleton instance.
func (s *InstanceService) StopSingleton(ctx context.Context) error {
	s.singletonMu.Lock()
	defer s.singletonMu.Unlock()

	if s.singleton == nil {
		return nil
	}

	err := s.manager.Stop(ctx, s.singleton.ID)
	if err != nil {
		return err
	}

	s.singleton.Status = instance.StatusStopped
	return nil
}

func (s *InstanceService) CreateInstance(ctx context.Context, cfg *InstanceConfig) (*instance.BrowserInstance, error) {
	return s.manager.Start(ctx, cfg)
}

func (s *InstanceService) DestroyInstance(ctx context.Context, id string) error {
	return s.manager.Stop(ctx, id)
}

func (s *InstanceService) StopInstance(ctx context.Context, id string) error {
	// Cleanup all contexts and tabs before stopping
	if store, ok := s.contextStores.LoadAndDelete(id); ok {
		mainClient, err := s.GetCDPClient(ctx, id)
		if err == nil {
			defer s.CloseCDPClient(id)
			store.(*ContextStore).CloseAll(ctx, mainClient)
		}
	}

	return s.manager.Stop(ctx, id)
}

func (s *InstanceService) RestartInstance(ctx context.Context, id string) (*instance.BrowserInstance, error) {
	inst, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	cfg := &instance.InstanceConfig{
		Name:         inst.Name,
		AccountLabel: "",
		Fingerprint:  inst.Fingerprint,
		Proxy:        nil,
		AccountID:    inst.AccountID,
		Group:        inst.Group,
		Headless:     inst.Headless,
	}
	if inst.ProxyID != "" || inst.ProxyURL != "" {
		cfg.Proxy = &instance.ProxyConfig{ID: inst.ProxyID, URL: inst.ProxyURL}
	}
	return s.RestartInstanceWithConfig(ctx, id, cfg)
}

func (s *InstanceService) DeleteInstance(ctx context.Context, id string) error {
	inst, err := s.store.Get(id)
	if err != nil {
		return err
	}
	if inst.Status != instance.StatusStopped {
		return fmt.Errorf("instance must be stopped before delete")
	}
	return s.manager.Delete(id)
}

func (s *InstanceService) RestartInstanceWithConfig(ctx context.Context, id string, cfg *instance.InstanceConfig) (*instance.BrowserInstance, error) {
	inst, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}
	if inst.Status != instance.StatusStopped {
		if err := s.manager.Stop(ctx, id); err != nil {
			return nil, err
		}
		inst, err = s.store.Get(id)
		if err != nil {
			return nil, err
		}
	}
	return s.manager.Restart(ctx, inst, cfg)
}

func (s *InstanceService) GetInstance(ctx context.Context, id string) (*instance.BrowserInstance, error) {
	return s.store.Get(id)
}

func (s *InstanceService) ListInstances(ctx context.Context, filter *instance.InstanceFilter) ([]*instance.BrowserInstance, error) {
	log.Println("[ListInstances] Querying instances with filter:", filter)
	result, err := s.store.List(filter)
	if err != nil {
		log.Printf("[ListInstances] Error: %v", err)
		return nil, err
	}
	log.Printf("[ListInstances] Found %d instances", len(result))
	return result, nil
}

func (s *InstanceService) GetCDPClient(ctx context.Context, id string) (instance.CDPClientInterface, error) {
	inst, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}

	if inst.Status != instance.StatusRunning {
		return nil, instance.ErrInstanceNotRunning
	}

	// Build WebSocket URL from CDP endpoint
	wsURL := inst.CDPEndpoint
	if !strings.HasPrefix(wsURL, "ws://") {
		wsURL = "ws://" + wsURL
	}

	// Query /json/version to get the browser-level WebSocket URL
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

	// Connect to browser-level WebSocket for commands that require browser-level access
	// (e.g., createBrowserContext, disposeBrowserContext, getTargets)
	conn, _, err := instance.DefaultDialer.DialContext(ctx, "tcp", browserWSURL)
	if err != nil {
		return nil, fmt.Errorf("failed to dial CDP browser endpoint: %w", err)
	}

	client := instance.NewCDPClient(conn)
	s.cdpClients.Store(id, client)
	return client, nil
}

// GetPageCDPClient returns a page-level CDP client for the instance.
// Use this for page-specific operations like navigate, evaluate.
func (s *InstanceService) GetPageCDPClient(ctx context.Context, id string) (instance.CDPClientInterface, error) {
	inst, err := s.store.Get(id)
	if err != nil {
		return nil, err
	}

	if inst.Status != instance.StatusRunning {
		return nil, instance.ErrInstanceNotRunning
	}

	// Build WebSocket URL from CDP endpoint
	wsURL := inst.CDPEndpoint
	if !strings.HasPrefix(wsURL, "ws://") {
		wsURL = "ws://" + wsURL
	}

	var targetWSURL string

	// First, try to use cached target URL
	if cachedURL, ok := s.targetURLs.Load(id); ok {
		targetWSURL = cachedURL.(string)
		conn, _, err := instance.DefaultDialer.DialContext(ctx, "tcp", targetWSURL)
		if err == nil {
			client := instance.NewCDPClient(conn)
			return client, nil
		}
		// Cached URL failed, will re-query
		log.Printf("[GetPageCDPClient] cached target URL failed: %v, re-querying /json", err)
		s.targetURLs.Delete(id)
	}

	// Query /json to get available targets
	jsonURL := strings.Replace(wsURL, "ws://", "http://", 1) + "/json"
	resp, err := http.Get(jsonURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query CDP targets: %w", err)
	}
	defer resp.Body.Close()

	var targets []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("failed to decode CDP targets: %w", err)
	}

	// Find target matching instance ID (first 8 chars) - this is a fallback when no cached URL
	// After browser navigates, title/URL change so we rely on cached URL instead
	targetID := id[:8]
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

	// If still no target URL, fail - don't use random fallback
	if targetWSURL == "" {
		return nil, fmt.Errorf("could not find CDP target for instance %s (browser may have navigated before connection)", id)
	}

	// Cache the target URL for future use
	s.targetURLs.Store(id, targetWSURL)

	conn, _, err := instance.DefaultDialer.DialContext(ctx, "tcp", targetWSURL)
	if err != nil {
		s.targetURLs.Delete(id)
		return nil, fmt.Errorf("failed to dial CDP target: %w", err)
	}

	client := instance.NewCDPClient(conn)
	return client, nil
}

func (s *InstanceService) CloseCDPClient(id string) error {
	if client, ok := s.cdpClients.LoadAndDelete(id); ok {
		return client.(instance.CDPClientInterface).Close()
	}
	return nil
}

// GetCDPClientForTab gets a CDP client for a specific tab
func (s *InstanceService) GetCDPClientForTab(ctx context.Context, instanceID, tabID string) (instance.CDPClientInterface, error) {
	inst, err := s.store.Get(instanceID)
	if err != nil {
		return nil, err
	}

	wsURL := inst.CDPEndpoint
	if !strings.HasPrefix(wsURL, "ws://") {
		wsURL = "ws://" + wsURL
	}

	jsonURL := strings.Replace(wsURL, "ws://", "http://", 1) + "/json"
	resp, err := http.Get(jsonURL)
	if err != nil {
		return nil, fmt.Errorf("failed to query CDP targets: %w", err)
	}
	defer resp.Body.Close()

	var targets []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&targets); err != nil {
		return nil, fmt.Errorf("failed to decode CDP targets: %w", err)
	}

	for _, t := range targets {
		if tid, ok := t["id"].(string); ok && tid == tabID {
			if wsu, ok := t["webSocketDebuggerUrl"].(string); ok {
				conn, _, err := instance.DefaultDialer.DialContext(ctx, "tcp", wsu)
				if err != nil {
					return nil, fmt.Errorf("failed to dial tab CDP: %w", err)
				}
				return instance.NewCDPClient(conn), nil
			}
		}
	}

	return nil, fmt.Errorf("tab not found in CDP targets")
}

// ClearCachedTargetURL removes the cached CDP target URL for an instance.
// Called when the browser restarts or target is closed.
func (s *InstanceService) ClearCachedTargetURL(id string) {
	s.targetURLs.Delete(id)
}

type InstanceConfig = instance.InstanceConfig
type InstanceFilter = instance.InstanceFilter
type InstanceStatus = instance.InstanceStatus

type BrowserInstance struct {
	ID           string                   `json:"id"`
	Name         string                   `json:"name"`
	Status       instance.InstanceStatus  `json:"status"`
	Fingerprint  *fingerprint.Fingerprint `json:"fingerprint"`
	ProxyID      string                   `json:"proxy_id"`
	ProxyURL     string                   `json:"proxy_url"`
	AccountID    string                   `json:"account_id"`
	AccountLabel string                   `json:"account_label"`
	CDPEndpoint  string                   `json:"cdp_endpoint"`
	PID          int                      `json:"pid"`
	Port         int                      `json:"port"`
	UserDataDir  string                   `json:"user_data_dir"`
	Group        string                   `json:"group"`
	Headless     bool                     `json:"headless"`
	StartedAt    string                   `json:"started_at"`
	LastActiveAt string                   `json:"last_active_at"`
	CreatedAt    string                   `json:"created_at"`
}

func ToBrowserInstance(inst *instance.BrowserInstance) *BrowserInstance {
	if inst == nil {
		return nil
	}
	return &BrowserInstance{
		ID:           inst.ID,
		Name:         inst.Name,
		Status:       inst.Status,
		Fingerprint:  inst.Fingerprint,
		ProxyID:      inst.ProxyID,
		ProxyURL:     inst.ProxyURL,
		AccountID:    inst.AccountID,
		CDPEndpoint:  inst.CDPEndpoint,
		PID:          inst.PID,
		Port:         inst.Port,
		UserDataDir:  inst.UserDataDir,
		Group:        inst.Group,
		Headless:     inst.Headless,
		StartedAt:    inst.StartedAt.Format(time.RFC3339Nano),
		LastActiveAt: inst.LastActiveAt.Format(time.RFC3339Nano),
		CreatedAt:    inst.CreatedAt.Format(time.RFC3339Nano),
	}
}

func ToBrowserInstances(list []*instance.BrowserInstance) []*BrowserInstance {
	if len(list) == 0 {
		return []*BrowserInstance{}
	}
	result := make([]*BrowserInstance, 0, len(list))
	for _, inst := range list {
		result = append(result, ToBrowserInstance(inst))
	}
	return result
}

// InstanceStatus constants
const (
	StatusPending  = instance.StatusPending
	StatusStarting = instance.StatusStarting
	StatusRunning  = instance.StatusRunning
	StatusStopping = instance.StatusStopping
	StatusStopped  = instance.StatusStopped
	StatusError    = instance.StatusError
)
