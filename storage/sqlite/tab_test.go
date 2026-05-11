package sqlite

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func TestTabStore_SaveAndGet(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "tabtest*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	store := NewTabStore(db)

	// Test Save and Get
	now := time.Now().Format(time.RFC3339)
	tab := &TabRecord{
		ID:              "test-tab-1",
		ContextID:       "test-context-1",
		InstanceID:      "test-instance-1",
		FingerprintSeed: "test-seed-123",
		URL:             "https://example.com",
		Title:           "Example",
		CreatedAt:       now,
		LastActiveAt:    now,
	}

	if err := store.Save(tab); err != nil {
		t.Fatalf("Failed to save tab: %v", err)
	}

	saved, err := store.Get("test-tab-1")
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}

	if saved.ID != tab.ID {
		t.Errorf("Expected ID %s, got %s", tab.ID, saved.ID)
	}
	if saved.ContextID != tab.ContextID {
		t.Errorf("Expected ContextID %s, got %s", tab.ContextID, saved.ContextID)
	}
	if saved.InstanceID != tab.InstanceID {
		t.Errorf("Expected InstanceID %s, got %s", tab.InstanceID, saved.InstanceID)
	}
	if saved.FingerprintSeed != tab.FingerprintSeed {
		t.Errorf("Expected FingerprintSeed %s, got %s", tab.FingerprintSeed, saved.FingerprintSeed)
	}
	if saved.URL != tab.URL {
		t.Errorf("Expected URL %s, got %s", tab.URL, saved.URL)
	}
}

func TestTabStore_ListOpenByInstance(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "tabtest*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	store := NewTabStore(db)

	now := time.Now().Format(time.RFC3339)

	// Create tabs for instance 1
	tab1 := &TabRecord{
		ID:              "tab-1",
		ContextID:       "ctx-1",
		InstanceID:      "instance-1",
		FingerprintSeed: "seed-1",
		URL:             "https://example1.com",
		CreatedAt:       now,
		LastActiveAt:    now,
	}
	tab2 := &TabRecord{
		ID:              "tab-2",
		ContextID:       "ctx-2",
		InstanceID:      "instance-1",
		FingerprintSeed: "seed-2",
		URL:             "https://example2.com",
		CreatedAt:       now,
		LastActiveAt:    now,
	}

	// Create tab for instance 2
	tab3 := &TabRecord{
		ID:              "tab-3",
		ContextID:       "ctx-3",
		InstanceID:      "instance-2",
		FingerprintSeed: "seed-3",
		URL:             "https://example3.com",
		CreatedAt:       now,
		LastActiveAt:    now,
	}

	// Create closed tab for instance 1
	tab4 := &TabRecord{
		ID:              "tab-4",
		ContextID:       "ctx-4",
		InstanceID:      "instance-1",
		FingerprintSeed: "seed-4",
		URL:             "https://example4.com",
		CreatedAt:       now,
		LastActiveAt:    now,
		ClosedAt:        nullTime(time.Now()),
	}

	if err := store.Save(tab1); err != nil {
		t.Fatalf("Failed to save tab1: %v", err)
	}
	if err := store.Save(tab2); err != nil {
		t.Fatalf("Failed to save tab2: %v", err)
	}
	if err := store.Save(tab3); err != nil {
		t.Fatalf("Failed to save tab3: %v", err)
	}
	if err := store.Save(tab4); err != nil {
		t.Fatalf("Failed to save tab4: %v", err)
	}

	// Test ListOpenByInstance for instance-1 (should return 2 open tabs)
	openTabs, err := store.ListOpenByInstance("instance-1")
	if err != nil {
		t.Fatalf("Failed to list open tabs: %v", err)
	}
	if len(openTabs) != 2 {
		t.Errorf("Expected 2 open tabs for instance-1, got %d", len(openTabs))
	}

	// Test ListOpenByInstance for instance-2 (should return 1 open tab)
	openTabs2, err := store.ListOpenByInstance("instance-2")
	if err != nil {
		t.Fatalf("Failed to list open tabs: %v", err)
	}
	if len(openTabs2) != 1 {
		t.Errorf("Expected 1 open tab for instance-2, got %d", len(openTabs2))
	}

	// Test ListAllOpen (should return 3 open tabs)
	allOpen, err := store.ListAllOpen()
	if err != nil {
		t.Fatalf("Failed to list all open tabs: %v", err)
	}
	if len(allOpen) != 3 {
		t.Errorf("Expected 3 open tabs total, got %d", len(allOpen))
	}
}

func TestTabStore_UpdateClosedAt(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "tabtest*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	store := NewTabStore(db)

	now := time.Now().Format(time.RFC3339)
	tab := &TabRecord{
		ID:              "tab-to-close",
		ContextID:       "ctx-1",
		InstanceID:      "instance-1",
		FingerprintSeed: "seed-1",
		URL:             "https://example.com",
		CreatedAt:       now,
		LastActiveAt:    now,
	}

	if err := store.Save(tab); err != nil {
		t.Fatalf("Failed to save tab: %v", err)
	}

	// Verify tab is not closed
	saved, err := store.Get("tab-to-close")
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}
	if saved.ClosedAt.Valid {
		t.Error("Tab should not be closed initially")
	}

	// Close the tab
	closedAt := time.Now().Add(time.Hour)
	if err := store.UpdateClosedAt("tab-to-close", closedAt); err != nil {
		t.Fatalf("Failed to update closed_at: %v", err)
	}

	// Verify tab is now closed
	saved, err = store.Get("tab-to-close")
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}
	if !saved.ClosedAt.Valid {
		t.Error("Tab should be closed")
	}
	// Note: Due to RFC3339 formatting (no monotonic clock), we just verify it's close in time
	if saved.ClosedAt.Time.Unix() != closedAt.Unix() {
		t.Errorf("Expected closed_at around %v, got %v", closedAt.Unix(), saved.ClosedAt.Time.Unix())
	}

	// Verify tab is no longer in open list
	openTabs, err := store.ListOpenByInstance("instance-1")
	if err != nil {
		t.Fatalf("Failed to list open tabs: %v", err)
	}
	if len(openTabs) != 0 {
		t.Errorf("Expected 0 open tabs after closing, got %d", len(openTabs))
	}
}

func TestTabStore_UpdateURL(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "tabtest*.db")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("Failed to create DB: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	store := NewTabStore(db)

	now := time.Now().Format(time.RFC3339)
	tab := &TabRecord{
		ID:              "tab-url-test",
		ContextID:       "ctx-1",
		InstanceID:      "instance-1",
		FingerprintSeed: "seed-1",
		URL:             "https://example.com",
		CreatedAt:       now,
		LastActiveAt:    now,
	}

	if err := store.Save(tab); err != nil {
		t.Fatalf("Failed to save tab: %v", err)
	}

	// Update URL
	newURL := "https://newexample.com"
	if err := store.UpdateURL("tab-url-test", newURL); err != nil {
		t.Fatalf("Failed to update URL: %v", err)
	}

	// Verify URL was updated
	saved, err := store.Get("tab-url-test")
	if err != nil {
		t.Fatalf("Failed to get tab: %v", err)
	}
	if saved.URL != newURL {
		t.Errorf("Expected URL %s, got %s", newURL, saved.URL)
	}
}

// Helper function to create sql.NullTime
func nullTime(t time.Time) sql.NullTime {
	return sql.NullTime{Time: t, Valid: true}
}
