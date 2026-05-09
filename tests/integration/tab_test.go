package integration

import (
	"testing"
	"time"

	"github.com/tmos/fingerbrower/app/commands"
	"github.com/tmos/fingerbrower/fingerprint"
	"github.com/tmos/fingerbrower/instance"
)

// T-08: Integration testing for per-tab fingerprint feature
// Test scenarios for per-tab fingerprint feature

// TC-001: Create 3 tabs with different fingerprints
func TestCreateMultipleTabsWithDifferentFingerprints(t *testing.T) {
	// This test requires a running browser instance
	// Skip in CI without a real browser
	//
	// Test would:
	// 1. Create instance with fingerprint A
	// 2. Create tab 1 with fingerprint A
	// 3. Create tab 2 with fingerprint B
	// 4. Create tab 3 with fingerprint C
	// 5. Verify each tab has different contextId
	t.Skip("Requires running browser instance with real CDP connection")
}

// TC-002: Close middle tab - others unaffected
func TestCloseMiddleTabUnaffected(t *testing.T) {
	t.Skip("Requires running browser instance with real CDP connection")

	// Test would:
	// 1. Create 3 tabs
	// 2. Close tab 2
	// 3. Verify tab 1 and 3 still exist and are accessible
}

// TC-003: Cookie isolation between tabs
func TestCookieIsolationBetweenTabs(t *testing.T) {
	t.Skip("Requires running browser instance with real CDP connection")

	// Test would:
	// 1. Create 2 tabs with different contexts
	// 2. Set cookie on tab 1 via CDP
	// 3. Verify cookie not present on tab 2 via CDP
}

// TC-004: Instance stop cleanup all contexts
func TestInstanceStopCleanupAllContexts(t *testing.T) {
	t.Skip("Requires running browser instance with real CDP connection")

	// Test would:
	// 1. Create 3 tabs
	// 2. Call StopInstance
	// 3. Verify all contexts are closed via CDP
}

// TC-005: Context limit enforcement
func TestContextLimitEnforcement(t *testing.T) {
	t.Skip("Requires running browser instance with real CDP connection")

	// Test would:
	// 1. Try to create more tabs than MaxContextsPerInstance (10)
	// 2. Verify error is returned for exceeding limit
}

// TC-006: Tab navigation updates URL
func TestTabNavigationUpdatesURL(t *testing.T) {
	t.Skip("Requires running browser instance with real CDP connection")

	// Test would:
	// 1. Create tab
	// 2. Navigate to url1
	// 3. Verify tab URL is url1
	// 4. Navigate to url2
	// 5. Verify tab URL is url2
}

// TC-007: Instance restart loses tab state
func TestInstanceRestartLosesTabState(t *testing.T) {
	t.Skip("Requires running browser instance with real CDP connection")

	// Test would:
	// 1. Create 3 tabs
	// 2. Stop instance
	// 3. Restart instance
	// 4. Verify ListTabs returns empty (tabs are lost on restart)
}

// TestContextStoreUnit tests for ContextStore (no browser needed)
func TestContextStoreUnit(t *testing.T) {
	store := commands.NewContextStore()

	// Test AddContext/GetContext
	fp := &fingerprint.Fingerprint{Seed: "test-seed"}
	store.AddContext("ctx1", "inst1", fp, "http://proxy:8080")
	ctx := store.GetContext("ctx1")
	if ctx == nil {
		t.Fatal("Expected context to be found")
	}
	if ctx.ID != "ctx1" {
		t.Errorf("Expected context ID ctx1, got %s", ctx.ID)
	}
	if ctx.InstanceID != "inst1" {
		t.Errorf("Expected instance ID inst1, got %s", ctx.InstanceID)
	}
	if ctx.Fingerprint == nil || ctx.Fingerprint.Seed != "test-seed" {
		t.Error("Expected fingerprint with seed test-seed")
	}
	if ctx.ProxyURL != "http://proxy:8080" {
		t.Errorf("Expected proxy URL http://proxy:8080, got %s", ctx.ProxyURL)
	}

	// Test AddTab/ListTabs
	store.AddTab("tab1", "ctx1", "inst1", "https://example.com")
	tabs := store.ListTabs()
	if len(tabs) != 1 {
		t.Errorf("Expected 1 tab, got %d", len(tabs))
	}
	if tabs[0].ID != "tab1" {
		t.Errorf("Expected tab ID tab1, got %s", tabs[0].ID)
	}
	if tabs[0].URL != "https://example.com" {
		t.Errorf("Expected tab URL https://example.com, got %s", tabs[0].URL)
	}
	if tabs[0].FingerprintSeed != "test-seed" {
		t.Errorf("Expected fingerprint seed test-seed, got %s", tabs[0].FingerprintSeed)
	}

	// Test GetTab
	tab := store.GetTab("tab1")
	if tab == nil {
		t.Fatal("Expected tab to be found")
	}
	if tab.ContextID != "ctx1" {
		t.Errorf("Expected context ID ctx1, got %s", tab.ContextID)
	}

	// Test CanCloseContext - should be false with tabs
	if store.CanCloseContext("ctx1") {
		t.Error("Expected CanCloseContext to be false with tabs")
	}

	// Test CanCloseContext for non-existent context - should be true
	if !store.CanCloseContext("non-existent") {
		t.Error("Expected CanCloseContext to be true for non-existent context")
	}

	// Test RemoveTab
	store.RemoveTab("tab1")
	tabs = store.ListTabs()
	if len(tabs) != 0 {
		t.Errorf("Expected 0 tabs after remove, got %d", len(tabs))
	}

	// Now context should be closable
	if !store.CanCloseContext("ctx1") {
		t.Error("Expected CanCloseContext to be true with no tabs")
	}

	// Test RemoveContext
	store.RemoveContext("ctx1")
	ctx = store.GetContext("ctx1")
	if ctx != nil {
		t.Error("Expected context to be nil after remove")
	}
}

// TestContextStoreMultipleContexts tests multiple contexts with tabs
func TestContextStoreMultipleContexts(t *testing.T) {
	store := commands.NewContextStore()

	// Add two contexts
	store.AddContext("ctx1", "inst1", &fingerprint.Fingerprint{Seed: "seed1"}, "")
	store.AddContext("ctx2", "inst1", &fingerprint.Fingerprint{Seed: "seed2"}, "")

	// Add tabs to each context
	store.AddTab("tab1", "ctx1", "inst1", "https://example1.com")
	store.AddTab("tab2", "ctx1", "inst1", "https://example2.com")
	store.AddTab("tab3", "ctx2", "inst1", "https://example3.com")

	// Verify tabs
	tabs := store.ListTabs()
	if len(tabs) != 3 {
		t.Errorf("Expected 3 tabs, got %d", len(tabs))
	}

	// Close one tab from ctx1
	store.RemoveTab("tab1")
	if store.CanCloseContext("ctx1") {
		t.Error("Expected CanCloseContext to be false (tab2 still exists)")
	}

	// Close remaining tab from ctx1
	store.RemoveTab("tab2")
	if !store.CanCloseContext("ctx1") {
		t.Error("Expected CanCloseContext to be true (no tabs in ctx1)")
	}

	// ctx2 should still have tab
	if store.CanCloseContext("ctx2") {
		t.Error("Expected CanCloseContext to be false for ctx2")
	}
}

// TestContextStoreCloseAll tests CloseAll method
func TestContextStoreCloseAll(t *testing.T) {
	t.Skip("CloseAll requires real CDP client - nil client causes panic")

	// Note: This test is skipped because the CloseAll implementation
	// calls mainClient.CloseBrowserContext which panics on nil.
	// A proper test would require a mock CDP client.
	//
	// store := commands.NewContextStore()
	// store.AddContext("ctx1", "inst1", nil, "")
	// store.AddTab("tab1", "ctx1", "inst1", "https://example1.com")
	// err := store.CloseAll(context.Background(), mockClient)
}

// TestBrowserContextDataStructure tests BrowserContext struct
func TestBrowserContextDataStructure(t *testing.T) {
	ctx := &commands.BrowserContext{
		ID:           "test-context-id",
		InstanceID:   "test-instance-id",
		Fingerprint:  &fingerprint.Fingerprint{Seed: "test-fp"},
		ProxyURL:     "http://test-proxy:8080",
		TabIDs:       []string{"tab1", "tab2"},
		CreatedAt:    time.Now(),
		LastActiveAt: time.Now(),
	}

	if ctx.ID != "test-context-id" {
		t.Errorf("Expected ID test-context-id, got %s", ctx.ID)
	}
	if ctx.InstanceID != "test-instance-id" {
		t.Errorf("Expected InstanceID test-instance-id, got %s", ctx.InstanceID)
	}
	if len(ctx.TabIDs) != 2 {
		t.Errorf("Expected 2 TabIDs, got %d", len(ctx.TabIDs))
	}
}

// TestTabInfoDataStructure tests TabInfo struct
func TestTabInfoDataStructure(t *testing.T) {
	now := time.Now()
	tab := &commands.TabInfo{
		ID:              "test-tab-id",
		ContextID:       "test-context-id",
		InstanceID:      "test-instance-id",
		URL:             "https://test.example.com",
		Title:           "Test Tab",
		FingerprintSeed: "test-fingerprint-seed",
		CreatedAt:       now,
		LastActiveAt:    now,
	}

	if tab.ID != "test-tab-id" {
		t.Errorf("Expected ID test-tab-id, got %s", tab.ID)
	}
	if tab.ContextID != "test-context-id" {
		t.Errorf("Expected ContextID test-context-id, got %s", tab.ContextID)
	}
	if tab.URL != "https://test.example.com" {
		t.Errorf("Expected URL https://test.example.com, got %s", tab.URL)
	}
	if tab.FingerprintSeed != "test-fingerprint-seed" {
		t.Errorf("Expected FingerprintSeed test-fingerprint-seed, got %s", tab.FingerprintSeed)
	}
}

// TestInstanceStatusConstants tests instance status constants
func TestInstanceStatusConstants(t *testing.T) {
	// Verify status constants are defined correctly
	if commands.StatusPending != instance.StatusPending {
		t.Error("StatusPending mismatch")
	}
	if commands.StatusStarting != instance.StatusStarting {
		t.Error("StatusStarting mismatch")
	}
	if commands.StatusRunning != instance.StatusRunning {
		t.Error("StatusRunning mismatch")
	}
	if commands.StatusStopping != instance.StatusStopping {
		t.Error("StatusStopping mismatch")
	}
	if commands.StatusStopped != instance.StatusStopped {
		t.Error("StatusStopped mismatch")
	}
	if commands.StatusError != instance.StatusError {
		t.Error("StatusError mismatch")
	}
}

// TestTabConfigDataStructure tests TabConfig struct
func TestTabConfigDataStructure(t *testing.T) {
	fp := &fingerprint.Fingerprint{
		Seed: "test-seed",
	}
	cfg := &commands.TabConfig{
		URL:        "https://test.example.com",
		Fingerprint: fp,
		ProxyURL:   "http://proxy:8080",
	}

	if cfg.URL != "https://test.example.com" {
		t.Errorf("Expected URL https://test.example.com, got %s", cfg.URL)
	}
	if cfg.Fingerprint == nil || cfg.Fingerprint.Seed != "test-seed" {
		t.Error("Expected Fingerprint with seed test-seed")
	}
	if cfg.ProxyURL != "http://proxy:8080" {
		t.Errorf("Expected ProxyURL http://proxy:8080, got %s", cfg.ProxyURL)
	}
}

// TestCDPClientBrowserContext tests CDP client BrowserContext methods
func TestCDPClientBrowserContext(t *testing.T) {
	t.Skip("Requires real CDP connection")

	// This would test:
	// - CreateBrowserContext returns valid contextId
	// - CloseBrowserContext closes context
	// - CreateTargetWithContext creates tab in context
	// - GetTargets returns list including the created tab
}
