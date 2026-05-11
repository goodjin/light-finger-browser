package commands

import (
	"os"
	"testing"
	"time"

	"github.com/tmos/fingerbrower/storage/sqlite"
)

// TestTabService_LogAccess tests the LogAccess method of TabService
func TestTabService_LogAccess(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_tab_access_log_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := sqlite.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	// Create TabService with AccessLogStore
	tabSvc := &TabService{
		accessLogStore: sqlite.NewAccessLogStore(db),
	}

	// Test LogAccess
	tabID := "test-tab-123"
	url := "https://example.com/page1"
	title := "Example Page"

	err = tabSvc.LogAccess(tabID, url, title)
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}

	// Verify the log was saved
	logs, err := tabSvc.accessLogStore.ListByTab(tabID)
	if err != nil {
		t.Fatalf("ListByTab failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	log := logs[0]
	if log.TabID != tabID {
		t.Errorf("TabID mismatch: got %v, want %v", log.TabID, tabID)
	}
	if log.URL != url {
		t.Errorf("URL mismatch: got %v, want %v", log.URL, url)
	}
	if log.Title != title {
		t.Errorf("Title mismatch: got %v, want %v", log.Title, title)
	}
	if log.VisitedAt == "" {
		t.Error("VisitedAt should not be empty")
	}

	// Verify VisitedAt is a valid RFC3339 time
	_, err = time.Parse(time.RFC3339, log.VisitedAt)
	if err != nil {
		t.Errorf("VisitedAt is not valid RFC3339: %v", err)
	}
}

// TestTabService_LogAccess_MultipleEntries tests logging multiple entries for the same tab
func TestTabService_LogAccess_MultipleEntries(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_tab_access_log_multi_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := sqlite.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	tabSvc := &TabService{
		accessLogStore: sqlite.NewAccessLogStore(db),
	}

	tabID := "test-tab-456"

	// Log multiple URLs
	urls := []string{
		"https://example.com/page1",
		"https://example.com/page2",
		"https://example.com/page3",
	}

	for _, url := range urls {
		err := tabSvc.LogAccess(tabID, url, "")
		if err != nil {
			t.Fatalf("LogAccess failed for %s: %v", url, err)
		}
	}

	// Verify all logs were saved
	logs, err := tabSvc.accessLogStore.ListByTab(tabID)
	if err != nil {
		t.Fatalf("ListByTab failed: %v", err)
	}

	if len(logs) != len(urls) {
		t.Errorf("Expected %d log entries, got %d", len(urls), len(logs))
	}
}

// TestTabService_LogAccess_EmptyTitle tests logging with empty title
func TestTabService_LogAccess_EmptyTitle(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_tab_access_log_empty_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := sqlite.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	tabSvc := &TabService{
		accessLogStore: sqlite.NewAccessLogStore(db),
	}

	// Log with empty title
	err = tabSvc.LogAccess("tab-789", "https://example.com", "")
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}

	// Verify the log was saved with empty title
	logs, err := tabSvc.accessLogStore.ListByTab("tab-789")
	if err != nil {
		t.Fatalf("ListByTab failed: %v", err)
	}

	if len(logs) != 1 {
		t.Fatalf("Expected 1 log entry, got %d", len(logs))
	}

	if logs[0].Title != "" {
		t.Errorf("Expected empty title, got %v", logs[0].Title)
	}
}

// TestTabService_LogAccess_UniqueIDs tests that each log entry has a unique ID
func TestTabService_LogAccess_UniqueIDs(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_tab_access_log_unique_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := sqlite.NewDB(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	tabSvc := &TabService{
		accessLogStore: sqlite.NewAccessLogStore(db),
	}

	// Log multiple entries
	tabID := "tab-unique"
	ids := make(map[string]bool)

	for i := 0; i < 10; i++ {
		err := tabSvc.LogAccess(tabID, "https://example.com", "")
		if err != nil {
			t.Fatalf("LogAccess failed: %v", err)
		}
	}

	// Get all logs and verify unique IDs
	logs, err := tabSvc.accessLogStore.ListByTab(tabID)
	if err != nil {
		t.Fatalf("ListByTab failed: %v", err)
	}

	if len(logs) != 10 {
		t.Fatalf("Expected 10 log entries, got %d", len(logs))
	}

	for _, log := range logs {
		if ids[log.ID] {
			t.Errorf("Duplicate ID found: %s", log.ID)
		}
		ids[log.ID] = true
	}
}
