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
	manager     browserRuntimeManager
	store       *sqlite.InstanceStore
	cdpClients  sync.Map // instanceID -> CDPClient
	targetURLs  sync.Map // instanceID -> string (cached webSocketDebuggerUrl)
}

func NewInstanceService(db *sqlite.DB) *InstanceService {
	manager := newBrowserRuntimeManager(db)
	store := sqlite.NewInstanceStore(db)
	return &InstanceService{
		manager: manager,
		store:   store,
	}
}

func (s *InstanceService) CreateInstance(ctx context.Context, cfg *InstanceConfig) (*instance.BrowserInstance, error) {
	return s.manager.Start(ctx, cfg)
}

func (s *InstanceService) DestroyInstance(ctx context.Context, id string) error {
	return s.manager.Stop(ctx, id)
}

func (s *InstanceService) StopInstance(ctx context.Context, id string) error {
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

	var targetWSURL string

	// First, try to use cached target URL
	if cachedURL, ok := s.targetURLs.Load(id); ok {
		targetWSURL = cachedURL.(string)
		conn, _, err := instance.DefaultDialer.DialContext(ctx, "tcp", targetWSURL)
		if err == nil {
			client := instance.NewCDPClient(conn)
			s.cdpClients.Store(id, client)
			return client, nil
		}
		// Cached URL failed, will re-query
		log.Printf("[GetCDPClient] cached target URL failed: %v, re-querying /json", err)
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
	s.cdpClients.Store(id, client)
	return client, nil
}

func (s *InstanceService) CloseCDPClient(id string) error {
	if client, ok := s.cdpClients.LoadAndDelete(id); ok {
		return client.(instance.CDPClientInterface).Close()
	}
	return nil
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
