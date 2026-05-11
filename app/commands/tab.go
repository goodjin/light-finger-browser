package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
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
	tabStore      *sqlite.TabStore
	contextStores sync.Map // instanceID -> *ContextStore
}

// NewTabService creates a new TabService with database persistence
func NewTabService(instanceSvc *InstanceService, db *sqlite.DB) *TabService {
	return &TabService{
		instanceSvc: instanceSvc,
		tabStore:    sqlite.NewTabStore(db),
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

	// 5. Store context and tab info in memory
	store := s.getOrCreateContextStore(instanceID)
	store.AddContext(contextId, instanceID, cfg.Fingerprint, cfg.ProxyURL)
	store.AddTab(targetId, contextId, instanceID, url)

	now := time.Now()
	tabInfo := &TabInfo{
		ID:              targetId,
		ContextID:       contextId,
		InstanceID:      instanceID,
		URL:             url,
		CreatedAt:       now,
		LastActiveAt:    now,
		FingerprintSeed: cfg.Fingerprint.Seed,
	}

	// 6. Persist to database
	tabRecord := &sqlite.TabRecord{
		ID:              tabInfo.ID,
		ContextID:       tabInfo.ContextID,
		InstanceID:      tabInfo.InstanceID,
		FingerprintSeed: tabInfo.FingerprintSeed,
		URL:             tabInfo.URL,
		CreatedAt:       tabInfo.CreatedAt.Format(time.RFC3339),
		LastActiveAt:    tabInfo.LastActiveAt.Format(time.RFC3339),
	}
	if err := s.tabStore.Save(tabRecord); err != nil {
		// Log but don't fail - tab is created in browser
		fmt.Printf("warning: failed to persist tab to database: %v\n", err)
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

	// 3. Close the tab via CDP Target.closeTarget
	if err := mainClient.CloseTarget(ctx, tabID); err != nil {
		// Log but don't fail - the tab might already be closed or unavailable
		fmt.Printf("warning: failed to close tab target %s via CDP: %v\n", tabID, err)
	}

	// 4. Remove tab from store
	store.RemoveTab(tabID)

	// 5. Update database - mark tab as closed
	now := time.Now()
	if err := s.tabStore.UpdateClosedAt(tabID, now); err != nil {
		fmt.Printf("warning: failed to update tab closed_at in database: %v\n", err)
	}

	// 6. If context has no more tabs, close the context
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
	// Get in-memory tabs from context store
	store := s.getOrCreateContextStore(instanceID)
	memoryTabs := store.ListTabs()

	// Also load tabs from database that might not be in memory
	// (e.g., tabs created before this session or after app restart)
	dbTabs, err := s.tabStore.ListOpenByInstance(instanceID)
	if err != nil {
		fmt.Printf("warning: failed to load tabs from database: %v\n", err)
		dbTabs = nil
	}

	// Merge tabs from both sources, preferring in-memory state
	tabMap := make(map[string]*TabInfo)
	for _, tab := range memoryTabs {
		tabMap[tab.ID] = tab
	}

	for _, dbTab := range dbTabs {
		if _, exists := tabMap[dbTab.ID]; !exists {
			// Tab exists in DB but not in memory - add it
			createdAt, _ := time.Parse(time.RFC3339, dbTab.CreatedAt)
			lastActiveAt, _ := time.Parse(time.RFC3339, dbTab.LastActiveAt)
			tabMap[dbTab.ID] = &TabInfo{
				ID:              dbTab.ID,
				ContextID:       dbTab.ContextID,
				InstanceID:      dbTab.InstanceID,
				URL:             dbTab.URL,
				Title:           dbTab.Title,
				FingerprintSeed: dbTab.FingerprintSeed,
				CreatedAt:       createdAt,
				LastActiveAt:    lastActiveAt,
			}
		}
	}

	// Convert map to slice
	tabs := make([]*TabInfo, 0, len(tabMap))
	for _, tab := range tabMap {
		tabs = append(tabs, tab)
	}

	return tabs, nil
}

// NavigateTab navigates a specific tab to a URL
func (s *TabService) NavigateTab(ctx context.Context, instanceID, tabID, url string) error {
	// 1. Get tab info from store
	store := s.getOrCreateContextStore(instanceID)
	tab := store.GetTab(tabID)
	if tab == nil {
		return fmt.Errorf("tab not found: %s", tabID)
	}

	// 2. Get CDP client for the specific tab
	tabClient, err := s.instanceSvc.GetCDPClientForTab(ctx, instanceID, tabID)
	if err != nil {
		return fmt.Errorf("failed to connect to tab: %w", err)
	}
	defer tabClient.Close()

	// 3. Navigate the tab
	if err := tabClient.Navigate(ctx, url); err != nil {
		return fmt.Errorf("failed to navigate tab: %w", err)
	}

	// 4. Update tab URL in store and database
	store.mu.Lock()
	if t, ok := store.tabs[tabID]; ok {
		t.URL = url
		t.LastActiveAt = time.Now()
	}
	store.mu.Unlock()

	// 5. Persist URL update to database
	if err := s.tabStore.UpdateURL(tabID, url); err != nil {
		fmt.Printf("warning: failed to update tab URL in database: %v\n", err)
	}

	return nil
}
