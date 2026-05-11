package commands

import (
	"context"
	"testing"
	"time"
)

// TestInstanceService_MonitorStartStop tests that the monitor can be started and stopped.
func TestInstanceService_MonitorStartStop(t *testing.T) {
	// Create a minimal instance service for testing
	svc := &InstanceService{
		monitorStopCh: nil,
		monitorDoneCh: nil,
		monitoring:    false,
		restartCount:  0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Test starting the monitor
	svc.StartAutoRecoveryMonitor(ctx)
	time.Sleep(100 * time.Millisecond) // Give goroutine time to start

	svc.monitorMu.RLock()
	monitoring := svc.monitoring
	svc.monitorMu.RUnlock()

	if !monitoring {
		t.Error("Expected monitor to be running after StartAutoRecoveryMonitor")
	}

	// Test stopping the monitor
	svc.StopAutoRecoveryMonitor()
	time.Sleep(100 * time.Millisecond) // Give goroutine time to stop

	svc.monitorMu.RLock()
	monitoring = svc.monitoring
	svc.monitorMu.RUnlock()

	if monitoring {
		t.Error("Expected monitor to be stopped after StopAutoRecoveryMonitor")
	}
}

// TestInstanceService_GetRestartCount tests that restart count is tracked.
func TestInstanceService_GetRestartCount(t *testing.T) {
	svc := &InstanceService{
		restartCount: 0,
	}

	// Initial count should be 0
	if count := svc.GetRestartCount(); count != 0 {
		t.Errorf("Expected initial restart count to be 0, got %d", count)
	}

	// Increment count
	svc.monitorMu.Lock()
	svc.restartCount = 5
	svc.monitorMu.Unlock()

	if count := svc.GetRestartCount(); count != 5 {
		t.Errorf("Expected restart count to be 5, got %d", count)
	}
}

// TestInstanceService_IsProcessRunning tests process detection.
func TestInstanceService_IsProcessRunning(t *testing.T) {
	svc := &InstanceService{}

	// Test with an invalid PID (should return false)
	if svc.isProcessRunning(999999) {
		t.Error("Expected isProcessRunning to return false for invalid PID")
	}

	// Note: Testing with a valid PID requires spawning a process
	// This is a basic sanity check
}

// TestInstanceService_MultipleStartCalls tests that multiple Start calls don't create multiple monitors.
func TestInstanceService_MultipleStartCalls(t *testing.T) {
	svc := &InstanceService{
		monitorStopCh: nil,
		monitorDoneCh: nil,
		monitoring:    false,
		restartCount:  0,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start monitor multiple times
	svc.StartAutoRecoveryMonitor(ctx)
	svc.StartAutoRecoveryMonitor(ctx) // Second call should be no-op
	svc.StartAutoRecoveryMonitor(ctx) // Third call should be no-op

	time.Sleep(100 * time.Millisecond)

	svc.monitorMu.RLock()
	monitoring := svc.monitoring
	svc.monitorMu.RUnlock()

	if !monitoring {
		t.Error("Expected monitor to be running")
	}

	// Stop should only be called once (implicitly, by the first Start)
	svc.StopAutoRecoveryMonitor()
	time.Sleep(100 * time.Millisecond)
}
