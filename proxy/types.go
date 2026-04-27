package proxy

import (
	"time"
)

// ProxyType represents the type of proxy
type ProxyType string

const (
	ProxyTypeResidential ProxyType = "residential"
	ProxyTypeDatacenter  ProxyType = "datacenter"
)

// ProxyStatus represents the current status of a proxy
type ProxyStatus string

const (
	ProxyStatusAvailable ProxyStatus = "available"
	ProxyStatusInUse    ProxyStatus = "in_use"
	ProxyStatusChecking  ProxyStatus = "checking"
	ProxyStatusDead      ProxyStatus = "dead"
)

// Proxy represents a proxy instance
type Proxy struct {
	ID          string      `json:"id"`
	IP          string      `json:"ip"`
	Port        int         `json:"port"`
	Country     string      `json:"country"`
	City        string      `json:"city"`
	Type        ProxyType   `json:"type"`
	Username    string      `json:"username"`
	Password    string      `json:"password"`
	Status      ProxyStatus `json:"status"`
	BindID      string      `json:"bind_id"`
	BoundAt     time.Time   `json:"bound_at"`
	LastCheckAt time.Time   `json:"last_check_at"`
	SuccessRate float64     `json:"success_rate"`
	Latency     int         `json:"latency"`
	Provider    string      `json:"provider"`
	CreatedAt   time.Time   `json:"created_at"`
}

// ProxyFilter represents filter criteria for listing proxies
type ProxyFilter struct {
	ID       *string
	Country  *string
	Type     *ProxyType
	Status   *ProxyStatus
	BindID   *string
	Provider *string
}

// ProxyStatusPtr returns a pointer to the given ProxyStatus
func ProxyStatusPtr(s ProxyStatus) *ProxyStatus {
	return &s
}

// ProxyTypePtr returns a pointer to the given ProxyType
func ProxyTypePtr(t ProxyType) *ProxyType {
	return &t
}