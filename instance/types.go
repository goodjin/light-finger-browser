package instance

import (
	"time"

	"github.com/tmos/fingerbrower/fingerprint"
)

// InstanceStatus represents the current state of a browser instance.
type InstanceStatus string

const (
	StatusPending   InstanceStatus = "pending"
	StatusStarting  InstanceStatus = "starting"
	StatusRunning   InstanceStatus = "running"
	StatusStopping  InstanceStatus = "stopping"
	StatusStopped   InstanceStatus = "stopped"
	StatusError     InstanceStatus = "error"
)

// BrowserInstance represents a managed browser instance.
type BrowserInstance struct {
	ID           string            `json:"id"`
	Status       InstanceStatus    `json:"status"`
	Fingerprint  *fingerprint.Fingerprint `json:"fingerprint"`
	ProxyID      string            `json:"proxy_id"`
	AccountID    string            `json:"account_id"`
	CDPEndpoint string            `json:"cdp_endpoint"`
	PID          int               `json:"pid"`
	Port         int               `json:"port"`
	UserDataDir  string            `json:"user_data_dir"`
	Group        string            `json:"group"`
	StartedAt    time.Time         `json:"started_at"`
	LastActiveAt time.Time         `json:"last_active_at"`
	CreatedAt    time.Time         `json:"created_at"`
}

// InstanceConfig contains configuration for creating a new instance.
type InstanceConfig struct {
	Fingerprint *fingerprint.Fingerprint `json:"fingerprint"`
	Proxy       *ProxyConfig             `json:"proxy"`
	AccountID   string                   `json:"account_id"`
	Group       string                   `json:"group"`
	Headless    bool                     `json:"headless"`
}

// ProxyConfig contains proxy configuration for an instance.
type ProxyConfig struct {
	ID   string `json:"id"`
	URL  string `json:"url"`
	Type string `json:"type"`
}

// InstanceFilter contains filter criteria for listing instances.
type InstanceFilter struct {
	Status   *InstanceStatus `json:"status"`
	Group    string          `json:"group"`
	ProxyID  string          `json:"proxy_id"`
	AccountID string         `json:"account_id"`
}

// MaxInstancesPerServer is the maximum number of instances per server.
var MaxInstancesPerServer = 200

// Errors for the instance module.
var (
	ErrNoAvailablePort      = &InstanceError{Message: "no available port"}
	ErrInstanceNotFound     = &InstanceError{Message: "instance not found"}
	ErrInstanceLimitReached = &InstanceError{Message: "instance limit reached"}
	ErrInstanceNotRunning   = &InstanceError{Message: "instance not running"}
	ErrPortAlreadyAllocated = &InstanceError{Message: "port already allocated"}
)

// InstanceError represents an instance-related error.
type InstanceError struct {
	Message string
}

func (e *InstanceError) Error() string {
	return e.Message
}

// StatusPtr returns a pointer to the given status.
func StatusPtr(s InstanceStatus) *InstanceStatus {
	return &s
}