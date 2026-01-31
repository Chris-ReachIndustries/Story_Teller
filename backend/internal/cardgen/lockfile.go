package cardgen

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

const (
	// LockFileName is the name of the lock file
	LockFileName = ".cardgen.lock"
	// StaleLockDuration is how long before a lock is considered stale
	StaleLockDuration = 30 * time.Minute
)

// LockFile prevents concurrent generation
type LockFile struct {
	path string
}

// LockInfo contains lock metadata
type LockInfo struct {
	PID       int       `json:"pid"`
	Hostname  string    `json:"hostname"`
	StartedAt time.Time `json:"startedAt"`
}

// NewLockFile creates a lock file manager
func NewLockFile(cardsPath string) *LockFile {
	return &LockFile{
		path: cardsPath + "/" + LockFileName,
	}
}

// Acquire attempts to acquire the lock
func (l *LockFile) Acquire() error {
	// Check if lock exists and is stale
	if info, err := l.readLock(); err == nil {
		if time.Since(info.StartedAt) < StaleLockDuration {
			return fmt.Errorf("lock held by PID %d on %s since %s",
				info.PID, info.Hostname, info.StartedAt.Format(time.RFC3339))
		}
		// Stale lock, remove it
		os.Remove(l.path)
	}

	// Create new lock
	hostname, _ := os.Hostname()
	info := LockInfo{
		PID:       os.Getpid(),
		Hostname:  hostname,
		StartedAt: time.Now(),
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal lock info: %w", err)
	}

	if err := os.WriteFile(l.path, data, 0644); err != nil {
		return fmt.Errorf("failed to create lock: %w", err)
	}

	return nil
}

// Release removes the lock
func (l *LockFile) Release() {
	os.Remove(l.path)
}

// readLock reads existing lock info
func (l *LockFile) readLock() (*LockInfo, error) {
	data, err := os.ReadFile(l.path)
	if err != nil {
		return nil, err
	}

	var info LockInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// IsLocked checks if a lock exists and is not stale
func (l *LockFile) IsLocked() bool {
	info, err := l.readLock()
	if err != nil {
		return false
	}
	return time.Since(info.StartedAt) < StaleLockDuration
}
