package adapter

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/tmos/facebook/internal/proxy"
)

// OxylabsAdapter implements ProxyProvider for Oxylabs service
type OxylabsAdapter struct {
	username  string
	password  string
	httpClient *http.Client
}

// NewOxylabsAdapter creates a new Oxylabs adapter
func NewOxylabsAdapter(username, password string) *OxylabsAdapter {
	return &OxylabsAdapter{
		username: username,
		password: password,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetProxy retrieves a proxy from Oxylabs
func (a *OxylabsAdapter) GetProxy(ctx context.Context, country string, proxyType proxy.ProxyType) (*proxy.Proxy, error) {
	// Oxylabs API response parsing
	// In production, this would call their API endpoint
	resp := &OxylabsResponse{
		IP:       fmt.Sprintf("pr.oxylabs.io"),
		Port:     7777,
		Username: a.username,
		Password: a.password,
	}

	proxyID := fmt.Sprintf("ox-%d", time.Now().UnixNano())

	return &proxy.Proxy{
		ID:       proxyID,
		IP:       resp.IP,
		Port:     resp.Port,
		Country:  country,
		Type:     proxyType,
		Username: resp.Username,
		Password: resp.Password,
		Status:   proxy.ProxyStatusAvailable,
		Provider: "oxylabs",
	}, nil
}

// ReleaseProxy releases a proxy back to Oxylabs
func (a *OxylabsAdapter) ReleaseProxy(ctx context.Context, id string) error {
	// Oxylabs releases automatically when session ends
	return nil
}

// CheckProxy tests if a proxy is working
func (a *OxylabsAdapter) CheckProxy(ctx context.Context, p *proxy.Proxy) (bool, int, error) {
	start := time.Now()

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	proxyURL := fmt.Sprintf("http://%s:%s@%s:%d", p.Username, p.Password, p.IP, p.Port)
	transport := &http.Transport{
		Proxy: func(*http.Request) (*url.URL, error) {
			return url.Parse(proxyURL)
		},
	}

	client.Transport = transport

	req, err := http.NewRequestWithContext(ctx, "GET", "http://httpbin.org/ip", nil)
	if err != nil {
		return false, 0, err
	}

	resp, err := client.Do(req)
	if err != nil {
		return false, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, 0, errors.New("proxy check failed")
	}

	latency := int(time.Since(start).Milliseconds())
	return true, latency, nil
}

// OxylabsResponse represents Oxylabs API response
type OxylabsResponse struct {
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
}