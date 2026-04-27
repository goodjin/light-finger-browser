package proxy

import (
	"context"
)

// ProxyProvider defines the interface for proxy service providers
type ProxyProvider interface {
	// GetProxy retrieves a new proxy from the provider
	GetProxy(ctx context.Context, country string, proxyType ProxyType) (*Proxy, error)

	// ReleaseProxy releases a proxy back to the provider
	ReleaseProxy(ctx context.Context, id string) error

	// CheckProxy tests if a proxy is still working
	// Returns: success, latency (ms), error
	CheckProxy(ctx context.Context, p *Proxy) (bool, int, error)
}