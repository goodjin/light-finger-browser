package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
)

// TabConfig contains configuration for creating a new tab
type TabConfig struct {
	URL        string
	Fingerprint *fingerprint.Fingerprint
	ProxyURL   string
}

// TabService manages browser tabs within instances
type TabService struct {
	instanceSvc   *InstanceService
	contextStores sync.Map // instanceID -> *ContextStore
}

// NewTabService creates a new TabService
func NewTabService(instanceSvc *InstanceService) *TabService {
	return &TabService{
		instanceSvc: instanceSvc,
	}
}

// getOrCreateContextStore gets or creates a ContextStore for the given instance
func (s *TabService) getOrCreateContextStore(instanceID string) *ContextStore {
	if store, ok := s.contextStores.Load(instanceID); ok {
		return store.(*ContextStore)
	}
	store := NewContextStore()
	s.contextStores.Store(instanceID, store)
	return store
}

// CreateTab creates a new tab with the specified fingerprint in an existing instance
func (s *TabService) CreateTab(ctx context.Context, instanceID string, cfg *TabConfig) (*TabInfo, error) {
	// 1. Verify instance is running
	inst, err := s.instanceSvc.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	if inst.Status != instance.StatusRunning {
		return nil, fmt.Errorf("instance is not running: %s", inst.Status)
	}

	// 2. Get main CDP client
	mainClient, err := s.instanceSvc.GetCDPClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseCDPClient(instanceID)

	// 3. Create isolated browser context
	contextId, err := mainClient.CreateBrowserContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}

	// 4. Create tab within that context
	url := cfg.URL
	if url == "" {
		url = "about:blank"
	}

	targetId, err := mainClient.CreateTargetWithContext(ctx, url, contextId)
	if err != nil {
		_ = mainClient.CloseBrowserContext(ctx, contextId)
		return nil, fmt.Errorf("failed to create tab: %w", err)
	}

	// 5. Store context and tab info
	store := s.getOrCreateContextStore(instanceID)
	store.AddContext(contextId, instanceID, cfg.Fingerprint, cfg.ProxyURL)
	store.AddTab(targetId, contextId, instanceID, url)

	tabInfo := &TabInfo{
		ID:              targetId,
		ContextID:       contextId,
		InstanceID:      instanceID,
		URL:             url,
		CreatedAt:       time.Now(),
		LastActiveAt:    time.Now(),
		FingerprintSeed: cfg.Fingerprint.Seed,
	}

	return tabInfo, nil
}

// CloseTab closes a specific tab and its context if no other tabs exist
func (s *TabService) CloseTab(ctx context.Context, instanceID, tabID string) error {
	// 1. Get instance CDP client
	mainClient, err := s.instanceSvc.GetCDPClient(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseCDPClient(instanceID)

	// 2. Get tab info from store
	store := s.getOrCreateContextStore(instanceID)
	tab := store.GetTab(tabID)
	if tab == nil {
		return fmt.Errorf("tab not found: %s", tabID)
	}

	contextId := tab.ContextID

	// 3. Close the tab via CDP (close target)
	// Note: In CDP, closing a target (tab) will also close its context if it's the last tab
	// We'll close the target, then try to close the context
	// The actual close of tab target is handled by the browser when we navigate away or close

	// For now, we just remove from our store. The actual tab closing happens when
	// the browser process terminates or we explicitly close the context.
	// TODO: Add CloseTarget method to CDPClient if needed

	// 4. Remove tab from store
	store.RemoveTab(tabID)

	// 5. If context has no more tabs, close the context
	if store.CanCloseContext(contextId) {
		if err := mainClient.CloseBrowserContext(ctx, contextId); err != nil {
			// Log but don't fail - the context might already be closed
			fmt.Printf("warning: failed to close context %s: %v\n", contextId, err)
		}
		store.RemoveContext(contextId)
	}

	return nil
}

// ListTabs returns all tabs in an instance
func (s *TabService) ListTabs(ctx context.Context, instanceID string) ([]*TabInfo, error) {
	store := s.getOrCreateContextStore(instanceID)
	return store.ListTabs(), nil
}

// NavigateTab navigates a specific tab to a URL
func (s *TabService) NavigateTab(ctx context.Context, instanceID, tabID, url string) error {
	// 1. Get tab info from store
	store := s.getOrCreateContextStore(instanceID)
	tab := store.GetTab(tabID)
	if tab == nil {
		return fmt.Errorf("tab not found: %s", tabID)
	}

	// 2. Get CDP client for the instance
	tabClient, err := s.instanceSvc.GetCDPClient(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseCDPClient(instanceID)

	// 3. Navigate the tab
	if err := tabClient.Navigate(ctx, url); err != nil {
		return fmt.Errorf("failed to navigate tab: %w", err)
	}

	// 4. Update tab URL in store
	store.mu.Lock()
	if t, ok := store.tabs[tabID]; ok {
		t.URL = url
		t.LastActiveAt = time.Now()
	}
	store.mu.Unlock()

	return nil
}
