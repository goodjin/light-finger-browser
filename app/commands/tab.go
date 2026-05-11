package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

// TabConfig contains configuration for creating a new tab
type TabConfig struct {
	URL        string
	Fingerprint *fingerprint.Fingerprint
	ProxyURL   string
	Country    string // Country code for fingerprint (e.g., "US", "UK", "DE")
}

// TabService manages browser tabs within instances
type TabService struct {
	instanceSvc    *InstanceService
	tabStore       *sqlite.TabStore
	accessLogStore *sqlite.AccessLogStore
	contextStores  sync.Map // instanceID -> *ContextStore
	fingerprintSvc *FingerprintService
}

// NewTabService creates a new TabService with database persistence
func NewTabService(instanceSvc *InstanceService, db *sqlite.DB) *TabService {
	return &TabService{
		instanceSvc:    instanceSvc,
		tabStore:       sqlite.NewTabStore(db),
		accessLogStore: sqlite.NewAccessLogStore(db),
		fingerprintSvc: NewFingerprintService(),
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
		ID:                 tabInfo.ID,
		ContextID:          tabInfo.ContextID,
		InstanceID:         tabInfo.InstanceID,
		FingerprintSeed:    tabInfo.FingerprintSeed,
		FingerprintCountry: cfg.Country,
		URL:                tabInfo.URL,
		CreatedAt:          tabInfo.CreatedAt.Format(time.RFC3339),
		LastActiveAt:       tabInfo.LastActiveAt.Format(time.RFC3339),
	}
	if err := s.tabStore.Save(tabRecord); err != nil {
		// Log but don't fail - tab is created in browser
		fmt.Printf("warning: failed to persist tab to database: %v\n", err)
	}

	// 7. Record access log for tab creation (AL-001)
	// Only record if URL is not about:blank (meaning user specified a URL)
	if url != "about:blank" {
		if err := s.LogAccess(tabInfo.ID, url, ""); err != nil {
			fmt.Printf("warning: failed to record access log: %v\n", err)
		}
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
				ID:                 dbTab.ID,
				ContextID:          dbTab.ContextID,
				InstanceID:         dbTab.InstanceID,
				URL:               dbTab.URL,
				Title:             dbTab.Title,
				FingerprintSeed:    dbTab.FingerprintSeed,
				FingerprintCountry: dbTab.FingerprintCountry,
				CreatedAt:         createdAt,
				LastActiveAt:      lastActiveAt,
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

	// 6. Record access log (AL-001: NavigateTab calls LogAccess to insert access_logs)
	// AL-004: LogAccess records tab_id, url, title, visited_at, duration_ms
	if err := s.LogAccess(tabID, url, ""); err != nil {
		fmt.Printf("warning: failed to record access log: %v\n", err)
	}

	return nil
}

// LogAccess records a navigation event in the access logs
// AL-001: Called by NavigateTab to insert access log
// AL-004: Records tab_id, url, title, visited_at, duration_ms
func (s *TabService) LogAccess(tabID, url, title string) error {
	log := &sqlite.AccessLogRecord{
		ID:        uuid.New().String(),
		TabID:     tabID,
		URL:       url,
		Title:     title,
		VisitedAt: time.Now().Format(time.RFC3339),
	}
	return s.accessLogStore.Save(log)
}

// ReopenTab reopens a closed tab with the same fingerprint configuration
// TM-005: ReopenTab API accepts tabID, gets fingerprint_seed and url from database,
// creates new BrowserContext with saved fingerprint config, creates new tab and updates database
func (s *TabService) ReopenTab(ctx context.Context, tabID string) (*TabInfo, error) {
	// 1. Get the closed tab record from database
	closedTab, err := s.tabStore.Get(tabID)
	if err != nil {
		return nil, fmt.Errorf("failed to get tab record: %w", err)
	}

	// Verify this is a closed tab
	if !closedTab.ClosedAt.Valid {
		return nil, fmt.Errorf("tab %s is not closed", tabID)
	}

	// 2. Get instance ID from the tab record
	instanceID := closedTab.InstanceID
	if instanceID == "" {
		// Try to get singleton instance ID if not set
		instanceID = s.instanceSvc.GetSingletonInstanceID()
		if instanceID == "" {
			return nil, fmt.Errorf("instance ID not found for tab and no singleton instance available")
		}
	}

	// 3. Verify instance is running
	inst, err := s.instanceSvc.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	if inst.Status != instance.StatusRunning {
		return nil, fmt.Errorf("instance is not running: %s", inst.Status)
	}

	// 4. Get main CDP client
	mainClient, err := s.instanceSvc.GetCDPClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseCDPClient(instanceID)

	// 5. Create new isolated browser context
	contextId, err := mainClient.CreateBrowserContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create browser context: %w", err)
	}

	// 6. Regenerate fingerprint from saved seed and country
	var fp *fingerprint.Fingerprint
	country := closedTab.FingerprintCountry
	if country == "" {
		country = "US" // Default country if not set
	}

	if closedTab.FingerprintSeed != "" {
		fp, err = s.fingerprintSvc.GenerateFingerprint(ctx, closedTab.FingerprintSeed, country)
		if err != nil {
			// Fallback: generate random fingerprint
			fmt.Printf("warning: failed to regenerate fingerprint from seed: %v, generating random\n", err)
			fp, err = s.fingerprintSvc.GenerateRandomFingerprint(ctx, country)
			if err != nil {
				_ = mainClient.CloseBrowserContext(ctx, contextId)
				return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
			}
		}
	} else {
		// No seed, generate random fingerprint
		fp, err = s.fingerprintSvc.GenerateRandomFingerprint(ctx, country)
		if err != nil {
			_ = mainClient.CloseBrowserContext(ctx, contextId)
			return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
		}
	}

	// 7. Create new tab within that context (use about:blank or original URL)
	url := closedTab.URL
	if url == "" {
		url = "about:blank"
	}

	newTabID, err := mainClient.CreateTargetWithContext(ctx, url, contextId)
	if err != nil {
		_ = mainClient.CloseBrowserContext(ctx, contextId)
		return nil, fmt.Errorf("failed to create tab: %w", err)
	}

	// 8. Store context and tab info in memory
	store := s.getOrCreateContextStore(instanceID)
	store.AddContext(contextId, instanceID, fp, "")
	store.AddTab(newTabID, contextId, instanceID, url)

	now := time.Now()
	tabInfo := &TabInfo{
		ID:              newTabID,
		ContextID:       contextId,
		InstanceID:      instanceID,
		URL:             url,
		CreatedAt:       now,
		LastActiveAt:    now,
		FingerprintSeed: fp.Seed,
	}

	// 9. Persist new tab to database
	newTabRecord := &sqlite.TabRecord{
		ID:                 tabInfo.ID,
		ContextID:          tabInfo.ContextID,
		InstanceID:         tabInfo.InstanceID,
		FingerprintSeed:    tabInfo.FingerprintSeed,
		FingerprintCountry: country,
		URL:                tabInfo.URL,
		CreatedAt:          tabInfo.CreatedAt.Format(time.RFC3339),
		LastActiveAt:       tabInfo.LastActiveAt.Format(time.RFC3339),
	}
	if err := s.tabStore.Save(newTabRecord); err != nil {
		// Log but don't fail - tab is created in browser
		fmt.Printf("warning: failed to persist reopened tab to database: %v\n", err)
	}

	return tabInfo, nil
}
