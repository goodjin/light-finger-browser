package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// lockFileName is the name of the singleton lock file.
	lockFileName = "instance.lock"

	// lockFileTimeout is how long to wait for the lock.
	lockFileTimeout = 5 * time.Second
)

// SingletonLock provides a file-based lock mechanism to ensure only one instance runs.
type SingletonLock struct {
	lockPath string
	file     *os.File
	mu       sync.Mutex
}

// NewSingletonLock creates a new SingletonLock.
func NewSingletonLock() *SingletonLock {
	appDataDir := filepath.Join(os.Getenv("HOME"), "Library", "Application Support", "fingerbrower")
	lockPath := filepath.Join(appDataDir, lockFileName)
	return &SingletonLock{
		lockPath: lockPath,
		file:     nil,
	}
}

// Acquire tries to acquire the singleton lock.
// Returns true if the lock was acquired, false if another instance is already running.
func (l *SingletonLock) Acquire() (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Ensure directory exists
	dir := filepath.Dir(l.lockPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return false, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Try to create the lock file with exclusive access
	file, err := os.OpenFile(l.lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		// Lock file already exists, check if the process is still running
		if os.IsExist(err) {
			return l.checkExistingLock()
		}
		return false, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Write current PID to lock file
	pid := os.Getpid()
	_, err = file.WriteString(strconv.Itoa(pid))
	if err != nil {
		file.Close()
		os.Remove(l.lockPath)
		return false, fmt.Errorf("failed to write PID to lock file: %w", err)
	}

	// Write timestamp
	timestamp := time.Now().Format(time.RFC3339)
	_, err = file.WriteString("\n" + timestamp)
	if err != nil {
		file.Close()
		os.Remove(l.lockPath)
		return false, fmt.Errorf("failed to write timestamp to lock file: %w", err)
	}

	file.Close()
	l.file = file
	return true, nil
}

// checkExistingLock checks if an existing lock file belongs to a running process.
func (l *SingletonLock) checkExistingLock() (bool, error) {
	// Read the lock file
	data, err := os.ReadFile(l.lockPath)
	if err != nil {
		// Lock file doesn't exist or can't be read, try to acquire
		return true, nil
	}

	// Parse PID
	pidStr := string(data)
	for i, c := range pidStr {
		if c == '\n' || c == '\r' {
			pidStr = pidStr[:i]
			break
		}
	}

	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		// Invalid PID in lock file, remove and acquire
		os.Remove(l.lockPath)
		return true, nil
	}

	// Check if process is still running
	if l.isProcessRunning(pid) {
		return false, nil
	}

	// Process is not running, remove stale lock file and acquire
	os.Remove(l.lockPath)
	return true, nil
}

// isProcessRunning checks if a process with the given PID is running.
func (l *SingletonLock) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Try to signal the process - if it's running, this will succeed or fail differently than "no such process"
	err = process.Signal(syscall.Signal(0))
	if err == nil {
		return true
	}

	// On Unix systems, Signal(0) returns EPERM if process exists but we don't have permission,
	// and returns ESRCH (error "os: process already finished") if process doesn't exist
	errStr := err.Error()
	if errStr == "os: process already finished" || errStr == "no such process" {
		return false
	}

	// EPERM or other errors mean the process exists
	return true
}

// Release releases the singleton lock.
func (l *SingletonLock) Release() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
		l.file = nil
	}

	if _, err := os.Stat(l.lockPath); err == nil {
		if err := os.Remove(l.lockPath); err != nil {
			return fmt.Errorf("failed to remove lock file: %w", err)
		}
	}

	return nil
}

// IsLocked checks if another instance is already running.
// Returns true if another process holds the lock (excluding current process).
func (l *SingletonLock) IsLocked() (bool, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := os.Stat(l.lockPath); os.IsNotExist(err) {
		return false, nil
	}

	return l.checkExistingLock()
}

// IsLockFileExists checks if the lock file exists (regardless of who holds it).
// This is different from IsLocked() which checks if ANOTHER instance is running.
func (l *SingletonLock) IsLockFileExists() bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	if _, err := os.Stat(l.lockPath); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetLockInfo returns information about the current lock holder.
func (l *SingletonLock) GetLockInfo() (*LockInfo, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	data, err := os.ReadFile(l.lockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read lock file: %w", err)
	}

	info := &LockInfo{}
	content := string(data)
	
	// Split by newline
	for i, c := range content {
		if c == '\n' || c == '\r' {
			line := content[:i]
			if info.PID == 0 {
				info.PID, _ = strconv.Atoi(line)
			} else if info.AcquiredAt == "" {
				info.AcquiredAt = line
				break
			}
			// Skip past this line
			for i+1 < len(content) && (content[i+1] == '\n' || content[i+1] == '\r') {
				continue
			}
		}
	}
	
	// If AcquiredAt is still empty and there's remaining content, use the rest
	if info.AcquiredAt == "" && len(content) > 0 {
		info.AcquiredAt = strings.TrimSpace(content)
	}

	return info, nil
}

// LockInfo contains information about a lock holder.
type LockInfo struct {
	PID        int    `json:"pid"`
	AcquiredAt string `json:"acquired_at"`
}
