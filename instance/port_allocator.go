package instance

import (
	"errors"
	"sync"
)

// PortAllocator manages port allocation for browser instances.
type PortAllocator struct {
	mu       sync.Mutex
	used     map[int]bool
	basePort int
	maxPort  int
}

// NewPortAllocator creates a new PortAllocator with the specified port range.
func NewPortAllocator(basePort, maxPort int) *PortAllocator {
	return &PortAllocator{
		used:     make(map[int]bool),
		basePort: basePort,
		maxPort:  maxPort,
	}
}

// Allocate assigns a new port from the available pool.
// Returns the port number or ErrNoAvailablePort if no ports are available.
func (p *PortAllocator) Allocate() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for port := p.basePort; port <= p.maxPort; port++ {
		if !p.used[port] {
			p.used[port] = true
			return port, nil
		}
	}
	return 0, ErrNoAvailablePort
}

// Release returns a port to the available pool.
func (p *PortAllocator) Release(port int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	delete(p.used, port)
}

// IsAllocated checks if a port is currently allocated.
func (p *PortAllocator) IsAllocated(port int) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.used[port]
}

// AvailableCount returns the number of available ports.
func (p *PortAllocator) AvailableCount() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.maxPort - p.basePort + 1 - len(p.used)
}

// Reset clears all port allocations.
func (p *PortAllocator) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.used = make(map[int]bool)
}

// ValidatePortRange checks if the port is within the valid range.
func ValidatePortRange(port, basePort, maxPort int) error {
	if port < basePort || port > maxPort {
		return errors.New("port outside valid range")
	}
	return nil
}