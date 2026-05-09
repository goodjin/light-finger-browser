package commands

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
)

// BrowserContext represents an isolated browser context
type BrowserContext struct {
	ID           string
	InstanceID   string
	Fingerprint  *fingerprint.Fingerprint
	ProxyURL     string
	TabIDs       []string
	CreatedAt    time.Time
	LastActiveAt time.Time
}

// TabInfo represents a browser tab
type TabInfo struct {
	ID              string
	ContextID       string
	InstanceID      string
	URL             string
	Title           string
	FingerprintSeed string
	CreatedAt       time.Time
	LastActiveAt    time.Time
}

// ContextStore holds all contexts for an instance
type ContextStore struct {
	mu       sync.Mutex
	contexts map[string]*BrowserContext
	tabs     map[string]*TabInfo
}

// NewContextStore creates a new ContextStore
func NewContextStore() *ContextStore {
	return &ContextStore{
		contexts: make(map[string]*BrowserContext),
		tabs:     make(map[string]*TabInfo),
	}
}

// AddContext adds a new browser context to the store
func (s *ContextStore) AddContext(contextId string, instanceID string, fp *fingerprint.Fingerprint, proxyURL string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.contexts[contextId] = &BrowserContext{
		ID:           contextId,
		InstanceID:   instanceID,
		Fingerprint:  fp,
		ProxyURL:     proxyURL,
		TabIDs:       []string{},
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
}

// AddTab adds a new tab to the store
func (s *ContextStore) AddTab(tabId string, contextId string, instanceID string, url string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	fpSeed := ""
	if ctx, ok := s.contexts[contextId]; ok && ctx.Fingerprint != nil {
		fpSeed = ctx.Fingerprint.Seed
	}

	s.tabs[tabId] = &TabInfo{
		ID:              tabId,
		ContextID:       contextId,
		InstanceID:      instanceID,
		URL:             url,
		CreatedAt:       time.Now(),
		LastActiveAt:    time.Now(),
		FingerprintSeed: fpSeed,
	}

	// Add tab to context's TabIDs
	if ctx, ok := s.contexts[contextId]; ok {
		ctx.TabIDs = append(ctx.TabIDs, tabId)
		ctx.LastActiveAt = time.Now()
	}
}

// RemoveTab removes a tab from the store
func (s *ContextStore) RemoveTab(tabId string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if tab, ok := s.tabs[tabId]; ok {
		// Remove tab from context's TabIDs
		if ctx, ok := s.contexts[tab.ContextID]; ok {
			newTabIDs := make([]string, 0, len(ctx.TabIDs))
			for _, id := range ctx.TabIDs {
				if id != tabId {
					newTabIDs = append(newTabIDs, id)
				}
			}
			ctx.TabIDs = newTabIDs
		}
		delete(s.tabs, tabId)
	}
}

// RemoveContext removes a context from the store
func (s *ContextStore) RemoveContext(contextId string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.contexts, contextId)
}

// GetTab returns a tab by ID
func (s *ContextStore) GetTab(tabId string) *TabInfo {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.tabs[tabId]
}

// ListTabs returns all tabs
func (s *ContextStore) ListTabs() []*TabInfo {
	s.mu.Lock()
	defer s.mu.Unlock()

	tabs := make([]*TabInfo, 0, len(s.tabs))
	for _, tab := range s.tabs {
		tabs = append(tabs, tab)
	}
	return tabs
}

// GetContext returns a context by ID
func (s *ContextStore) GetContext(contextId string) *BrowserContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.contexts[contextId]
}

// CanCloseContext checks if a context can be closed (no tabs remaining)
func (s *ContextStore) CanCloseContext(contextId string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.contexts[contextId]; ok {
		return len(ctx.TabIDs) == 0
	}
	return true
}

// CloseAll closes all contexts and tabs when instance stops
func (s *ContextStore) CloseAll(ctx context.Context, mainClient instance.CDPClientInterface) error {
	s.mu.Lock()
	// Copy contexts map to avoid holding lock while closing
	contexts := make(map[string]*BrowserContext)
	for k, v := range s.contexts {
		contexts[k] = v
	}
	tabs := make(map[string]*TabInfo)
	for k, v := range s.tabs {
		tabs[k] = v
	}
	s.contexts = make(map[string]*BrowserContext)
	s.tabs = make(map[string]*TabInfo)
	s.mu.Unlock()

	var lastErr error

	// Close all tabs first
	for tabId := range tabs {
		// Tabs are closed by closing their context, no need to close individually
		_ = tabId
	}

	// Then close all contexts
	for contextId := range contexts {
		if err := mainClient.CloseBrowserContext(ctx, contextId); err != nil {
			lastErr = fmt.Errorf("failed to close context %s: %w", contextId, err)
		}
	}

	return lastErr
}
