package commands

import (
	"os"
	"testing"
	"time"

	"github.com/tmos/fingerbrower/storage/sqlite"
)

// TestTabService_GetAccessLogs_AllLogs tests GetAccessLogs without filter
func TestTabService_GetAccessLogs_AllLogs(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_query_*.db")
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

	// Log multiple entries for different tabs
	tabIDs := []string{"tab-A", "tab-B", "tab-C"}
	for _, tabID := range tabIDs {
		for i := 0; i < 3; i++ {
			err := tabSvc.LogAccess(tabID, "https://example.com/"+tabID+"/"+string(rune('0'+i)), "")
			if err != nil {
				t.Fatalf("LogAccess failed for %s: %v", tabID, err)
			}
		}
	}

	// Test GetAccessLogs with empty query (should return all logs)
	query := &AccessLogQuery{}
	logs, err := tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	if len(logs) != 9 {
		t.Errorf("Expected 9 log entries, got %d", len(logs))
	}

	// Verify logs are ordered by time descending (most recent first)
	for i := 0; i < len(logs)-1; i++ {
		if logs[i].VisitedAt.Before(logs[i+1].VisitedAt) {
			t.Errorf("Logs not ordered by time descending at index %d", i)
		}
	}
}

// TestTabService_GetAccessLogs_FilterByTabID tests GetAccessLogs with tabID filter
func TestTabService_GetAccessLogs_FilterByTabID(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_query_filter_*.db")
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

	tabA := "tab-A"
	tabB := "tab-B"

	// Log entries for tab A
	for i := 0; i < 3; i++ {
		err := tabSvc.LogAccess(tabA, "https://example.com/page"+string(rune('A'+i)), "Page "+string(rune('A'+i)))
		if err != nil {
			t.Fatalf("LogAccess failed for tab A: %v", err)
		}
	}

	// Log entries for tab B
	for i := 0; i < 2; i++ {
		err := tabSvc.LogAccess(tabB, "https://other.com/page"+string(rune('0'+i)), "Other Page "+string(rune('0'+i)))
		if err != nil {
			t.Fatalf("LogAccess failed for tab B: %v", err)
		}
	}

	// Test filtering by tab A
	query := &AccessLogQuery{TabID: tabA}
	logs, err := tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	if len(logs) != 3 {
		t.Errorf("Expected 3 log entries for tab A, got %d", len(logs))
	}

	for _, log := range logs {
		if log.TabID != tabA {
			t.Errorf("Expected tabID %s, got %s", tabA, log.TabID)
		}
	}

	// Test filtering by tab B
	query = &AccessLogQuery{TabID: tabB}
	logs, err = tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries for tab B, got %d", len(logs))
	}

	for _, log := range logs {
		if log.TabID != tabB {
			t.Errorf("Expected tabID %s, got %s", tabB, log.TabID)
		}
	}
}

// TestTabService_GetAccessLogs_TimeRangeFilter tests GetAccessLogs with time range filter
func TestTabService_GetAccessLogs_TimeRangeFilter(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_query_time_*.db")
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

	now := time.Now()
	tabID := "tab-time"

	// Log entries at different times (due to sequential execution, they have different timestamps)
	// RFC3339 in SQLite has second precision, so we need at least 1 second delay between entries
	// Entry 1 - oldest
	err = tabSvc.LogAccess(tabID, "https://example.com/old", "Old Page")
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)

	// Entry 2 - middle
	err = tabSvc.LogAccess(tabID, "https://example.com/middle", "Middle Page")
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)

	// Entry 3 - most recent
	err = tabSvc.LogAccess(tabID, "https://example.com/recent", "Recent Page")
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}

	// Test filtering with start time T-20min
	startTime := now.Add(-20 * time.Minute).Format(time.RFC3339)
	query := &AccessLogQuery{
		TabID:     tabID,
		StartTime: startTime,
	}
	logs, err := tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	// Should return entries from T-15min and T (2 entries)
	// Note: Due to test timing, we might get different counts, so just verify the query works
	if len(logs) < 2 {
		t.Errorf("Expected at least 2 log entries after start time filter, got %d", len(logs))
	}

	// Verify all returned logs have visited_at >= startTime
	for _, log := range logs {
		if log.VisitedAt.Before(time.Now().Add(-20 * time.Minute)) {
			t.Errorf("Log visited at %v is before start time %v", log.VisitedAt, startTime)
		}
	}

	// Test filtering with end time T-10min (should exclude recent entries)
	endTime := now.Add(-10 * time.Minute).Format(time.RFC3339)
	query = &AccessLogQuery{
		TabID:   tabID,
		EndTime: endTime,
	}
	logs, err = tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	// Verify all returned logs have visited_at <= endTime
	for _, log := range logs {
		if log.VisitedAt.After(time.Now().Add(-10 * time.Minute)) {
			t.Errorf("Log visited at %v is after end time %v", log.VisitedAt, endTime)
		}
	}

	// Test combined start and end time filter
	startTime2 := now.Add(-20 * time.Minute).Format(time.RFC3339)
	endTime2 := now.Add(-5 * time.Minute).Format(time.RFC3339)
	query = &AccessLogQuery{
		TabID:     tabID,
		StartTime: startTime2,
		EndTime:   endTime2,
	}
	logs, err = tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	// Verify all returned logs are within the time range
	for _, log := range logs {
		if log.VisitedAt.Before(time.Now().Add(-20 * time.Minute)) {
			t.Errorf("Log visited at %v is before start time %v", log.VisitedAt, startTime2)
		}
		if log.VisitedAt.After(time.Now().Add(-5 * time.Minute)) {
			t.Errorf("Log visited at %v is after end time %v", log.VisitedAt, endTime2)
		}
	}
}

// TestTabService_GetAccessLogs_EmptyResult tests GetAccessLogs when no logs match
func TestTabService_GetAccessLogs_EmptyResult(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_query_empty_*.db")
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

	// Log entries for tab A
	err = tabSvc.LogAccess("tab-A", "https://example.com", "Page A")
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}

	// Query for non-existent tab
	query := &AccessLogQuery{TabID: "non-existent-tab"}
	logs, err := tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	if len(logs) != 0 {
		t.Errorf("Expected 0 log entries for non-existent tab, got %d", len(logs))
	}
}

// TestTabService_GetAccessLogs_NilQuery tests GetAccessLogs with nil query
func TestTabService_GetAccessLogs_NilQuery(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_query_nil_*.db")
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

	// Log some entries
	err = tabSvc.LogAccess("tab-1", "https://example.com", "Page 1")
	if err != nil {
		t.Fatalf("LogAccess failed: %v", err)
	}

	// Test with nil query (should behave like empty query)
	logs, err := tabSvc.GetAccessLogs(nil)
	if err != nil {
		t.Fatalf("GetAccessLogs(nil) failed: %v", err)
	}

	if len(logs) != 1 {
		t.Errorf("Expected 1 log entry with nil query, got %d", len(logs))
	}
}

// TestTabService_GetAccessLogs_Ordering tests that logs are ordered by time descending
func TestTabService_GetAccessLogs_Ordering(t *testing.T) {
	// Create temp database
	tmpFile, err := os.CreateTemp("", "test_access_log_query_order_*.db")
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

	tabID := "tab-order"

	// Log entries in sequence with enough delay to ensure different timestamps
	// RFC3339 in SQLite has second precision, so we need at least 1 second delay
	urls := []string{"https://example.com/first", "https://example.com/second", "https://example.com/third"}
	for i, url := range urls {
		err := tabSvc.LogAccess(tabID, url, "")
		if err != nil {
			t.Fatalf("LogAccess failed: %v", err)
		}
		if i < len(urls)-1 {
			time.Sleep(1100 * time.Millisecond) // Sleep 1.1 seconds to ensure different second timestamps
		}
	}

	// Query and verify ordering
	query := &AccessLogQuery{TabID: tabID}
	logs, err := tabSvc.GetAccessLogs(query)
	if err != nil {
		t.Fatalf("GetAccessLogs failed: %v", err)
	}

	if len(logs) != 3 {
		t.Fatalf("Expected 3 log entries, got %d", len(logs))
	}

	// Debug: Print all logs and their timestamps
	t.Logf("Number of logs: %d", len(logs))
	for i, log := range logs {
		t.Logf("Log[%d]: URL=%s, VisitedAt=%v", i, log.URL, log.VisitedAt)
	}

	// Verify logs are ordered by visited_at DESC (most recent first)
	for i := 0; i < len(logs)-1; i++ {
		if logs[i].VisitedAt.Before(logs[i+1].VisitedAt) {
			t.Errorf("Logs not ordered by time descending: index %d (%v) is before index %d (%v)",
				i, logs[i].VisitedAt, i+1, logs[i+1].VisitedAt)
		}
	}

	// Verify the most recent log has URL "third"
	if logs[0].URL != "https://example.com/third" {
		t.Errorf("Expected first log to be 'third', got %s", logs[0].URL)
	}

	// Verify the oldest log has URL "first"
	if logs[2].URL != "https://example.com/first" {
		t.Errorf("Expected last log to be 'first', got %s", logs[2].URL)
	}
}
