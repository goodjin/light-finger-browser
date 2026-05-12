package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"syscall"
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

	// SI-005: Auto-recovery monitoring
	monitorStopCh chan struct{}     // Channel to signal monitor to stop
	monitorDoneCh chan struct{}     // Channel to signal monitor has stopped
	monitorMu     sync.RWMutex      // Protects monitor state
	monitoring    bool              // Whether monitoring is active
	restartCount  int              // Number of times instance was restarted
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

// StartAutoRecoveryMonitor starts the monitoring goroutine for SI-005.
// This goroutine periodically checks the browser process health and restarts if crashed.
func (s *InstanceService) StartAutoRecoveryMonitor(ctx context.Context) {
	s.monitorMu.Lock()
	if s.monitoring {
		s.monitorMu.Unlock()
		return
	}
	s.monitorStopCh = make(chan struct{})
	s.monitorDoneCh = make(chan struct{})
	s.monitoring = true
	s.monitorMu.Unlock()

	go s.monitorLoop(ctx)
	log.Println("[StartAutoRecoveryMonitor] Auto-recovery monitor started")
}

// StopAutoRecoveryMonitor stops the monitoring goroutine.
func (s *InstanceService) StopAutoRecoveryMonitor() {
	s.monitorMu.Lock()
	if !s.monitoring {
		s.monitorMu.Unlock()
		return
	}
	s.monitoring = false
	s.monitorMu.Unlock()

	close(s.monitorStopCh)
	<-s.monitorDoneCh
	log.Println("[StopAutoRecoveryMonitor] Auto-recovery monitor stopped")
}

// monitorLoop is the main loop for the monitoring goroutine.
// It implements SI-005: periodically checks browser process health and restarts on crash.
func (s *InstanceService) monitorLoop(ctx context.Context) {
	defer close(s.monitorDoneCh)

	// Check interval: every 10 seconds
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-s.monitorStopCh:
			log.Println("[monitorLoop] Received stop signal, exiting")
			return
		case <-ctx.Done():
			log.Println("[monitorLoop] Context cancelled, exiting")
			return
		case <-ticker.C:
			s.checkAndRecoverSingleton(ctx)
		}
	}
}

// checkAndRecoverSingleton checks if the singleton instance is still running and recovers if crashed.
// This implements SI-005: Detect crash and restart.
func (s *InstanceService) checkAndRecoverSingleton(ctx context.Context) {
	s.singletonMu.RLock()
	inst := s.singleton
	needRestart := inst != nil && inst.Status == instance.StatusRunning
	s.singletonMu.RUnlock()

	if !needRestart {
		return
	}

	// Check if the process is still running
	if !s.isProcessRunning(inst.PID) {
		log.Printf("[checkAndRecoverSingleton] Browser process (PID %d) is not running, initiating recovery", inst.PID)
		s.recoverSingleton(ctx)
		return
	}

	// Also check if the CDP port is still accessible
	if !s.isPortAccessible(inst.Port) {
		log.Printf("[checkAndRecoverSingleton] Browser port %d is not accessible, initiating recovery", inst.Port)
		s.recoverSingleton(ctx)
		return
	}
}

// isProcessRunning checks if the process with the given PID is still running.
func (s *InstanceService) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Printf("[isProcessRunning] FindProcess error: %v", err)
		return false
	}

	// Signal 0 check - see if process exists
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	// ESRCH means process doesn't exist
	errStr := err.Error()
	if errStr == "os: process already finished" || strings.Contains(errStr, "no such process") {
		return false
	}

	// EPERM or other errors mean the process exists
	return true
}

// isPortAccessible checks if the CDP port is still accepting connections.
func (s *InstanceService) isPortAccessible(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

// recoverSingleton attempts to restart the singleton instance after a crash.
// This implements SI-005: Call Restart() to recover.
func (s *InstanceService) recoverSingleton(ctx context.Context) {
	s.singletonMu.Lock()
	defer s.singletonMu.Unlock()

	// Double-check singleton state
	if s.singleton == nil {
		return
	}

	// Don't restart if already in a restart sequence or stopped intentionally
	if s.singleton.Status == instance.StatusStopped {
		return
	}

	log.Printf("[recoverSingleton] Attempting to restart singleton instance: %s", s.singleton.ID)
	s.singleton.Status = instance.StatusError

	// Get config for restart
	cfg := &instance.InstanceConfig{
		Name:      s.singleton.Name,
		Headless:  s.singleton.Headless,
		Fingerprint: s.singleton.Fingerprint,
	}

	// Clean up CDP clients before restart
	s.cleanupAllCDPClients()

	// Try to restart
	inst, err := s.manager.Restart(ctx, s.singleton, cfg)
	if err != nil {
		log.Printf("[recoverSingleton] Failed to restart singleton instance: %v", err)
		s.singleton.Status = instance.StatusError
		return
	}

	s.singleton = inst
	s.restartCount++
	log.Printf("[recoverSingleton] Successfully recovered singleton instance: %s on port %d (restart count: %d)", inst.ID, inst.Port, s.restartCount)
}

// GetRestartCount returns the number of times the singleton instance was restarted.
func (s *InstanceService) GetRestartCount() int {
	s.monitorMu.RLock()
	defer s.monitorMu.RUnlock()
	return s.restartCount
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

	// SI-005: Start auto-recovery monitor after singleton is created
	s.StartAutoRecoveryMonitor(ctx)

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

// StopSingleton stops the singleton instance and its auto-recovery monitor.
func (s *InstanceService) StopSingleton(ctx context.Context) error {
	// SI-005: Stop auto-recovery monitor first
	s.StopAutoRecoveryMonitor()

	s.singletonMu.Lock()
	defer s.singletonMu.Unlock()

	if s.singleton == nil {
		return nil
	}

	// Clean up all CDP clients before stopping
	s.cleanupAllCDPClients()

	err := s.manager.Stop(ctx, s.singleton.ID)
	if err != nil {
		return err
	}

	s.singleton.Status = instance.StatusStopped
	return nil
}

// RestartSingleton restarts the singleton instance.
// This can be used for manual restart or by the auto-recovery monitor.
func (s *InstanceService) RestartSingleton(ctx context.Context) (*instance.BrowserInstance, error) {
	s.singletonMu.Lock()
	defer s.singletonMu.Unlock()

	if s.singleton == nil {
		return nil, fmt.Errorf("no singleton instance to restart")
	}

	log.Printf("[RestartSingleton] Restarting singleton instance: %s", s.singleton.ID)

	// Get config for restart
	cfg := &instance.InstanceConfig{
		Name:        s.singleton.Name,
		Headless:    s.singleton.Headless,
		Fingerprint: s.singleton.Fingerprint,
	}

	// Clean up CDP clients before restart
	s.cleanupAllCDPClients()

	// Try to restart using manager.Restart
	inst, err := s.manager.Restart(ctx, s.singleton, cfg)
	if err != nil {
		log.Printf("[RestartSingleton] Failed to restart singleton instance: %v", err)
		s.singleton.Status = instance.StatusError
		return nil, fmt.Errorf("failed to restart singleton instance: %w", err)
	}

	s.singleton = inst
	s.restartCount++
	log.Printf("[RestartSingleton] Successfully restarted singleton instance: %s on port %d (restart count: %d)", inst.ID, inst.Port, s.restartCount)

	// Restart monitoring
	s.StartAutoRecoveryMonitor(ctx)

	return inst, nil
}

// cleanupAllCDPClients closes all CDP clients in the cdpClients map.
func (s *InstanceService) cleanupAllCDPClients() {
	s.cdpClients.Range(func(k, v interface{}) bool {
		if client, ok := v.(instance.CDPClientInterface); ok {
			if err := client.Close(); err != nil {
				log.Printf("[cleanupAllCDPClients] error closing CDP client for %v: %v", k, err)
			} else {
				log.Printf("[cleanupAllCDPClients] closed CDP client for %v", k)
			}
		}
		return true
	})
	s.cdpClients.Clear()
	// Also clear cached target URLs
	s.targetURLs.Clear()
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

	// Clean up all CDP clients for this instance (including page-level clients)
	s.CloseCDPClient(id)
	// Clear cached target URL for this instance
	s.ClearCachedTargetURL(id)

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

	// Validate the connection is still alive by sending a simple command
	// This catches cases where the browser process crashed but the WebSocket is still connected
	if !client.IsConnected(ctx) {
		client.Close()
		return nil, fmt.Errorf("CDP connection is not alive: browser process may have crashed")
	}

	// Use separate key for browser-level CDP client to avoid overwriting page-level clients
	s.cdpClients.Store(id+":browser", client)
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
			// Store client for cleanup - use page-level key
			s.cdpClients.Store(id+":page", client)
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

	// Validate the connection is still alive by sending a simple command
	// This catches cases where the browser process crashed but the WebSocket is still connected
	if !client.IsConnected(ctx) {
		client.Close()
		s.targetURLs.Delete(id)
		return nil, fmt.Errorf("CDP connection is not alive: browser process may have crashed")
	}

	// Store client for cleanup - use page-level key
	s.cdpClients.Store(id+":page", client)
	return client, nil
}

func (s *InstanceService) CloseCDPClient(id string) error {
	// Close both browser-level and page-level CDP clients for this instance
	var lastErr error
	if client, ok := s.cdpClients.LoadAndDelete(id + ":browser"); ok {
		if err := client.(instance.CDPClientInterface).Close(); err != nil {
			lastErr = err
			log.Printf("[CloseCDPClient] error closing browser-level CDP client: %v", err)
		}
	}
	if client, ok := s.cdpClients.LoadAndDelete(id + ":page"); ok {
		if err := client.(instance.CDPClientInterface).Close(); err != nil {
			lastErr = err
			log.Printf("[CloseCDPClient] error closing page-level CDP client: %v", err)
		}
	}
	return lastErr
}

// GetBrowserCDPClient returns a browser-level CDP client for the instance.
// Use this for browser-level operations like createBrowserContext, createTarget, closeBrowserContext.
func (s *InstanceService) GetBrowserCDPClient(ctx context.Context, id string) (instance.CDPClientInterface, error) {
	return s.GetCDPClient(ctx, id)
}

// CloseBrowserCDPClient closes only the browser-level CDP client for the instance.
func (s *InstanceService) CloseBrowserCDPClient(id string) error {
	var lastErr error
	if client, ok := s.cdpClients.LoadAndDelete(id + ":browser"); ok {
		if err := client.(instance.CDPClientInterface).Close(); err != nil {
			lastErr = err
			log.Printf("[CloseBrowserCDPClient] error closing browser-level CDP client: %v", err)
		}
	}
	return lastErr
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
