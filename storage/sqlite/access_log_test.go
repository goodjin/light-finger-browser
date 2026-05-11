package sqlite

import (
	"os"
	"testing"
	"time"
)

func TestAccessLogStore_SaveAndGet(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	store := NewAccessLogStore(db)

	// Create test log
	log := &AccessLogRecord{
		ID:         "test-log-1",
		TabID:      "test-tab-1",
		URL:        "https://example.com",
		Title:      "Example Domain",
		VisitedAt:  time.Now().Format(time.RFC3339),
		DurationMs: 5000,
	}

	// Save
	if err := store.Save(log); err != nil {
		t.Fatal(err)
	}

	// Get
	saved, err := store.Get(log.ID)
	if err != nil {
		t.Fatal(err)
	}

	if saved.ID != log.ID {
		t.Errorf("ID mismatch: got %v, want %v", saved.ID, log.ID)
	}
	if saved.TabID != log.TabID {
		t.Errorf("TabID mismatch: got %v, want %v", saved.TabID, log.TabID)
	}
	if saved.URL != log.URL {
		t.Errorf("URL mismatch: got %v, want %v", saved.URL, log.URL)
	}
	if saved.Title != log.Title {
		t.Errorf("Title mismatch: got %v, want %v", saved.Title, log.Title)
	}
}

func TestAccessLogStore_ListByTab(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_*.db")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	db, err := NewDB(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.Migrate(); err != nil {
		t.Fatal(err)
	}

	store := NewAccessLogStore(db)
	tabID := "tab-123"

	// Save multiple logs for different tabs
	logs := []*AccessLogRecord{
		{ID: "log-1", TabID: tabID, URL: "https://example.com/1", VisitedAt: time.Now().Format(time.RFC3339)},
		{ID: "log-2", TabID: tabID, URL: "https://example.com/2", VisitedAt: time.Now().Format(time.RFC3339)},
		{ID: "log-3", TabID: "other-tab", URL: "https://other.com", VisitedAt: time.Now().Format(time.RFC3339)},
	}

	for _, log := range logs {
		if err := store.Save(log); err != nil {
			t.Fatal(err)
		}
	}

	// List by tab
	tabLogs, err := store.ListByTab(tabID)
	if err != nil {
		t.Fatal(err)
	}

	if len(tabLogs) != 2 {
		t.Errorf("Expected 2 logs for tab %s, got %d", tabID, len(tabLogs))
	}

	// List all
	allLogs, err := store.ListAll()
	if err != nil {
		t.Fatal(err)
	}

	if len(allLogs) != 3 {
		t.Errorf("Expected 3 total logs, got %d", len(allLogs))
	}
}
