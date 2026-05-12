package commands

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/tmos/fingerbrower/instance"
	"github.com/tmos/fingerbrower/storage/sqlite"
)

// WindowStatus represents the status of a fingerprint window
type WindowStatus string

const (
	WindowStatusActive  WindowStatus = "active"
	WindowStatusClosed WindowStatus = "closed"
	WindowStatusError  WindowStatus = "error"
)

// WindowType represents the type of window
type WindowType string

const (
	WindowTypeWindow WindowType = "window"
	WindowTypeTab    WindowType = "tab"
)

// FingerprintWindow represents a fingerprint window data structure
type FingerprintWindow struct {
	ID             string       `json:"id"`
	Country        string       `json:"country"`
	Seed           string       `json:"seed"`
	ContextID      string       `json:"context_id"`
	InstanceID     string       `json:"instance_id"`
	Status         WindowStatus `json:"status"`
	CreatedAt      time.Time    `json:"created_at"`
	LastActiveAt   time.Time    `json:"last_active_at"`
	ClosedAt       *time.Time   `json:"closed_at,omitempty"`
	WindowType     WindowType   `json:"window_type"`
	ParentWindowID string       `json:"parent_window_id,omitempty"`
	Title          string       `json:"title,omitempty"`
	URL            string       `json:"url,omitempty"`
}

// FingerprintWindowService manages fingerprint windows
type FingerprintWindowService struct {
	instanceSvc   *InstanceService
	windowStore   *sqlite.FingerprintWindowStore
	tabStore      *sqlite.TabStore
	fingerprintSvc *FingerprintService
	contextStores sync.Map // instanceID -> *FingerprintContextStore
}

// NewFingerprintWindowService creates a new FingerprintWindowService
func NewFingerprintWindowService(instanceSvc *InstanceService, db *sqlite.DB) *FingerprintWindowService {
	return &FingerprintWindowService{
		instanceSvc:   instanceSvc,
		windowStore:   sqlite.NewFingerprintWindowStore(db),
		tabStore:      sqlite.NewTabStore(db),
		fingerprintSvc: NewFingerprintService(),
	}
}

// FingerprintContextStore holds contexts and windows for an instance
type FingerprintContextStore struct {
	mu       sync.Mutex
	contexts map[string]*FingerprintContext
	windows  map[string]*FingerprintWindow // keyed by window ID
}

// FingerprintContext represents a browser context with its metadata
type FingerprintContext struct {
	ID           string
	InstanceID   string
	Seed         string
	Country      string
	WindowIDs    []string
	CreatedAt    time.Time
	LastActiveAt time.Time
}

// NewFingerprintContextStore creates a new FingerprintContextStore
func NewFingerprintContextStore() *FingerprintContextStore {
	return &FingerprintContextStore{
		contexts: make(map[string]*FingerprintContext),
		windows:  make(map[string]*FingerprintWindow),
	}
}

// getOrCreateContextStore gets or creates a FingerprintContextStore for the given instance
func (s *FingerprintWindowService) getOrCreateContextStore(instanceID string) *FingerprintContextStore {
	if store, ok := s.contextStores.Load(instanceID); ok {
		return store.(*FingerprintContextStore)
	}
	store := NewFingerprintContextStore()
	s.contextStores.Store(instanceID, store)
	return store
}

// CreateWindow creates a new fingerprint window with BrowserContext
func (s *FingerprintWindowService) CreateWindow(ctx context.Context, instanceID string, country string, parentWindowID string) (*FingerprintWindow, error) {
	// 1. Verify instance is running
	inst, err := s.instanceSvc.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	if inst.Status != instance.StatusRunning {
		return nil, fmt.Errorf("instance is not running: %s", inst.Status)
	}

	// 2. Get browser-level CDP client for context operations
	mainClient, err := s.instanceSvc.GetBrowserCDPClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseBrowserCDPClient(instanceID)

	// 3. Generate fingerprint for this window (validates seed works with country)
	seed := uuid.New().String()
	_, err = s.fingerprintSvc.GenerateFingerprint(ctx, seed, country)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fingerprint: %w", err)
	}

	// 4. Determine window type and parent context
	windowType := WindowTypeWindow
	if parentWindowID != "" {
		windowType = WindowTypeTab
	}

	// 5. Create BrowserContext for the window with retry logic
	var contextID string
	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		contextID, err = mainClient.CreateBrowserContext(ctx)
		if err == nil {
			break
		}
		if attempt == maxRetries {
			return nil, fmt.Errorf("failed to create browser context after %d attempts: %w", maxRetries, err)
		}
		log.Printf("warning: CreateBrowserContext attempt %d failed: %v, retrying...", attempt, err)
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}

	// 6. Create window ID
	windowID := uuid.New().String()
	now := time.Now()

	window := &FingerprintWindow{
		ID:           windowID,
		Country:      country,
		Seed:         seed,
		ContextID:    contextID,
		InstanceID:   instanceID,
		Status:       WindowStatusActive,
		CreatedAt:    now,
		LastActiveAt: now,
		WindowType:   windowType,
		ParentWindowID: parentWindowID,
		URL:          "about:blank",
	}

	// 7. Store in memory
	store := s.getOrCreateContextStore(instanceID)
	store.AddContext(contextID, instanceID, seed, country, windowID)
	store.AddWindow(window)

	// 8. Persist to database
	record := &sqlite.FingerprintWindowRecord{
		ID:               windowID,
		Country:          country,
		Seed:             seed,
		ContextID:        contextID,
		InstanceID:       instanceID,
		Status:           string(WindowStatusActive),
		CreatedAt:        now.Format(time.RFC3339),
		LastActiveAt:     now.Format(time.RFC3339),
		WindowType:       string(windowType),
		ParentWindowID:   parentWindowID,
		Title:            "",
		URL:              "about:blank",
	}
	if err := s.windowStore.Save(record); err != nil {
		log.Printf("warning: failed to persist window to database: %v", err)
	}

	log.Printf("[CreateWindow] Created window %s with context %s for instance %s", windowID, contextID, instanceID)
	return window, nil
}

// CreateTabInWindow creates a new tab in an existing window's context
func (s *FingerprintWindowService) CreateTabInWindow(ctx context.Context, windowID string, url string) (*FingerprintWindow, error) {
	// 1. Get window from store - find the window's instance and context
	var instanceID, contextID, country, seed string
	var found bool

	s.contextStores.Range(func(key, value interface{}) bool {
		cs := value.(*FingerprintContextStore)
		cs.mu.Lock()
		defer cs.mu.Unlock()
		if w, ok := cs.windows[windowID]; ok {
			instanceID = w.InstanceID
			contextID = w.ContextID
			country = w.Country
			seed = w.Seed
			found = true
			return false // stop iteration
		}
		return true
	})

	if !found {
		return nil, fmt.Errorf("window not found: %s", windowID)
	}

	if contextID == "" {
		return nil, fmt.Errorf("window context not found: %s", windowID)
	}

	// 2. Verify instance is running
	inst, err := s.instanceSvc.GetInstance(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}
	if inst.Status != instance.StatusRunning {
		return nil, fmt.Errorf("instance is not running: %s", inst.Status)
	}

	// 3. Get browser-level CDP client for context operations
	mainClient, err := s.instanceSvc.GetBrowserCDPClient(ctx, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseBrowserCDPClient(instanceID)

	// 4. Create tab in the existing context with retry logic
	if url == "" {
		url = "about:blank"
	}

	var targetID string
	const maxRetries = 3
	for attempt := 1; attempt <= maxRetries; attempt++ {
		targetID, err = mainClient.CreateTargetWithContext(ctx, url, contextID)
		if err == nil {
			break
		}
		if attempt == maxRetries {
			return nil, fmt.Errorf("failed to create tab after %d attempts: %w", maxRetries, err)
		}
		log.Printf("warning: CreateTargetWithContext attempt %d failed: %v, retrying...", attempt, err)
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}

	// 5. Create window record for the tab
	tabWindowID := uuid.New().String()
	now := time.Now()

	tabWindow := &FingerprintWindow{
		ID:           tabWindowID,
		Country:      country,
		Seed:         seed,
		ContextID:    contextID,
		InstanceID:   instanceID,
		Status:       WindowStatusActive,
		CreatedAt:    now,
		LastActiveAt: now,
		WindowType:   WindowTypeTab,
		ParentWindowID: windowID,
		URL:          url,
	}

	// 6. Store in memory
	foundStore := s.getOrCreateContextStore(instanceID)
	foundStore.AddWindow(tabWindow)

	// Add tab to parent window's tab list
	foundStore.mu.Lock()
	if parent, ok := foundStore.windows[windowID]; ok {
		parent.LastActiveAt = now
	}
	foundStore.mu.Unlock()

	// 7. Persist to database
	record := &sqlite.FingerprintWindowRecord{
		ID:               tabWindowID,
		Country:          country,
		Seed:             seed,
		ContextID:        contextID,
		InstanceID:       instanceID,
		Status:           string(WindowStatusActive),
		CreatedAt:        now.Format(time.RFC3339),
		LastActiveAt:     now.Format(time.RFC3339),
		WindowType:       string(WindowTypeTab),
		ParentWindowID:   windowID,
		Title:            "",
		URL:              url,
	}
	if err := s.windowStore.Save(record); err != nil {
		log.Printf("warning: failed to persist tab window to database: %v", err)
	}

	log.Printf("[CreateTabInWindow] Created tab %s with target %s for window %s", tabWindowID, targetID, windowID)
	return tabWindow, nil
}

// DeleteWindow closes and deletes a fingerprint window
func (s *FingerprintWindowService) DeleteWindow(ctx context.Context, windowID string) error {
	// 1. Find window in stores
	var instanceID, contextID string
	windowFound := false

	s.contextStores.Range(func(key, value interface{}) bool {
		cs := value.(*FingerprintContextStore)
		cs.mu.Lock()
		defer cs.mu.Unlock()

		if w, ok := cs.windows[windowID]; ok {
			instanceID = w.InstanceID
			contextID = w.ContextID
			windowFound = true
			return false // stop iteration
		}
		return true
	})

	if !windowFound {
		// Try to load from database
		record, err := s.windowStore.Get(windowID)
		if err != nil {
			return fmt.Errorf("window not found: %s", windowID)
		}
		instanceID = record.InstanceID
		contextID = record.ContextID
	}

	// 2. Get browser-level CDP client for context operations
	mainClient, err := s.instanceSvc.GetBrowserCDPClient(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseBrowserCDPClient(instanceID)

	// 3. Close BrowserContext
	if contextID != "" {
		if err := mainClient.CloseBrowserContext(ctx, contextID); err != nil {
			log.Printf("warning: failed to close browser context %s: %v", contextID, err)
		}
	}

	// 4. Remove from memory store
	s.contextStores.Range(func(key, value interface{}) bool {
		cs := value.(*FingerprintContextStore)
		cs.mu.Lock()
		defer cs.mu.Unlock()
		delete(cs.windows, windowID)
		return true
	})

	// 5. Update database
	now := time.Now()
	if err := s.windowStore.UpdateClosedAt(windowID, now); err != nil {
		log.Printf("warning: failed to update window closed_at in database: %v", err)
	}

	log.Printf("[DeleteWindow] Deleted window %s", windowID)
	return nil
}

// GetWindow retrieves a fingerprint window by ID
func (s *FingerprintWindowService) GetWindow(ctx context.Context, windowID string) (*FingerprintWindow, error) {
	// Try to get from memory first
	var window *FingerprintWindow
	found := false

	s.contextStores.Range(func(key, value interface{}) bool {
		cs := value.(*FingerprintContextStore)
		cs.mu.Lock()
		defer cs.mu.Unlock()

		if w, ok := cs.windows[windowID]; ok {
			window = w
			found = true
			return false
		}
		return true
	})

	if found {
		return window, nil
	}

	// Load from database
	record, err := s.windowStore.Get(windowID)
	if err != nil {
		return nil, fmt.Errorf("window not found: %s", windowID)
	}

	return s.recordToWindow(record), nil
}

// ListWindows returns all fingerprint windows
func (s *FingerprintWindowService) ListWindows(ctx context.Context, instanceID string, includeClosed bool) ([]*FingerprintWindow, error) {
	// Get from database
	records, err := s.windowStore.List(instanceID, includeClosed)
	if err != nil {
		return nil, fmt.Errorf("failed to list windows: %w", err)
	}

	windows := make([]*FingerprintWindow, 0, len(records))
	for _, record := range records {
		windows = append(windows, s.recordToWindow(record))
	}

	return windows, nil
}

// ListOpenWindows returns all open fingerprint windows for an instance
func (s *FingerprintWindowService) ListOpenWindows(ctx context.Context, instanceID string) ([]*FingerprintWindow, error) {
	return s.ListWindows(ctx, instanceID, false)
}

// UpdateWindowURL updates the URL of a fingerprint window
func (s *FingerprintWindowService) UpdateWindowURL(ctx context.Context, windowID string, url string) error {
	// Update in memory
	s.contextStores.Range(func(key, value interface{}) bool {
		cs := value.(*FingerprintContextStore)
		cs.mu.Lock()
		defer cs.mu.Unlock()

		if w, ok := cs.windows[windowID]; ok {
			w.URL = url
			w.LastActiveAt = time.Now()
			return false
		}
		return true
	})

	// Update in database
	if err := s.windowStore.UpdateURL(windowID, url); err != nil {
		return fmt.Errorf("failed to update window URL: %w", err)
	}

	return nil
}

// UpdateWindowTitle updates the title of a fingerprint window
func (s *FingerprintWindowService) UpdateWindowTitle(ctx context.Context, windowID string, title string) error {
	// Update in memory
	s.contextStores.Range(func(key, value interface{}) bool {
		cs := value.(*FingerprintContextStore)
		cs.mu.Lock()
		defer cs.mu.Unlock()

		if w, ok := cs.windows[windowID]; ok {
			w.Title = title
			w.LastActiveAt = time.Now()
			return false
		}
		return true
	})

	// Update in database
	if err := s.windowStore.UpdateTitle(windowID, title); err != nil {
		return fmt.Errorf("failed to update window title: %w", err)
	}

	return nil
}

// GetWindowByContextID retrieves a window by its browser context ID
func (s *FingerprintWindowService) GetWindowByContextID(ctx context.Context, contextID string) (*FingerprintWindow, error) {
	record, err := s.windowStore.GetByContextID(contextID)
	if err != nil {
		return nil, fmt.Errorf("window not found for context: %s", contextID)
	}
	return s.recordToWindow(record), nil
}

// CloseAllWindowsForInstance closes all windows for an instance
func (s *FingerprintWindowService) CloseAllWindowsForInstance(ctx context.Context, instanceID string) error {
	// Get browser-level CDP client for context operations
	mainClient, err := s.instanceSvc.GetBrowserCDPClient(ctx, instanceID)
	if err != nil {
		return fmt.Errorf("failed to connect to instance: %w", err)
	}
	defer s.instanceSvc.CloseBrowserCDPClient(instanceID)

	// Get all context IDs
	contextIDs, err := s.windowStore.GetActiveContextIDs(instanceID)
	if err != nil {
		log.Printf("warning: failed to get context IDs: %v", err)
	}

	// Close all contexts
	for _, contextID := range contextIDs {
		if err := mainClient.CloseBrowserContext(ctx, contextID); err != nil {
			log.Printf("warning: failed to close context %s: %v", contextID, err)
		}
	}

	// Clear memory store
	if store, ok := s.contextStores.LoadAndDelete(instanceID); ok {
		cs := store.(*FingerprintContextStore)
		cs.mu.Lock()
		defer cs.mu.Unlock()

		now := time.Now()
		for windowID := range cs.windows {
			if err := s.windowStore.UpdateClosedAt(windowID, now); err != nil {
				log.Printf("warning: failed to update window closed_at: %v", err)
			}
		}
	}

	log.Printf("[CloseAllWindowsForInstance] Closed all windows for instance %s", instanceID)
	return nil
}

// AddContext adds a context to the store
func (s *FingerprintContextStore) AddContext(contextID, instanceID, seed, country, windowID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.contexts[contextID] = &FingerprintContext{
		ID:           contextID,
		InstanceID:   instanceID,
		Seed:         seed,
		Country:      country,
		WindowIDs:    []string{windowID},
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}
}

// AddWindow adds a window to the store
func (s *FingerprintContextStore) AddWindow(window *FingerprintWindow) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.windows[window.ID] = window

	// Add window to context's window list
	if ctx, ok := s.contexts[window.ContextID]; ok {
		ctx.WindowIDs = append(ctx.WindowIDs, window.ID)
		ctx.LastActiveAt = time.Now()
	}
}

// RemoveWindow removes a window from the store
func (s *FingerprintContextStore) RemoveWindow(windowID string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if window, ok := s.windows[windowID]; ok {
		// Remove from context's window list
		if ctx, ok := s.contexts[window.ContextID]; ok {
			newWindowIDs := make([]string, 0, len(ctx.WindowIDs))
			for _, id := range ctx.WindowIDs {
				if id != windowID {
					newWindowIDs = append(newWindowIDs, id)
				}
			}
			ctx.WindowIDs = newWindowIDs
		}
		delete(s.windows, windowID)
	}
}

// GetWindow returns a window by ID
func (s *FingerprintContextStore) GetWindow(windowID string) *FingerprintWindow {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.windows[windowID]
}

// ListWindows returns all windows
func (s *FingerprintContextStore) ListWindows() []*FingerprintWindow {
	s.mu.Lock()
	defer s.mu.Unlock()

	windows := make([]*FingerprintWindow, 0, len(s.windows))
	for _, window := range s.windows {
		windows = append(windows, window)
	}
	return windows
}

// GetContext returns a context by ID
func (s *FingerprintContextStore) GetContext(contextID string) *FingerprintContext {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.contexts[contextID]
}

// CanCloseContext checks if a context can be closed (no windows remaining)
func (s *FingerprintContextStore) CanCloseContext(contextID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if ctx, ok := s.contexts[contextID]; ok {
		return len(ctx.WindowIDs) == 0
	}
	return true
}

// recordToWindow converts a database record to a FingerprintWindow
func (s *FingerprintWindowService) recordToWindow(record *sqlite.FingerprintWindowRecord) *FingerprintWindow {
	window := &FingerprintWindow{
		ID:             record.ID,
		Country:        record.Country,
		Seed:           record.Seed,
		ContextID:      record.ContextID,
		InstanceID:    record.InstanceID,
		Status:         WindowStatus(record.Status),
		WindowType:     WindowType(record.WindowType),
		ParentWindowID: record.ParentWindowID,
		Title:          record.Title,
		URL:            record.URL,
	}

	if record.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, record.CreatedAt); err == nil {
			window.CreatedAt = t
		}
	}
	if record.LastActiveAt != "" {
		if t, err := time.Parse(time.RFC3339, record.LastActiveAt); err == nil {
			window.LastActiveAt = t
		}
	}
	if record.ClosedAt.Valid {
		window.ClosedAt = &record.ClosedAt.Time
	}

	return window
}
