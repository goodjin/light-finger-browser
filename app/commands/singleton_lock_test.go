package commands

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSingletonLockAcquireRelease(t *testing.T) {
	// Create a temp lock directory
	tmpDir := filepath.Join(os.TempDir(), "fingerbrower-lock-test")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	// Create first lock
	lock1 := &SingletonLock{
		lockPath: filepath.Join(tmpDir, "instance.lock"),
	}

	// Acquire first lock
	acquired, err := lock1.Acquire()
	if err != nil {
		t.Fatalf("First Acquire() failed: %v", err)
	}
	if !acquired {
		t.Error("Expected first lock to be acquired")
	}

	// Try to acquire second lock (should fail)
	lock2 := &SingletonLock{
		lockPath: filepath.Join(tmpDir, "instance.lock"),
	}

	acquired2, err := lock2.Acquire()
	if err != nil {
		t.Fatalf("Second Acquire() failed: %v", err)
	}
	if acquired2 {
		t.Error("Expected second lock acquisition to fail")
	}

	// Release first lock
	err = lock1.Release()
	if err != nil {
		t.Fatalf("Release() failed: %v", err)
	}

	// Now lock2 should be able to acquire
	acquired3, err := lock2.Acquire()
	if err != nil {
		t.Fatalf("Third Acquire() failed: %v", err)
	}
	if !acquired3 {
		t.Error("Expected third lock to be acquired after release")
	}

	// Cleanup
	lock2.Release()
}

func TestSingletonLockIsLocked(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "fingerbrower-lock-test2")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	lock := &SingletonLock{
		lockPath: filepath.Join(tmpDir, "instance.lock"),
	}

	// Initially no lock file
	if lock.IsLockFileExists() {
		t.Error("Expected no lock file initially")
	}

	// Acquire lock
	acquired, err := lock.Acquire()
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}
	if !acquired {
		t.Error("Expected lock to be acquired")
	}

	// Lock file should now exist
	if !lock.IsLockFileExists() {
		t.Error("Expected lock file to exist after acquire")
	}

	// IsLocked returns whether ANOTHER instance is running (not us)
	// Since we're the same process, this returns false
	locked, err := lock.IsLocked()
	if err != nil {
		t.Fatalf("IsLocked() failed: %v", err)
	}
	if locked {
		t.Error("IsLocked should return false for current process")
	}

	// Release
	lock.Release()

	// Lock file should be gone
	if lock.IsLockFileExists() {
		t.Error("Expected no lock file after release")
	}
}

func TestSingletonLockGetLockInfo(t *testing.T) {
	tmpDir := filepath.Join(os.TempDir(), "fingerbrower-lock-test3")
	os.MkdirAll(tmpDir, 0755)
	defer os.RemoveAll(tmpDir)

	lock := &SingletonLock{
		lockPath: filepath.Join(tmpDir, "instance.lock"),
	}

	// Acquire lock
	_, err := lock.Acquire()
	if err != nil {
		t.Fatalf("Acquire() failed: %v", err)
	}

	// Get lock info
	info, err := lock.GetLockInfo()
	if err != nil {
		t.Fatalf("GetLockInfo() failed: %v", err)
	}

	if info.PID != os.Getpid() {
		t.Errorf("Expected PID %d, got %d", os.Getpid(), info.PID)
	}

	if info.AcquiredAt == "" {
		t.Error("Expected non-empty AcquiredAt")
	}

	// Cleanup
	lock.Release()
}
